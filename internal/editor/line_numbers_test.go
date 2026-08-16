package editor

import (
	"strings"
	"testing"
)

func TestShowLineNumbersAddsGutterAndPreservesContentWidth(t *testing.T) {
	m := New(4, 10, GruvboxTheme())
	m.buffer.SetContent("first\nsecond")
	m.SetSize(20, 4)
	m.SetShowLineNumbers(true)

	if got := m.editWidth(); got != 10 {
		t.Fatalf("edit width = %d, want configured content width 10", got)
	}
	if got := m.lineNumberGutterWidth(); got != 4 {
		t.Fatalf("gutter width = %d, want 4", got)
	}

	view := m.View()
	if !strings.Contains(view, "1 │ ") || !strings.Contains(view, "2 │ ") {
		t.Fatalf("line-number gutter missing from view:\n%s", view)
	}
}

func TestLineNumberGutterReducesFullWidthEditor(t *testing.T) {
	m := New(4, 0, GruvboxTheme())
	m.SetSize(20, 4)
	if got := m.editWidth(); got != 20 {
		t.Fatalf("edit width without gutter = %d, want 20", got)
	}

	m.SetShowLineNumbers(true)
	if got := m.editWidth(); got != 16 {
		t.Fatalf("edit width with gutter = %d, want 16", got)
	}
}
