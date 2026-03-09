package v1

import (
	"fmt"
	"os"
	"sync"

	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/internal/converter"

	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/google/uuid"
)

// TOOD: should they be added to env vars?
var WIDGET_CHART_CPU_UUID = "c11b97f0-6b2e-40cd-abc6-b721e38778b9"
var WIDGET_CHART_MEMORY_UUID = "c11b97f0-6b2e-40cd-abc6-b721e38778b9"

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

	// Initialize metrics client for this org
	if err := metrics.Init(org); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing metrics client: %v\n", err)
		utils.ReturnError(c, err)
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
			data, err := metrics.GetChartImage(org, clusterName, clusterType, from, to, timeZone, c.widgetUuid) // pass widget UUID
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting chart image for widget %s: %v\n", c.widgetUuid, err)
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
