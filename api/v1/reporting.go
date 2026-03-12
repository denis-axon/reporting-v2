package v1

import (
	"fmt"
	"os"
	"strconv"
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
	// clusterDetails, err := axonserver.GetClusterDetails(org, clusterType, clusterName)
	err := metrics.GetClusters(org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cluster details: %v\n", err)
		// Continue with default values if cluster details fail
		// clusterDetails = axonserver.ClusterDetails{}
	}

	reportData := converter.ReportData{
		Organization: org,
		Dashboard:    "Reporting",
		DateFrom:     dateFromFormatted,
		DateTo:       dateToFormatted,
		Timezone:     timeZone,
		GeneratedAt:  now.Format(dateFormat),
		ClusterType:  clusterType,
		ClusterName:  clusterName,
		// NodeCount:        strconv.Itoa(clusterDetails.NodeCount),
		// DataCenters:      clusterDetails.DataCenters,
		// CassandraVersion: clusterDetails.CassandraVersion,
		// OSVersion:        clusterDetails.OSVersion,
		// JavaVersion:      clusterDetails.JavaVersion,
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
