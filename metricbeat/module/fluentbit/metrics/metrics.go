package metrics

import (
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/pkg/errors"
)

const (
	defaultScheme = "http"

	defaultPath = "/api/v1/metrics"
)

var (
	debugf = logp.MakeDebug("Fluentbit-metrics")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "path",
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("fluentbit", "metrics", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The fluentbit metrics metricset is beta.")

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
	json, err := m.http.FetchJSON()
	if err != nil {
		return errors.Wrap(err, "error in http fetch")
	}

	reporter.Event(mb.Event{
		MetricSetFields: json,
	})

	return nil
}
