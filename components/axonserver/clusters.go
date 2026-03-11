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

// ClusterDetails contains detailed information about a specific cluster
// type ClusterDetails struct {
// 	NodeCount        int
// 	DataCenters      string
// 	CassandraVersion string
// 	OSVersion        string
// 	JavaVersion      string
// }

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

// GetClusterDetails retrieves detailed information about a specific cluster
// func GetClusterDetails(org, clusterType, clusterName string) (ClusterDetails, error) {
// 	res, err := doApiCall(org, "nodes/"+org+"/"+clusterType+"/"+clusterName)
// 	if err != nil {
// 		return ClusterDetails{}, err
// 	}

// 	var data []interface{}
// 	err = json.Unmarshal(res, &data)
// 	if err != nil {
// 		return ClusterDetails{}, err
// 	}

// 	// log the raw response for debugging
// 	log.Info(fmt.Sprintf("GetClusterDetails raw data for %s/%s/%s: %+v", org, clusterType, clusterName, data))

// 	details := ClusterDetails{
// 		NodeCount:   len(data),
// 		DataCenters: "All",
// 	}

// 	// Gather versions from all nodes
// 	cassandraVersions := make([]string, 0)
// 	osVersions := make([]string, 0)
// 	javaVersions := make([]string, 0)
// 	dcNames := make([]string, 0)

// 	for _, n := range data {
// 		node, ok := n.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		// Get datacenter name
// 		if dc, ok := node["dc"].(string); ok && dc != "" {
// 			dcNames = append(dcNames, dc)
// 		}

// 		nodeDetails, ok := node["Details"].(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		// Cassandra version
// 		if ver, ok := nodeDetails["comp_releaseVersion"].(string); ok && ver != "" {
// 			cassandraVersions = append(cassandraVersions, strings.TrimSpace(ver))
// 		}

// 		// OS version
// 		if osName, ok := nodeDetails["host_osName"].(string); ok {
// 			osVer := ""
// 			if v, ok := nodeDetails["host_osVersion"].(string); ok {
// 				osVer = v
// 			}
// 			fullOS := strings.TrimSpace(osName + " " + osVer)
// 			if fullOS != "" {
// 				osVersions = append(osVersions, fullOS)
// 			}
// 		}

// 		// Java version
// 		if javaVer, ok := nodeDetails["jvm_version"].(string); ok && javaVer != "" {
// 			javaVersions = append(javaVersions, strings.TrimSpace(javaVer))
// 		}
// 	}

// 	// Get unique values
// 	slices.Sort(cassandraVersions)
// 	cassandraVersions = slices.Compact(cassandraVersions)
// 	details.CassandraVersion = strings.Join(cassandraVersions, ", ")

// 	slices.Sort(osVersions)
// 	osVersions = slices.Compact(osVersions)
// 	details.OSVersion = strings.Join(osVersions, ", ")

// 	slices.Sort(javaVersions)
// 	javaVersions = slices.Compact(javaVersions)
// 	details.JavaVersion = strings.Join(javaVersions, ", ")

// 	slices.Sort(dcNames)
// 	dcNames = slices.Compact(dcNames)
// 	if len(dcNames) > 0 {
// 		details.DataCenters = strings.Join(dcNames, ", ")
// 	}

// 	// Set defaults if empty
// 	if details.CassandraVersion == "" {
// 		details.CassandraVersion = "N/A"
// 	}
// 	if details.OSVersion == "" {
// 		details.OSVersion = "N/A"
// 	}
// 	if details.JavaVersion == "" {
// 		details.JavaVersion = "N/A"
// 	}

// 	return details, nil
// }
