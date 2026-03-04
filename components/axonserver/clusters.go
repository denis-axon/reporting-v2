package axonserver

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	log "bitbucket.org/digitalisio/go/logger"
)

type ClusterInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  int    `json:"status"`
	Nodes   int    `json:"nodes"`
	Version string `json:"version"`
}

func GetClusters(org string) ([]ClusterInfo, error) {
	res, err := doApiCall(org, "orgs")
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(res, &data)
	if err != nil {
		return nil, err
	}

	var allClusters []ClusterInfo
	if orgsResp, ok := data["children"].([]interface{}); ok {
		for _, oIntf := range orgsResp {
			o, ok := oIntf.(map[string]interface{})
			if !ok {
				continue
			}
			if t, ok := o["type"]; !ok || t != "org" {
				continue
			}
			if n, ok := o["name"]; !ok || n != org {
				continue
			}
			if clusterTypes, ok := o["children"].([]interface{}); ok {
				for _, ctIntf := range clusterTypes {
					ct, ok := ctIntf.(map[string]interface{})
					if !ok {
						continue
					}
					if clusters, ok := ct["children"].([]interface{}); ok {
						for _, cIntf := range clusters {
							c, ok := cIntf.(map[string]interface{})
							if !ok {
								continue
							}
							n, ok := c["name"].(string)
							if !ok {
								continue
							}
							t, ok := c["type"].(string)
							if !ok {
								continue
							}
							s, ok := c["status"].(float64)
							if !ok {
								continue
							}

							numNodes, clusterVersion, err := getNodeCountAndClusterVersion(org, t, n)
							if err != nil {
								log.Error(fmt.Sprintf("Error getting nodes for cluster %s/%s/%s: %s", org, t, n, err.Error()))
								numNodes = 0
							}

							allClusters = append(allClusters, ClusterInfo{
								Name:    n,
								Type:    t,
								Nodes:   numNodes,
								Status:  int(s),
								Version: clusterVersion,
							})
						}
					}
				}
			}
		}
	}
	return allClusters, nil
}

func getNodeCountAndClusterVersion(org, clusterType, cluster string) (int, string, error) {
	res, err := doApiCall(org, "nodes/"+org+"/"+clusterType+"/"+cluster)
	if err != nil {
		return 0, "", err
	}

	var data []interface{}
	err = json.Unmarshal(res, &data)
	if err != nil {
		return 0, "", err
	}

	// get the cluster version - if there are multiple versions then return them as a comma-separated list
	nodeVersions := make([]string, 0, len(data))
	for _, n := range data {
		node := n.(map[string]interface{})
		details, ok := node["Details"].(map[string]interface{})
		if !ok {
			continue
		}
		nodeVer, ok := details["comp_releaseVersion"].(string)
		nodeVer = strings.TrimSpace(nodeVer)
		if ok && nodeVer != "" {
			nodeVersions = append(nodeVersions, nodeVer)
		}
	}

	// get only unique versions from the list
	slices.Sort(nodeVersions)
	nodeVersions = slices.Compact(nodeVersions)

	return len(data), strings.Join(nodeVersions, ", "), nil
}

func cacheKey(org string) string {
	return "cloudapi-nodescache:" + org
}
