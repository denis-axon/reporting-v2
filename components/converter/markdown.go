package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mandolyte/mdtopdf"
)

// MarkdownToPDF converts a markdown file to a PDF
func MarkdownToPDF(inputFile string, outputFile string) error {
	// Read the markdown file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	// Create a new PDF renderer (Portrait, A4, output file, LIGHT theme)
	pf := mdtopdf.NewPdfRenderer("P", "A4", outputFile, "", nil, mdtopdf.LIGHT)

	// Process the markdown content into PDF
	err = pf.Process(data)
	if err != nil {
		return err
	}

	return nil
}

// ImageData holds image bytes and its placeholder name
type ImageData struct {
	Placeholder string // e.g., "{{CHART_1}}", "{{CHART_2}}"
	Data        []byte
	Filename    string // e.g., "chart_1.png"
}

// ReportData holds text data for report template placeholders
type ReportData struct {
	Organization     string
	Dashboard        string
	DateFrom         string
	DateTo           string
	Timezone         string
	GeneratedAt      string
	ClusterType      string
	ClusterName      string
	NodeCount        string
	DataCenters      string
	CassandraVersion string
	JavaVersion      string
	OSVersion        string
	BackupsSection   string
	SecuritySection  string
	Consistency      string
	Percentile       string
	GroupBy          string
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
	content = strings.Replace(content, "{{CLUSTER_TYPE}}", data.ClusterType, 1)
	content = strings.Replace(content, "{{CLUSTER_NAME}}", data.ClusterName, 1)
	content = strings.Replace(content, "{{NODE_COUNT}}", data.NodeCount, 1)
	content = strings.Replace(content, "{{DATA_CENTERS}}", data.DataCenters, 1)
	content = strings.Replace(content, "{{CASSANDRA_VERSION}}", data.CassandraVersion, 1)
	content = strings.Replace(content, "{{OS_VERSION}}", data.OSVersion, 1)
	content = strings.Replace(content, "{{JAVA_VERSION}}", data.JavaVersion, 1)
	content = strings.Replace(content, "{{BACKUPS_SECTION}}", data.BackupsSection, 1)
	content = strings.Replace(content, "{{SECURITY_SECTION}}", data.SecuritySection, 1)
	content = strings.ReplaceAll(content, "$consistency", data.Consistency)
	content = strings.ReplaceAll(content, "$percentile", data.Percentile)
	content = strings.ReplaceAll(content, "$groupBy", data.GroupBy)

	// Save each image and replace placeholder
	for _, img := range images {
		imgPath := filepath.Join(tempDir, img.Filename)
		if err := os.WriteFile(imgPath, img.Data, 0644); err != nil {
			return fmt.Errorf("failed to write image %s: %w", img.Filename, err)
		}
		mdImage := fmt.Sprintf("![](%s)", imgPath)
		if strings.Contains(content, img.Placeholder) {
			fmt.Printf("Replacing placeholder %s with image %s\n", img.Placeholder, imgPath)
			content = strings.Replace(content, img.Placeholder, mdImage, 1)
		} else {
			fmt.Printf("WARNING: placeholder %s not found in template!\n", img.Placeholder)
		}
	}

	// Write to a temp file first, then rename to output path
	// This prevents partial/corrupt files if generation fails
	tempOutputPath := filepath.Join(tempDir, "output.pdf")

	// Generate PDF — create renderer pointing to temp output
	pf := mdtopdf.NewPdfRenderer("P", "A4", tempOutputPath, "", nil, mdtopdf.LIGHT)

	// Use Helvetica with no bold for all headings to achieve a thin appearance
	thinHeading := func(size, spacing float64) mdtopdf.Styler {
		return mdtopdf.Styler{
			Font:      "Helvetica",
			Style:     "",
			Size:      size,
			Spacing:   spacing,
			TextColor: mdtopdf.Color{Red: 0, Green: 0, Blue: 0},
			FillColor: mdtopdf.Color{Red: 255, Green: 255, Blue: 255},
		}
	}
	pf.H1 = thinHeading(28, 8)
	pf.H2 = thinHeading(22, 6)
	pf.H3 = thinHeading(18, 5)
	pf.H4 = thinHeading(14, 4)
	pf.H5 = thinHeading(12, 3)
	pf.H6 = thinHeading(11, 3)

	// Override both header and body with light colors
	lightStyle := mdtopdf.Styler{
		Font:      "Arial",
		Style:     "",
		Size:      12,
		Spacing:   2,
		TextColor: mdtopdf.Color{Red: 0, Green: 0, Blue: 0},
		FillColor: mdtopdf.Color{Red: 255, Green: 255, Blue: 255},
	}
	pf.THeader = mdtopdf.Styler{
		Font:      "Arial",
		Style:     "B",
		Size:      12,
		Spacing:   2,
		TextColor: mdtopdf.Color{Red: 0, Green: 0, Blue: 0},
		FillColor: mdtopdf.Color{Red: 245, Green: 245, Blue: 245},
	}
	pf.TBody = lightStyle

	pf.Blockquote = mdtopdf.Styler{
		Font:      "Arial",
		Style:     "",
		Size:      11,
		Spacing:   2,
		TextColor: mdtopdf.Color{Red: 192, Green: 0, Blue: 0},
		FillColor: mdtopdf.Color{Red: 255, Green: 255, Blue: 255},
	}
	pf.UpdateBlockquoteStyler()

	// Also lighten the page background
	pf.Backtick = mdtopdf.Styler{
		Font:      "Courier",
		Style:     "",
		Size:      12,
		Spacing:   2,
		TextColor: mdtopdf.Color{Red: 0, Green: 0, Blue: 0},
		FillColor: mdtopdf.Color{Red: 255, Green: 255, Blue: 255},
	}

	// Process markdown to PDF — only call this ONCE
	if err := pf.Process([]byte(content)); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Read the generated PDF and write to final output path
	pdfData, err := os.ReadFile(tempOutputPath)
	if err != nil {
		return fmt.Errorf("failed to read generated PDF: %w", err)
	}

	if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
		return fmt.Errorf("failed to write final PDF: %w", err)
	}

	return nil
}
