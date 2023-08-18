package mbeaninfo

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"

	defaultPath = "/mbeanclient/MBeanInfo"
)

var (
	debugf = logp.MakeDebug("SM-mbeaninfo")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "path",
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("mfsm", "mbeaninfo", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The mfsm mbeaninfo metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
	}, nil
}

func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := m.http.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error fetching status")
	}
	
	data, _ := eventMapping(&content, m.Host())

	if reported := reporter.Event(mb.Event{MetricSetFields: data}); !reported {
		m.Logger().Error("error reporting event")
	}

	return nil
}
