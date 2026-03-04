package main

import (
	"fmt"
	"os"

	"github.com/denis-axon/reporting-v2/components/axonserver"
	"github.com/denis-axon/reporting-v2/internal/converter"
)

func main() {
	orgId := "testorg1"
	cl, err := axonserver.GetClusters(orgId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clusters for org %s: %v\n", orgId, err)
		os.Exit(1)
	}
	fmt.Printf("Clusters for org %s: %+v\n", orgId, cl)
	fmt.Printf("Successfully fetched clusters for org %s\n", orgId)
	os.Exit(0)

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input_file.md> <output_file.pdf>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	if err := converter.MarkdownToPDF(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting %s to PDF: %v\n", inputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted %s to %s\n", inputFile, outputFile)
}