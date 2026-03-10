package converter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// MarkdownToPDF converts a markdown file to a PDF
func MarkdownToPDF(inputFile string, outputFile string) error {
	// Read the markdown file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	// Convert markdown to HTML
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	var buf bytes.Buffer
	if err := md.Convert(data, &buf); err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}

	// Wrap in HTML with styling
	html := wrapHTML(buf.String(), nil)

	// Generate PDF using Chrome
	return htmlToPDF(html, outputFile)
}

// ImageData holds image bytes and its placeholder name
type ImageData struct {
	Placeholder string // e.g., "{{CHART_1}}", "{{CHART_2}}"
	Data        []byte
	Filename    string // e.g., "chart_1.png"
}

// ReportData holds text data for report template placeholders
type ReportData struct {
	Organization string
	Dashboard    string
	DateFrom     string
	DateTo       string
	Timezone     string
	GeneratedAt  string
}

// GeneratePDFWithImages creates a PDF from markdown template with embedded images
func GeneratePDFWithImages(templatePath string, outputPath string, images []ImageData, data ReportData) error {
	// Create unique temp directory for this request
	tempDir := filepath.Join(os.TempDir(), "pdf-gen-"+uuid.New().String())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup after PDF generation

	// Read markdown template
	mdContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	content := string(mdContent)

	// Replace text placeholders with report data
	content = strings.Replace(content, "{{ORGANIZATION}}", data.Organization, 1)
	content = strings.Replace(content, "{{DASHBOARD}}", data.Dashboard, 1)
	content = strings.Replace(content, "{{DATE_FROM}}", data.DateFrom, 1)
	content = strings.Replace(content, "{{DATE_TO}}", data.DateTo, 1)
	content = strings.Replace(content, "{{TIMEZONE}}", data.Timezone, 1)
	content = strings.Replace(content, "{{GENERATED_AT}}", data.GeneratedAt, 1)

	// Save each image and replace placeholder
	for _, img := range images {
		imgPath := filepath.Join(tempDir, img.Filename)
		if err := os.WriteFile(imgPath, img.Data, 0644); err != nil {
			return fmt.Errorf("failed to write image %s: %w", img.Filename, err)
		}
		// Replace placeholder with markdown image reference
		mdImage := fmt.Sprintf("![%s](%s)", img.Filename, imgPath)
		content = strings.Replace(content, img.Placeholder, mdImage, 1)
	}

	// Write processed markdown to temp file
	tempMdPath := filepath.Join(tempDir, "report.md")
	if err := os.WriteFile(tempMdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp markdown: %w", err)
	}

	// Generate PDF
	pf := mdtopdf.NewPdfRenderer("P", "A4", outputPath, "", nil, mdtopdf.LIGHT)
	return pf.Process([]byte(content))
}
