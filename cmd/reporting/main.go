package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/ledongthuc/pdf"
)

func main() {
	// Define command line flags
	markdownFile := flag.String("input", "input.md", "Path to the input markdown file")
	pdfFile := flag.String("output", "output.pdf", "Path to the output PDF file")
	flag.Parse()

	// Read the markdown file
	content, err := os.ReadFile(*markdownFile)
	if err != nil {
		fmt.Println("Error reading markdown file:", err)
		return
	}

	// Convert markdown to PDF
	pdf := pdf.NewPDFWriter()
	pdf.AddPage()
	page := pdf.GetPage(1)
	page.DrawText(10, 10, string(content))

	// Save the PDF to file
	err = pdf.WriteFile(*pdfFile)
	if err != nil {
		fmt.Println("Error writing PDF file:", err)
	}

	fmt.Printf("Converted %s to %s successfully!\n", *markdownFile, *pdfFile)
}