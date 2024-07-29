package shell

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/heartbeat/eventext"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"

	"github.com/elastic/beats/v7/heartbeat/monitors/active/shell/docker"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/shell/local"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/shell/ssh"

	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func createClient(addr string, config *Config) (Client, error) {

	if config.Docker {
		docker := docker.NewDockerClient()
		docker.Endpoint = addr
		docker.Timeout = config.Timeout
		docker.Filter = config.Dockerfilter
		return docker, nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(host) == "localhost" {
		lclient := local.NewLocalClient()
		lclient.Timeout = config.Timeout
		return lclient, nil
	}
	sshClient := ssh.NewSSHClient()
	sshClient.Addr = addr
	sshClient.Username = config.Username
	sshClient.Password = config.Password
	sshClient.Timeout = config.Timeout
	sshClient.Sudo = config.Sudo
	sshClient.Key = config.Key
	return sshClient, nil
}

type ShellMonitorJob struct {
	clientLocker *sync.Once
}

func (s *ShellMonitorJob) newShellMonitorJob(
	addr string,
	config *Config,
	validator OutputCheck) (jobs.Job, error) {

	cli, err := createClient(addr, config)
	if err != nil {
		return nil, err
	}
	okstr := ""
	for _, ok := range config.Check.Response.Ok {
		okstr = okstr + ok.String() + ","
	}

	criticalStr := ""
	for _, critical := range config.Check.Response.Critical {
		criticalStr = criticalStr + critical.String() + " ,"
	}

	eventFields := common.MapStr{
		"monitor": common.MapStr{
			"scheme":       monitorName,
			"command":      config.Check.Request.Command,
			"args":         strings.Join(config.Check.Request.Args, " "),
			"dir":          config.Check.Request.Dir,
			"username":     config.Username,
			"docker":       config.Docker,
			"occurrence":   config.Occurrence,
			"dockerfilter": strings.Join(config.Dockerfilter, " "),
		},
		"check": common.MapStr{
			"ok":       okstr,
			"critical": criticalStr,
		},
	}

	logp.Info("Start newShellMonitorJob %v", config.Name)

	return jobs.MakeSimpleJob(func(beatEvent *beat.Event) error {
		_, _, err = s.runCommand(config, beatEvent, eventFields, cli, validator)
		return err
	}), nil
}

func (s *ShellMonitorJob) failedEvent(beatEvent *beat.Event, err error) (end time.Time, errReason reason.Reason) {
	eventext.MergeEventFields(beatEvent, makeOutput("", 1, err))
	s.clientLocker = &sync.Once{}
	end = time.Now()
	errReason = reason.ValidateFailed(err)
	return
}

func (s *ShellMonitorJob) runCommand(config *Config, beatEvent *beat.Event, eventFields common.MapStr, cli Client, validate func(string) error) (start, end time.Time, errReason reason.Reason) {
	logp.Info("Start to check %v", config.Name)
	defer logp.Info("Finish check %v", config.Name)
	failedOnUpload := false
	eventext.MergeEventFields(beatEvent, eventFields)
	downCount := 0
	start = time.Now()

	s.clientLocker.Do(func() {
		for _, uploadFile := range config.Upload {
			upload, ok := cli.(UploadClient)
			if ok {
				fromandto := strings.Split(uploadFile, ":")
				mode := "0755"
				if len(fromandto) == 3 {
					mode = fromandto[2]
				}
				logp.Info("upload file from %v to %v mode is %v", fromandto[0], fromandto[1], mode)
				if err := upload.UploadFile(strings.Trim(fromandto[0], " "), strings.Trim(fromandto[1], " "), mode); err != nil {

					failedOnUpload = true
					end, errReason = s.failedEvent(beatEvent, err)
					logp.Err("faild to upload the file for %v", uploadFile)
				}
			} else {
				failedOnUpload = true
				failedReason := fmt.Errorf("%v doesn't have the upload function", config.Name)
				end, errReason = s.failedEvent(beatEvent, failedReason)

				logp.Err("%v doesn't have the upload function", config.Name)
			}
		}
	})

	if failedOnUpload {
		return
	}

	if !config.LiveConnection {
		defer cli.Close()
	}

	var err error
	var output string
	for downCount < config.Occurrence {
		output, err = cli.Run(config.Check.Request.Dir, config.Check.Request.Command, config.Check.Request.Args...)
		if err == nil {
			if err = validate(output); err == nil {
				break
			}
		}
		downCount++
		time.Sleep(5 * time.Second)
	}

	end = time.Now()
	eventext.MergeEventFields(beatEvent, makeOutput(output, downCount, err))
	errReason = reason.ValidateFailed(err)

	return
}

func makeOutput(output string, downCount int, err error) common.MapStr {
	if err != nil {
		output = fmt.Sprintf("%v. Error:%v ", output, err.Error())
	}
	return common.MapStr{"shell": common.MapStr{
		"response": common.MapStr{
			"output":    output,
			"downcount": downCount,
		},
	}}
}
