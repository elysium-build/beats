package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/heartbeat/monitors/active/shell/util"
)

/*
*

	Add a eofcheck command to check whether the command is completed .
	The length from header is not accurate if the command takes a while to run

*
*/
func (d *DockerClient) readerToString(reader *bufio.Reader, eofcheck string) (string, error) {
	header := make([]byte, 8) // [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}[]byte{OUTPUT}
	var retByte bytes.Buffer
	_, err := reader.Read(header)
	if err != nil {
		return "", err
	}
	// retType := int(header[0])                          // 1 --stdout , 2 -- stderr
	// length := int(binary.BigEndian.Uint32(header[4:])) // It's inaccurate if the streaming takes a bit longer
	// fmt.Println((fmt.Sprintf("Estimate: %v", length)))
	// if retType == 2 {
	// 	output := make([]byte, length)
	// 	readLen, _ := reader.Read(output)
	// 	return (string(output[0:readLen])), errors.New(string(output[0:readLen]))
	// }

	for line, _, err := reader.ReadLine(); !strings.Contains(string(line), eofcheck) && err == nil; {
		retByte.Write(line)
		line, _, err = reader.ReadLine()
	}
	// fmt.Println("end")

	if err != nil {
		return "", err
	}

	return string(retByte.Bytes()), nil
}

func (d *DockerClient) Run(dir, command string, args ...string) (string, error) {
	d.commandMutex.Lock()
	defer d.commandMutex.Unlock()

	err := d.Connect()
	if err != nil {
		// logp.Err("Failed to connect for %v", d.Filter)
		// return "", err
		logp.Info("Reconnect for %v", d.Filter)
		err = d.Reconnect() // always Reconnect if it's failed in first connect
		if err != nil {
			logp.Err("Reconnect failed for %v ,reason is %v", d.Filter, err.Error())
			return "", err
		}
	}
	hijacked := d.hijackedResponse
	if hijacked == nil {
		err = fmt.Errorf("connection is closed  for %v ", d.name)
		d.execErr = err
		return "", err
	}
	if hijacked.Conn == nil || hijacked.Reader == nil {
		err = fmt.Errorf("connection is closed  for %v ", d.name)
		d.execErr = err
		return "", err
	}
	eofCheck := fmt.Sprintf("%v%v", "epicon-eof", time.Now().UnixNano())
	eofCommand := fmt.Sprintf(`
			echo "%v" 
	`, eofCheck)

	logp.Debug("Run command %v for %v", util.BuildCmd(dir, command, args...)+eofCommand, d.name)
	_, err = hijacked.Conn.Write([]byte(util.BuildCmd(dir, command, args...) + eofCommand))
	if err != nil {
		d.execErr = err
		return "", err
	}
	hijacked.Conn.SetDeadline(time.Now().Add(d.Timeout))
	output, err := d.readerToString(hijacked.Reader, eofCheck)
	d.execErr = err
	return strings.Trim(string(output), "\n"), err
}
