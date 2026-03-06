package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/denis-axon/reporting-v2/components/axonserver"
	"github.com/denis-axon/reporting-v2/internal/converter"

	// "github.com/denis-axon/reporting-v2/components/cloudapi"
	"encoding/base64"
	"path/filepath"

	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/google/uuid"
)

// Convert bytes to base64 markdown image
func ImageToMarkdown(data []byte, alt string) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("![%s](data:image/png;base64,%s)", alt, b64)
}

// func handleGeneratePDF(w http.ResponseWriter, r *http.Request) {
func handleGeneratePDF() {
	// Fetch all chart images concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var images []converter.ImageData

	chartConfigs := []struct {
		placeholder string
		widgetUuid  string
	}{
		{"{{CHART_CPU}}", "uuid-1"},
		{"{{CHART_MEMORY}}", "uuid-2"},
		{"{{CHART_DISK}}", "uuid-3"},
	}

	for i, cfg := range chartConfigs {
		wg.Add(1)
		go func(idx int, c struct{ placeholder, widgetUuid string }) {
			defer wg.Done()
			// data, err := metrics.GetChartImage(c.widgetUuid) // pass widget UUID
			data, err := metrics.GetChartImage() // pass widget UUID
			if err != nil {
				return // handle error appropriately
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

	// Generate PDF with unique output path
	outputPath := filepath.Join(os.TempDir(), uuid.New().String()+".pdf")
	err := converter.GeneratePDFWithImages("templates/report.md", outputPath, images)
	if err != nil {
		// http.Error(w, err.Error(), 500)
		fmt.Fprintf(os.Stderr, "Error generating PDF: %v\n", err)
		return
	}
	// defer os.Remove(outputPath)

	// Serve PDF
	// http.ServeFile(w, r, outputPath)
	fmt.Printf("PDF generated successfully at %s\n", outputPath)
}

func main() {
	err := metrics.Init("testorg3") // Initialize metrics client
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing metrics client: %v\n", err)
		os.Exit(1)
	}
	// err, healthy := metrics.Healthy() // Check if metrics client is healthy
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error checking metrics client health: %v\n", err)
	// 	os.Exit(1)
	// }
	// if !healthy {
	// 	fmt.Fprintf(os.Stderr, "Metrics client is not healthy\n")
	// 	os.Exit(1)
	// }
	// fmt.Println("Metrics client is healthy")
	// byteData, err := metrics.GetChartImage()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error getting chart image: %v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("Chart image data: %v\n", byteData)
	// os.WriteFile("chart.png", byteData, 0644)
	handleGeneratePDF()

	// test fetching Cloud API
	// orgs, err := cloudapi.ListOrgs()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error listing orgs: %v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("Orgs: %+v\n", orgs)

	// validate args
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "No args provided\n")
		os.Exit(1)
	}

	// fetch clusters for org if only 1 arg provided, otherwise convert markdown to PDF if 2 args provided
	if len(os.Args) == 2 {
		orgId := os.Args[1]
		cl, err := axonserver.GetClusters(orgId)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting clusters for org %s: %v\n", orgId, err)
			os.Exit(1)
		}
		fmt.Printf("Clusters for org %s: %+v\n", orgId, cl)
		fmt.Printf("Successfully fetched clusters for org %s\n", orgId)
		os.Exit(0)
	}

	// if we have 2 args, convert markdown to PDF
	if len(os.Args) == 3 {
		inputFile := os.Args[1]
		outputFile := os.Args[2]

		if err := converter.MarkdownToPDF(inputFile, outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting %s to PDF: %v\n", inputFile, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully converted %s to %s\n", inputFile, outputFile)
		os.Exit(0)
	}

	// if we have more than 2 args, print usage and exit
	fmt.Fprintf(os.Stderr, "Invalid number of arguments. Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s <orgId> - Fetch clusters for the given org\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s <input.md> <output.pdf> - Convert Markdown to PDF\n", os.Args[0])
	os.Exit(1)
}
