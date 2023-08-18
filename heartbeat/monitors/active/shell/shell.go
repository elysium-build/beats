package shell

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	plugin.Register("shell", create, "synthetics/shell")
}

var debugf = logp.MakeDebug("icmp")

func create(
	name string,
	cfg *conf.C,
) (p plugin.Plugin, err error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return plugin.Plugin{}, err
	}

	validator := makeValidator(&config)

	js := make([]jobs.Job, len(config.Commands))

	for i, command := range config.Commands {
		js[i], err = newShellMonitorJob(command, &config, validator)
		if err != nil {
			return plugin.Plugin{}, err
		}
	}
	return plugin.Plugin{Jobs: js, Endpoints: len(config.Commands)}, nil
}

func newShellMonitorJob(
	command string,
	config *Config,
	validator OutputCheck,
) (jobs.Job, error) {
	okstr := ""
	for _, ok := range config.Check.Response.Ok {
		okstr = okstr + ok.String() + ","
	}

	criticalStr := ""
	for _, critical := range config.Check.Response.Critical {
		criticalStr = criticalStr + critical.String() + " ,"
	}

	eventFields := mapstr.M{
		"monitor": mapstr.M{
			"scheme":  "shell",
			"command": command,
		},
		"check": mapstr.M{
			"ok":       okstr,
			"critical": criticalStr,
		},
	}

	customs := config.CustomeFields
	if len(customs) != 0 {
		customMap := mapstr.M{}
		for _, v := range customs {
			splitPos := strings.Index(v, ":")
			if splitPos > 0 && splitPos != len(v)-1 {
				customMap[string(v[0:splitPos])] = string(v[splitPos+1:])
			}
		}
		if len(customMap) > 0 {
			eventFields["custom"] = customMap
		}
	}

	return wrappers.WithFields(eventFields,
		jobs.MakeSimpleJob(func(event *beat.Event) error {
			err := execute(command, event, config.Timeout, validator)
			return err
		})), nil
}

func execute(command string, event *beat.Event, duration time.Duration, validate func(string) error) (errReason reason.Reason) {
	execution := ExecutionRequest{
		Command: command,
		Env:     nil,
		Timeout: duration,
	}

	cmdExec, err := execution.Execute(context.Background(), execution)
	outputStr := cmdExec.Output
	exitStatus := cmdExec.Status

	// ctx, cancel := context.WithTimeout(context.Background(), duration)
	// defer cancel()

	// cmd := exec.CommandContext(ctx, command)
	// outputByte, err := cmd.Output()
	// outputStr := string(outputByte)

	eventext.MergeEventFields(event, mapstr.M{"shell": mapstr.M{
		"response": mapstr.M{
			"output": outputStr,
		},
	}})

	if err != nil {
		return reason.IOFailed(err)
	}

	if exitStatus == 2 {
		ErrTimeOut := errors.New("Execution timed out")
		return reason.IOFailed(ErrTimeOut)
	}

	if exitStatus == 3 {
		ErrFallback := errors.New("Execution fallback")
		return reason.IOFailed(ErrFallback)
	}

	err = validate(outputStr)

	return reason.ValidateFailed(err)
}
