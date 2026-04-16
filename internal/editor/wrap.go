package editor

import (
	"github.com/mattn/go-runewidth"
)

// WrapResult holds the result of wrapping logical lines into visual lines.
type WrapResult struct {
	// VisualLines contains each visual row as a slice of runes.
	VisualLines []VisualLine
	// LineMap maps logical line index → range of visual line indices [start, end).
	LineMap []LineRange
}

// VisualLine represents a single visual row on screen.
type VisualLine struct {
	Runes       []rune
	LogicalLine int // which logical line this belongs to
	LogicalCol  int // rune offset within the logical line where this visual line starts
}

// LineRange represents a range of visual lines for a logical line.
type LineRange struct {
	Start int
	End   int
}

// WrapLines wraps logical buffer lines into visual lines at the given width.
// It performs word-aware wrapping using display width.
func WrapLines(lines [][]rune, width int) WrapResult {
	if width <= 0 {
		width = 80
	}

	result := WrapResult{
		LineMap: make([]LineRange, len(lines)),
	}

	for logIdx, line := range lines {
		startVisual := len(result.VisualLines)

		if len(line) == 0 {
			result.VisualLines = append(result.VisualLines, VisualLine{
				Runes:       nil,
				LogicalLine: logIdx,
				LogicalCol:  0,
			})
			result.LineMap[logIdx] = LineRange{Start: startVisual, End: startVisual + 1}
			continue
		}

		wrapped := wrapLine(line, width, logIdx)
		result.VisualLines = append(result.VisualLines, wrapped...)
		result.LineMap[logIdx] = LineRange{Start: startVisual, End: startVisual + len(wrapped)}
	}

	return result
}

// wrapLine wraps a single logical line into one or more visual lines.
func wrapLine(line []rune, width int, logicalLine int) []VisualLine {
	var result []VisualLine
	lineStart := 0

	for lineStart < len(line) {
		// Find how many runes fit within width
		displayW := 0
		end := lineStart
		lastBreak := -1

		for end < len(line) {
			rw := runewidth.RuneWidth(line[end])
			if displayW+rw > width {
				break
			}
			displayW += rw
			if line[end] == ' ' || line[end] == '\t' {
				lastBreak = end
			}
			end++
		}

		if end == len(line) {
			// Rest of line fits
			segment := make([]rune, end-lineStart)
			copy(segment, line[lineStart:end])
			result = append(result, VisualLine{
				Runes:       segment,
				LogicalLine: logicalLine,
				LogicalCol:  lineStart,
			})
			break
		}

		// Need to wrap — prefer breaking at word boundary
		breakAt := end
		if lastBreak > lineStart {
			breakAt = lastBreak + 1 // break after the space
		}

		segment := make([]rune, breakAt-lineStart)
		copy(segment, line[lineStart:breakAt])
		result = append(result, VisualLine{
			Runes:       segment,
			LogicalLine: logicalLine,
			LogicalCol:  lineStart,
		})
		lineStart = breakAt
	}

	if len(result) == 0 {
		result = append(result, VisualLine{
			Runes:       nil,
			LogicalLine: logicalLine,
			LogicalCol:  0,
		})
	}

	return result
}

// LogicalToVisual converts a logical (line, col) to a visual (row, col).
func (w *WrapResult) LogicalToVisual(logLine, logCol int) (visRow, visCol int) {
	if logLine < 0 || logLine >= len(w.LineMap) {
		return 0, 0
	}

	lr := w.LineMap[logLine]
	for i := lr.Start; i < lr.End; i++ {
		vl := w.VisualLines[i]
		nextStart := vl.LogicalCol + len(vl.Runes)
		if i+1 < lr.End {
			nextStart = w.VisualLines[i+1].LogicalCol
		}

		if logCol < nextStart || i == lr.End-1 {
			visRow = i
			visCol = logCol - vl.LogicalCol
			return visRow, visCol
		}
	}

	return lr.Start, logCol
}

// VisualToLogical converts a visual (row, col) to a logical (line, col).
func (w *WrapResult) VisualToLogical(visRow, visCol int) (logLine, logCol int) {
	if visRow < 0 {
		visRow = 0
	}
	if visRow >= len(w.VisualLines) {
		visRow = len(w.VisualLines) - 1
	}
	if visRow < 0 {
		return 0, 0
	}

	vl := w.VisualLines[visRow]
	logLine = vl.LogicalLine
	logCol = vl.LogicalCol + visCol

	// Clamp to visual line length
	if visCol > len(vl.Runes) {
		logCol = vl.LogicalCol + len(vl.Runes)
	}
	if visCol < 0 {
		logCol = vl.LogicalCol
	}

	return logLine, logCol
}

// VisualLineCount returns the total number of visual lines.
func (w *WrapResult) VisualLineCount() int {
	return len(w.VisualLines)
}
