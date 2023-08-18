package cluster_stats

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/opendistro"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var schema = s.Schema{
	"status": c.Str("status"),
	"nodes": c.Dict("nodes", s.Schema{
		"versions": c.Ifc("versions"),
		"count":    c.Int("count.total"),
		"master":   c.Int("count.master"),
		"data":     c.Int("count.data"),
		"fs": c.Dict("fs", s.Schema{
			"total": s.Object{
				"bytes": c.Int("total_in_bytes"),
			},
			"available": s.Object{
				"bytes": c.Int("available_in_bytes"),
			},
		}),
		"jvm": c.Dict("jvm", s.Schema{
			"max_uptime": s.Object{
				"ms": c.Int("max_uptime_in_millis"),
			},
			"memory": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"used": s.Object{
						"bytes": c.Int("heap_used_in_bytes"),
					},
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
				},
			}),
		}),
	}),

	"indices": c.Dict("indices", s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"total": c.Int("count"),
		}),
		"total": c.Int("count"),
		"shards": c.Dict("shards", s.Schema{
			"count":     c.Int("total"),
			"primaries": c.Int("primaries"),
		}),
		"store": c.Dict("store", s.Schema{
			"size": s.Object{"bytes": c.Int("size_in_bytes")},
		}),
		"fielddata": c.Dict("fielddata", s.Schema{
			"memory": s.Object{
				"bytes": c.Int("memory_size_in_bytes"),
			},
		}),
	}),
}

// computeNodesHash computes a simple hash value that can be used to determine if the nodes listing has changed since the last report.
func computeNodesHash(clusterState mapstr.M) (int32, error) {
	value, err := clusterState.GetValue("nodes")
	if err != nil {
		return 0, elastic.MakeErrorForMissingField("nodes", elastic.Elasticsearch)
	}

	nodes, ok := value.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("nodes is not a map")
	}

	var nodeEphemeralIDs []string
	for _, value := range nodes {
		nodeData, ok := value.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("node data is not a map")
		}

		value, ok := nodeData["ephemeral_id"]
		if !ok {
			return 0, fmt.Errorf("node data does not contain ephemeral ID")
		}

		ephemeralID, ok := value.(string)
		if !ok {
			return 0, fmt.Errorf("node ephemeral ID is not a string")
		}

		nodeEphemeralIDs = append(nodeEphemeralIDs, ephemeralID)
	}

	sort.Strings(nodeEphemeralIDs)

	combinedNodeEphemeralIDs := strings.Join(nodeEphemeralIDs, "")
	return hash(combinedNodeEphemeralIDs), nil
}

func hash(s string) int32 {
	h := fnv.New32()
	h.Write([]byte(s))
	return int32(h.Sum32()) // This cast is needed because the ES mapping is for a 32-bit *signed* integer
}

func getClusterMetadataSettings(httpClient *helper.HTTP) (mapstr.M, error) {
	// For security reasons we only get the display_name setting
	filterPaths := []string{"*.cluster.metadata.display_name"}
	clusterSettings, err := opendistro.GetClusterSettingsWithDefaults(httpClient, httpClient.GetURI(), filterPaths)
	if err != nil {
		return nil, fmt.Errorf("failure to get cluster settings: %w", err)
	}

	clusterSettings, err = opendistro.MergeClusterSettings(clusterSettings)
	if err != nil {
		return nil, fmt.Errorf("failure to merge cluster settings: %w", err)
	}

	return clusterSettings, nil
}

func eventMapping(r mb.ReporterV2, httpClient *helper.HTTP, info opendistro.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Cluster Stats API response: %w", err)
	}

	clusterStats := mapstr.M(data)
	clusterStats.Delete("_nodes")

	clusterStateMetrics := []string{"version", "master_node", "nodes", "routing_table"}
	clusterState, err := opendistro.GetClusterState(httpClient, httpClient.GetURI(), clusterStateMetrics)
	if err != nil {
		return fmt.Errorf("failed to get cluster state from Elasticsearch: %w", err)
	}
	clusterState.Delete("cluster_name")

	clusterStateReduced := mapstr.M{}
	if err = opendistro.PassThruField("status", clusterStats, clusterStateReduced); err != nil {
		return fmt.Errorf("failed to pass through status field: %w", err)
	}
	clusterStateReduced.Delete("status")

	if err = opendistro.PassThruField("master_node", clusterState, clusterStateReduced); err != nil {
		return fmt.Errorf("failed to pass through master_node field: %w", err)
	}

	if err = opendistro.PassThruField("state_uuid", clusterState, clusterStateReduced); err != nil {
		return fmt.Errorf("failed to pass through state_uuid field: %w", err)
	}

	if err = opendistro.PassThruField("nodes", clusterState, clusterStateReduced); err != nil {
		return fmt.Errorf("failed to pass through nodes field: %w", err)
	}

	nodesHash, err := computeNodesHash(clusterState)
	if err != nil {
		return fmt.Errorf("failed to compute nodes hash: %w", err)
	}
	clusterStateReduced.Put("nodes_hash", nodesHash)

	delete(clusterState, "routing_table") // We don't want to index the routing table in monitoring indices

	event := mb.Event{
		ModuleFields: mapstr.M{},
		RootFields:   mapstr.M{},
	}
	event.ModuleFields.Put("cluster.name", info.ClusterName)
	event.ModuleFields.Put("cluster.id", info.ClusterID)

	clusterSettings, err := getClusterMetadataSettings(httpClient)
	if err != nil {
		return err
	}
	if clusterSettings != nil {
		event.RootFields.Put("cluster_settings", clusterSettings)
	}

	metricSetFields, _ := schema.Apply(data)

	metricSetFields.Put("state", clusterStateReduced)

	if err = opendistro.PassThruField("version", clusterState, event.ModuleFields); err != nil {
		return fmt.Errorf("failed to pass through version field: %w", err)
	}

	event.MetricSetFields = metricSetFields

	r.Event(event)

	return nil
}
