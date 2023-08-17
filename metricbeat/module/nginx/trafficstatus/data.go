package trafficstatus

import (
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(content map[string]interface{}) ([]mapstr.M, error) {
	// remove unnecessary keys from serverZones
	delete(content, "serverZones")

	// remove unnecessary keys from serverZones
	for _, value := range content["upstreamZones"].(map[string]interface{}) {
		for _, s := range value.([]interface{}) {
			delete(s.(map[string]interface{}), "requestMsecs")
			delete(s.(map[string]interface{}), "responseMsecs")
			delete(s.(map[string]interface{}), "responseBuckets")
			delete(s.(map[string]interface{}), "requestBuckets")
		}
	}

	buf := []mapstr.M{}

	for key, value := range content["upstreamZones"].(map[string]interface{}) {
		for _, s := range value.([]interface{}) {
			buf = append(buf, mapstr.M{
				"type": "upstream",
				"name": key,
				"data": s.(map[string]interface{}),
			})
		}
	}

	delete(content, "upstreamZones")

	buf = append(buf, mapstr.M{
		"type": "server",
		"data": content,
	})

	// event := mapstr.M(content)

	return buf, nil
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
