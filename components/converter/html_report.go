package converter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// HTMLToPDF converts an HTML file to a PDF using the WeasyPrint CLI tool.
// It executes: weasyprint <inputFile> <outputFile>
func HTMLToPDF(inputFile string, outputFile string) error {
	cmd := exec.Command("weasyprint", inputFile, outputFile)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w\noutput: %s", err, out.String())
	}

	return nil
}

// GenerateHTMLReportPDF creates a PDF from an HTML template using WeasyPrint.
// Chart images are embedded as base64 data URIs so WeasyPrint needs no file access.
// The logo (if present at templates/logo.png) is also embedded as a data URI.
func GenerateHTMLReportPDF(templatePath string, outputPath string, images []ImageData, data ReportData) error {
	// Create a unique temp directory — WeasyPrint writes the PDF here first.
	tempDir := filepath.Join(os.TempDir(), "pdf-gen-"+uuid.New().String())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Read the HTML template.
	htmlBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read HTML template: %w", err)
	}
	content := string(htmlBytes)

	// ── Text placeholder substitution ───────────────────────────────────────
	content = strings.Replace(content, "{{ORGANIZATION}}", escapeHTML(data.Organization), 1)
	content = strings.Replace(content, "{{DASHBOARD}}", escapeHTML(data.Dashboard), 1)
	content = strings.Replace(content, "{{DATE_FROM}}", escapeHTML(data.DateFrom), 1)
	content = strings.Replace(content, "{{DATE_TO}}", escapeHTML(data.DateTo), 1)
	content = strings.Replace(content, "{{TIMEZONE}}", escapeHTML(data.Timezone), 1)
	content = strings.Replace(content, "{{GENERATED_AT}}", escapeHTML(data.GeneratedAt), 1)
	content = strings.Replace(content, "{{CLUSTER_TYPE}}", escapeHTML(data.ClusterType), 1)
	content = strings.Replace(content, "{{CLUSTER_NAME}}", escapeHTML(data.ClusterName), 1)
	content = strings.Replace(content, "{{NODE_COUNT}}", escapeHTML(data.NodeCount), 1)
	content = strings.Replace(content, "{{DATA_CENTERS}}", escapeHTML(data.DataCenters), 1)
	content = strings.Replace(content, "{{CASSANDRA_VERSION}}", escapeHTML(data.CassandraVersion), 1)
	content = strings.Replace(content, "{{OS_VERSION}}", escapeHTML(data.OSVersion), 1)
	content = strings.Replace(content, "{{JAVA_VERSION}}", escapeHTML(data.JavaVersion), 1)
	// BackupsSection and SecuritySection are already HTML — do NOT escape.
	content = strings.Replace(content, "{{BACKUPS_SECTION}}", data.BackupsSection, 1)
	content = strings.Replace(content, "{{SECURITY_SECTION}}", data.SecuritySection, 1)
	content = strings.ReplaceAll(content, "$consistency", escapeHTML(data.Consistency))
	content = strings.ReplaceAll(content, "$percentile", escapeHTML(data.Percentile))
	content = strings.ReplaceAll(content, "$groupBy", escapeHTML(data.GroupBy))

	// ── Logo ─────────────────────────────────────────────────────────────────
	logoTag := ""
	if logoBytes, err := os.ReadFile("templates/logo.png"); err == nil {
		logoTag = fmt.Sprintf(`<img class="logo" src="data:image/png;base64,%s" alt="Logo" />`,
			base64.StdEncoding.EncodeToString(logoBytes))
	}
	content = strings.Replace(content, "{{LOGO}}", logoTag, 1)

	// ── Chart images (base64 data URIs) ──────────────────────────────────────
	for _, img := range images {
		if !strings.Contains(content, img.Placeholder) {
			fmt.Printf("WARNING: placeholder %s not found in HTML template\n", img.Placeholder)
			continue
		}
		mime := "image/png"
		if strings.HasSuffix(img.Filename, ".jpg") {
			mime = "image/jpeg"
		}
		imgTag := fmt.Sprintf(`<img src="data:%s;base64,%s" alt="%s" style="width:100%%;height:auto;" />`,
			mime, base64.StdEncoding.EncodeToString(img.Data), img.Placeholder)
		content = strings.Replace(content, img.Placeholder, imgTag, 1)
		fmt.Printf("Replaced placeholder %s with base64 image (%d bytes)\n", img.Placeholder, len(img.Data))
	}

	// Remove any placeholders that were not filled (e.g. missing charts).
	for _, cfg := range []string{
		"{{CHART_DISK_USAGE}}", "{{CHART_DISK_READ}}", "{{CHART_CPU}}",
		"{{CHART_DISK_WRITE}}", "{{CHART_DISK_ALL_USAGE}}",
		"{{CHART_COORDINATOR_READS}}", "{{CHART_COORDINATOR_WRITES}}",
		"{{CHART_COORDINATOR_READ_THROUGHPUT}}", "{{CHART_COORDINATOR_WRITE_THROUGHPUT}}",
		"{{CHART_COORDINATOR_READ_LATENCY}}", "{{CHART_COORDINATOR_WRITE_LATENCY}}",
	} {
		content = strings.ReplaceAll(content, cfg, "")
	}

	// ── Write rendered HTML to temp file ─────────────────────────────────────
	htmlPath := filepath.Join(tempDir, "report.html")
	if err := os.WriteFile(htmlPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write rendered HTML: %w", err)
	}

	// ── Run WeasyPrint ────────────────────────────────────────────────────────
	tempPDF := filepath.Join(tempDir, "output.pdf")
	cmd := exec.Command("weasyprint", htmlPath, tempPDF)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("weasyprint failed: %w\noutput: %s", err, out.String())
	}

	// ── Copy to final output path ─────────────────────────────────────────────
	pdfData, err := os.ReadFile(tempPDF)
	if err != nil {
		return fmt.Errorf("failed to read weasyprint output: %w", err)
	}
	if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
		return fmt.Errorf("failed to write final PDF: %w", err)
	}

	return nil
}

// escapeHTML replaces the five characters that must be escaped inside HTML text nodes.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
