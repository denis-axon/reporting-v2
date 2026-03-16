package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// Snapshot types for strongly-typed parsing of cassandraSnapshot response

type NodeDetail struct {
	HostID              string `json:"HostID"`
	LocalSnapshotState  int    `json:"LocalSnapshotState"`
	RemoteSnapshotState int    `json:"RemoteSnapshotState"`
	RemoteError         string `json:"RemoteError,omitempty"`
	LocalError          string `json:"LocalError,omitempty"`
}

type SnapshotDescription struct {
	Tag                     string       `json:"tag"`
	SnapshotName            string       `json:"snapshotName"`
	Schedule                bool         `json:"schedule"`
	ScheduleExpr            string       `json:"scheduleExpr"`
	Datacenters             []string     `json:"datacenters"`
	CreationTime            int64        `json:"creationTime"`
	CompletionTime          int64        `json:"CompletionTime"`
	AllTables               bool         `json:"allTables"`
	AllNodes                bool         `json:"allNodes"`
	NodesDetails            []NodeDetail `json:"NodesDetails"`
	LocalRetentionDuration  string       `json:"LocalRetentionDuration"`
	RemoteRetentionDuration string       `json:"RemoteRetentionDuration"`
	ScheduleID              string       `json:"ScheduleID"`
	Remote                  *bool        `json:"Remote,omitempty"`
	RemoteConfig            string       `json:"RemoteConfig"`
	BackupDetails           string       `json:"BackupDetails"`
	Error                   string       `json:"error,omitempty"`
}

type Snapshot struct {
	Description SnapshotDescription `json:"description"`
	Status      string              `json:"status"`
}

type CassandraSnapshotResponse struct {
	Snapshots         []Snapshot             `json:"Snapshots"`
	SnapshotsCount    int                    `json:"SnapshotsCount"`
	RestoringSnapshot map[string]interface{} `json:"RestoringSnapshot"`
}

// BackupScheduleSummary groups snapshots by schedule and provides summary info
type BackupScheduleSummary struct {
	ScheduleID    string
	ScheduleExpr  string
	Tag           string
	Datacenters   string
	RemoteType    string
	Successful    int
	Failed        int
	FailedBackups []FailedBackupDetail
}

// FailedBackupDetail holds information about a single failed backup
type FailedBackupDetail struct {
	BackupTime      string
	FailedNodes     []string
	FailureMessages []string
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

func GetCassandraSnapshot(org string, clusterType string, clusterName string) (*CassandraSnapshotResponse, error) {
	c := GetClient(org)
	if c == nil {
		return nil, fmt.Errorf("metrics client not initialized for org %s", org)
	}
	ctx := context.Background()

	var allSnapshots []Snapshot
	page := 1
	perPage := 100

	for {
		req := client.NewRequest().
			WithMethod("GET").
			WithPath("/dashboard/api/v1/cassandraSnapshot/"+org+"/"+clusterType+"/"+clusterName).
			WithQueryParam("page", strconv.Itoa(page)).
			WithQueryParam("perPage", strconv.Itoa(perPage)).
			Build()

		resp, err := c.Do(ctx, req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var snapshot CassandraSnapshotResponse
		err = json.Unmarshal(resp.Data, &snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cluster snapshot response: %w", err)
		}

		log.Printf("getCassandraSnapshot page %d: got %d snapshots (total: %d)\n",
			page, len(snapshot.Snapshots), snapshot.SnapshotsCount)

		allSnapshots = append(allSnapshots, snapshot.Snapshots...)

		// Stop if we've fetched all snapshots
		if len(allSnapshots) >= snapshot.SnapshotsCount || len(snapshot.Snapshots) == 0 {
			result := &CassandraSnapshotResponse{
				Snapshots:         allSnapshots,
				SnapshotsCount:    snapshot.SnapshotsCount,
				RestoringSnapshot: snapshot.RestoringSnapshot,
			}
			return result, nil
		}

		page++
	}
}

// isSnapshotFailed determines if a snapshot failed based on the Remote flag and node states.
// If Remote is true, check RemoteSnapshotState == -2 on any node.
// If Remote is absent (nil), check LocalSnapshotState == -2 on any node.
func isSnapshotFailed(desc SnapshotDescription) bool {
	if desc.Remote != nil && *desc.Remote {
		// Remote backup: check RemoteSnapshotState
		for _, node := range desc.NodesDetails {
			if node.RemoteSnapshotState == -2 {
				return true
			}
		}
	} else if desc.Remote == nil {
		// No Remote property: check LocalSnapshotState
		for _, node := range desc.NodesDetails {
			if node.LocalSnapshotState == -2 {
				return true
			}
		}
	}
	return false
}

// getFailedNodeDetails extracts the HostIDs and error messages from failed nodes.
func getFailedNodeDetails(desc SnapshotDescription) ([]string, []string) {
	var failedNodes []string
	var failureMessages []string

	if desc.Remote != nil && *desc.Remote {
		for _, node := range desc.NodesDetails {
			if node.RemoteSnapshotState == -2 {
				failedNodes = append(failedNodes, node.HostID)
				if node.RemoteError != "" {
					failureMessages = append(failureMessages, node.RemoteError)
				}
			}
		}
	} else if desc.Remote == nil {
		for _, node := range desc.NodesDetails {
			if node.LocalSnapshotState == -2 {
				failedNodes = append(failedNodes, node.HostID)
				if node.LocalError != "" {
					failureMessages = append(failureMessages, node.LocalError)
				}
			}
		}
	}

	// If there's a top-level error on the description and no per-node messages collected
	if desc.Error != "" && len(failureMessages) == 0 {
		failureMessages = append(failureMessages, desc.Error)
	}

	return failedNodes, failureMessages
}

// getRemoteType extracts the remote type from BackupDetails JSON string
func getRemoteType(backupDetailsJSON string) string {
	if backupDetailsJSON == "" {
		return "local"
	}
	var bd map[string]interface{}
	if err := json.Unmarshal([]byte(backupDetailsJSON), &bd); err != nil {
		return "unknown"
	}
	if rt, ok := bd["remoteType"].(string); ok && rt != "" {
		return rt
	}
	return "local"
}

// humanizeScheduleExpr converts a cron expression into a human-readable string.
func humanizeScheduleExpr(expr string) string {
	expr = strings.TrimSpace(expr)
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return expr
	}

	minute, hour, dom, month, dow := parts[0], parts[1], parts[2], parts[3], parts[4]

	// Daily at specific time: M H * * *
	if minute != "*" && !strings.Contains(minute, "/") &&
		hour != "*" && !strings.Contains(hour, "/") &&
		dom == "*" && month == "*" && dow == "*" {
		return fmt.Sprintf("Daily at %s:%s", hour, fmt.Sprintf("%02s", minute))
	}

	// Every N minutes: */N * * * *
	if strings.HasPrefix(minute, "*/") && hour == "*" && dom == "*" && month == "*" && dow == "*" {
		interval := strings.TrimPrefix(minute, "*/")
		return fmt.Sprintf("Every %s minutes", interval)
	}

	// Top of every hour: 0 * * * *
	if minute == "0" && hour == "*" && dom == "*" && month == "*" && dow == "*" {
		return "Every hour"
	}

	// Every hour at minute M: M * * * *
	if minute != "*" && !strings.Contains(minute, "/") && hour == "*" && dom == "*" && month == "*" && dow == "*" {
		return fmt.Sprintf("Every hour at minute %s", minute)
	}

	// Every N hours: 0 */N * * *
	if minute == "0" && strings.HasPrefix(hour, "*/") && dom == "*" && month == "*" && dow == "*" {
		interval := strings.TrimPrefix(hour, "*/")
		return fmt.Sprintf("Every %s hours", interval)
	}

	return expr
}

// GetBackupSummaries parses a CassandraSnapshotResponse and returns per-schedule summaries.
func GetBackupSummaries(snapshotResp *CassandraSnapshotResponse) []BackupScheduleSummary {
	if snapshotResp == nil || len(snapshotResp.Snapshots) == 0 {
		return nil
	}

	// Group snapshots by ScheduleID when present, otherwise by scheduleExpr+tag
	type scheduleKey string
	groupOrder := []scheduleKey{}
	groups := make(map[scheduleKey][]Snapshot)

	for _, snap := range snapshotResp.Snapshots {
		var key scheduleKey
		if snap.Description.ScheduleID != "" {
			key = scheduleKey(snap.Description.ScheduleID)
		} else {
			// Fallback: group by scheduleExpr + tag for ad-hoc or unscheduled snapshots
			key = scheduleKey(snap.Description.ScheduleExpr + "|" + snap.Description.Tag)
		}
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], snap)
	}

	var summaries []BackupScheduleSummary

	for _, key := range groupOrder {
		snapshots := groups[key]
		if len(snapshots) == 0 {
			continue
		}

		first := snapshots[0].Description
		summary := BackupScheduleSummary{
			ScheduleID:   first.ScheduleID,
			ScheduleExpr: humanizeScheduleExpr(first.ScheduleExpr),
			Tag:          first.Tag,
			Datacenters:  strings.Join(first.Datacenters, ", "),
			RemoteType:   getRemoteType(first.BackupDetails),
		}

		for _, snap := range snapshots {
			if isSnapshotFailed(snap.Description) {
				summary.Failed++
				failedNodes, failureMessages := getFailedNodeDetails(snap.Description)

				backupTime := time.UnixMilli(snap.Description.CreationTime).UTC().Format("2006-01-02 15:04:05")

				summary.FailedBackups = append(summary.FailedBackups, FailedBackupDetail{
					BackupTime:      backupTime,
					FailedNodes:     failedNodes,
					FailureMessages: failureMessages,
				})
			} else {
				summary.Successful++
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries
}
