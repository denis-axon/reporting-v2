package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	apiv1 "github.com/denis-axon/reporting-v2/api/v1"
	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/config"
	"github.com/gin-gonic/gin"

	// "github.com/denis-axon/reporting-v2/components/cloudapi"
	log "bitbucket.org/digitalisio/go/logger"
)

func main() {
	// Enable pprof on localhost port 6060
	go func() {
		_ = http.ListenAndServe("127.0.0.1:6060", nil)
	}()
	r := gin.New()
	setupRoutes(r)

	// Start the HTTP server in the background
	go func() {
		err := r.Run(config.GetInstance().ListenAddress) // listen and serve on 0.0.0.0:8081 by default
		if err != nil {
			log.Error(fmt.Sprintf("HTTP server exited with error: %s", err.Error()))
		}
	}()

	// Wait for an exit signal
	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)
	signal.Notify(signalChan, syscall.SIGTERM, os.Interrupt)
	s := <-signalChan
	log.Warn(fmt.Sprintf("captured %v signal, stopping gracefully...", s))
	signal.Stop(signalChan)

	// err := metrics.Init("testorg3") // Initialize metrics client
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error initializing metrics client: %v\n", err)
	// 	os.Exit(1)
	// }
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
	// handleGeneratePDF()

	// test fetching Cloud API
	// orgs, err := cloudapi.ListOrgs()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error listing orgs: %v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("Orgs: %+v\n", orgs)

	// validate args
	// if len(os.Args) < 2 {
	// 	fmt.Fprintf(os.Stderr, "No args provided\n")
	// 	os.Exit(1)
	// }

	// // fetch clusters for org if only 1 arg provided, otherwise convert markdown to PDF if 2 args provided
	// if len(os.Args) == 2 {
	// 	orgId := os.Args[1]
	// 	cl, err := axonserver.GetClusters(orgId)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error getting clusters for org %s: %v\n", orgId, err)
	// 		os.Exit(1)
	// 	}
	// 	fmt.Printf("Clusters for org %s: %+v\n", orgId, cl)
	// 	fmt.Printf("Successfully fetched clusters for org %s\n", orgId)
	// 	os.Exit(0)
	// }

	// // if we have 2 args, convert markdown to PDF
	// if len(os.Args) == 3 {
	// 	inputFile := os.Args[1]
	// 	outputFile := os.Args[2]

	// 	if err := converter.MarkdownToPDF(inputFile, outputFile); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error converting %s to PDF: %v\n", inputFile, err)
	// 		os.Exit(1)
	// 	}

	// 	fmt.Printf("Successfully converted %s to %s\n", inputFile, outputFile)
	// 	os.Exit(0)
	// }

	// // if we have more than 2 args, print usage and exit
	// fmt.Fprintf(os.Stderr, "Invalid number of arguments. Usage:\n")
	// fmt.Fprintf(os.Stderr, "  %s <orgId> - Fetch clusters for the given org\n", os.Args[0])
	// fmt.Fprintf(os.Stderr, "  %s <input.md> <output.pdf> - Convert Markdown to PDF\n", os.Args[0])
	// os.Exit(1)
}

func authenticationCheck(c *gin.Context) {
	fmt.Println("Authentication check!")
	suppliedToken := c.GetHeader("X-AxonOps-Auth")
	expectedToken := config.AuthToken()
	// fmt.Printf("Authentication check: supplied token '%s', expected token '%s'\n", suppliedToken, expectedToken)
	if suppliedToken != expectedToken {
		c.JSON(http.StatusUnauthorized, utils.Response{Error: "Unauthorized"})
		c.Abort()
	}
}

func setupRoutes(r *gin.Engine) {
	// Health check endpoints
	r.GET("/healthz", apiv1.HealthCheck)

	// v1 API
	v1 := r.Group("/v1")
	// we don't use authentication for now because we'll need this to work
	// for both SaaS and on-prem installs and the authentication works differently in these environments
	// v1.Use(authenticationCheck)

	// Authenticated dummy endpoint
	v1.GET("/authcheck", apiv1.AuthCheck)

	v1.GET("/reporting", apiv1.GeneratePDF)

	// ensure there is a newline in the output because it doesn't always display correctly and it's annoying!
	fmt.Println()
}
