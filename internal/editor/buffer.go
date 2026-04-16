package editor

import "unicode"

// Buffer is a rune-aware line-based text buffer.
type Buffer struct {
	lines     [][]rune
	dirty     bool
	undoStack []undoEntry
	redoStack []undoEntry
	undoGroup int // when > 0, new entries get this group ID
}

type undoKind int

const (
	undoInsertChar undoKind = iota
	undoDeleteChar
	undoInsertNewline
	undoJoinLine
	undoDeleteRange
	undoInsertText
)

type undoEntry struct {
	kind    undoKind
	line    int
	col     int
	char    rune     // for single-char ops
	chars   []rune   // all chars in a coalesced insert
	text    [][]rune // for multi-line ops
	endLine int
	endCol  int
	coalesce bool   // can this be merged with previous entry?
	group    int    // entries with the same non-zero group are undone together
}

func NewBuffer() *Buffer {
	return &Buffer{
		lines: [][]rune{{}},
	}
}

func NewBufferFromString(s string) *Buffer {
	b := &Buffer{}
	b.SetContent(s)
	return b
}

func (b *Buffer) SetContent(s string) {
	b.lines = nil
	b.dirty = false
	b.undoStack = nil
	b.redoStack = nil

	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			b.lines = append(b.lines, []rune(s[start:i]))
			start = i + 1
		}
	}
	b.lines = append(b.lines, []rune(s[start:]))

	if len(b.lines) == 0 {
		b.lines = [][]rune{{}}
	}
}

func (b *Buffer) Content() string {
	if len(b.lines) == 0 {
		return ""
	}
	total := 0
	for _, line := range b.lines {
		total += len(line) + 1
	}
	total-- // no trailing newline

	buf := make([]byte, 0, total)
	for i, line := range b.lines {
		if i > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, string(line)...)
	}
	return string(buf)
}

func (b *Buffer) LineCount() int {
	return len(b.lines)
}

func (b *Buffer) Line(n int) []rune {
	if n < 0 || n >= len(b.lines) {
		return nil
	}
	return b.lines[n]
}

func (b *Buffer) LineLen(n int) int {
	if n < 0 || n >= len(b.lines) {
		return 0
	}
	return len(b.lines[n])
}

func (b *Buffer) IsDirty() bool {
	return b.dirty
}

func (b *Buffer) ClearDirty() {
	b.dirty = false
}

// InsertChar inserts a rune at the given position.
func (b *Buffer) InsertChar(line, col int, ch rune) {
	if line < 0 || line >= len(b.lines) {
		return
	}
	if col < 0 {
		col = 0
	}
	if col > len(b.lines[line]) {
		col = len(b.lines[line])
	}

	// Check if we can coalesce with the previous undo entry
	canCoalesce := false
	if len(b.undoStack) > 0 {
		prev := &b.undoStack[len(b.undoStack)-1]
		if prev.kind == undoInsertChar && prev.line == line &&
			prev.endCol == col && !unicode.IsSpace(ch) && !unicode.IsSpace(prev.char) {
			canCoalesce = true
		}
	}

	row := b.lines[line]
	newRow := make([]rune, len(row)+1)
	copy(newRow, row[:col])
	newRow[col] = ch
	copy(newRow[col+1:], row[col:])
	b.lines[line] = newRow
	b.dirty = true

	if canCoalesce {
		// Extend the previous entry
		prev := &b.undoStack[len(b.undoStack)-1]
		prev.endCol = col + 1
		prev.char = ch
		prev.chars = append(prev.chars, ch)
	} else {
		b.pushUndo(undoEntry{
			kind:    undoInsertChar,
			line:    line,
			col:     col,
			char:    ch,
			chars:   []rune{ch},
			endCol:  col + 1,
		})
	}
}

// DeleteChar deletes the rune at the given position (like pressing Delete key).
func (b *Buffer) DeleteChar(line, col int) (rune, bool) {
	if line < 0 || line >= len(b.lines) {
		return 0, false
	}
	if col < 0 || col >= len(b.lines[line]) {
		// At end of line — join with next line
		if col == len(b.lines[line]) && line+1 < len(b.lines) {
			b.joinLines(line)
			return '\n', true
		}
		return 0, false
	}

	ch := b.lines[line][col]
	row := b.lines[line]
	newRow := make([]rune, len(row)-1)
	copy(newRow, row[:col])
	copy(newRow[col:], row[col+1:])
	b.lines[line] = newRow
	b.dirty = true

	b.pushUndo(undoEntry{
		kind: undoDeleteChar,
		line: line,
		col:  col,
		char: ch,
	})
	return ch, true
}

// Backspace deletes the rune before the given position.
func (b *Buffer) Backspace(line, col int) (newLine, newCol int, ok bool) {
	if col > 0 {
		b.DeleteChar(line, col-1)
		return line, col - 1, true
	}
	if line > 0 {
		newCol = len(b.lines[line-1])
		b.joinLines(line - 1)
		return line - 1, newCol, true
	}
	return line, col, false
}

// InsertNewline splits the current line at the given position.
func (b *Buffer) InsertNewline(line, col int) {
	if line < 0 || line >= len(b.lines) {
		return
	}
	if col < 0 {
		col = 0
	}
	if col > len(b.lines[line]) {
		col = len(b.lines[line])
	}

	row := b.lines[line]
	before := make([]rune, col)
	copy(before, row[:col])
	after := make([]rune, len(row)-col)
	copy(after, row[col:])

	newLines := make([][]rune, len(b.lines)+1)
	copy(newLines, b.lines[:line])
	newLines[line] = before
	newLines[line+1] = after
	copy(newLines[line+2:], b.lines[line+1:])
	b.lines = newLines
	b.dirty = true

	b.pushUndo(undoEntry{
		kind: undoInsertNewline,
		line: line,
		col:  col,
	})
}

func (b *Buffer) joinLines(line int) {
	if line < 0 || line+1 >= len(b.lines) {
		return
	}

	joinCol := len(b.lines[line])
	b.lines[line] = append(b.lines[line], b.lines[line+1]...)

	newLines := make([][]rune, len(b.lines)-1)
	copy(newLines, b.lines[:line+1])
	copy(newLines[line+1:], b.lines[line+2:])
	b.lines = newLines
	b.dirty = true

	b.pushUndo(undoEntry{
		kind: undoJoinLine,
		line: line,
		col:  joinCol,
	})
}

// InsertText inserts a string at the given position, possibly spanning multiple lines.
func (b *Buffer) InsertText(line, col int, text string) (endLine, endCol int) {
	runes := []rune(text)
	if len(runes) == 0 {
		return line, col
	}

	// Build lines from the text
	var textLines [][]rune
	var current []rune
	for _, r := range runes {
		if r == '\n' {
			textLines = append(textLines, current)
			current = nil
		} else {
			current = append(current, r)
		}
	}
	textLines = append(textLines, current)

	if len(textLines) == 1 {
		// Single-line insert
		for _, r := range textLines[0] {
			b.InsertChar(line, col, r)
			col++
		}
		return line, col
	}

	// Multi-line: save for undo, then do the insert
	if line < 0 || line >= len(b.lines) {
		return line, col
	}

	origLine := make([]rune, len(b.lines[line]))
	copy(origLine, b.lines[line])

	// Split current line at col
	before := origLine[:col]
	after := origLine[col:]

	// First line: before + first text line
	firstLine := append(append([]rune{}, before...), textLines[0]...)

	// Middle lines: as-is
	// Last line: last text line + after
	lastIdx := len(textLines) - 1
	lastLine := append(append([]rune{}, textLines[lastIdx]...), after...)

	endLine = line + lastIdx
	endCol = len(textLines[lastIdx])

	// Build new lines slice
	newLines := make([][]rune, 0, len(b.lines)+len(textLines)-1)
	newLines = append(newLines, b.lines[:line]...)
	newLines = append(newLines, firstLine)
	for i := 1; i < lastIdx; i++ {
		newLines = append(newLines, textLines[i])
	}
	if lastIdx > 0 {
		newLines = append(newLines, lastLine)
	}
	newLines = append(newLines, b.lines[line+1:]...)
	b.lines = newLines
	b.dirty = true

	b.pushUndo(undoEntry{
		kind:    undoInsertText,
		line:    line,
		col:     col,
		endLine: endLine,
		endCol:  endCol,
		text:    textLines,
	})

	return endLine, endCol
}

// DeleteRange deletes text between two positions and returns the deleted text.
func (b *Buffer) DeleteRange(startLine, startCol, endLine, endCol int) string {
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, startCol, endLine, endCol = endLine, endCol, startLine, startCol
	}

	// Capture deleted text for undo
	var deleted [][]rune
	if startLine == endLine {
		line := b.lines[startLine]
		if startCol > len(line) {
			startCol = len(line)
		}
		if endCol > len(line) {
			endCol = len(line)
		}
		deleted = [][]rune{append([]rune{}, line[startCol:endCol]...)}
	} else {
		first := b.lines[startLine]
		if startCol > len(first) {
			startCol = len(first)
		}
		deleted = append(deleted, append([]rune{}, first[startCol:]...))
		for i := startLine + 1; i < endLine; i++ {
			deleted = append(deleted, append([]rune{}, b.lines[i]...))
		}
		last := b.lines[endLine]
		if endCol > len(last) {
			endCol = len(last)
		}
		deleted = append(deleted, append([]rune{}, last[:endCol]...))
	}

	// Build the result line
	var result []rune
	if startCol <= len(b.lines[startLine]) {
		result = append(result, b.lines[startLine][:startCol]...)
	}
	if endCol <= len(b.lines[endLine]) {
		result = append(result, b.lines[endLine][endCol:]...)
	}

	// Replace the lines
	newLines := make([][]rune, 0, len(b.lines)-(endLine-startLine))
	newLines = append(newLines, b.lines[:startLine]...)
	newLines = append(newLines, result)
	newLines = append(newLines, b.lines[endLine+1:]...)
	b.lines = newLines
	b.dirty = true

	b.pushUndo(undoEntry{
		kind:    undoDeleteRange,
		line:    startLine,
		col:     startCol,
		endLine: endLine,
		endCol:  endCol,
		text:    deleted,
	})

	// Build return string
	var s []byte
	for i, d := range deleted {
		if i > 0 {
			s = append(s, '\n')
		}
		s = append(s, string(d)...)
	}
	return string(s)
}

// WordAt returns the start and end column of the word at the given position.
func (b *Buffer) WordAt(line, col int) (start, end int) {
	if line < 0 || line >= len(b.lines) {
		return col, col
	}
	row := b.lines[line]
	if col < 0 || col >= len(row) {
		return col, col
	}

	// Expand left
	start = col
	for start > 0 && isWordChar(row[start-1]) {
		start--
	}
	// Expand right
	end = col
	for end < len(row) && isWordChar(row[end]) {
		end++
	}
	return start, end
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// WordCount returns the number of words in the buffer, optionally excluding frontmatter.
func (b *Buffer) WordCount() int {
	count := 0
	inFrontmatter := false
	frontmatterDone := false

	for i, line := range b.lines {
		// Check for frontmatter delimiters
		if i == 0 && len(line) >= 3 && string(line[:3]) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && !frontmatterDone && len(line) >= 3 && string(line[:3]) == "---" {
			frontmatterDone = true
			inFrontmatter = false
			continue
		}
		if inFrontmatter {
			continue
		}

		inWord := false
		for _, r := range line {
			if unicode.IsSpace(r) {
				inWord = false
			} else if !inWord {
				inWord = true
				count++
			}
		}
	}
	return count
}

// Undo/Redo

func (b *Buffer) pushUndo(e undoEntry) {
	if b.undoGroup > 0 {
		e.group = b.undoGroup
	}
	b.undoStack = append(b.undoStack, e)
	b.redoStack = nil // clear redo on new edit
}

// BeginUndoGroup starts a group so multiple edits are undone as one.
func (b *Buffer) BeginUndoGroup() {
	b.undoGroup++
}

// EndUndoGroup ends the current undo group.
func (b *Buffer) EndUndoGroup() {
	if b.undoGroup > 0 {
		b.undoGroup--
	}
}

// Undo reverses the last edit and returns the cursor position to restore.
func (b *Buffer) Undo() (line, col int, ok bool) {
	if len(b.undoStack) == 0 {
		return 0, 0, false
	}

	top := b.undoStack[len(b.undoStack)-1]
	line, col, ok = b.undoOne()
	if !ok {
		return
	}

	// If this entry belongs to a group, undo all entries in the same group
	if top.group > 0 {
		for len(b.undoStack) > 0 && b.undoStack[len(b.undoStack)-1].group == top.group {
			line, col, _ = b.undoOne()
		}
	}

	return line, col, true
}

// undoOne reverses a single undo entry.
func (b *Buffer) undoOne() (line, col int, ok bool) {
	if len(b.undoStack) == 0 {
		return 0, 0, false
	}

	e := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]

	// Temporarily disable undo recording
	savedUndo := b.undoStack
	savedRedo := b.redoStack

	switch e.kind {
	case undoInsertChar:
		// Undo insert: delete the range of coalesced chars
		// Find how many chars were coalesced
		count := e.endCol - e.col
		for i := 0; i < count; i++ {
			if e.col < len(b.lines[e.line]) {
				row := b.lines[e.line]
				b.lines[e.line] = append(row[:e.col], row[e.col+1:]...)
			}
		}
		line, col = e.line, e.col

	case undoDeleteChar:
		// Undo delete: re-insert the char
		row := b.lines[e.line]
		newRow := make([]rune, len(row)+1)
		copy(newRow, row[:e.col])
		newRow[e.col] = e.char
		copy(newRow[e.col+1:], row[e.col:])
		b.lines[e.line] = newRow
		line, col = e.line, e.col+1

	case undoInsertNewline:
		// Undo newline: join the two lines back
		if e.line+1 < len(b.lines) {
			b.lines[e.line] = append(b.lines[e.line], b.lines[e.line+1]...)
			b.lines = append(b.lines[:e.line+1], b.lines[e.line+2:]...)
		}
		line, col = e.line, e.col

	case undoJoinLine:
		// Undo join: split the line back
		row := b.lines[e.line]
		before := make([]rune, e.col)
		copy(before, row[:e.col])
		after := make([]rune, len(row)-e.col)
		copy(after, row[e.col:])
		newLines := make([][]rune, len(b.lines)+1)
		copy(newLines, b.lines[:e.line])
		newLines[e.line] = before
		newLines[e.line+1] = after
		copy(newLines[e.line+2:], b.lines[e.line+1:])
		b.lines = newLines
		line, col = e.line, e.col

	case undoInsertText:
		// Undo text insert: delete the range
		var result []rune
		if e.col <= len(b.lines[e.line]) {
			result = append(result, b.lines[e.line][:e.col]...)
		}
		if e.endLine < len(b.lines) && e.endCol <= len(b.lines[e.endLine]) {
			result = append(result, b.lines[e.endLine][e.endCol:]...)
		}
		newLines := make([][]rune, 0, len(b.lines)-(e.endLine-e.line))
		newLines = append(newLines, b.lines[:e.line]...)
		newLines = append(newLines, result)
		newLines = append(newLines, b.lines[e.endLine+1:]...)
		b.lines = newLines
		line, col = e.line, e.col

	case undoDeleteRange:
		// Undo delete range: re-insert the text
		if len(e.text) == 1 {
			row := b.lines[e.line]
			newRow := make([]rune, 0, len(row)+len(e.text[0]))
			newRow = append(newRow, row[:e.col]...)
			newRow = append(newRow, e.text[0]...)
			newRow = append(newRow, row[e.col:]...)
			b.lines[e.line] = newRow
		} else {
			origLine := b.lines[e.line]
			before := append([]rune{}, origLine[:e.col]...)
			after := append([]rune{}, origLine[e.col:]...)

			firstLine := append(before, e.text[0]...)
			lastLine := append(append([]rune{}, e.text[len(e.text)-1]...), after...)

			newLines := make([][]rune, 0, len(b.lines)+len(e.text)-1)
			newLines = append(newLines, b.lines[:e.line]...)
			newLines = append(newLines, firstLine)
			for i := 1; i < len(e.text)-1; i++ {
				newLines = append(newLines, e.text[i])
			}
			newLines = append(newLines, lastLine)
			newLines = append(newLines, b.lines[e.line+1:]...)
			b.lines = newLines
		}
		line, col = e.endLine, e.endCol
	}

	b.undoStack = savedUndo
	b.redoStack = append(savedRedo, e)
	b.dirty = true
	return line, col, true
}

// Redo re-applies the last undone edit.
func (b *Buffer) Redo() (line, col int, ok bool) {
	if len(b.redoStack) == 0 {
		return 0, 0, false
	}

	top := b.redoStack[len(b.redoStack)-1]
	line, col, ok = b.redoOne()
	if !ok {
		return
	}

	// If this entry belongs to a group, redo all entries in the same group
	if top.group > 0 {
		for len(b.redoStack) > 0 && b.redoStack[len(b.redoStack)-1].group == top.group {
			line, col, _ = b.redoOne()
		}
	}

	return line, col, true
}

// redoOne re-applies a single undo entry.
func (b *Buffer) redoOne() (line, col int, ok bool) {
	if len(b.redoStack) == 0 {
		return 0, 0, false
	}

	e := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]

	savedUndo := b.undoStack
	savedRedo := b.redoStack

	switch e.kind {
	case undoInsertChar:
		// Re-insert all coalesced chars
		for i, ch := range e.chars {
			pos := e.col + i
			row := b.lines[e.line]
			newRow := make([]rune, len(row)+1)
			copy(newRow, row[:pos])
			newRow[pos] = ch
			copy(newRow[pos+1:], row[pos:])
			b.lines[e.line] = newRow
		}
		line, col = e.line, e.endCol

	case undoDeleteChar:
		if e.col < len(b.lines[e.line]) {
			row := b.lines[e.line]
			b.lines[e.line] = append(row[:e.col], row[e.col+1:]...)
		}
		line, col = e.line, e.col

	case undoInsertNewline:
		// Re-split
		row := b.lines[e.line]
		before := make([]rune, e.col)
		copy(before, row[:e.col])
		after := make([]rune, len(row)-e.col)
		copy(after, row[e.col:])
		newLines := make([][]rune, len(b.lines)+1)
		copy(newLines, b.lines[:e.line])
		newLines[e.line] = before
		newLines[e.line+1] = after
		copy(newLines[e.line+2:], b.lines[e.line+1:])
		b.lines = newLines
		line, col = e.line+1, 0

	case undoJoinLine:
		// Re-join
		if e.line+1 < len(b.lines) {
			b.lines[e.line] = append(b.lines[e.line], b.lines[e.line+1]...)
			b.lines = append(b.lines[:e.line+1], b.lines[e.line+2:]...)
		}
		line, col = e.line, e.col

	case undoInsertText:
		// Re-insert the text
		origLine := b.lines[e.line]
		before := append([]rune{}, origLine[:e.col]...)
		after := append([]rune{}, origLine[e.col:]...)

		if len(e.text) == 1 {
			b.lines[e.line] = append(append(before, e.text[0]...), after...)
		} else {
			firstLine := append(before, e.text[0]...)
			lastLine := append(append([]rune{}, e.text[len(e.text)-1]...), after...)
			newLines := make([][]rune, 0, len(b.lines)+len(e.text)-1)
			newLines = append(newLines, b.lines[:e.line]...)
			newLines = append(newLines, firstLine)
			for i := 1; i < len(e.text)-1; i++ {
				newLines = append(newLines, e.text[i])
			}
			newLines = append(newLines, lastLine)
			newLines = append(newLines, b.lines[e.line+1:]...)
			b.lines = newLines
		}
		line, col = e.endLine, e.endCol

	case undoDeleteRange:
		// Re-delete the range
		var result []rune
		result = append(result, b.lines[e.line][:e.col]...)
		result = append(result, b.lines[e.endLine][e.endCol:]...)
		newLines := make([][]rune, 0, len(b.lines)-(e.endLine-e.line))
		newLines = append(newLines, b.lines[:e.line]...)
		newLines = append(newLines, result)
		newLines = append(newLines, b.lines[e.endLine+1:]...)
		b.lines = newLines
		line, col = e.line, e.col
	}

	b.undoStack = append(savedUndo, e)
	b.redoStack = savedRedo
	b.dirty = true
	return line, col, true
}
