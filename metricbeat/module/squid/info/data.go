package info

import (
	"reflect"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/squid"
	"github.com/pkg/errors"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"version": c.Str("Version"),

		"connection": s.Object{
			"clients_accessing_cache":       c.Int("NClientAccess"),
			"http_requests_received":        c.Int("NHTTPRecv"),
			"icp_messages_received":         c.Int("NICPRecv"),
			"icp_messages_sent":             c.Int("NICPSent"),
			"queued_icp_replies":            c.Int("NQueuedICP"),
			"htcp_messages_received":        c.Int("NHTCPRecv"),
			"htcp_messages_sent":            c.Int("NHTCPSent"),
			"request_failure_ratio":         c.Float("ReqFailRatio"),
			"average_http_requests_per_min": c.Float("AvHTTPperMin"),
			"average_icp_messages_per_min ": c.Float("AvICPperMin"),
			"select_loop_called":            c.Str("LoopCalled"),
		},

		"service": s.Object{
			"http_Requests":        c.Float("HTTPReqTime"),
			"cache_misses":         c.Float("CacheMisses"),
			"cache_hits":           c.Float("CacheHits"),
			"near_hits":            c.Float("NearHits"),
			"not_modified_replies": c.Float("NotModifiedReplies"),
			"dns_lookups":          c.Float("DNSLookup"),
			"icp_queries":          c.Float("ICPQueries"),
		},

		"resource": s.Object{
			"up_ime":            c.Float("Uptime"),
			"cpu_ime":           c.Float("CPUTime"),
			"cpu_usage":         c.Float("CPUUsage"),
			"max_resident_size": c.Int("MaxResident"),
			"page_fault":        c.Int("PageFaults"),
		},

		"memory": s.Object{
			"total":              c.Int("MemTotal"),
			"mempoolalloc_calls": c.Int("MemAllocCall"),
			"mempoolfree_calls":  c.Int("MemFreeCall"),
		},

		"file": s.Object{
			"max_file_desc":            c.Int("MaxFD"),
			"largest_file_desc_in_use": c.Int("LargestFDUsed"),
			"file_desc_in_use":         c.Int("NFDUsed"),
			"file_queued_for_open":     c.Int("FDQueued"),
			"available_file_desc":      c.Int("AvailableFD"),
			"reserved_file_desc":       c.Int("ReservedFD"),
			"store_disk_files_open":    c.Int("SDiskFiles"),
		},
	}
)

func eventMapping(info *squid.Info, r mb.ReporterV2) (mb.Event, error) {

	st := reflect.ValueOf(info).Elem()
	typeOfT := st.Type()
	source := map[string]interface{}{}

	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		source[typeOfT.Field(i).Name] = f.Interface()
	}

	event := mb.Event{
		RootFields: common.MapStr{},
	}
	fields, err := schema.Apply(source)
	if err != nil {
		return event, errors.Wrap(err, "error applying schema")
	}

	event.MetricSetFields = fields

	return event, nil
}
