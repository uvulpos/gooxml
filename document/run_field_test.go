// Copyright 2017 Baliance. All rights reserved.
//
// Use of this source code is governed by the terms of the Affero GNU General
// Public License version 3.0 as published by the Free Software Foundation and
// appearing in the file LICENSE included in the packaging of this file. A
// commercial license can be purchased by contacting sales@baliance.com.

package document_test

import (
	"testing"

	"github.com/uvulpos/gooxml/document"
	"github.com/uvulpos/gooxml/schema/soo/wml"
)

// fldCharTypes returns the ordered list of field-character types in a run.
func fldCharTypes(run document.Run) []wml.ST_FldCharType {
	types := []wml.ST_FldCharType{}
	for _, ic := range run.X().EG_RunInnerContent {
		if ic.FldChar != nil {
			types = append(types, ic.FldChar.FldCharTypeAttr)
		}
	}
	return types
}

// A plain field must be a well-formed begin/separate/end region so that readers
// which do not recalculate fields on open still render the field.
func TestAddFieldEmitsSeparate(t *testing.T) {
	doc := document.New()
	run := doc.AddParagraph().AddRun()
	run.AddField(document.FieldTOC)

	got := fldCharTypes(run)
	want := []wml.ST_FldCharType{
		wml.ST_FldCharTypeBegin,
		wml.ST_FldCharTypeSeparate,
		wml.ST_FldCharTypeEnd,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d fldChar markers, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fldChar[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// Instruction text carrying switches must be marked xml:space="preserve".
func TestAddFieldWithFormattingPreservesSpace(t *testing.T) {
	doc := document.New()
	run := doc.AddParagraph().AddRun()
	run.AddFieldWithFormatting(document.FieldTOC, `\o "1-3" \h`)

	var instr *wml.CT_Text
	for _, ic := range run.X().EG_RunInnerContent {
		if ic.InstrText != nil {
			instr = ic.InstrText
			break
		}
	}
	if instr == nil {
		t.Fatal("expected an instrText inner content item")
	}
	if want := `TOC \o "1-3" \h`; instr.Content != want {
		t.Errorf("instrText = %q, want %q", instr.Content, want)
	}
	if instr.SpaceAttr == nil || *instr.SpaceAttr != "preserve" {
		t.Errorf("instrText should carry xml:space=preserve, got %v", instr.SpaceAttr)
	}
}

// AddFieldTOC should produce the documented default switches, a preserved
// instruction run, a separate marker, a cached result and an end marker.
func TestAddFieldTOCStructure(t *testing.T) {
	doc := document.New()
	run := doc.AddParagraph().AddRun()
	run.AddFieldTOC(document.DefaultTOCOptions())

	ic := run.X().EG_RunInnerContent
	if len(ic) != 5 {
		t.Fatalf("expected 5 inner content items, got %d", len(ic))
	}
	if ic[0].FldChar == nil || ic[0].FldChar.FldCharTypeAttr != wml.ST_FldCharTypeBegin {
		t.Errorf("item 0 should be a begin fldChar")
	}
	if ic[1].InstrText == nil {
		t.Fatal("item 1 should be instrText")
	}
	if want := `TOC \o "1-3" \h \z \u`; ic[1].InstrText.Content != want {
		t.Errorf("instrText = %q, want %q", ic[1].InstrText.Content, want)
	}
	if ic[1].InstrText.SpaceAttr == nil || *ic[1].InstrText.SpaceAttr != "preserve" {
		t.Errorf("instrText should carry xml:space=preserve")
	}
	if ic[2].FldChar == nil || ic[2].FldChar.FldCharTypeAttr != wml.ST_FldCharTypeSeparate {
		t.Errorf("item 2 should be a separate fldChar")
	}
	if ic[3].T == nil || ic[3].T.Content == "" {
		t.Errorf("item 3 should be the cached result text")
	}
	if ic[4].FldChar == nil || ic[4].FldChar.FldCharTypeAttr != wml.ST_FldCharTypeEnd {
		t.Errorf("item 4 should be an end fldChar")
	}
}

func TestFieldTOCOptionsSwitchesDefaultLevels(t *testing.T) {
	// An empty OutlineLevels should fall back to "1-3".
	doc := document.New()
	run := doc.AddParagraph().AddRun()
	run.AddFieldTOC(document.FieldTOCOptions{Hyperlink: true})

	var instr *wml.CT_Text
	for _, ic := range run.X().EG_RunInnerContent {
		if ic.InstrText != nil {
			instr = ic.InstrText
			break
		}
	}
	if instr == nil {
		t.Fatal("expected an instrText inner content item")
	}
	if want := `TOC \o "1-3" \h`; instr.Content != want {
		t.Errorf("instrText = %q, want %q", instr.Content, want)
	}
}
