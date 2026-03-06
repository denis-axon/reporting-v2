package main

import (
	"fmt"
	"os"

	"github.com/denis-axon/reporting-v2/components/axonserver"
	"github.com/denis-axon/reporting-v2/internal/converter"

	// "github.com/denis-axon/reporting-v2/components/cloudapi"
	"github.com/denis-axon/reporting-v2/components/metrics"
)

func main() {
	err := metrics.Init("testorg1") // Initialize metrics client
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing metrics client: %v\n", err)
		os.Exit(1)
	}
	err, healthy := metrics.Healthy() // Check if metrics client is healthy
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking metrics client health: %v\n", err)
		os.Exit(1)
	}
	if !healthy {
		fmt.Fprintf(os.Stderr, "Metrics client is not healthy\n")
		os.Exit(1)
	}
	fmt.Println("Metrics client is healthy")

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
