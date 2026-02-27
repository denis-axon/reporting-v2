package converter

import (
	"os"

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