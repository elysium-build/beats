package shell

import (
	"sync"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/common"
)

const (
	monitorName = "shell"
	aliasName   = "epicon/shell"
)

type Client interface {
	Connect() error
	Reconnect() error
	Close()
	Run(dir, command string, args ...string) (string, error)
}

// Upload Implement this interface if the client needs to upload file
type UploadClient interface {
	UploadFile(sourceFile, destfolder string, mode string) error
}

// var debugf = logp.MakeDebug(monitorName)

func init() {
	plugin.Register(monitorName, create, aliasName)

}

func create(
	name string,
	cfg *common.Config,
) (p plugin.Plugin, err error) {

	// unpack the monitors configuration
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {

		return plugin.Plugin{}, err
	}

	validator := makeValidator(&config)

	js := make([]jobs.Job, 0)

	for _, host := range config.Hosts {

		shellmonitorjob := &ShellMonitorJob{
			clientLocker: &sync.Once{},
		}
		newjob, err := shellmonitorjob.newShellMonitorJob(host, &config, validator)

		if err != nil {
			return plugin.Plugin{}, err
		}
		js = append(js, newjob)
	}

	return plugin.Plugin{Jobs: js, Endpoints: len(config.Hosts)}, nil

}
