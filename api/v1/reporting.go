package v1

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/components/converter"

	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/google/uuid"
)

// TOOD: should they be added to env vars?
// for test41cluster the UUID was c11b97f0-6b2e-40cd-abc6-b721e38778b9
var WIDGET_CHART_CPU_UUID = "8d6ce9dc-77fe-4576-8b30-7986d8e0bf9b"
var WIDGET_CHART_MEMORY_UUID = "8d6ce9dc-77fe-4576-8b30-7986d8e0bf9b"

func GeneratePDF(c *gin.Context) {
	// Get org from query parameter
	org := c.Query("org")
	clusterName := c.Query("clusterName")
	clusterType := c.Query("clusterType")
	from := c.Query("from")
	to := c.Query("to")
	timeZone := c.Query("timeZone")

	// Validate required query parameters
	if org == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: org"))
		return
	}
	if clusterName == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: clusterName"))
		return
	}
	if clusterType == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: clusterType"))
		return
	}
	if from == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: from"))
		return
	}
	if to == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: to"))
		return
	}
	if timeZone == "" {
		utils.ReturnError(c, fmt.Errorf("missing required query parameter: timeZone"))
		return
	}

	// Fetch all chart images concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var images []converter.ImageData

	chartConfigs := []struct {
		placeholder string
		widgetUuid  string
	}{
		{"{{CHART_CPU}}", WIDGET_CHART_CPU_UUID},
		{"{{CHART_MEMORY}}", WIDGET_CHART_MEMORY_UUID},
	}

	for i, cfg := range chartConfigs {
		wg.Add(1)
		go func(idx int, c struct{ placeholder, widgetUuid string }) {
			defer wg.Done()
			data, err := metrics.GetChartImage(org, clusterName, clusterType, from, to, timeZone, c.widgetUuid)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting chart image for widget %s: %v\n", c.widgetUuid, err)
				return
			}
			mu.Lock()
			images = append(images, converter.ImageData{
				Placeholder: c.placeholder,
				Data:        data,
				Filename:    fmt.Sprintf("chart_%d.png", idx),
			})
			mu.Unlock()
		}(i, cfg)
	}
	wg.Wait()

	// Prepare report data
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)

	// Convert Unix timestamps to formatted dates
	dateFormat := "2006-01-02 15:04:05"
	fromUnix, _ := strconv.ParseInt(from, 10, 64)
	toUnix, _ := strconv.ParseInt(to, 10, 64)
	dateFromFormatted := time.Unix(fromUnix, 0).In(loc).Format(dateFormat)
	dateToFormatted := time.Unix(toUnix, 0).In(loc).Format(dateFormat)

	// Get cluster details
	allDetails, err := metrics.GetClusters(org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cluster details: %v\n", err)
	}

	fmt.Printf("Cluster details: %+v\n", allDetails)

	// Find the matching cluster details for the requested cluster
	var matchedCluster metrics.ClusterDetails
	for _, d := range allDetails {
		if d.ClusterName == clusterName && d.ClusterType == clusterType {
			matchedCluster = d
			break
		}
	}

	// Fetch backup/snapshot data and build the backups markdown section
	backupsSection := buildBackupsSection(org, clusterType, clusterName)

	reportData := converter.ReportData{
		Organization:     org,
		Dashboard:        "Reporting",
		DateFrom:         dateFromFormatted,
		DateTo:           dateToFormatted,
		Timezone:         timeZone,
		GeneratedAt:      now.Format(dateFormat),
		ClusterType:      clusterType,
		ClusterName:      clusterName,
		NodeCount:        strconv.Itoa(matchedCluster.NodeCount),
		DataCenters:      matchedCluster.DataCenters,
		CassandraVersion: matchedCluster.CassandraVersion,
		JavaVersion:      matchedCluster.JavaVersion,
		OSVersion:        matchedCluster.OSVersion,
		BackupsSection:   backupsSection,
	}

	// Generate PDF with unique output path
	outputPath := filepath.Join(os.TempDir(), uuid.New().String()+".pdf")
	err = converter.GeneratePDFWithImages("templates/report.md", outputPath, images, reportData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating PDF: %v\n", err)
		utils.ReturnError(c, err)
		return
	}
	defer os.Remove(outputPath)

	// Serve PDF
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(outputPath)))
	c.Header("Content-Type", "application/pdf")
	c.File(outputPath)

	fmt.Printf("PDF generated successfully at %s\n", outputPath)
}

// buildBackupsSection fetches cassandra snapshot data and renders a markdown string
// for the backups section of the report.
func buildBackupsSection(org, clusterType, clusterName string) string {
	snapshotResp, err := metrics.GetCassandraSnapshot(org, clusterType, clusterName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cassandra snapshot: %v\n", err)
		return "## Backups\n\nNo backup data available.\n"
	}

	summaries := metrics.GetBackupSummaries(snapshotResp)
	if len(summaries) == 0 {
		return "## Backups\n\nNo backup schedules found.\n"
	}

	var sb strings.Builder
	// sb.WriteString("## Backups\n\n")

	for i, summary := range summaries {
		sb.WriteString(fmt.Sprintf("##### Schedule %d\n\n", i+1))

		sb.WriteString("```text\n")
		sb.WriteString(fmt.Sprintf("Tag                       : %s\n", summary.Tag))
		sb.WriteString(fmt.Sprintf("Schedule                  : %s\n", summary.ScheduleExpr))
		sb.WriteString(fmt.Sprintf("Data Centers              : %s\n", summary.Datacenters))
		sb.WriteString(fmt.Sprintf("Remote Type               : %s\n", summary.RemoteType))
		sb.WriteString("```\n\n")

		sb.WriteString("###### Backups Summary\n\n")
		sb.WriteString("```text\n")
		sb.WriteString(fmt.Sprintf("Successful Backups        : %d\n", summary.Successful))
		sb.WriteString(fmt.Sprintf("Failed Backups            : %d\n", summary.Failed))
		sb.WriteString("```\n\n")

		if summary.Failed > 0 && len(summary.FailedBackups) > 0 {
			sb.WriteString("###### Failed Backups Details\n\n")
			for j, fb := range summary.FailedBackups {
				sb.WriteString(fmt.Sprintf("**Failure %d**\n\n", j+1))
				sb.WriteString("```text\n")
				sb.WriteString(fmt.Sprintf("Backup Time    : %s\n", fb.BackupTime))
				sb.WriteString(fmt.Sprintf("Failed Nodes   : %d\n", len(fb.FailedNodes)))
				for _, nodeID := range fb.FailedNodes {
					sb.WriteString(fmt.Sprintf("                 %s\n", nodeID))
				}
				sb.WriteString("```\n\n")

				// Render errors as regular text for smaller font rendering
				uniqueErrors := deduplicateErrors(fb.FailureMessages)
				for _, msg := range uniqueErrors {
					sb.WriteString(fmt.Sprintf("*%s*\n\n", msg))
				}
			}
		}
	}

	return sb.String()
}

func deduplicateErrors(messages []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, msg := range messages {
		key := msg
		if len(key) > 80 {
			key = key[:80]
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, msg)
		}
	}
	return unique
}

// truncateMessage shortens a message to maxLen characters, appending "..." if truncated.
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}
