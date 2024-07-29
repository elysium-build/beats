package cluster_stats

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/metricbeat/module/opendistro"
	"github.com/pkg/errors"
)

func eventMapping(r mb.ReporterV2, m *MetricSet, info opendistro.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Cluster Stats API response")
	}

	clusterStats := common.MapStr(data)
	clusterStats.Delete("_nodes")

	value, err := clusterStats.GetValue("cluster_name")
	if err != nil {
		return elastic.MakeErrorForMissingField("cluster_name", elastic.Elasticsearch)
	}
	clusterName, ok := value.(string)
	if !ok {
		return fmt.Errorf("cluster name is not a string")
	}
	clusterStats.Delete("cluster_name")

	clusterStateMetrics := []string{"version", "master_node", "nodes", "routing_table"}
	clusterState, err := elasticsearch.GetClusterState(m.HTTP, m.HTTP.GetURI(), clusterStateMetrics)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster state from Elasticsearch")
	}
	clusterState.Delete("cluster_name")

	if err = elasticsearch.PassThruField("status", clusterStats, clusterState); err != nil {
		return errors.Wrap(err, "failed to pass through status field")
	}

	nodesHash, err := computeNodesHash(clusterState)
	if err != nil {
		return errors.Wrap(err, "failed to compute nodes hash")
	}
	clusterState.Put("nodes_hash", nodesHash)

	delete(clusterState, "routing_table") // We don't want to index the routing table in monitoring indices

	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":  info.ClusterID,
		"cluster_name":  clusterName,
		"timestamp":     common.Time(time.Now()),
		"interval_ms":   m.Module().Config().Period / time.Millisecond,
		"type":          "cluster_stats",
		"version":       info.Version.Number.String(),
		"cluster_stats": clusterStats,
		"cluster_state": clusterState,
	}

	clusterSettings, err := getClusterMetadataSettings(m)
	if err != nil {
		return err
	}
	if clusterSettings != nil {
		event.RootFields.Put("cluster_settings", clusterSettings)
	}

	r.Event(event)
	return nil
}

func computeNodesHash(clusterState common.MapStr) (int32, error) {
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

func getClusterMetadataSettings(m *MetricSet) (common.MapStr, error) {
	// For security reasons we only get the display_name setting
	filterPaths := []string{"*.cluster.metadata.display_name"}
	clusterSettings, err := opendistro.GetClusterSettingsWithDefaults(m.HTTP, m.HTTP.GetURI(), filterPaths)
	if err != nil {
		return nil, errors.Wrap(err, "failure to get cluster settings")
	}

	clusterSettings, err = opendistro.MergeClusterSettings(clusterSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failure to merge cluster settings")
	}

	return clusterSettings, nil
}
