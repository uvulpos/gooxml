// Copyright 2017 Baliance. All rights reserved.
package main

import (
	"fmt"

	"github.com/uvulpos/gooxml/document"
	"github.com/uvulpos/gooxml/measurement"
	"github.com/uvulpos/gooxml/schema/soo/wml"
)

var lorem = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin lobortis, lectus dictum feugiat tempus, sem neque finibus enim, sed eleifend sem nunc ac diam. Vestibulum tempus sagittis elementum`

func main() {
	doc := document.New()

	// Build the document body first: a set of headings at different levels with
	// some body text in between.
	nd := doc.Numbering.AddDefinition()
	for i := 0; i < 9; i++ {
		lvl := nd.AddLevel()
		lvl.SetFormat(wml.ST_NumberFormatDecimal)
		lvl.SetAlignment(wml.ST_JcLeft)
		lvl.Properties().SetLeftIndent(0.5 * measurement.Distance(i) * measurement.Inch)
	}

	for i := 0; i < 4; i++ {
		para := doc.AddParagraph()
		para.SetNumberingDefinition(nd)
		para.Properties().SetHeadingLevel(1)
		para.AddRun().AddText(fmt.Sprintf("First Level %d", i+1))
		doc.AddParagraph().AddRun().AddText(lorem)

		for j := 0; j < 3; j++ {
			para := doc.AddParagraph()
			para.SetNumberingDefinition(nd)
			para.Properties().SetHeadingLevel(2)
			para.AddRun().AddText(fmt.Sprintf("Second Level %d.%d", i+1, j+1))
			doc.AddParagraph().AddRun().AddText(lorem)

			para = doc.AddParagraph()
			para.SetNumberingDefinition(nd)
			para.Properties().SetHeadingLevel(3)
			para.AddRun().AddText(fmt.Sprintf("Third Level %d.%d.1", i+1, j+1))
			doc.AddParagraph().AddRun().AddText(lorem)
		}
	}

	// Now generate a table of contents from the headings we just added and place
	// it at the top of the document. Unlike Run.AddFieldTOC - which inserts an
	// empty field that the reader must recalculate - GenerateTableOfContents
	// writes the current entries (heading text, hyperlink and PAGEREF page
	// number) directly into the document, so the directory is already filled in
	// when the file is opened.
	doc.GenerateTableOfContents(true, document.DefaultTOCOptions())

	doc.SaveToFile("toc-generate.docx")
}
