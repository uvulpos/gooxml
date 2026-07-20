// Copyright 2017 Baliance. All rights reserved.
//
// Use of this source code is governed by the terms of the Affero GNU General
// Public License version 3.0 as published by the Free Software Foundation and
// appearing in the file LICENSE included in the packaging of this file. A
// commercial license can be purchased by contacting sales@baliance.com.

package document

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uvulpos/gooxml"
	"github.com/uvulpos/gooxml/measurement"
	"github.com/uvulpos/gooxml/schema/soo/ofc/sharedTypes"
	"github.com/uvulpos/gooxml/schema/soo/wml"
)

// tocBookmarkBase is the starting value for the sequentially generated bookmark
// names (`_Toc...`) that anchor generated table of contents entries. Word uses
// the same `_Toc` prefix so the entries look native and are recognized as TOC
// bookmarks by readers.
const tocBookmarkBase = 800000000

// tocEntryTabPosition is the position of the right aligned, dot-leadered tab
// stop that separates an entry's heading text from its page number. It matches a
// US-letter page with one inch margins (8.5" - 2 * 1" = 6.5").
const tocEntryTabPosition = 6.5 * measurement.Inch

// tocEntry is a single heading collected from the document body that will be
// rendered as one line of the table of contents.
type tocEntry struct {
	level    int
	text     string
	bookmark string
}

// TableOfContents is the set of paragraphs produced by
// Document.GenerateTableOfContents. It wraps a well-formed TOC field whose
// cached result already lists every heading, so the entries are visible in
// readers that do not recalculate fields on open.
type TableOfContents struct {
	d     *Document
	paras []*wml.CT_P
}

// Paragraphs returns the paragraphs that make up the table of contents.
func (t TableOfContents) Paragraphs() []Paragraph {
	ret := make([]Paragraph, 0, len(t.paras))
	for _, p := range t.paras {
		ret = append(ret, Paragraph{t.d, p})
	}
	return ret
}

// GenerateTableOfContents scans the document body for heading paragraphs and
// builds a fully populated table of contents from them. Unlike
// Run.AddFieldTOC - which only inserts an empty TOC field that readers must
// recalculate - this method writes the current entries (heading text, a
// hyperlink to the heading and a PAGEREF page number) directly into the field's
// cached result, so the directory is already filled in when the document is
// opened.
//
// Headings are matched by their built-in style (Heading1..Heading9) filtered by
// opts.OutlineLevels (e.g. "1-3"). Bookmarks are added to the matched headings
// so the entries can link to them. The document is also flagged to update its
// fields on open (see Settings.SetUpdateFieldsOnOpen) so that page numbers are
// recomputed once the reader has laid the document out.
//
// When atTop is true the table of contents is inserted before the current first
// block of the body; otherwise it is appended at the end. The generated
// paragraphs are returned so the caller can, for example, add a heading in front
// of them.
func (d *Document) GenerateTableOfContents(atTop bool, opts FieldTOCOptions) TableOfContents {
	entries := d.collectTOCEntries(opts)
	d.ensureTOCStyles()

	paras := d.buildTOCParagraphs(entries, opts)

	// Ensure the page numbers (and the field itself) are recomputed by the
	// reader once it has laid out the pages, since the cached page numbers are
	// necessarily unknown at generation time.
	d.Settings.SetUpdateFieldsOnOpen(true)

	d.insertBlocks(paras, atTop)
	return TableOfContents{d, paras}
}

// collectTOCEntries walks the document paragraphs and returns one tocEntry for
// each heading that falls within the requested outline levels. A bookmark is
// added to every matched heading so the generated entry can link back to it.
func (d *Document) collectTOCEntries(opts FieldTOCOptions) []tocEntry {
	min, max := parseOutlineLevels(opts.OutlineLevels)
	needBookmark := opts.Hyperlink || opts.IncludePageNumbers
	nextID := d.nextBookmarkID()

	entries := []tocEntry{}
	for _, p := range d.Paragraphs() {
		lvl := headingLevel(p.Style())
		if lvl < min || lvl > max {
			continue
		}
		text := paragraphText(p)
		if text == "" {
			continue
		}
		e := tocEntry{level: lvl, text: text}
		if needBookmark {
			e.bookmark = fmt.Sprintf("_Toc%d", tocBookmarkBase+len(entries))
			addBookmarkWithID(p, e.bookmark, nextID)
			nextID++
		}
		entries = append(entries, e)
	}
	return entries
}

// nextBookmarkID returns an unused bookmark id, computed as one past the highest
// id currently in the document, so bookmarks added for the table of contents do
// not collide with existing ones.
func (d *Document) nextBookmarkID() int64 {
	max := int64(-1)
	for _, bm := range d.Bookmarks() {
		if id := bm.X().IdAttr; id > max {
			max = id
		}
	}
	return max + 1
}

// addBookmarkWithID adds an empty bookmark (a start immediately followed by an
// end) to the end of the paragraph, giving both markers the same explicit id so
// Word can pair them. Paragraph.AddBookmark leaves the id at its zero value,
// which is ambiguous once several bookmarks are present, so the TOC uses this
// helper instead.
func addBookmarkWithID(p Paragraph, name string, id int64) {
	pc := wml.NewEG_PContent()
	rc := wml.NewEG_ContentRunContent()
	pc.EG_ContentRunContent = append(pc.EG_ContentRunContent, rc)

	relt := wml.NewEG_RunLevelElts()
	rc.EG_RunLevelElts = append(rc.EG_RunLevelElts, relt)

	startEl := wml.NewEG_RangeMarkupElements()
	startEl.BookmarkStart = wml.NewCT_Bookmark()
	startEl.BookmarkStart.NameAttr = name
	startEl.BookmarkStart.IdAttr = id
	relt.EG_RangeMarkupElements = append(relt.EG_RangeMarkupElements, startEl)

	endEl := wml.NewEG_RangeMarkupElements()
	endEl.BookmarkEnd = wml.NewCT_MarkupRange()
	endEl.BookmarkEnd.IdAttr = id
	relt.EG_RangeMarkupElements = append(relt.EG_RangeMarkupElements, endEl)

	p.x.EG_PContent = append(p.x.EG_PContent, pc)
}

// buildTOCParagraphs turns the collected entries into the paragraphs of the TOC
// field. The field's begin/instruction/separate markers are emitted on the first
// paragraph and the end marker on the last, so the whole block is a single well
// formed field whose cached result is the visible list of entries.
func (d *Document) buildTOCParagraphs(entries []tocEntry, opts FieldTOCOptions) []*wml.CT_P {
	instr := " " + FieldTOC + " " + opts.switches() + " "

	if len(entries) == 0 {
		// Nothing to list, but still emit a valid (empty) field so a later
		// recalculation in the reader can populate it.
		p := wml.NewCT_P()
		para := Paragraph{d, p}
		addFieldRuns(para.AddRun, instr, "No table of contents entries found.")
		return []*wml.CT_P{p}
	}

	paras := make([]*wml.CT_P, 0, len(entries))
	for i, e := range entries {
		p := wml.NewCT_P()
		para := Paragraph{d, p}
		para.SetStyle(fmt.Sprintf("TOC%d", e.level))
		para.Properties().AddTabStop(tocEntryTabPosition, wml.ST_TabJcRight, wml.ST_TabTlcDot)

		// Open the field on the first entry.
		if i == 0 {
			fldCharRun(para.AddRun(), wml.ST_FldCharTypeBegin, true)
			instrRun(para.AddRun(), instr)
			fldCharRun(para.AddRun(), wml.ST_FldCharTypeSeparate, false)
		}

		d.addTOCEntryContent(para, e, opts)

		// Close the field on the last entry.
		if i == len(entries)-1 {
			fldCharRun(para.AddRun(), wml.ST_FldCharTypeEnd, false)
		}

		paras = append(paras, p)
	}
	return paras
}

// addTOCEntryContent appends the visible content of a single entry - the heading
// text, an optional dot-leadered tab and page number - to para, optionally
// wrapped in a hyperlink pointing at the heading's bookmark.
func (d *Document) addTOCEntryContent(para Paragraph, e tocEntry, opts FieldTOCOptions) {
	if opts.Hyperlink && e.bookmark != "" {
		hl := para.AddHyperLink()
		hl.X().AnchorAttr = gooxml.String(e.bookmark)
		hl.X().HistoryAttr = &sharedTypes.ST_OnOff{}
		hl.X().HistoryAttr.Bool = gooxml.Bool(true)
		hl.AddRun().AddText(e.text)
		if opts.IncludePageNumbers {
			hl.AddRun().AddTab()
			addFieldRuns(hl.AddRun, " PAGEREF "+e.bookmark+" \\h ", "")
		}
		return
	}

	para.AddRun().AddText(e.text)
	if opts.IncludePageNumbers && e.bookmark != "" {
		para.AddRun().AddTab()
		addFieldRuns(para.AddRun, " PAGEREF "+e.bookmark+" \\h ", "")
	}
}

// insertBlocks adds the generated paragraphs to the document body, either before
// the current first block (atTop) or appended after the last.
func (d *Document) insertBlocks(paras []*wml.CT_P, atTop bool) {
	if d.x.Body == nil {
		d.x.Body = wml.NewCT_Body()
	}

	blocks := make([]*wml.EG_BlockLevelElts, 0, len(paras))
	for _, p := range paras {
		elt := wml.NewEG_BlockLevelElts()
		c := wml.NewEG_ContentBlockContent()
		elt.EG_ContentBlockContent = append(elt.EG_ContentBlockContent, c)
		c.P = append(c.P, p)
		blocks = append(blocks, elt)
	}

	if atTop {
		d.x.Body.EG_BlockLevelElts = append(blocks, d.x.Body.EG_BlockLevelElts...)
	} else {
		d.x.Body.EG_BlockLevelElts = append(d.x.Body.EG_BlockLevelElts, blocks...)
	}
}

// ensureTOCStyles creates the TOC1..TOC9 paragraph styles referenced by the
// generated entries if they are not already defined, so the entries are indented
// by level and Word recognizes them as table of contents styles.
func (d *Document) ensureTOCStyles() {
	existing := map[string]bool{}
	for _, s := range d.Styles.Styles() {
		existing[s.StyleID()] = true
	}
	for i := 1; i <= 9; i++ {
		id := fmt.Sprintf("TOC%d", i)
		if existing[id] {
			continue
		}
		st := d.Styles.AddStyle(id, wml.ST_StyleTypeParagraph, false)
		st.SetName(fmt.Sprintf("toc %d", i))
		st.SetBasedOn("Normal")
		st.SetNextStyle("Normal")
		st.SetUISortOrder(39 + i)
		st.SetSemiHidden(true)
		st.SetUnhideWhenUsed(true)
		st.ParagraphProperties().SetLeftIndent(measurement.Distance(i-1) * 0.25 * measurement.Inch)
	}
}

// fldCharRun turns r into a run holding a single field character marker.
func fldCharRun(r Run, t wml.ST_FldCharType, dirty bool) {
	ic := r.newIC()
	ic.FldChar = wml.NewCT_FldChar()
	ic.FldChar.FldCharTypeAttr = t
	if dirty {
		ic.FldChar.DirtyAttr = &sharedTypes.ST_OnOff{}
		ic.FldChar.DirtyAttr.Bool = gooxml.Bool(true)
	}
}

// instrRun turns r into a run holding the instruction text of a field. The text
// is written with xml:space="preserve" so the space separated switches are not
// collapsed.
func instrRun(r Run, instr string) {
	ic := r.newIC()
	ic.InstrText = wml.NewCT_Text()
	ic.InstrText.Content = instr
	preserve := "preserve"
	ic.InstrText.SpaceAttr = &preserve
}

// addFieldRuns appends the begin/instruction/separate/(cached)/end runs of a
// field to a run container. add is Paragraph.AddRun or HyperLink.AddRun so the
// same helper works inside a paragraph and inside a hyperlink.
func addFieldRuns(add func() Run, instr, cached string) {
	fldCharRun(add(), wml.ST_FldCharTypeBegin, true)
	instrRun(add(), instr)
	fldCharRun(add(), wml.ST_FldCharTypeSeparate, false)
	if cached != "" {
		add().AddText(cached)
	}
	fldCharRun(add(), wml.ST_FldCharTypeEnd, false)
}

// headingLevel returns the 1-based heading level of a style name of the form
// "Heading1".."Heading9", or 0 if the style is not a heading.
func headingLevel(style string) int {
	const prefix = "Heading"
	if !strings.HasPrefix(style, prefix) {
		return 0
	}
	lvl, err := strconv.Atoi(style[len(prefix):])
	if err != nil || lvl < 1 || lvl > 9 {
		return 0
	}
	return lvl
}

// parseOutlineLevels parses a TOC \o range like "1-3" into its inclusive
// minimum and maximum. It falls back to 1-9 for empty or malformed input.
func parseOutlineLevels(s string) (int, int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1, 9
	}
	parts := strings.SplitN(s, "-", 2)
	min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err1 != nil {
		return 1, 9
	}
	if len(parts) == 1 {
		return min, min
	}
	max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err2 != nil {
		return min, 9
	}
	if max < min {
		min, max = max, min
	}
	return min, max
}

// paragraphText returns the concatenated visible text of a paragraph, including
// the text inside any hyperlinks.
func paragraphText(p Paragraph) string {
	sb := strings.Builder{}
	for _, r := range p.Runs() {
		sb.WriteString(r.Text())
	}
	return strings.TrimSpace(sb.String())
}
