package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/digitalisio/axon-metrics-go-client/client"
	"github.com/denis-axon/reporting-v2/components/cloudapi"
	"github.com/denis-axon/reporting-v2/config"
	"go.uber.org/zap"
)

var (
	clientMu sync.Mutex
	clients  = make(map[string]*client.Client)
)

func GetClient(org string) *client.Client {
	clientMu.Lock()
	c, exists := clients[org]
	clientMu.Unlock()
	if !exists {
		if err := InitClient(org); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing metrics client: %v\n", err)
			return nil
		}
		clientMu.Lock()
		c = clients[org]
		clientMu.Unlock()
	}
	return c
}

func InitClient(org string) error {
	clientMu.Lock()
	if _, exists := clients[org]; exists {
		clientMu.Unlock()
		return nil
	}
	clientMu.Unlock()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	w := bytes.NewBuffer(nil)
	cfg := config.GetInstance()

	// switch between Regular and SAML modes URLs based on org settings in Cloud API
	samlConfig, err := cloudapi.GetSamlConfig(org)
	if err != nil {
		logger.Error("Failed to get SAML config for org", zap.String("org", org), zap.Error(err))
		return err
	}
	isSaml := samlConfig.Provider != ""
	logger.Info("SAML config retrieved", zap.String("org", org), zap.Bool("isSaml", isSaml))

	currentUrlTemplate := cfg.AxonServerUrlTemplate
	if isSaml {
		currentUrlTemplate = cfg.AxonServerUrlTemplateSaml
	}
	err = currentUrlTemplate.Execute(w, map[string]string{"Org": org})
	if err != nil {
		return err
	}

	baseURL := w.String()

	c, err := client.New(client.Options{
		BaseURL:   baseURL,
		AuthToken: "Bearer " + cfg.AuthToken,
		Timeout:   10 * time.Second,
		Logger:    logger,
	})
	if err != nil {
		return err
	}
	logger.Info("Metrics client initialized successfully", zap.String("org", org), zap.Any("client", c))

	clientMu.Lock()
	clients[org] = c
	clientMu.Unlock()

	return nil
}

func Healthy(org string) (error, bool) {
	c := GetClient(org)
	if c == nil {
		return fmt.Errorf("metrics client not initialized for org %s", org), false
	}
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/v1/healthz").
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err, false
	}

	return nil, resp.StatusCode == 200
}

func GetChartImage(org string, clusterName string, clusterType string, from string, to string, timeZone string, widgetUuid string) ([]byte, error) {
	c := GetClient(org)
	if c == nil {
		return nil, fmt.Errorf("metrics client not initialized for org %s", org)
	}
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/dash/chartImage").
		WithQueryParam("org", org).
		WithQueryParam("cluster", clusterName).
		WithQueryParam("clusterType", clusterType).
		WithQueryParam("width", "800").
		WithQueryParam("height", "400").
		WithQueryParam("timeZone", timeZone).
		WithQueryParam("from", from).
		WithQueryParam("to", to).
		WithQueryParam("widgetUuid", widgetUuid).
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Data, nil
}

type ClusterDetails struct {
	ClusterName      string
	ClusterType      string
	NodeCount        int
	DataCenters      string
	CassandraVersion string
	JavaVersion      string
	OSVersion        string
}

type ClustersResponse struct {
	Children []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Children []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Children []struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Status int    `json:"status"`
			} `json:"children"`
		} `json:"children"`
	} `json:"children"`
}

func GetClusters(org string) ([]ClusterDetails, error) {

	c := GetClient(org)
	if c == nil {
		return nil, fmt.Errorf("metrics client not initialized for org %s", org)
	}
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/v1/orgs").
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("GetClusters response", zap.Any("resp", resp))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clustersResp ClustersResponse
	err = json.Unmarshal(resp.Data, &clustersResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clusters response: %w", err)
	}

	var allDetails []ClusterDetails

	for _, orgNode := range clustersResp.Children {
		for _, typeNode := range orgNode.Children {
			clusterType := typeNode.Name
			for _, cluster := range typeNode.Children {
				clusterName := cluster.Name
				logger.Info("Fetching details for cluster",
					zap.String("org", org),
					zap.String("clusterType", clusterType),
					zap.String("clusterName", clusterName),
					zap.Int("status", cluster.Status),
				)

				details, err := getClusterDetails(org, clusterType, clusterName)
				if err != nil {
					logger.Error("Failed to get cluster details",
						zap.String("clusterName", clusterName),
						zap.Error(err),
					)
					continue
				}
				allDetails = append(allDetails, details)
			}
		}
	}

	return allDetails, nil
}

func getClusterDetails(org string, clusterType string, clusterName string) (ClusterDetails, error) {
	c := GetClient(org)
	if c == nil {
		return ClusterDetails{}, fmt.Errorf("metrics client not initialized for org %s", org)
	}
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/v1/nodes/" + org + "/" + clusterType + "/" + clusterName).
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return ClusterDetails{}, err
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("getClusterDetails raw data",
		zap.String("clusterName", clusterName),
		zap.String("rawData", string(resp.Data)),
	)

	if resp.StatusCode != 200 {
		return ClusterDetails{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// The response can be either a JSON array or a JSON object.
	// Normalize to a slice of nodes.
	var nodes []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &nodes); err != nil {
		// Try as a single object
		var single map[string]interface{}
		if err2 := json.Unmarshal(resp.Data, &single); err2 != nil {
			return ClusterDetails{}, fmt.Errorf("failed to parse cluster details response: %w", err)
		}
		nodes = []map[string]interface{}{single}
	}

	details := ClusterDetails{
		ClusterName: clusterName,
		ClusterType: clusterType,
		NodeCount:   len(nodes),
	}

	dcSet := make(map[string]struct{})
	var lowestJava string
	var cassandraVersion string
	var osVersion string

	for _, node := range nodes {
		// Data centers
		if dc, ok := node["DC"].(string); ok && dc != "" {
			dcSet[dc] = struct{}{}
		}

		// Extract details sub-object
		detailsMap, ok := node["Details"].(map[string]interface{})
		if !ok {
			continue
		}

		// Cassandra version (take first non-empty)
		if cv, ok := detailsMap["comp_releaseVersion"].(string); ok && cv != "" && cassandraVersion == "" {
			cassandraVersion = cv
		}

		// Java version (find lowest)
		if jv, ok := detailsMap["comp_jvm_java.version"].(string); ok && jv != "" {
			if lowestJava == "" || compareVersions(jv, lowestJava) < 0 {
				lowestJava = jv
			}
		}

		// OS version (take first non-empty)
		if osVersion == "" {
			platform, _ := detailsMap["host_Platform"].(string)
			platformVersion, _ := detailsMap["host_PlatformVersion"].(string)
			if platform != "" {
				osVersion = platform
				if platformVersion != "" {
					osVersion += " " + platformVersion
				}
			}
		}
	}

	// Build unique data centers string
	dcs := make([]string, 0, len(dcSet))
	for dc := range dcSet {
		dcs = append(dcs, dc)
	}
	details.DataCenters = strings.Join(dcs, ", ")
	details.CassandraVersion = cassandraVersion
	details.JavaVersion = lowestJava
	details.OSVersion = osVersion

	return details, nil
}

// compareVersions compares two version strings (e.g. "11.0.2" vs "17.0.1").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}
	return 0
}
