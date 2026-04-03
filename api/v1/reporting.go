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
	err = converter.GenerateHTMLReportPDF("templates/report.html", outputPath, images, reportData)
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

func buildBackupsSection(org, clusterType, clusterName string) string {
	snapshotResp, err := metrics.GetCassandraSnapshot(org, clusterType, clusterName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cassandra snapshot: %v\n", err)
		return "<p>No backup data available.</p>"
	}

	summaries := metrics.GetBackupSummaries(snapshotResp)
	if len(summaries) == 0 {
		return "<p>No backup schedules found.</p>"
	}

	var sb strings.Builder

	for i, summary := range summaries {
		// Schedule badge only (shortened label, outlined style)
		scheduleBadge := ""
		if summary.ScheduleExpr != "" {
			scheduleBadge = fmt.Sprintf(`<span class="badge badge-schedule">%s</span>`,
				escapeHTMLStr(shortenScheduleExpr(strings.ToUpper(summary.ScheduleExpr))))
		}

		sb.WriteString(`<div class="backup-card">`)

		// ── Card top: header + stats kept together on one page ───────────────
		sb.WriteString(`<div class="backup-card-top">`)

		// ── Card header ──────────────────────────────────────────────────────
		sb.WriteString(`<div class="backup-card-header">`)
		sb.WriteString(`<div>`)
		sb.WriteString(fmt.Sprintf(`<div class="schedule-title">Schedule %d: %s</div>`, i+1, escapeHTMLStr(summary.Tag)))
		sb.WriteString(`</div>`)
		sb.WriteString(`<div style="display:flex;gap:6px;align-items:center;">`)
		sb.WriteString(scheduleBadge)
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`) // backup-card-header

		// ── Stats body ───────────────────────────────────────────────────────
		sb.WriteString(`<div class="backup-card-body">`)
		sb.WriteString(`<div class="backup-stats-row">`)

		// Left cell: Tag / Data Centers / Remote Type
		sb.WriteString(`<div class="backup-stats-cell">`)
		sb.WriteString(`<div class="stat-inline-row">`)
		sb.WriteString(`<div class="stat-inline-label">Tag:</div>`)
		sb.WriteString(fmt.Sprintf(`<div class="stat-inline-value">%s</div>`, escapeHTMLStr(summary.Tag)))
		sb.WriteString(`</div>`)
		sb.WriteString(`<div class="stat-inline-row">`)
		sb.WriteString(`<div class="stat-inline-label">Data Centers:</div>`)
		sb.WriteString(fmt.Sprintf(`<div class="stat-inline-value">%s</div>`, escapeHTMLStr(summary.Datacenters)))
		sb.WriteString(`</div>`)
		sb.WriteString(`<div class="stat-inline-row">`)
		sb.WriteString(`<div class="stat-inline-label">Remote Type:</div>`)
		sb.WriteString(fmt.Sprintf(`<div class="stat-inline-value">%s</div>`, escapeHTMLStr(summary.RemoteType)))
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`) // backup-stats-cell left

		// Right cell: counts (each pair wrapped in stat-row for horizontal layout)
		sb.WriteString(`<div class="backup-stats-cell">`)
		sb.WriteString(`<div class="stat-row">`)
		sb.WriteString(`<div class="stat-label">Successful Backups:</div>`)
		sb.WriteString(fmt.Sprintf(`<div class="stat-value success">%d</div>`, summary.Successful))
		sb.WriteString(`</div>`)
		sb.WriteString(`<div class="stat-row">`)
		sb.WriteString(`<div class="stat-label">Failed Backups:</div>`)
		sb.WriteString(fmt.Sprintf(`<div class="stat-value failure">%d</div>`, summary.Failed))
		sb.WriteString(`</div>`)
		sb.WriteString(`</div>`) // backup-stats-cell right

		sb.WriteString(`</div>`) // backup-stats-row
		sb.WriteString(`</div>`) // backup-card-body

		sb.WriteString(`</div>`) // backup-card-top

		// ── Failed backup details (each entry avoids page splits) ────────────
		if summary.Failed > 0 && len(summary.FailedBackups) > 0 {
			sb.WriteString(`<div class="failed-section-heading">Failed Backups Details</div>`)

			for j, fb := range summary.FailedBackups {
				// Each grey block stays on one page — if it doesn't fit, it
				// moves entirely to the next page.
				sb.WriteString(`<div class="failed-detail-block">`)

				// Detail rows
				sb.WriteString(`<div class="detail-row">`)
				sb.WriteString(`<span class="detail-label">Backup Time :</span>`)
				sb.WriteString(fmt.Sprintf(`<span class="detail-value">%s</span>`, escapeHTMLStr(fb.BackupTime)))
				sb.WriteString(`</div>`)
				sb.WriteString(`<div class="detail-row">`)
				sb.WriteString(`<span class="detail-label">Failed Nodes :</span>`)
				sb.WriteString(fmt.Sprintf(`<span class="detail-value">%d</span>`, len(fb.FailedNodes)))
				sb.WriteString(`</div>`)

				// Node IDs — inside the same grey block
				for _, nodeID := range fb.FailedNodes {
					sb.WriteString(fmt.Sprintf(`<div class="node-id">%s</div>`, escapeHTMLStr(nodeID)))
				}

				// Error rows — inside the same grey block
				uniqueErrors := deduplicateErrors(fb.FailureMessages)
				for _, msg := range uniqueErrors {
					sb.WriteString(`<div class="error-row">`)
					sb.WriteString(`<svg class="error-dot" viewBox="0 0 14 14" xmlns="http://www.w3.org/2000/svg">` +
						`<circle cx="7" cy="7" r="7" fill="#e53935"/>` +
						`<text x="7" y="10.5" text-anchor="middle" font-family="Helvetica,Arial,sans-serif" ` +
						`font-size="9" font-weight="bold" fill="#ffffff">!</text>` +
						`</svg>`)

					sb.WriteString(fmt.Sprintf(`<div class="error-text">%s</div>`, escapeHTMLStr(msg)))
					sb.WriteString(`</div>`)
				}

				sb.WriteString(`</div>`) // close failed-detail-block
				_ = j
			}
		}

		sb.WriteString(`</div>`) // backup-card
	}

	return sb.String()
}

func buildSecuritySection(org, clusterType, clusterName, eventType, from, to string, loc *time.Location, nodeIdentifiers map[string]string) string {
	var sb strings.Builder
	title := strings.ToUpper(eventType[:1]) + eventType[1:]
	titleUpper := strings.ToUpper(eventType)

	// Choose icon colour: green tick for auth types
	iconColor := "#43a047"
	iconPath := `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="` + iconColor + `" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><polyline points="9 12 11 14 15 10"/></svg>`
	if eventType == "authorization" {
		iconPath = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fb8c00" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`
	}

	eventsResp, err := metrics.GetEvents(org, clusterType, clusterName, eventType, from, to)
	hasError := err != nil || len(eventsResp.Data) == 0

	sb.WriteString(`<div class="security-event-block">`)
	sb.WriteString(fmt.Sprintf(`<div class="sec-icon">%s</div>`, iconPath))
	sb.WriteString(`<div class="sec-content">`)
	sb.WriteString(fmt.Sprintf(`<div class="sec-type">%s</div>`, titleUpper))

	if hasError || len(eventsResp.Data) == 0 {
		sb.WriteString(fmt.Sprintf(`<div class="sec-text">No failed %ss during this period.</div>`, title))
		sb.WriteString(`</div></div>`)
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf(`<div class="sec-text"><strong>%d event(s) found.</strong></div>`, len(eventsResp.Data)))

	for i, event := range eventsResp.Data {
		eventTime := time.Unix(event.Time/1000, (event.Time%1000)*int64(time.Millisecond)).In(loc)
		host := event.HostID
		if id, ok := nodeIdentifiers[event.HostID]; ok && id != "" {
			host = id
		}
		sb.WriteString(fmt.Sprintf(`<div style="margin-top:6px;font-size:9px;color:#888;">Event %d</div>`, i+1))
		sb.WriteString(`<table class="sec-event-table">`)
		sb.WriteString(fmt.Sprintf(`<tr><td class="el">Time</td><td class="ev">%s</td></tr>`, eventTime.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf(`<tr><td class="el">Type</td><td class="ev">%s</td></tr>`, escapeHTMLStr(event.Type)))
		sb.WriteString(fmt.Sprintf(`<tr><td class="el">Source</td><td class="ev">%s</td></tr>`, escapeHTMLStr(event.Source)))
		sb.WriteString(fmt.Sprintf(`<tr><td class="el">Host</td><td class="ev">%s</td></tr>`, escapeHTMLStr(host)))
		sb.WriteString(fmt.Sprintf(`<tr><td class="el">Message</td><td class="ev">%s</td></tr>`, escapeHTMLStr(event.Message)))
		sb.WriteString(`</table>`)
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}

// escapeHTMLStr is a local alias — reporting.go can't import from converter directly
// for this helper, so we duplicate the five-char escaping here.
func escapeHTMLStr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
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

// shortenScheduleExpr abbreviates schedule expressions for the badge.
// e.g. "EVERY 15 MINUTES" → "EVERY 15 MIN"
func shortenScheduleExpr(expr string) string {
	expr = strings.ReplaceAll(expr, "MINUTES", "MIN")
	expr = strings.ReplaceAll(expr, "MINUTE", "MIN")
	expr = strings.ReplaceAll(expr, "HOURS", "HR")
	expr = strings.ReplaceAll(expr, "HOUR", "HR")
	return expr
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

// TableTest renders templates/table-test.html to a PDF via WeasyPrint and streams it back.
func TableTest(c *gin.Context) {
	outputPath := filepath.Join(os.TempDir(), uuid.New().String()+".pdf")
	err := converter.HTMLToPDF("templates/table-test.html", outputPath)
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
