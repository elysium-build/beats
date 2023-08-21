package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/active/shell/util"
	"github.com/elastic/beats/v7/libbeat/logp"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	sshclient  *ssh.Client
	sshError   error
	initClient *sync.Once

	Addr     string
	Username string
	Password string
	Key      string
	Sudo     bool
	Timeout  time.Duration
}

type TimeoutConn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *TimeoutConn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *TimeoutConn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func NewSSHClient() *SSHClient {
	return &SSHClient{
		initClient: &sync.Once{},
	}
}

func (c *SSHClient) buildSSHConfig() (*ssh.ClientConfig, error) {
	sshConfig := &ssh.ClientConfig{
		User:            c.Username,
		Timeout:         c.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if c.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(c.Password)}
	}
	if c.Key != "" {
		var data []byte
		var err error
		if strings.Index(c.Key, "@") == 0 {
			data, err = ioutil.ReadFile(string(c.Key[1:]))
			if err != nil {
				return nil, err
			}
		} else {
			data = []byte(c.Key)
		}

		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, err
		}

		sshConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}
	return sshConfig, nil
}

func (c *SSHClient) Reconnect() error {
	c.Close()
	logp.Info("Reconnect for %v ", c.Addr)

	return c.Connect()
}

func (c *SSHClient) Connect() error {

	c.initClient.Do(func() {

		sshConfig, err := c.buildSSHConfig()
		if err != nil {
			c.wrapError(err)
			return
		}
		conn, err := net.DialTimeout("tcp", c.Addr, c.Timeout)
		if err != nil {
			c.wrapError(err)
			return
		}
		TimeoutConn := &TimeoutConn{conn, c.Timeout, c.Timeout}
		cli, chans, reqs, err := ssh.NewClientConn(TimeoutConn, c.Addr, sshConfig)
		if err != nil {
			c.wrapError(err)
			return
		}
		client := ssh.NewClient(cli, chans, reqs)
		c.sshclient = client
		c.wrapError(nil)
		go func() {
			t := time.NewTicker(3 * time.Second)
			defer t.Stop()
			for {
				<-t.C
				_, _, err := client.Conn.SendRequest("keepalive@epicon.com", true, nil)
				if err != nil {
					return
				}
			}
		}()
	})
	return c.sshError
}

func (c *SSHClient) Close() {
	logp.Info("Close the connection for %v", c.Addr)
	if c.sshclient != nil {
		c.sshclient.Close()
		c.initClient = &sync.Once{}
		c.sshError = nil
	}
}

func (c *SSHClient) wrapError(err error) error {
	c.sshError = err
	return err
}

func pty(session *ssh.Session) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	return session.RequestPty("xterm", 80, 40, modes)
}

func (c *SSHClient) Run(dir, command string, args ...string) (string, error) {
	err := c.Connect()
	if err != nil {
		err = c.Reconnect() // always Reconnect if it's failed in first connect
		if err != nil {
			c.Close()
			return "", err
		}
	}
	// start := time.Now()
	session, err := c.sshclient.NewSession()
	defer session.Close()
	if err != nil {
		return "", c.wrapError(err)
	}

	if c.Sudo {
		err = pty(session)
		if err != nil {
			return "", c.wrapError(err)
		}
	}

	var stdoutB bytes.Buffer
	session.Stdout = &stdoutB
	in, _ := session.StdinPipe()

	quitSudo := make(chan bool, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func(in io.Writer, output *bytes.Buffer, quit chan bool) {
		for {
			if output != nil && output.Bytes() != nil {
				if strings.Contains(string(output.Bytes()), "[sudo] password for ") {
					_, err = in.Write([]byte(c.Password + "\n"))
					c.wrapError(err)
					wg.Done()
					return
				}
			}
			select {
			case <-quit:
				wg.Done()
				return
			default:
			}
		}
	}(in, &stdoutB, quitSudo)

	if c.Sudo {
		if !strings.HasPrefix(strings.Trim(command, " "), "sudo") {
			command = fmt.Sprintf("sudo sh -c '%v' ", command)
		}
	}
	fullcommand := util.BuildCmd(dir, command, args...)
	err = session.Run(fullcommand)

	quitSudo <- true
	wg.Wait()

	if err != nil {
		switch err.(type) {
		default:
			return strings.Trim(string(stdoutB.Bytes()), "\n"), c.wrapError(fmt.Errorf(err.Error()))
		case *ssh.ExitMissingError:
			return "", c.wrapError(fmt.Errorf("Connection is disconnected by the timeout or lost"))
		case *ssh.ExitError:
			return strings.Trim(fmt.Sprintf("%v. %v ", string(stdoutB.Bytes()), err.Error()), "\n"), nil
		}
	}
	// fmt.Println(time.Since(start))
	err = nil
	c.wrapError(err)

	return strings.Trim(string(stdoutB.Bytes()), "\n"), err

}

func (c *SSHClient) UploadFile(source, dest string, mode string) error {
	const (
		scpOK = "\x00"
	)
	sourceBytes, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}
	sourceReader := bytes.NewReader(sourceBytes)
	size := len(sourceBytes)
	oldTimeout := c.Timeout
	c.Timeout = 120 * time.Second // temprorary increase the timeout to 120 second
	defer func() {
		c.Timeout = oldTimeout
	}()
	fileName := filepath.Base(source)

	err = c.Connect()

	if err != nil {
		return err
	}
	session, err := c.sshclient.NewSession()
	defer session.Close()
	// defer c.Close()
	if err != nil {
		return err
	}
	pi, err := session.StdinPipe()
	if err != nil {
		return err
	}
	po, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	destDir, destName := filepath.Split(dest)
	if destName == "" {
		destName = fileName
	}
	go func() {
		session.Run("scp -t " + destDir)

	}()

	perms := fmt.Sprintf("C%v", mode)
	fmt.Fprintln(pi, perms, size, destName)
	if _, err := io.CopyN(pi, sourceReader, int64(size)); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(pi, scpOK); err != nil {
		return err
	}
	return checkSCPStatus(bufio.NewReader(po))
}

func checkSCPStatus(r *bufio.Reader) error {
	code, err := r.ReadByte()
	code, err = r.ReadByte() // 2nd byte is for the return code
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	fmt.Println(code)
	if code != 0 {
		message, _, err := r.ReadLine()
		if err != nil {
			return fmt.Errorf("Error reading error message: %s", err)
		}

		return errors.New(string(message))
	}

	return nil
}
