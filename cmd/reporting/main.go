package main

import (
	"flag"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

func main() {
	title := flag.String("title", "Default Title", "Title of the PDF document")
	author := flag.String("author", "Default Author", "Author of the document")
	theme := flag.String("theme", "default", "Theme of the document")

	flag.Parse()

	doc := gofpdf.New("P", "mm", "A4", "")
	doc.SetFont("Arial", "B", 16)
	doc.AddPage()
	doc.Cell(40, 10, *title)

	if *theme == "dark" {
		doc.SetFillColor(0, 0, 0)
		doc.SetTextColor(255, 255, 255)
	} else {
		doc.SetFillColor(255, 255, 255)
		doc.SetTextColor(0, 0, 0)
	}

	doc.Cell(40, 10, fmt.Sprintf("Author: %s", *author))
	doc.OutputFileAndClose("output.pdf")
}
