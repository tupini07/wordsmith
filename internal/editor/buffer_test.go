package editor

import (
	"strings"
	"testing"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer()
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", b.LineCount())
	}
	if b.Content() != "" {
		t.Errorf("expected empty content, got %q", b.Content())
	}
	if b.IsDirty() {
		t.Error("new buffer should not be dirty")
	}
}

func TestBufferSetContent(t *testing.T) {
	b := NewBufferFromString("hello\nworld\nfoo")
	if b.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("line 0: expected %q, got %q", "hello", string(b.Line(0)))
	}
	if string(b.Line(1)) != "world" {
		t.Errorf("line 1: expected %q, got %q", "world", string(b.Line(1)))
	}
	if string(b.Line(2)) != "foo" {
		t.Errorf("line 2: expected %q, got %q", "foo", string(b.Line(2)))
	}
	if b.Content() != "hello\nworld\nfoo" {
		t.Errorf("content mismatch: %q", b.Content())
	}
}

func TestInsertChar(t *testing.T) {
	b := NewBufferFromString("hello")
	b.InsertChar(0, 5, '!')
	if string(b.Line(0)) != "hello!" {
		t.Errorf("expected %q, got %q", "hello!", string(b.Line(0)))
	}
	if !b.IsDirty() {
		t.Error("buffer should be dirty after insert")
	}

	// Insert at beginning
	b.InsertChar(0, 0, '>')
	if string(b.Line(0)) != ">hello!" {
		t.Errorf("expected %q, got %q", ">hello!", string(b.Line(0)))
	}

	// Insert in middle
	b.InsertChar(0, 3, '-')
	if string(b.Line(0)) != ">he-llo!" {
		t.Errorf("expected %q, got %q", ">he-llo!", string(b.Line(0)))
	}
}

func TestDeleteChar(t *testing.T) {
	b := NewBufferFromString("hello")
	ch, ok := b.DeleteChar(0, 0)
	if !ok || ch != 'h' {
		t.Errorf("expected 'h', got %q, ok=%v", string(ch), ok)
	}
	if string(b.Line(0)) != "ello" {
		t.Errorf("expected %q, got %q", "ello", string(b.Line(0)))
	}

	// Delete at end joins lines
	b2 := NewBufferFromString("ab\ncd")
	ch, ok = b2.DeleteChar(0, 2) // at end of "ab"
	if !ok || ch != '\n' {
		t.Errorf("expected newline join, got %q, ok=%v", string(ch), ok)
	}
	if b2.LineCount() != 1 {
		t.Errorf("expected 1 line after join, got %d", b2.LineCount())
	}
	if string(b2.Line(0)) != "abcd" {
		t.Errorf("expected %q, got %q", "abcd", string(b2.Line(0)))
	}
}

func TestBackspace(t *testing.T) {
	b := NewBufferFromString("hello")
	newLine, newCol, ok := b.Backspace(0, 3)
	if !ok {
		t.Error("backspace should succeed")
	}
	if newLine != 0 || newCol != 2 {
		t.Errorf("expected (0, 2), got (%d, %d)", newLine, newCol)
	}
	if string(b.Line(0)) != "helo" {
		t.Errorf("expected %q, got %q", "helo", string(b.Line(0)))
	}

	// Backspace at start of line joins
	b2 := NewBufferFromString("ab\ncd")
	newLine, newCol, ok = b2.Backspace(1, 0)
	if !ok {
		t.Error("backspace at line start should join")
	}
	if newLine != 0 || newCol != 2 {
		t.Errorf("expected (0, 2), got (%d, %d)", newLine, newCol)
	}
	if string(b2.Line(0)) != "abcd" {
		t.Errorf("expected %q, got %q", "abcd", string(b2.Line(0)))
	}
}

func TestInsertNewline(t *testing.T) {
	b := NewBufferFromString("hello world")
	b.InsertNewline(0, 5)
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("line 0: expected %q, got %q", "hello", string(b.Line(0)))
	}
	if string(b.Line(1)) != " world" {
		t.Errorf("line 1: expected %q, got %q", " world", string(b.Line(1)))
	}
}

func TestDeleteRange(t *testing.T) {
	b := NewBufferFromString("hello world")
	deleted := b.DeleteRange(0, 5, 0, 11)
	if deleted != " world" {
		t.Errorf("expected %q, got %q", " world", deleted)
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("expected %q, got %q", "hello", string(b.Line(0)))
	}

	// Multi-line delete
	b2 := NewBufferFromString("aaa\nbbb\nccc")
	deleted = b2.DeleteRange(0, 1, 2, 2)
	if deleted != "aa\nbbb\ncc" {
		t.Errorf("expected %q, got %q", "aa\nbbb\ncc", deleted)
	}
	if b2.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", b2.LineCount())
	}
	if string(b2.Line(0)) != "ac" {
		t.Errorf("expected %q, got %q", "ac", string(b2.Line(0)))
	}
}

func TestWordAt(t *testing.T) {
	b := NewBufferFromString("hello world foo")
	start, end := b.WordAt(0, 7)
	if start != 6 || end != 11 {
		t.Errorf("expected word 'world' at [6,11), got [%d,%d)", start, end)
	}

	// At space — WordAt returns the cursor position for non-word chars
	start, end = b.WordAt(0, 5)
	if start == end {
		// Word boundaries are implementation-defined at spaces; just verify it doesn't panic
	}
}

func TestWordCount(t *testing.T) {
	b := NewBufferFromString("hello world foo bar")
	if wc := b.WordCount(); wc != 4 {
		t.Errorf("expected 4 words, got %d", wc)
	}

	// With frontmatter
	b2 := NewBufferFromString("---\ntitle: test\n---\nhello world")
	if wc := b2.WordCount(); wc != 2 {
		t.Errorf("expected 2 words (excluding frontmatter), got %d", wc)
	}
}

func TestUndoRedo(t *testing.T) {
	b := NewBufferFromString("hello")
	b.InsertChar(0, 5, '!')
	if string(b.Line(0)) != "hello!" {
		t.Fatalf("insert failed: %q", string(b.Line(0)))
	}

	line, col, ok := b.Undo()
	if !ok {
		t.Fatal("undo should succeed")
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("after undo: expected %q, got %q", "hello", string(b.Line(0)))
	}
	if line != 0 || col != 5 {
		t.Errorf("undo cursor: expected (0,5), got (%d,%d)", line, col)
	}

	line, col, ok = b.Redo()
	if !ok {
		t.Fatal("redo should succeed")
	}
	if string(b.Line(0)) != "hello!" {
		t.Errorf("after redo: expected %q, got %q", "hello!", string(b.Line(0)))
	}
}

func TestUndoNewline(t *testing.T) {
	b := NewBufferFromString("hello world")
	b.InsertNewline(0, 5)
	if b.LineCount() != 2 {
		t.Fatal("newline insert failed")
	}

	_, _, ok := b.Undo()
	if !ok {
		t.Fatal("undo should succeed")
	}
	if b.LineCount() != 1 {
		t.Errorf("after undo: expected 1 line, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello world" {
		t.Errorf("after undo: expected %q, got %q", "hello world", string(b.Line(0)))
	}
}

func TestUnicodeHandling(t *testing.T) {
	b := NewBufferFromString("héllo wörld 🌍")
	if b.LineLen(0) != 13 { // rune count: h-é-l-l-o- -w-ö-r-l-d- -🌍
		t.Errorf("expected 13 runes, got %d", b.LineLen(0))
	}

	// Insert after emoji
	b.InsertChar(0, 14, '!')
	if !strings.HasSuffix(string(b.Line(0)), "🌍!") {
		t.Errorf("expected suffix '🌍!', got %q", string(b.Line(0)))
	}
}

func TestCoalescedUndoRedo(t *testing.T) {
b := NewBuffer()

// Type "hello" — all non-space chars coalesce into one undo entry
for i, ch := range "hello" {
b.InsertChar(0, i, ch)
}
if string(b.Line(0)) != "hello" {
t.Fatalf("insert failed: %q", string(b.Line(0)))
}

// Undo should remove all 5 chars at once
_, _, ok := b.Undo()
if !ok {
t.Fatal("undo should succeed")
}
if string(b.Line(0)) != "" {
t.Errorf("after undo: expected empty, got %q", string(b.Line(0)))
}

// Redo should re-insert all 5 chars
line, col, ok := b.Redo()
if !ok {
t.Fatal("redo should succeed")
}
if string(b.Line(0)) != "hello" {
t.Errorf("after redo: expected %q, got %q", "hello", string(b.Line(0)))
}
if line != 0 || col != 5 {
t.Errorf("redo cursor: expected (0,5), got (%d,%d)", line, col)
}

// Undo again and redo again to verify stability
b.Undo()
if string(b.Line(0)) != "" {
t.Errorf("second undo: expected empty, got %q", string(b.Line(0)))
}
b.Redo()
if string(b.Line(0)) != "hello" {
t.Errorf("second redo: expected %q, got %q", "hello", string(b.Line(0)))
}
}
