package trafficstatus

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/status/format/json"
)

var hostParser = parse.URLHostParserBuilder{
	DefaultScheme: defaultScheme,
	PathConfigKey: "server_status_path",
	DefaultPath:   defaultPath,
}.Build()

var logger = logp.NewLogger("nginx.trafficstatus")

func init() {
	mb.Registry.MustAddMetricSet("nginx", "trafficstatus", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http                *helper.HTTP
	previousNumRequests int // Total number of requests as returned in the previous fetch.
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
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
	content, err := m.http.FetchJSON()
	if err != nil {
		return fmt.Errorf("error fetching status: %w", err)
	}

	events, err := eventMapping(content)
	if err != nil {
		return fmt.Errorf("error fetching status")
	}

	for _, event := range events {
		reporter.Event(mb.Event{MetricSetFields: event})
	}

	return nil
}
