package document

import (
	"testing"

	"github.com/uvulpos/gooxml/schema/soo/wml"
)

// headingDoc builds a document with a handful of headings at levels 1-3 plus
// some body text between them.
func headingDoc() *Document {
	doc := New()
	for i := 0; i < 2; i++ {
		h1 := doc.AddParagraph()
		h1.Properties().SetHeadingLevel(1)
		h1.AddRun().AddText("Chapter")
		doc.AddParagraph().AddRun().AddText("Body text")

		h2 := doc.AddParagraph()
		h2.Properties().SetHeadingLevel(2)
		h2.AddRun().AddText("Section")

		h3 := doc.AddParagraph()
		h3.Properties().SetHeadingLevel(3)
		h3.AddRun().AddText("Subsection")
	}
	return doc
}

func runFldCharCount(r *wml.CT_R, t wml.ST_FldCharType) int {
	n := 0
	for _, ic := range r.EG_RunInnerContent {
		if ic.FldChar != nil && ic.FldChar.FldCharTypeAttr == t {
			n++
		}
	}
	return n
}

// fldCharCount counts field-character markers across the body, descending into
// hyperlinks (Paragraph.Runs does not, and the PAGEREF fields live there).
func fldCharCount(d *Document, t wml.ST_FldCharType) int {
	n := 0
	for _, p := range d.Paragraphs() {
		for _, pc := range p.X().EG_PContent {
			for _, rc := range pc.EG_ContentRunContent {
				if rc.R != nil {
					n += runFldCharCount(rc.R, t)
				}
			}
			if pc.Hyperlink != nil {
				for _, rc := range pc.Hyperlink.EG_ContentRunContent {
					if rc.R != nil {
						n += runFldCharCount(rc.R, t)
					}
				}
			}
		}
	}
	return n
}

func TestGenerateTableOfContents(t *testing.T) {
	doc := headingDoc()
	beforeParas := len(doc.Paragraphs())

	toc := doc.GenerateTableOfContents(true, DefaultTOCOptions())

	// Two chapters, each with one section and one subsection => 6 headings.
	if got := len(toc.Paragraphs()); got != 6 {
		t.Fatalf("expected 6 TOC entry paragraphs, got %d", got)
	}

	// The TOC paragraphs must have been prepended to the body.
	if got := len(doc.Paragraphs()); got != beforeParas+6 {
		t.Errorf("expected %d paragraphs after generation, got %d", beforeParas+6, got)
	}
	if style := doc.Paragraphs()[0].Style(); style != "TOC1" {
		t.Errorf("expected first paragraph to be a TOC1 entry, got %q", style)
	}

	// Exactly one enclosing TOC field: a single begin/separate pair whose
	// instruction is the TOC field, plus a matching begin/separate for each
	// PAGEREF. begin markers: 1 (TOC) + 6 (PAGEREF) = 7; likewise for end.
	if got := fldCharCount(doc, wml.ST_FldCharTypeBegin); got != 7 {
		t.Errorf("expected 7 begin field chars, got %d", got)
	}
	if got := fldCharCount(doc, wml.ST_FldCharTypeEnd); got != 7 {
		t.Errorf("expected 7 end field chars, got %d", got)
	}

	// A bookmark should have been added to every heading, with unique ids.
	ids := map[int64]bool{}
	names := 0
	for _, bm := range doc.Bookmarks() {
		names++
		if ids[bm.X().IdAttr] {
			t.Errorf("duplicate bookmark id %d", bm.X().IdAttr)
		}
		ids[bm.X().IdAttr] = true
	}
	if names != 6 {
		t.Errorf("expected 6 bookmarks, got %d", names)
	}

	// Fields should be flagged for recalculation on open so page numbers fill in.
	if doc.Settings.X().UpdateFields == nil {
		t.Error("expected update-fields-on-open to be enabled")
	}

	if err := doc.Validate(); err != nil {
		t.Errorf("generated document did not validate: %s", err)
	}
}

func TestGenerateTableOfContentsOutlineLevels(t *testing.T) {
	doc := headingDoc()
	opts := DefaultTOCOptions()
	opts.OutlineLevels = "1-2"

	toc := doc.GenerateTableOfContents(false, opts)

	// Level 3 headings must be excluded => 2 chapters + 2 sections = 4.
	if got := len(toc.Paragraphs()); got != 4 {
		t.Fatalf("expected 4 TOC entry paragraphs for levels 1-2, got %d", got)
	}
	// atTop == false, so the TOC entries are appended after the content.
	paras := doc.Paragraphs()
	if style := paras[len(paras)-1].Style(); style != "TOC2" {
		t.Errorf("expected last paragraph to be a TOC2 entry, got %q", style)
	}
}

func TestGenerateTableOfContentsNoHeadings(t *testing.T) {
	doc := New()
	doc.AddParagraph().AddRun().AddText("Just body text")

	toc := doc.GenerateTableOfContents(true, DefaultTOCOptions())
	if got := len(toc.Paragraphs()); got != 1 {
		t.Fatalf("expected a single placeholder paragraph, got %d", got)
	}
	// Even with no headings the field must be well-formed (begin + end).
	if got := fldCharCount(doc, wml.ST_FldCharTypeBegin); got != 1 {
		t.Errorf("expected 1 begin field char, got %d", got)
	}
	if got := fldCharCount(doc, wml.ST_FldCharTypeEnd); got != 1 {
		t.Errorf("expected 1 end field char, got %d", got)
	}
	if err := doc.Validate(); err != nil {
		t.Errorf("empty TOC document did not validate: %s", err)
	}
}

func TestHeadingLevel(t *testing.T) {
	cases := map[string]int{
		"Heading1":  1,
		"Heading9":  9,
		"Heading10": 0,
		"Heading0":  0,
		"Normal":    0,
		"":          0,
		"HeadingX":  0,
	}
	for style, want := range cases {
		if got := headingLevel(style); got != want {
			t.Errorf("headingLevel(%q) = %d, want %d", style, got, want)
		}
	}
}

func TestParseOutlineLevels(t *testing.T) {
	cases := []struct {
		in       string
		min, max int
	}{
		{"1-3", 1, 3},
		{"2-5", 2, 5},
		{"4", 4, 4},
		{"", 1, 9},
		{"garbage", 1, 9},
		{"3-1", 1, 3},
	}
	for _, c := range cases {
		min, max := parseOutlineLevels(c.in)
		if min != c.min || max != c.max {
			t.Errorf("parseOutlineLevels(%q) = %d,%d want %d,%d", c.in, min, max, c.min, c.max)
		}
	}
}
