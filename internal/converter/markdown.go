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

	// Create a PDF renderer and process the markdown content
	pf := mdtopdf.NewPdfRenderer("portrait", "A4", outputFile, "", nil, mdtopdf.LIGHT)
	return pf.Process(data)
}