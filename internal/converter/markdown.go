package converter

import (
	"github.com/mandolyte/mdtopdf"
	"io/ioutil"
	"log"
)

// MarkdownToPDF converts a markdown file to a PDF
func MarkdownToPDF(inputFile string, outputFile string) error {
	// Read the markdown file
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}

	// Convert to PDF
	err = md2pdf.Convert(data, outputFile)
	if err != nil {
		return err
	}

	return nil
}