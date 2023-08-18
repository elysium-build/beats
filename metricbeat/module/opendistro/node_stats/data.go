package node_stats

import (
	"encoding/json"
	"fmt"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/opendistro"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/joeshaw/multierror"
)

var schema = s.Schema{
	"name":              c.Str("name"),
	"transport_address": c.Str("transport_address"),
	"indices": c.Dict("indices", s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count": c.Int("count"),
		}),
		"store": c.Dict("store", s.Schema{
			"size_in_bytes": c.Int("size_in_bytes"),
		}),
		"indexing": c.Dict("indexing", s.Schema{
			"index_total":             c.Int("index_total"),
			"index_time_in_millis":    c.Int("index_time_in_millis"),
			"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
		}),
		"search": c.Dict("search", s.Schema{
			"query_total":          c.Int("query_total"),
			"query_time_in_millis": c.Int("query_time_in_millis"),
		}),
		"query_cache": c.Dict("query_cache", s.Schema{
			"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
			"hit_count":            c.Int("hit_count"),
			"miss_count":           c.Int("miss_count"),
			"evictions":            c.Int("evictions"),
		}),
		"fielddata": c.Dict("fielddata", s.Schema{
			"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
			"evictions":            c.Int("evictions"),
		}),
		"segments": c.Dict("segments", s.Schema{
			"count":                         c.Int("count"),
			"memory_in_bytes":               c.Int("memory_in_bytes"),
			"terms_memory_in_bytes":         c.Int("terms_memory_in_bytes"),
			"stored_fields_memory_in_bytes": c.Int("stored_fields_memory_in_bytes"),
			"term_vectors_memory_in_bytes":  c.Int("term_vectors_memory_in_bytes"),
			"norms_memory_in_bytes":         c.Int("norms_memory_in_bytes"),
			"points_memory_in_bytes":        c.Int("points_memory_in_bytes"),
			"doc_values_memory_in_bytes":    c.Int("doc_values_memory_in_bytes"),
			"index_writer_memory_in_bytes":  c.Int("index_writer_memory_in_bytes"),
			"version_map_memory_in_bytes":   c.Int("version_map_memory_in_bytes"),
			"fixed_bit_set_memory_in_bytes": c.Int("fixed_bit_set_memory_in_bytes"),
		}),
		"request_cache": c.Dict("request_cache", s.Schema{
			"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
			"evictions":            c.Int("evictions"),
			"hit_count":            c.Int("hit_count"),
			"miss_count":           c.Int("miss_count"),
		}),
	}),
	"os": c.Dict("os", s.Schema{
		"cpu": c.Dict("cpu", s.Schema{
			"percent": c.Int("percent"),
			"load_average": c.Dict("load_average", s.Schema{
				"1m":  c.Float("1m", s.Optional),
				"5m":  c.Float("5m", s.Optional),
				"15m": c.Float("15m", s.Optional),
			}, c.DictOptional), // No load average reported by ES on Windows
		}),
		"cgroup": c.Dict("cgroup", s.Schema{
			"cpuacct": c.Dict("cpuacct", s.Schema{
				"control_group": c.Str("control_group"),
				"usage_nanos":   c.Int("usage_nanos"),
			}),
			"cpu": c.Dict("cpu", s.Schema{
				"control_group":     c.Str("control_group"),
				"cfs_period_micros": c.Int("cfs_period_micros"),
				"cfs_quota_micros":  c.Int("cfs_quota_micros"),
				"stat": c.Dict("stat", s.Schema{
					"number_of_elapsed_periods": c.Int("number_of_elapsed_periods"),
					"number_of_times_throttled": c.Int("number_of_times_throttled"),
					"time_throttled_nanos":      c.Int("time_throttled_nanos"),
				}),
			}),
			"memory": c.Dict("memory", s.Schema{
				"control_group":  c.Str("control_group"),
				"limit_in_bytes": c.Str("limit_in_bytes"),
				"usage_in_bytes": c.Str("usage_in_bytes"),
			}),
		}, c.DictOptional),
	}),
	"process": c.Dict("process", s.Schema{
		"open_file_descriptors": c.Int("open_file_descriptors"),
		"max_file_descriptors":  c.Int("max_file_descriptors"),
		"cpu": c.Dict("cpu", s.Schema{
			"percent": c.Int("percent"),
		}),
	}),
	"jvm": c.Dict("jvm", s.Schema{
		"mem": c.Dict("mem", s.Schema{
			"heap_used_in_bytes": c.Int("heap_used_in_bytes"),
			"heap_used_percent":  c.Int("heap_used_percent"),
			"heap_max_in_bytes":  c.Int("heap_max_in_bytes"),
		}),
		"gc": c.Dict("gc", s.Schema{
			"collectors": c.Dict("collectors", s.Schema{
				"young": c.Dict("young", s.Schema{
					"collection_count":          c.Int("collection_count"),
					"collection_time_in_millis": c.Int("collection_time_in_millis"),
				}),
				"old": c.Dict("young", s.Schema{
					"collection_count":          c.Int("collection_count"),
					"collection_time_in_millis": c.Int("collection_time_in_millis"),
				}),
			}),
		}),
	}),
	"fs": c.Dict("fs", s.Schema{
		"total": c.Dict("total", s.Schema{
			"total_in_bytes":     c.Int("total_in_bytes"),
			"free_in_bytes":      c.Int("free_in_bytes"),
			"available_in_bytes": c.Int("available_in_bytes"),
		}),
		"io_stats": c.Dict("io_stats", s.Schema{
			"total": c.Dict("total", s.Schema{
				"operations":       c.Int("operations"),
				"read_kilobytes":   c.Int("read_kilobytes"),
				"read_operations":  c.Int("read_operations"),
				"write_kilobytes":  c.Int("write_kilobytes"),
				"write_operations": c.Int("write_operations"),
			}, c.DictOptional),
		}, c.DictOptional),
	}),
}

type nodesStruct struct {
	Nodes map[string]map[string]interface{} `json:"nodes"`
}

func eventsMapping(r mb.ReporterV2, m opendistro.MetricSetAPI, info opendistro.Info, content []byte) error {
	nodeData := &nodesStruct{}
	err := json.Unmarshal(content, nodeData)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Node Stats API response: %w", err)
	}

	masterNodeID, err := m.GetMasterNodeID()
	if err != nil {
		return err
	}

	var errs multierror.Errors
	for nodeID, node := range nodeData.Nodes {
		isMaster := nodeID == masterNodeID

		mlockall, err := m.IsMLockAllEnabled(nodeID)
		if err != nil {
			errs = append(errs, fmt.Errorf("error determining if mlockall is set on Elasticsearch node: %w", err))
			continue
		}

		event := mb.Event{}

		event.RootFields = mapstr.M{}
		_, err = event.RootFields.Put("service.name", opendistro.ModuleName)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to put field service.name: %w", err))
			continue
		}

		roles := node["roles"]

		event.ModuleFields = mapstr.M{
			"node": mapstr.M{
				"id":       nodeID,
				"mlockall": mlockall,
				"master":   isMaster,
				"roles":    roles,
			},
			"cluster": mapstr.M{
				"name": info.ClusterName,
				"id":   info.ClusterID,
			},
		}

		event.MetricSetFields, err = schema.Apply(node)
		if err != nil {
			errs = append(errs, fmt.Errorf("failure to apply node schema: %w", err))
			continue
		}

		name, err := event.MetricSetFields.GetValue("name")
		if err != nil {
			errs = append(errs, elastic.MakeErrorForMissingField("name", elastic.Elasticsearch))
			continue
		}

		nameStr, ok := name.(string)
		if !ok {
			errs = append(errs, fmt.Errorf("name is not a string"))
			continue
		}
		_, err = event.ModuleFields.Put("node.name", nameStr)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to put field node.name: %w", err))
			continue
		}
		err = event.MetricSetFields.Delete("name")
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to delete field name: %w", err))
			continue
		}

		r.Event(event)
	}
	return errs.Err()
}
