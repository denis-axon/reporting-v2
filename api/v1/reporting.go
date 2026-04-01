package v1

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/components/converter"

	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/google/uuid"
)

// TOOD: should they be added to env vars?
// Max Disk Read Per Second
var WIDGET_CHART_DISK_READ_UUID = "34c7b8c9-e355-44eb-8801-1cda450627f4"

// Used Disk Space Per Node
var WIDGET_CHART_DISK_USAGE_UUID = "d20d4c11-abdb-4f57-8039-54d4678d7400"

// Average CPU Usage per DC
var WIDGET_CHART_CPU_UUID = "323fc1c5-fd7e-4485-ae10-724b146735b3"

// Max Disk Write Per Second
var WIDGET_CHART_DISK_WRITE_UUID = "237fddeb-594c-4a84-9916-83dc30d1675e"

// Average Disk % Usage All
var WIDGET_CHART_DISK_ALL_USAGE_UUID = "483f11de-feaf-4275-8d92-68315ae5f236"

// Coordinator Reads distribution
var WIDGET_CHART_COORDINATOR_READS_UUID = "1851ace3-976f-459e-843c-e0119f718c8d"

// Coordinator Writes distribution
var WIDGET_CHART_COORDINATOR_WRITES_UUID = "e697843a-2526-4630-8616-519377dc51f7"

// Coordinator Read Throughput Per $groupBy ($consistency) - Count Per Second
var WIDGET_CHART_COORDINATOR_READ_THROUGHPUT_UUID = "f1900e0d-4044-400f-a485-2f64258e4227"

// Total Coordinator Write Throughput Per $groupBy ($consistency) - Count Per Second
var WIDGET_CHART_COORDINATOR_WRITE_THROUGHPUT_UUID = "b844ea65-bf98-45a3-b833-b0621fed84ff"

// Max Coordinator Read $consistency Latency - $percentile
var WIDGET_CHART_COORDINATOR_READ_LATENCY_UUID = "dc1fe7f8-dc9a-4d11-9efc-fd99269e114b"

// Max Coordinator Write $consistency Latency - $percentile
var WIDGET_CHART_COORDINATOR_WRITE_LATENCY_UUID = "da8dbb16-2ce2-49be-81ea-cbdbe0bb1eb1"

func GeneratePDF(c *gin.Context) {
	// Get org from query parameter
	org := c.Query("org")
	clusterName := c.Query("clusterName")
	clusterType := c.Query("clusterType")
	from := c.Query("from")
	to := c.Query("to")
	timeZone := c.Query("timeZone")

	width := c.DefaultQuery("width", "680")
	height := c.DefaultQuery("height", "425")

	consistency := c.DefaultQuery("consistency", "")
	percentile := c.DefaultQuery("percentile", "99thPercentile")
	groupBy := c.DefaultQuery("groupBy", "dc")

	sharedChartVars := map[string]string{
		"consistency": consistency,
		"percentile":  percentile,
		"groupBy":     groupBy,
		"width":       width,
		"height":      height,

		"org":         org,
		"cluster":     clusterName,
		"clusterType": clusterType,
		"timeZone":    timeZone,
		"from":        from,
		"to":          to,
	}

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

	// Fetch chart images sequentially to avoid overwhelming the chart
	// rendering server (concurrent requests cause timeouts).
	var images []converter.ImageData

	chartConfigs := []struct {
		placeholder string
		widgetUuid  string
		chartType   string
		title       string
	}{
		{"{{CHART_DISK_READ}}", WIDGET_CHART_DISK_READ_UUID, "line", "Disk Read Per Node"},
		{"{{CHART_CPU}}", WIDGET_CHART_CPU_UUID, "line", "CPU Usage Per Node"},
		{"{{CHART_DISK_WRITE}}", WIDGET_CHART_DISK_WRITE_UUID, "line", "Disk Write Per Node"},
		{"{{CHART_DISK_ALL_USAGE}}", WIDGET_CHART_DISK_ALL_USAGE_UUID, "pie", "Disk Usage Distribution"},
		{"{{CHART_DISK_USAGE}}", WIDGET_CHART_DISK_USAGE_UUID, "line", "Used Disk Space Per Node"},
		{"{{CHART_COORDINATOR_READS}}", WIDGET_CHART_COORDINATOR_READS_UUID, "pie", "Coordinator Reads Distribution"},
		{"{{CHART_COORDINATOR_WRITES}}", WIDGET_CHART_COORDINATOR_WRITES_UUID, "pie", "Coordinator Writes Distribution"},
		{"{{CHART_COORDINATOR_READ_THROUGHPUT}}", WIDGET_CHART_COORDINATOR_READ_THROUGHPUT_UUID, "line", "Coordinator Read Throughput"},
		{"{{CHART_COORDINATOR_WRITE_THROUGHPUT}}", WIDGET_CHART_COORDINATOR_WRITE_THROUGHPUT_UUID, "line", "Coordinator Write Throughput"},
		{"{{CHART_COORDINATOR_READ_LATENCY}}", WIDGET_CHART_COORDINATOR_READ_LATENCY_UUID, "line", "Coordinator Read Latency"},
		{"{{CHART_COORDINATOR_WRITE_LATENCY}}", WIDGET_CHART_COORDINATOR_WRITE_LATENCY_UUID, "line", "Coordinator Write Latency"},
	}

	for i, cfg := range chartConfigs {
		data, err := metrics.GetChartImage(sharedChartVars, cfg.widgetUuid, cfg.chartType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting chart image for widget %s: %v\n", cfg.widgetUuid, err)
			continue
		}
		if len(data) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: empty chart image returned for widget %s\n", cfg.widgetUuid)
			continue
		}

		// Detect image format from magic bytes
		ext := detectImageFormat(data)
		fmt.Printf("Received chart image for widget %s (idx=%d): %d bytes (format: %s)\n", cfg.widgetUuid, i, len(data), ext)

		// mdtopdf only supports PNG and JPEG; skip unsupported formats
		if ext != "png" && ext != "jpg" {
			fmt.Fprintf(os.Stderr, "Warning: unsupported image format '%s' for widget %s, skipping\n", ext, cfg.widgetUuid)
			continue
		}

		if ext == "png" {
			data = converter.AddTitleToImage(data, cfg.title) // add title first
		}

		images = append(images, converter.ImageData{
			Placeholder: cfg.placeholder,
			Data:        data,
			Filename:    fmt.Sprintf("chart_%d.%s", i, ext),
		})
		fmt.Printf("Added chart %d: placeholder=%s filename=%s\n", i, cfg.placeholder, fmt.Sprintf("chart_%d.%s", i, ext))
	}

	fmt.Printf("Total charts fetched: %d/%d\n", len(images), len(chartConfigs))
	for _, img := range images {
		fmt.Printf("  -> placeholder=%s filename=%s size=%d bytes\n", img.Placeholder, img.Filename, len(img.Data))
	}

	// Prepend logo if available
	if logoBytes, err := os.ReadFile("templates/logo.png"); err == nil {
		images = append([]converter.ImageData{{
			Placeholder: "{{LOGO}}",
			Data:        logoBytes,
			Filename:    "logo.png",
		}}, images...)
	}

	// Prepare report data
	loc, _ := time.LoadLocation(timeZone)
	now := time.Now().In(loc)

	// Convert Unix timestamps to formatted dates
	dateFormat := "2006-01-02 15:04:05"
	fromUnix, _ := strconv.ParseInt(from, 10, 64)
	toUnix, _ := strconv.ParseInt(to, 10, 64)
	dateFromFormatted := time.Unix(fromUnix, 0).In(loc).Format(dateFormat)
	dateToFormatted := time.Unix(toUnix, 0).In(loc).Format(dateFormat)

	// Get cluster details directly for the requested cluster
	matchedCluster, err := metrics.GetClusterDetails(org, clusterType, clusterName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cluster details: %v\n", err)
	}

	// Fetch backup/snapshot data and build the backups markdown section
	backupsSection := buildBackupsSection(org, clusterType, clusterName)

	// Fetch security events and build the security section
	var securitySB strings.Builder
	for _, eventType := range []string{"authentication", "authorization"} {
		securitySB.WriteString(buildSecuritySection(org, clusterType, clusterName, eventType, from, to, loc, matchedCluster.NodeIdentifiersByHostID))
	}
	securitySection := securitySB.String()

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
		SecuritySection:  securitySection,
		Consistency:      consistency,
		Percentile:       percentile,
		GroupBy:          groupBy,
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

				// Render errors as blockquotes so they display in red
				uniqueErrors := deduplicateErrors(fb.FailureMessages)
				for _, msg := range uniqueErrors {
					sb.WriteString(fmt.Sprintf("> %s\n\n", msg))
				}
			}
		}
	}

	return sb.String()
}

// buildSecuritySection fetches security events for a given eventType and renders
// a markdown string for the security section of the report.
func buildSecuritySection(org, clusterType, clusterName, eventType, from, to string, loc *time.Location, nodeIdentifiers map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("##### %s\n\n", strings.Title(eventType)))

	eventsResp, err := metrics.GetEvents(org, clusterType, clusterName, eventType, from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting %s events: %v\n", eventType, err)
		sb.WriteString(fmt.Sprintf("No Failed %ss during this period.\n\n", strings.Title(eventType)))
		return sb.String()
	}

	if len(eventsResp.Data) == 0 {
		sb.WriteString(fmt.Sprintf("No Failed %ss during this period.\n\n", strings.Title(eventType)))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("**%d event(s) found.**\n\n", len(eventsResp.Data)))

	for i, event := range eventsResp.Data {
		eventTime := time.Unix(event.Time/1000, (event.Time%1000)*int64(time.Millisecond)).In(loc)
		sb.WriteString(fmt.Sprintf("###### Event %d\n\n", i+1))
		sb.WriteString("```text\n")
		sb.WriteString(fmt.Sprintf("Time       : %s\n", eventTime.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("Type       : %s\n", event.Type))
		sb.WriteString(fmt.Sprintf("Source     : %s\n", event.Source))
		host := event.HostID
		if id, ok := nodeIdentifiers[event.HostID]; ok && id != "" {
			host = id
		}
		sb.WriteString(fmt.Sprintf("Host       : %s\n", host))
		sb.WriteString(fmt.Sprintf("Message    : %s\n", event.Message))
		sb.WriteString("```\n\n")
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

// detectImageFormat returns the file extension based on magic bytes.
func detectImageFormat(data []byte) string {
	if len(data) < 8 {
		return "unknown"
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpg"
	}
	// GIF: 47 49 46
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "gif"
	}
	// SVG: starts with '<' (XML)
	if data[0] == 0x3C {
		return "svg"
	}
	// WebP: RIFF....WEBP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "webp"
	}
	return "unknown"
}
