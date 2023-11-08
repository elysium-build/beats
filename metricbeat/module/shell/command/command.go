package command

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("shell", "command", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	CMD     string
	Format  string
	Timeout time.Duration
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The shell command metricset is beta.")

	config := struct {
		CMD     string        `config:"command"`
		Format  string        `config:"format"`
		Timeout time.Duration `config:"timeout"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		CMD:           config.CMD,
		Format:        config.Format,
		Timeout:       config.Timeout,
	}, nil
}

func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	execution := ExecutionRequest{
		Command: m.CMD,
		Env:     nil,
		Timeout: m.Timeout,
	}

	cmdExec, err := execution.Execute(context.Background(), execution)
	if err != nil {
		return err
	}
	outputStr := cmdExec.Output
	exitStatus := cmdExec.Status

	m.Logger().Debugf("outputStr=%s", outputStr)
	m.Logger().Debugf("exitStatus=%d", exitStatus)

	if exitStatus == 2 {
		return errors.New("Execution timed out")
	}

	if exitStatus == 3 {
		return errors.New("Execution fallback")
	}

	if m.Format == "json" {
		var data map[string]interface{}

		err = json.Unmarshal([]byte(outputStr), &data)
		if err != nil {
			return err
		}
		report.Event(mb.Event{
			MetricSetFields: data,
		})
	} else {
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{"message": outputStr},
		})
	}

	return nil
}
