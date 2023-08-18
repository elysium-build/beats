package opendistro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

var (
	// CCRStatsAPIAvailableVersion is the version of Elasticsearch since when the CCR stats API is available.
	CCRStatsAPIAvailableVersion = version.MustNew("6.5.0")

	// EnrichStatsAPIAvailableVersion is the version of Elasticsearch since when the Enrich stats API is available.
	EnrichStatsAPIAvailableVersion = version.MustNew("7.5.0")

	// BulkStatsAvailableVersion is the version since when bulk indexing stats are available
	BulkStatsAvailableVersion = version.MustNew("8.0.0")

	// ExpandWildcardsHiddenAvailableVersion is the version since when the "expand_wildcards" query parameter to
	// the Indices Stats API can accept "hidden" as a value.
	ExpandWildcardsHiddenAvailableVersion = version.MustNew("7.7.0")

	// Global clusterIdCache. Assumption is that the same node id never can belong to a different cluster id.
	clusterIDCache = map[string]string{}
)

const ModuleName = "opendistro"

type Info struct {
	ClusterName string  `json:"cluster_name"`
	ClusterID   string  `json:"cluster_uuid"`
	Version     Version `json:"version"`
	Name        string  `json:"name"`
}

type Version struct {
	Number *version.V `json:"number"`
}

type NodeInfo struct {
	Host             string `json:"host"`
	TransportAddress string `json:"transport_address"`
	IP               string `json:"ip"`
	Name             string `json:"name"`
	ID               string
}

func GetClusterID(http *helper.HTTP, uri string, nodeID string) (string, error) {
	// Check if cluster id already cached. If yes, return it.
	if clusterID, ok := clusterIDCache[nodeID]; ok {
		return clusterID, nil
	}

	info, err := GetInfo(http, uri)
	if err != nil {
		return "", err
	}

	clusterIDCache[nodeID] = info.ClusterID
	return info.ClusterID, nil
}

func isMaster(http *helper.HTTP, uri string) (bool, error) {
	node, err := getNodeName(http, uri)
	if err != nil {
		return false, err
	}

	master, err := getMasterName(http, uri)
	if err != nil {
		return false, err
	}

	return master == node, nil
}

func getNodeName(http *helper.HTTP, uri string) (string, error) {
	content, err := fetchPath(http, uri, "/_nodes/_local/nodes", "")
	if err != nil {
		return "", err
	}

	nodesStruct := struct {
		Nodes map[string]interface{} `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	// _local will only fetch one node info. First entry is node name
	for k := range nodesStruct.Nodes {
		return k, nil
	}
	return "", fmt.Errorf("No local node found")
}

func getMasterName(http *helper.HTTP, uri string) (string, error) {
	// TODO: evaluate on why when run with ?local=true request does not contain master_node field
	content, err := fetchPath(http, uri, "_cluster/state/master_node", "")
	if err != nil {
		return "", err
	}

	clusterStruct := struct {
		MasterNode string `json:"master_node"`
	}{}

	json.Unmarshal(content, &clusterStruct)

	return clusterStruct.MasterNode, nil
}

func GetInfo(http *helper.HTTP, uri string) (Info, error) {
	content, err := fetchPath(http, uri, "/", "")
	if err != nil {
		return Info{}, err
	}

	info := Info{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		return Info{}, err
	}

	return info, nil
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

func GetNodeInfo(http *helper.HTTP, uri string, nodeID string) (*NodeInfo, error) {
	content, err := fetchPath(http, uri, "/_nodes/_local/nodes", "")
	if err != nil {
		return nil, err
	}

	nodesStruct := struct {
		Nodes map[string]*NodeInfo `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	// _local will only fetch one node info. First entry is node name
	for k, v := range nodesStruct.Nodes {
		// In case the nodeID is empty, first node info will be returned
		if k == nodeID || nodeID == "" {
			v.ID = k
			return v, nil
		}
	}
	return nil, fmt.Errorf("no node matched id %s", nodeID)
}

func GetClusterState(http *helper.HTTP, resetURI string, metrics []string) (mapstr.M, error) {
	clusterStateURI := "_cluster/state"
	if metrics != nil && len(metrics) > 0 {
		clusterStateURI += "/" + strings.Join(metrics, ",")
	}

	content, err := fetchPath(http, resetURI, clusterStateURI, "")
	if err != nil {
		return nil, err
	}

	var clusterState map[string]interface{}
	err = json.Unmarshal(content, &clusterState)
	return clusterState, err
}

// GetClusterSettingsWithDefaults returns cluster settings.
func GetClusterSettingsWithDefaults(http *helper.HTTP, resetURI string, filterPaths []string) (mapstr.M, error) {
	return GetClusterSettings(http, resetURI, true, filterPaths)
}

// GetClusterSettings returns cluster settings
func GetClusterSettings(http *helper.HTTP, resetURI string, includeDefaults bool, filterPaths []string) (mapstr.M, error) {
	clusterSettingsURI := "_cluster/settings"
	var queryParams []string
	if includeDefaults {
		queryParams = append(queryParams, "include_defaults=true")
	}

	if filterPaths != nil && len(filterPaths) > 0 {
		filterPathQueryParam := "filter_path=" + strings.Join(filterPaths, ",")
		queryParams = append(queryParams, filterPathQueryParam)
	}

	queryString := strings.Join(queryParams, "&")

	content, err := fetchPath(http, resetURI, clusterSettingsURI, queryString)
	if err != nil {
		return nil, err
	}

	var clusterSettings map[string]interface{}
	err = json.Unmarshal(content, &clusterSettings)
	return clusterSettings, err
}

// GetStackUsage returns stack usage information.
func GetStackUsage(http *helper.HTTP, resetURI string) (map[string]interface{}, error) {
	content, err := fetchPath(http, resetURI, "_xpack/usage", "")
	if err != nil {
		return nil, err
	}

	var stackUsage map[string]interface{}
	err = json.Unmarshal(content, &stackUsage)
	return stackUsage, err
}

type boolStr bool

func (b *boolStr) UnmarshalJSON(raw []byte) error {
	var bs string
	err := json.Unmarshal(raw, &bs)
	if err != nil {
		return err
	}

	bv, err := strconv.ParseBool(bs)
	if err != nil {
		return err
	}

	*b = boolStr(bv)
	return nil
}

type IndexSettings struct {
	Hidden bool
}

func GetIndicesSettings(http *helper.HTTP, resetURI string) (map[string]IndexSettings, error) {
	content, err := fetchPath(http, resetURI, "*/_settings", "filter_path=*.settings.index.hidden&expand_wildcards=all")
	if err != nil {
		return nil, fmt.Errorf("could not fetch indices settings: %w", err)
	}

	var resp map[string]struct {
		Settings struct {
			Index struct {
				Hidden boolStr `json:"hidden"`
			} `json:"index"`
		} `json:"settings"`
	}

	err = json.Unmarshal(content, &resp)
	if err != nil {
		return nil, fmt.Errorf("could not parse indices settings response: %w", err)
	}

	ret := make(map[string]IndexSettings, len(resp))
	for index, settings := range resp {
		ret[index] = IndexSettings{
			Hidden: bool(settings.Settings.Index.Hidden),
		}
	}

	return ret, nil
}

func IsMLockAllEnabled(http *helper.HTTP, resetURI, nodeID string) (bool, error) {
	content, err := fetchPath(http, resetURI, "_nodes/"+nodeID, "filter_path=nodes.*.process.mlockall")
	if err != nil {
		return false, err
	}

	var response map[string]map[string]map[string]map[string]bool
	err = json.Unmarshal(content, &response)
	if err != nil {
		return false, err
	}

	for _, nodeInfo := range response["nodes"] {
		mlockall := nodeInfo["process"]["mlockall"]
		return mlockall, nil
	}

	return false, fmt.Errorf("could not determine if mlockall is enabled on node ID = %v", nodeID)
}

func GetMasterNodeID(http *helper.HTTP, resetURI string) (string, error) {
	content, err := fetchPath(http, resetURI, "_nodes/_master", "filter_path=nodes.*.name")
	if err != nil {
		return "", err
	}

	var response struct {
		Nodes map[string]interface{} `json:"nodes"`
	}

	if err := json.Unmarshal(content, &response); err != nil {
		return "", err
	}

	for nodeID := range response.Nodes {
		return nodeID, nil
	}

	return "", errors.New("could not determine master node ID")
}

func PassThruField(fieldPath string, sourceData, targetData mapstr.M) error {
	fieldValue, err := sourceData.GetValue(fieldPath)
	if err != nil {
		return elastic.MakeErrorForMissingField(fieldPath, elastic.Elasticsearch)
	}

	targetData.Put(fieldPath, fieldValue)
	return nil
}

// MergeClusterSettings merges cluster settings in the correct precedence order
func MergeClusterSettings(clusterSettings mapstr.M) (mapstr.M, error) {
	transientSettings, err := getSettingGroup(clusterSettings, "transient")
	if err != nil {
		return nil, err
	}

	persistentSettings, err := getSettingGroup(clusterSettings, "persistent")
	if err != nil {
		return nil, err
	}

	settings, err := getSettingGroup(clusterSettings, "default")
	if err != nil {
		return nil, err
	}

	// Transient settings override persistent settings which override default settings
	if settings == nil {
		settings = persistentSettings
	}

	if settings == nil {
		settings = transientSettings
	}

	if settings == nil {
		return nil, nil
	}

	if persistentSettings != nil {
		settings.DeepUpdate(persistentSettings)
	}

	if transientSettings != nil {
		settings.DeepUpdate(transientSettings)
	}

	return settings, nil
}

func getSettingGroup(allSettings mapstr.M, groupKey string) (mapstr.M, error) {
	hasSettingGroup, err := allSettings.HasKey(groupKey)
	if err != nil {
		return nil, fmt.Errorf("failure to determine if "+groupKey+" settings exist: %w", err)
	}

	if !hasSettingGroup {
		return nil, nil
	}

	settings, err := allSettings.GetValue(groupKey)
	if err != nil {
		return nil, fmt.Errorf("failure to extract "+groupKey+" settings: %w", err)
	}

	v, ok := settings.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(groupKey + " settings are not a map")
	}

	return mapstr.M(v), nil
}
