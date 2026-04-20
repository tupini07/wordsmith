package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Token represents a highlighted segment of text.
type Token struct {
	Text  string
	Style lipgloss.Style
}

// TokenKind identifies a markdown token type.
type TokenKind int

const (
	TokenNormal TokenKind = iota
	TokenBold
	TokenItalic
	TokenBoldItalic
	TokenHeading
	TokenLink
	TokenLinkURL
	TokenInlineCode
	TokenBlockquote
	TokenListMarker
	TokenFrontmatter
	TokenHR
)

// HighlightState tracks persistent state across lines.
type HighlightState struct {
	InFrontmatter bool
	InCodeBlock   bool
	InBlockquote  bool
}

// HighlightLine tokenizes a single line of markdown for rendering.
// logicalLine is the 0-based buffer line number, used to restrict
// frontmatter detection to the very first line of the document.
func HighlightLine(line []rune, theme Theme, state HighlightState, logicalLine int) ([]Token, HighlightState) {
	s := string(line)
	trimmed := strings.TrimSpace(s)

	// Code fence delimiter (``` or ~~~)
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		if state.InCodeBlock {
			return []Token{{Text: s, Style: theme.CodeFence}}, HighlightState{}
		}
		if !state.InFrontmatter {
			return []Token{{Text: s, Style: theme.CodeFence}}, HighlightState{InCodeBlock: true}
		}
	}

	// Inside code block — render as code, no inline parsing
	if state.InCodeBlock {
		return []Token{{Text: s, Style: theme.CodeBlock}}, state
	}

	// Frontmatter delimiter
	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		if state.InFrontmatter {
			return []Token{{Text: s, Style: theme.Frontmatter}}, HighlightState{}
		}
		if trimmed == "---" && logicalLine == 0 {
			return []Token{{Text: s, Style: theme.Frontmatter}}, HighlightState{InFrontmatter: true}
		}
		// Horizontal rule
		return []Token{{Text: s, Style: theme.HR}}, state
	}
	if state.InFrontmatter {
		return []Token{{Text: s, Style: theme.Frontmatter}}, state
	}

	// Heading
	if len(trimmed) > 0 && trimmed[0] == '#' {
		level := 0
		for _, c := range trimmed {
			if c == '#' {
				level++
			} else {
				break
			}
		}
		if level <= 6 && level < len(trimmed) && trimmed[level] == ' ' {
			style := theme.Heading
			if level == 2 {
				style = theme.Heading2
			} else if level >= 3 {
				style = theme.Heading3
			}
			return []Token{{Text: s, Style: style}}, state
		}
	}

	// Blockquote
	if len(trimmed) > 0 && trimmed[0] == '>' {
		bqState := state
		bqState.InBlockquote = true
		return []Token{{Text: s, Style: theme.Blockquote}}, bqState
	}

	// Blockquote continuation (soft-wrapped sub-lines)
	if state.InBlockquote {
		return []Token{{Text: s, Style: theme.Blockquote}}, state
	}

	// Table row: starts with |
	if len(trimmed) > 0 && trimmed[0] == '|' {
		return tokenizeTableRow(line, theme), state
	}

	// Inline tokenization
	tokens := tokenizeInline(line, theme)
	return tokens, state
}

// tokenizeInline handles bold, italic, code, and links within a line.
func tokenizeInline(line []rune, theme Theme) []Token {
	var tokens []Token
	i := 0
	n := len(line)

	// Check for list markers at the start
	listEnd := detectListMarker(line)
	if listEnd > 0 {
		tokens = append(tokens, Token{
			Text:  string(line[:listEnd]),
			Style: theme.ListMarker,
		})
		i = listEnd
	}

	var current []rune

	flushCurrent := func() {
		if len(current) > 0 {
			tokens = append(tokens, Token{Text: string(current), Style: theme.Normal})
			current = nil
		}
	}

	for i < n {
		// Inline code: `...`
		if line[i] == '`' {
			flushCurrent()
			end := findClosing(line, i+1, '`')
			if end > 0 {
				tokens = append(tokens, Token{
					Text:  string(line[i : end+1]),
					Style: theme.InlineCode,
				})
				i = end + 1
				continue
			}
		}

		// Bold+Italic: ***...*** or ___...___
		if i+2 < n && ((line[i] == '*' && line[i+1] == '*' && line[i+2] == '*') ||
			(line[i] == '_' && line[i+1] == '_' && line[i+2] == '_')) {
			marker := line[i]
			end := findTriple(line, i+3, marker)
			if end > 0 {
				flushCurrent()
				tokens = append(tokens, Token{
					Text:  string(line[i : end+3]),
					Style: theme.BoldItalic,
				})
				i = end + 3
				continue
			}
		}

		// Bold: **...** or __...__
		if i+1 < n && ((line[i] == '*' && line[i+1] == '*') || (line[i] == '_' && line[i+1] == '_')) {
			marker := line[i]
			end := findDouble(line, i+2, marker)
			if end > 0 {
				flushCurrent()
				tokens = append(tokens, Token{
					Text:  string(line[i : end+2]),
					Style: theme.Bold,
				})
				i = end + 2
				continue
			}
		}

		// Italic: *...* or _..._
		if (line[i] == '*' || line[i] == '_') && i+1 < n && line[i+1] != ' ' {
			marker := line[i]
			end := findClosing(line, i+1, marker)
			if end > 0 && end > i+1 {
				flushCurrent()
				tokens = append(tokens, Token{
					Text:  string(line[i : end+1]),
					Style: theme.Italic,
				})
				i = end + 1
				continue
			}
		}

		// Footnote reference: [^id]
		if line[i] == '[' && i+2 < n && line[i+1] == '^' {
			// Find closing ]
			end := -1
			for j := i + 2; j < n; j++ {
				if line[j] == ']' {
					end = j
					break
				}
			}
			if end > i+2 {
				flushCurrent()
				tokens = append(tokens, Token{
					Text:  string(line[i : end+1]),
					Style: theme.FootnoteRef,
				})
				i = end + 1
				continue
			}
		}

		// Link: [text](url)
		if line[i] == '[' {
			linkEnd := parseLinkAt(line, i)
			if linkEnd > 0 {
				flushCurrent()
				// Find the ] and ( positions
				bracketEnd := -1
				for j := i + 1; j < n; j++ {
					if line[j] == ']' {
						bracketEnd = j
						break
					}
				}
				if bracketEnd > 0 && bracketEnd+1 < n && line[bracketEnd+1] == '(' {
					// [text]
					tokens = append(tokens, Token{
						Text:  string(line[i : bracketEnd+1]),
						Style: theme.Link,
					})
					// (url)
					tokens = append(tokens, Token{
						Text:  string(line[bracketEnd+1 : linkEnd+1]),
						Style: theme.LinkURL,
					})
				}
				i = linkEnd + 1
				continue
			}
		}

		// Raw URL: http:// or https://
		if line[i] == 'h' && matchesAt(line, i, "http://") || matchesAt(line, i, "https://") {
			urlEnd := findURLEnd(line, i)
			if urlEnd > i {
				flushCurrent()
				tokens = append(tokens, Token{
					Text:  string(line[i:urlEnd]),
					Style: theme.LinkURL,
				})
				i = urlEnd
				continue
			}
		}

		current = append(current, line[i])
		i++
	}

	flushCurrent()
	return tokens
}

// tokenizeTableRow splits a table row into pipe (border) and cell tokens.
// Separator rows (|---|---| etc.) are rendered entirely as border style.
func tokenizeTableRow(line []rune, theme Theme) []Token {
	s := string(line)
	if isTableSeparator(s) {
		return []Token{{Text: s, Style: theme.TableBorder}}
	}

	var tokens []Token
	i := 0
	n := len(line)
	for i < n {
		if line[i] == '|' {
			tokens = append(tokens, Token{Text: "|", Style: theme.TableBorder})
			i++
		} else {
			// Collect cell content until next | or end
			start := i
			for i < n && line[i] != '|' {
				i++
			}
			tokens = append(tokens, Token{Text: string(line[start:i]), Style: theme.Normal})
		}
	}
	return tokens
}

// isTableSeparator returns true for lines like |---|---|
func isTableSeparator(s string) bool {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 3 || trimmed[0] != '|' {
		return false
	}
	for _, r := range trimmed {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

func detectListMarker(line []rune) int {
	i := 0
	// Skip leading whitespace
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	if i >= len(line) {
		return 0
	}

	// Unordered: - or * or + followed by space
	if (line[i] == '-' || line[i] == '*' || line[i] == '+') && i+1 < len(line) && line[i+1] == ' ' {
		return i + 2
	}

	// Ordered: digits followed by . or ) and space
	start := i
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i > start && i < len(line) && (line[i] == '.' || line[i] == ')') && i+1 < len(line) && line[i+1] == ' ' {
		return i + 2
	}

	return 0
}

func findClosing(line []rune, start int, marker rune) int {
	for i := start; i < len(line); i++ {
		if line[i] == marker && (i == 0 || line[i-1] != '\\') {
			return i
		}
	}
	return -1
}

func findDouble(line []rune, start int, marker rune) int {
	for i := start; i+1 < len(line); i++ {
		if line[i] == marker && line[i+1] == marker {
			return i
		}
	}
	return -1
}

func findTriple(line []rune, start int, marker rune) int {
	for i := start; i+2 < len(line); i++ {
		if line[i] == marker && line[i+1] == marker && line[i+2] == marker {
			return i
		}
	}
	return -1
}

func parseLinkAt(line []rune, start int) int {
	// [text](url)
	i := start + 1
	n := len(line)

	// Find closing ]
	for i < n && line[i] != ']' {
		i++
	}
	if i >= n {
		return -1
	}

	// Must be followed by (
	if i+1 >= n || line[i+1] != '(' {
		return -1
	}

	// Find closing )
	i += 2
	depth := 1
	for i < n && depth > 0 {
		if line[i] == '(' {
			depth++
		} else if line[i] == ')' {
			depth--
		}
		if depth > 0 {
			i++
		}
	}
	if depth != 0 {
		return -1
	}
	return i
}

// matchesAt checks if substr appears at position pos in line.
func matchesAt(line []rune, pos int, substr string) bool {
	sr := []rune(substr)
	if pos+len(sr) > len(line) {
		return false
	}
	for i, r := range sr {
		if line[pos+i] != r {
			return false
		}
	}
	return true
}

// findURLEnd returns the index one past the end of a URL starting at pos.
// URLs end at whitespace or common trailing punctuation that's unlikely part of the URL.
func findURLEnd(line []rune, pos int) int {
	i := pos
	n := len(line)
	for i < n {
		r := line[i]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '<' || r == '>' || r == '"' || r == '\'' {
			break
		}
		i++
	}
	// Strip common trailing punctuation that's likely sentence-ending, not part of the URL
	for i > pos {
		last := line[i-1]
		if last == '.' || last == ',' || last == ';' || last == ':' ||
			last == '!' || last == '?' || last == ')' || last == ']' {
			i--
		} else {
			break
		}
	}
	// Must have some content after the scheme
	if i <= pos+7 { // len("http://") == 7
		return pos
	}
	return i
}

// RenderTokens renders a list of tokens into a styled string with cursor and selection.
// When activeLine is true, the token backgrounds are replaced with the active-line
// highlight color for a subtle "current line" indicator.
func RenderTokens(tokens []Token, cursorCol int, selStart, selEnd int, theme Theme, activeLine bool) string {
	var sb strings.Builder
	col := 0

	for _, tok := range tokens {
		for _, r := range tok.Text {
			rw := runewidth.RuneWidth(r)
			style := tok.Style

			// Apply active line background (preserves foreground + bold/italic)
			if activeLine {
				style = style.Background(theme.ActiveLine.GetBackground())
			}

			// Selection takes priority
			if selStart >= 0 && selEnd >= 0 && col >= selStart && col < selEnd {
				style = theme.Selection
			}

			// Cursor on top
			if col == cursorCol {
				style = theme.Cursor
			}

			sb.WriteString(style.Render(string(r)))
			col += rw
		}
	}

	// If cursor is at end of line, render a cursor space
	if cursorCol == col {
		sb.WriteString(theme.Cursor.Render(" "))
	}

	return sb.String()
}
