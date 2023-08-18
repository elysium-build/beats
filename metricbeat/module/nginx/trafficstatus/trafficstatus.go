package trafficstatus

import (
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/pkg/errors"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/status/format/json"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "server_status_path",
		DefaultPath:   defaultPath,
	}.Build()
)

var logger = logp.NewLogger("nginx.trafficstatus")

func init() {
	mb.Registry.MustAddMetricSet("nginx", "trafficstatus", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)

}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
	// upstreamzones []string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The nginx trafficstatus metricset is beta.")

	config := struct {
		// UpstreamZones []string `config:"upstreamzones"`
	}{}
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
		// upstreamzones: config.UpstreamZones,
	}, nil
}

func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	content, err := m.http.FetchJSON()
	if err != nil {
		return errors.Wrap(err, "error fetching status")
	}

	events, err := eventMapping(content, m)
	if err != nil {
		return errors.Wrap(err, "error fetching status")
	}

	for _, event := range events {
		report.Event(mb.Event{MetricSetFields: event})
	}

	return nil
}
