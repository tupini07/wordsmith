package editor

import (
	"testing"
)

func TestWrapEmptyLine(t *testing.T) {
	lines := [][]rune{{}}
	result := WrapLines(lines, 80)
	if len(result.VisualLines) != 1 {
		t.Errorf("expected 1 visual line, got %d", len(result.VisualLines))
	}
}

func TestWrapShortLine(t *testing.T) {
	lines := [][]rune{[]rune("hello world")}
	result := WrapLines(lines, 80)
	if len(result.VisualLines) != 1 {
		t.Errorf("expected 1 visual line, got %d", len(result.VisualLines))
	}
	if string(result.VisualLines[0].Runes) != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", string(result.VisualLines[0].Runes))
	}
}

func TestWrapLongLine(t *testing.T) {
	line := []rune("this is a test of word wrapping behavior in the editor")
	lines := [][]rune{line}
	result := WrapLines(lines, 20)

	if len(result.VisualLines) < 2 {
		t.Errorf("expected at least 2 visual lines, got %d", len(result.VisualLines))
	}

	// All visual lines should reference logical line 0
	for i, vl := range result.VisualLines {
		if vl.LogicalLine != 0 {
			t.Errorf("visual line %d: expected logical line 0, got %d", i, vl.LogicalLine)
		}
	}
}

func TestWrapMultipleLogicalLines(t *testing.T) {
	lines := [][]rune{
		[]rune("short"),
		[]rune("this is a longer line that should wrap"),
		[]rune("end"),
	}
	result := WrapLines(lines, 20)

	// Verify line map
	if len(result.LineMap) != 3 {
		t.Fatalf("expected 3 line map entries, got %d", len(result.LineMap))
	}

	// First logical line shouldn't wrap
	lr0 := result.LineMap[0]
	if lr0.End-lr0.Start != 1 {
		t.Errorf("logical line 0: expected 1 visual line, got %d", lr0.End-lr0.Start)
	}

	// Second logical line should wrap
	lr1 := result.LineMap[1]
	if lr1.End-lr1.Start < 2 {
		t.Errorf("logical line 1: expected ≥2 visual lines, got %d", lr1.End-lr1.Start)
	}
}

func TestLogicalToVisual(t *testing.T) {
	lines := [][]rune{
		[]rune("short"),
		[]rune("hello world again"),
	}
	result := WrapLines(lines, 10)

	// Cursor at start of logical line 0
	vr, vc := result.LogicalToVisual(0, 0)
	if vr != 0 || vc != 0 {
		t.Errorf("expected visual (0,0), got (%d,%d)", vr, vc)
	}

	// Cursor at start of logical line 1
	vr, vc = result.LogicalToVisual(1, 0)
	lr := result.LineMap[1]
	if vr != lr.Start {
		t.Errorf("expected visual row %d, got %d", lr.Start, vr)
	}
	if vc != 0 {
		t.Errorf("expected visual col 0, got %d", vc)
	}
}

func TestVisualToLogical(t *testing.T) {
	lines := [][]rune{
		[]rune("hello world this is a test"),
	}
	result := WrapLines(lines, 12)

	// First visual line maps to logical (0, x)
	ll, lc := result.VisualToLogical(0, 3)
	if ll != 0 {
		t.Errorf("expected logical line 0, got %d", ll)
	}
	if lc != 3 {
		t.Errorf("expected logical col 3, got %d", lc)
	}

	// Second visual line maps to logical (0, offset+x)
	if len(result.VisualLines) > 1 {
		vl1 := result.VisualLines[1]
		ll, lc = result.VisualToLogical(1, 2)
		if ll != 0 {
			t.Errorf("expected logical line 0, got %d", ll)
		}
		expected := vl1.LogicalCol + 2
		if lc != expected {
			t.Errorf("expected logical col %d, got %d", expected, lc)
		}
	}
}

func TestWrapWordBoundary(t *testing.T) {
	// "hello world" with width 8 should break at the space
	line := []rune("hello world")
	lines := [][]rune{line}
	result := WrapLines(lines, 8)

	if len(result.VisualLines) != 2 {
		t.Fatalf("expected 2 visual lines, got %d", len(result.VisualLines))
	}

	// First line should be "hello " (breaks after space)
	first := string(result.VisualLines[0].Runes)
	if first != "hello " {
		t.Errorf("first visual line: expected %q, got %q", "hello ", first)
	}

	second := string(result.VisualLines[1].Runes)
	if second != "world" {
		t.Errorf("second visual line: expected %q, got %q", "world", second)
	}
}

func TestCursorUpDownAcrossWrappedLine(t *testing.T) {
	// Simulate: one long line that wraps into 2 visual lines, then a short line.
	// "The quick brown fox jumps over" wraps at width 15:
	// vis 0: "The quick brown " (LogicalCol 0)
	// vis 1: "fox jumps over"  (LogicalCol 16)
	// vis 2: "end"             (logical line 1)
	lines := [][]rune{
		[]rune("The quick brown fox jumps over"),
		[]rune("end"),
	}
	w := WrapLines(lines, 16)

	if len(w.VisualLines) < 3 {
		t.Fatalf("expected ≥3 visual lines, got %d", len(w.VisualLines))
	}

	// Cursor at logical(0, 20) = middle of second visual line of the wrap
	vr, vc := w.LogicalToVisual(0, 20)
	if vr != 1 {
		t.Errorf("expected visual row 1, got %d", vr)
	}
	if vc != 4 {
		t.Errorf("expected visual col 4 (20-16), got %d", vc)
	}

	// Move up from visual row 1 → visual row 0, using visual col 4
	ll, lc := w.VisualToLogical(0, 4)
	if ll != 0 || lc != 4 {
		t.Errorf("VisualToLogical(0, 4): expected (0, 4), got (%d, %d)", ll, lc)
	}
	// Verify it maps back to visual row 0
	backRow, _ := w.LogicalToVisual(ll, lc)
	if backRow != 0 {
		t.Errorf("round-trip: expected visual row 0, got %d", backRow)
	}
}

func TestVisualToLogicalClampOnWrapBoundary(t *testing.T) {
	// A line that wraps: visual line 0 is full-width, visual line 1 continues.
	// Test that VisualToLogical with visCol > visual line width clamps correctly.
	lines := [][]rune{
		[]rune("hello world this is a test of wrapping"),
	}
	w := WrapLines(lines, 12)

	if len(w.VisualLines) < 2 {
		t.Fatalf("expected ≥2 visual lines, got %d", len(w.VisualLines))
	}

	vl0 := w.VisualLines[0]

	// Request visCol far past the visual line width
	ll, lc := w.VisualToLogical(0, 100)
	if ll != 0 {
		t.Errorf("expected logical line 0, got %d", ll)
	}
	maxLogCol := vl0.LogicalCol + len(vl0.Runes)
	if lc != maxLogCol {
		t.Errorf("expected clamped logCol %d, got %d", maxLogCol, lc)
	}
}

func TestCursorNotStuckAtWrapBoundary(t *testing.T) {
	// Regression: cursor at wrap boundary (end of visual line 0 = start of
	// visual line 1) should not get stuck when pressing up.
	lines := [][]rune{
		[]rune("hello world this is a test of wrapping around"),
	}
	w := WrapLines(lines, 12)

	if len(w.VisualLines) < 2 {
		t.Fatalf("expected ≥2 visual lines, got %d", len(w.VisualLines))
	}

	vl0 := w.VisualLines[0]
	boundaryCol := vl0.LogicalCol + len(vl0.Runes)

	// The boundary position maps to visual line 1, not 0
	vr, _ := w.LogicalToVisual(0, boundaryCol)
	if vr != 1 {
		t.Fatalf("expected boundary col %d to map to visual row 1, got %d", boundaryCol, vr)
	}

	// The snapToVisualRow mechanism in the editor handles this:
	// After VisualToLogical(0, largeVisCol) → clamps to boundary → maps to row 1.
	// The editor detects row mismatch and snaps cursor back to end of row 0.
	// Here we verify the raw VisualToLogical → LogicalToVisual round-trip,
	// and confirm the boundary position is on row 1.
	ll, lc := w.VisualToLogical(0, len(vl0.Runes))
	if ll != 0 {
		t.Errorf("expected logical line 0, got %d", ll)
	}
	if lc != boundaryCol {
		t.Errorf("expected logCol %d, got %d", boundaryCol, lc)
	}
	roundRow, _ := w.LogicalToVisual(ll, lc)
	// This IS the boundary ambiguity — it maps to row 1
	// The editor's snapToVisualRow fixes this during navigation
	if roundRow != 1 {
		t.Logf("boundary maps to row %d (expected row 1, editor snap handles this)", roundRow)
	}
}

func TestVisualToLogicalSingleCharVisualLine(t *testing.T) {
	// Edge case: a very narrow wrap producing single-char visual lines.
	lines := [][]rune{[]rune("abc")}
	w := WrapLines(lines, 1)

	// Each char should be its own visual line
	if len(w.VisualLines) != 3 {
		t.Fatalf("expected 3 visual lines, got %d", len(w.VisualLines))
	}

	// visCol 0 and 1 should both be valid on visual row 0
	ll, lc := w.VisualToLogical(0, 0)
	if ll != 0 || lc != 0 {
		t.Errorf("VisualToLogical(0,0): expected (0,0), got (%d,%d)", ll, lc)
	}

	// visCol=1 on row 0 → caret at position 1 (end of visual line, boundary)
	ll, lc = w.VisualToLogical(0, 1)
	if ll != 0 || lc != 1 {
		t.Errorf("VisualToLogical(0,1): expected (0,1), got (%d,%d)", ll, lc)
	}

	// visCol=0 on row 1
	ll, lc = w.VisualToLogical(1, 0)
	if ll != 0 || lc != 1 {
		t.Errorf("VisualToLogical(1,0): expected (0,1), got (%d,%d)", ll, lc)
	}
}
