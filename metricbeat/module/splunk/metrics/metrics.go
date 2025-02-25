package metrics

import (
	"encoding/json"
	"encoding/xml"
	"net/url"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"

	defaultPath = "/services/server/introspection/kvstore/serverstatus"
)

var (
	debugf = logp.MakeDebug("Splunk-metrics")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "path",
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("splunk", "metrics", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The splunk metrics metricset is beta.")

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




func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	content, err := fetchPath(m.http, m.HostData().SanitizedURI, "/services/server/introspection/kvstore/serverstatus", "")
	if err != nil {
		return err
	}

	type Key struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",innerxml"` // Capture inner XML, including CDATA
	}

	type Feed struct {
		XMLName   xml.Name  `xml:"feed"`
		Keys      []Key     `xml:"entry>content>dict>key"`
	}


	var feed Feed
	if err := xml.Unmarshal(content, &feed); err != nil {
		return err
	}




	var cdata string

	for _, key := range feed.Keys {
		if key.Name == "data" {
			cdata = extractCDATA(key.Value)
		}
	}

	type Async struct {
		CurrentWorkQueueLength int `json:"current work queue length"`
		MaximumWorkQueueLength int `json:"maximum work queue length"`
	}

	type Capacity struct {
		TimeWaitingDueToTotalCapacity int `json:"time waiting due to total capacity (usecs)"`
		TimeWaitingDuringCheckpoint int `json:"time waiting during checkpoint (usecs)"`
	}

	type Perf struct {
		FileSystemReadLatencyHistogram int `json:"file system read latency histogram (bucket 1) - 10-49ms"`
		FileSystemWriteLatencyHistogram int `json:"file system write latency histogram (bucket 1) - 10-49ms"`
		OperationReadLatencyHistogram int `json:"operation read latency histogram (bucket 1) - 100-249us"`
		OperationWriteLatencyHistogram int `json:"operation write latency histogram (bucket 1) - 100-249us"`

	}

	type WiredTiger struct {
		Async Async `json:"async"`
		Capacity Capacity `json:"capacity"`
		Perf Perf `json:"perf"`
	}

	type Mem struct {
		Bits int `json:"bits"`
		Resident int `json:"resident"`
		Virtual int `json:"virtual"`
		Supported bool `json:"supported"`

	}

	type Connections struct {
		Current int `json:"current"`
		Available int `json:"available"`
		TotalCreated int `json:"totalCreated"`
		Active int `json:"active"`

	}

	type ServerStats struct {
		Connections Connections `json:"connections"`
		Mem Mem `json:"mem"`
		WiredTiger WiredTiger `json:"wiredTiger"`
	}

	var serverStatus ServerStats
	if err := json.Unmarshal([]byte(cdata), &serverStatus); err != nil {
		return err
	}

	content, err = fetchPath(m.http, m.HostData().SanitizedURI, "/services/server/status/resource-usage/hostwide", "")
	if err != nil {
		return err
	}

	if err := xml.Unmarshal(content, &feed); err != nil {
		return err
	}

	var cpu_count int
	var cpu_idle_pct float64
	var cpu_system_pct float64
	var cpu_user_pct float64
	var mem_used float64
	var swap float64
	var swap_used float64

	for _, key := range feed.Keys {
		if key.Name == "cpu_count" {
			cpu_count, _ = strconv.Atoi(key.Value)
		}else if key.Name == "cpu_idle_pct" {
			cpu_idle_pct, _ = strconv.ParseFloat(key.Value,64)
		}else if key.Name == "cpu_system_pct" {
			cpu_system_pct, _ = strconv.ParseFloat(key.Value,64)
		}else if key.Name == "cpu_user_pct" {
			cpu_user_pct, _ = strconv.ParseFloat(key.Value,64)
		}else if key.Name == "mem_used" {
			mem_used, _ = strconv.ParseFloat(key.Value,64)
		}else if key.Name == "swap" {
			swap, _ = strconv.ParseFloat(key.Value,64)
		}else if key.Name == "swap_used" {
			swap_used, _ = strconv.ParseFloat(key.Value,64)
		}
	}


	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"connections":              serverStatus.Connections,
			"mem":                      serverStatus.Mem,
			"async":                    common.MapStr{
																		"current_work_queue_length": serverStatus.WiredTiger.Async.CurrentWorkQueueLength,
																		"maximum_work_queue_length": serverStatus.WiredTiger.Async.MaximumWorkQueueLength,
																	},
			"capacity":                 common.MapStr{
																		"time_waiting_dueto_total_capacity": serverStatus.WiredTiger.Capacity.TimeWaitingDueToTotalCapacity,
																		"time_waiting_during_checkpoint": serverStatus.WiredTiger.Capacity.TimeWaitingDuringCheckpoint,
																	},
			"perf":                     common.MapStr{
																		"filesystem_read_latency_histogram": serverStatus.WiredTiger.Perf.FileSystemReadLatencyHistogram,
																		"filesystem_write_latency_histogram": serverStatus.WiredTiger.Perf.FileSystemWriteLatencyHistogram,
																		"operation_read_latency_histogram": serverStatus.WiredTiger.Perf.OperationReadLatencyHistogram,
																		"operation_write_latency_histogram": serverStatus.WiredTiger.Perf.OperationWriteLatencyHistogram,
																	},
			"host":                     common.MapStr{
																		"cpu_count": cpu_count,
																		"cpu_idle_pct": cpu_idle_pct,
																		"cpu_system_pct": cpu_system_pct,
																		"cpu_user_pct": cpu_user_pct,
																		"mem_used": mem_used,
																		"swap": swap,
																		"swap_used": swap_used,
																	},





		},
	})

	return nil
}

func fetchPath(http *helper.HTTP, uri, path string, query string) ([]byte, error) {
	defer http.SetURI(uri)

	// Parses the uri to replace the path
	u, _ := url.Parse(uri)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}


func extractCDATA(input string) string {
	start := strings.Index(input, "<![CDATA[")
	end := strings.LastIndex(input, "]]>")
	if start != -1 && end != -1 {
		return input[start+9 : end] // Extract inside CDATA
	}
	return input // Return as-is if no CDATA found
}