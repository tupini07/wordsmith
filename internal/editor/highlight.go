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

// Theme defines colors for each token kind.
// IMPORTANT: Every style MUST set an explicit Background to prevent the
// terminal's own background from bleeding through.
type Theme struct {
	// Base colors used by components that derive styles from the theme
	Bg          lipgloss.Color
	ChromeBg    lipgloss.Color
	Fg          lipgloss.Color
	AccentColor lipgloss.Color
	DimColor    lipgloss.Color
	DirColor    lipgloss.Color

	// Token styles (all have explicit backgrounds)
	Normal      lipgloss.Style
	Bold        lipgloss.Style
	Italic      lipgloss.Style
	BoldItalic  lipgloss.Style
	Heading     lipgloss.Style
	Heading2    lipgloss.Style
	Heading3    lipgloss.Style
	Link        lipgloss.Style
	LinkURL     lipgloss.Style
	InlineCode  lipgloss.Style
	Blockquote  lipgloss.Style
	ListMarker  lipgloss.Style
	Frontmatter lipgloss.Style
	HR          lipgloss.Style
	Cursor      lipgloss.Style
	Selection   lipgloss.Style
	StatusBar   lipgloss.Style
	TitleBar    lipgloss.Style
	FileTree    lipgloss.Style
	FileTreeSel lipgloss.Style
	FileTreeDir lipgloss.Style
	Margin      lipgloss.Style
	Dimmed      lipgloss.Style
	ActiveLine  lipgloss.Style // background for the cursor's line
}

// DefaultTheme returns a warm, cozy color palette.
func DefaultTheme() Theme {
	bg := lipgloss.Color("#1E1C1F")
	chromeBg := lipgloss.Color("#2D2A2E")
	midBg := lipgloss.Color("#3E3B40")

	cream := lipgloss.Color("#F5E6D3")
	warmWhite := lipgloss.Color("#E8DCC8")
	orange := lipgloss.Color("#E8A87C")
	coral := lipgloss.Color("#E27D60")
	teal := lipgloss.Color("#41B3A3")
	purple := lipgloss.Color("#C38D9E")
	dimText := lipgloss.Color("#8A8488")

	return Theme{
		Bg:          bg,
		ChromeBg:    chromeBg,
		Fg:          cream,
		AccentColor: orange,
		DimColor:    dimText,
		DirColor:    teal,

		Normal:      lipgloss.NewStyle().Foreground(cream).Background(bg),
		Bold:        lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true),
		Italic:      lipgloss.NewStyle().Foreground(warmWhite).Background(bg).Italic(true),
		BoldItalic:  lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true).Italic(true),
		Heading:     lipgloss.NewStyle().Foreground(coral).Background(bg).Bold(true),
		Heading2:    lipgloss.NewStyle().Foreground(coral).Background(bg).Bold(true),
		Heading3:    lipgloss.NewStyle().Foreground(coral).Background(bg),
		Link:        lipgloss.NewStyle().Foreground(teal).Background(bg).Underline(true),
		LinkURL:     lipgloss.NewStyle().Foreground(dimText).Background(bg),
		InlineCode:  lipgloss.NewStyle().Foreground(purple).Background(midBg),
		Blockquote:  lipgloss.NewStyle().Foreground(dimText).Background(bg).Italic(true),
		ListMarker:  lipgloss.NewStyle().Foreground(teal).Background(bg),
		Frontmatter: lipgloss.NewStyle().Foreground(dimText).Background(bg),
		HR:          lipgloss.NewStyle().Foreground(dimText).Background(bg),
		Cursor:      lipgloss.NewStyle().Foreground(bg).Background(cream),
		Selection:   lipgloss.NewStyle().Background(lipgloss.Color("#4A4550")),
		StatusBar:   lipgloss.NewStyle().Background(chromeBg).Foreground(cream).Padding(0, 1),
		TitleBar:    lipgloss.NewStyle().Background(chromeBg).Foreground(cream).Padding(0, 1).Bold(true),
		FileTree:    lipgloss.NewStyle().Foreground(cream).Background(bg),
		FileTreeSel: lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true),
		FileTreeDir: lipgloss.NewStyle().Foreground(teal).Background(bg).Bold(true),
		Margin:      lipgloss.NewStyle().Background(bg),
		Dimmed:      lipgloss.NewStyle().Foreground(dimText).Background(bg),
		ActiveLine:  lipgloss.NewStyle().Background(chromeBg),
	}
}

// NordTheme returns a cool blue-gray palette inspired by Nord.
func NordTheme() Theme {
	bg := lipgloss.Color("#2E3440")
	chromeBg := lipgloss.Color("#3B4252")
	midBg := lipgloss.Color("#434C5E")

	snow := lipgloss.Color("#ECEFF4")
	snow2 := lipgloss.Color("#D8DEE9")
	frost1 := lipgloss.Color("#8FBCBB")
	frost2 := lipgloss.Color("#88C0D0")
	frost3 := lipgloss.Color("#81A1C1")
	aurora1 := lipgloss.Color("#BF616A") // red
	aurora4 := lipgloss.Color("#B48EAD") // purple
	dimText := lipgloss.Color("#7B88A1") // brightened for readability (original #616E88 too dim)

	return Theme{
		Bg:          bg,
		ChromeBg:    chromeBg,
		Fg:          snow,
		AccentColor: frost2,
		DimColor:    dimText,
		DirColor:    frost1,

		Normal:      lipgloss.NewStyle().Foreground(snow).Background(bg),
		Bold:        lipgloss.NewStyle().Foreground(frost2).Background(bg).Bold(true),
		Italic:      lipgloss.NewStyle().Foreground(snow2).Background(bg).Italic(true),
		BoldItalic:  lipgloss.NewStyle().Foreground(frost2).Background(bg).Bold(true).Italic(true),
		Heading:     lipgloss.NewStyle().Foreground(aurora1).Background(bg).Bold(true),
		Heading2:    lipgloss.NewStyle().Foreground(frost3).Background(bg).Bold(true),
		Heading3:    lipgloss.NewStyle().Foreground(frost3).Background(bg),
		Link:        lipgloss.NewStyle().Foreground(frost1).Background(bg).Underline(true),
		LinkURL:     lipgloss.NewStyle().Foreground(dimText).Background(bg),
		InlineCode:  lipgloss.NewStyle().Foreground(aurora4).Background(midBg),
		Blockquote:  lipgloss.NewStyle().Foreground(dimText).Background(bg).Italic(true),
		ListMarker:  lipgloss.NewStyle().Foreground(frost1).Background(bg),
		Frontmatter: lipgloss.NewStyle().Foreground(dimText).Background(bg),
		HR:          lipgloss.NewStyle().Foreground(dimText).Background(bg),
		Cursor:      lipgloss.NewStyle().Foreground(bg).Background(snow),
		Selection:   lipgloss.NewStyle().Background(midBg),
		StatusBar:   lipgloss.NewStyle().Background(chromeBg).Foreground(snow).Padding(0, 1),
		TitleBar:    lipgloss.NewStyle().Background(chromeBg).Foreground(snow).Padding(0, 1).Bold(true),
		FileTree:    lipgloss.NewStyle().Foreground(snow).Background(bg),
		FileTreeSel: lipgloss.NewStyle().Foreground(frost2).Background(bg).Bold(true),
		FileTreeDir: lipgloss.NewStyle().Foreground(frost1).Background(bg).Bold(true),
		Margin:      lipgloss.NewStyle().Background(bg),
		Dimmed:      lipgloss.NewStyle().Foreground(dimText).Background(bg),
		ActiveLine:  lipgloss.NewStyle().Background(chromeBg),
	}
}

// DraculaTheme returns a dark purple-accented palette inspired by Dracula.
func DraculaTheme() Theme {
	bg := lipgloss.Color("#282A36")
	chromeBg := lipgloss.Color("#343746")
	midBg := lipgloss.Color("#44475A")

	fg := lipgloss.Color("#F8F8F2")
	fgItalic := lipgloss.Color("#E8E4DE") // soft cream (yellow too jarring for prose)
	purple := lipgloss.Color("#BD93F9")
	pink := lipgloss.Color("#FF79C6")
	green := lipgloss.Color("#50FA7B")
	cyan := lipgloss.Color("#8BE9FD")
	orange := lipgloss.Color("#FFB86C")
	dimText := lipgloss.Color("#6272A4")

	return Theme{
		Bg:          bg,
		ChromeBg:    chromeBg,
		Fg:          fg,
		AccentColor: purple,
		DimColor:    dimText,
		DirColor:    cyan,

		Normal:      lipgloss.NewStyle().Foreground(fg).Background(bg),
		Bold:        lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true),
		Italic:      lipgloss.NewStyle().Foreground(fgItalic).Background(bg).Italic(true),
		BoldItalic:  lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true).Italic(true),
		Heading:     lipgloss.NewStyle().Foreground(pink).Background(bg).Bold(true),
		Heading2:    lipgloss.NewStyle().Foreground(purple).Background(bg).Bold(true),
		Heading3:    lipgloss.NewStyle().Foreground(purple).Background(bg),
		Link:        lipgloss.NewStyle().Foreground(cyan).Background(bg).Underline(true),
		LinkURL:     lipgloss.NewStyle().Foreground(dimText).Background(bg),
		InlineCode:  lipgloss.NewStyle().Foreground(green).Background(midBg),
		Blockquote:  lipgloss.NewStyle().Foreground(dimText).Background(bg).Italic(true),
		ListMarker:  lipgloss.NewStyle().Foreground(pink).Background(bg),
		Frontmatter: lipgloss.NewStyle().Foreground(dimText).Background(bg),
		HR:          lipgloss.NewStyle().Foreground(dimText).Background(bg),
		Cursor:      lipgloss.NewStyle().Foreground(bg).Background(fg),
		Selection:   lipgloss.NewStyle().Background(midBg),
		StatusBar:   lipgloss.NewStyle().Background(chromeBg).Foreground(fg).Padding(0, 1),
		TitleBar:    lipgloss.NewStyle().Background(chromeBg).Foreground(fg).Padding(0, 1).Bold(true),
		FileTree:    lipgloss.NewStyle().Foreground(fg).Background(bg),
		FileTreeSel: lipgloss.NewStyle().Foreground(purple).Background(bg).Bold(true),
		FileTreeDir: lipgloss.NewStyle().Foreground(cyan).Background(bg).Bold(true),
		Margin:      lipgloss.NewStyle().Background(bg),
		Dimmed:      lipgloss.NewStyle().Foreground(dimText).Background(bg),
		ActiveLine:  lipgloss.NewStyle().Background(chromeBg),
	}
}

// GruvboxTheme returns a warm retro palette inspired by Gruvbox.
func GruvboxTheme() Theme {
	bg := lipgloss.Color("#282828")
	chromeBg := lipgloss.Color("#3C3836")
	midBg := lipgloss.Color("#504945")

	fg := lipgloss.Color("#EBDBB2")
	fg2 := lipgloss.Color("#D5C4A1")
	orange := lipgloss.Color("#FE8019")
	yellow := lipgloss.Color("#FABD2F")
	aqua := lipgloss.Color("#8EC07C")
	blue := lipgloss.Color("#83A598")
	purple := lipgloss.Color("#D3869B")
	red := lipgloss.Color("#FB4934")
	dimText := lipgloss.Color("#928374")

	return Theme{
		Bg:          bg,
		ChromeBg:    chromeBg,
		Fg:          fg,
		AccentColor: orange,
		DimColor:    dimText,
		DirColor:    aqua,

		Normal:      lipgloss.NewStyle().Foreground(fg).Background(bg),
		Bold:        lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true),
		Italic:      lipgloss.NewStyle().Foreground(fg2).Background(bg).Italic(true),
		BoldItalic:  lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true).Italic(true),
		Heading:     lipgloss.NewStyle().Foreground(red).Background(bg).Bold(true),
		Heading2:    lipgloss.NewStyle().Foreground(yellow).Background(bg).Bold(true),
		Heading3:    lipgloss.NewStyle().Foreground(yellow).Background(bg),
		Link:        lipgloss.NewStyle().Foreground(blue).Background(bg).Underline(true),
		LinkURL:     lipgloss.NewStyle().Foreground(dimText).Background(bg),
		InlineCode:  lipgloss.NewStyle().Foreground(purple).Background(midBg),
		Blockquote:  lipgloss.NewStyle().Foreground(dimText).Background(bg).Italic(true),
		ListMarker:  lipgloss.NewStyle().Foreground(aqua).Background(bg),
		Frontmatter: lipgloss.NewStyle().Foreground(dimText).Background(bg),
		HR:          lipgloss.NewStyle().Foreground(dimText).Background(bg),
		Cursor:      lipgloss.NewStyle().Foreground(bg).Background(fg),
		Selection:   lipgloss.NewStyle().Background(midBg),
		StatusBar:   lipgloss.NewStyle().Background(chromeBg).Foreground(fg).Padding(0, 1),
		TitleBar:    lipgloss.NewStyle().Background(chromeBg).Foreground(fg).Padding(0, 1).Bold(true),
		FileTree:    lipgloss.NewStyle().Foreground(fg).Background(bg),
		FileTreeSel: lipgloss.NewStyle().Foreground(orange).Background(bg).Bold(true),
		FileTreeDir: lipgloss.NewStyle().Foreground(aqua).Background(bg).Bold(true),
		Margin:      lipgloss.NewStyle().Background(bg),
		Dimmed:      lipgloss.NewStyle().Foreground(dimText).Background(bg),
		ActiveLine:  lipgloss.NewStyle().Background(chromeBg),
	}
}

// ThemeByName returns a theme by its config name. Falls back to DefaultTheme.
func ThemeByName(name string) Theme {
	switch strings.ToLower(name) {
	case "nord":
		return NordTheme()
	case "dracula":
		return DraculaTheme()
	case "gruvbox":
		return GruvboxTheme()
	default:
		return DefaultTheme()
	}
}

// HighlightLine tokenizes a single line of markdown for rendering.
func HighlightLine(line []rune, theme Theme, inFrontmatter bool) ([]Token, bool) {
	s := string(line)

	// Frontmatter delimiter
	trimmed := strings.TrimSpace(s)
	if trimmed == "---" {
		return []Token{{Text: s, Style: theme.Frontmatter}}, !inFrontmatter
	}
	if inFrontmatter {
		return []Token{{Text: s, Style: theme.Frontmatter}}, true
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
			return []Token{{Text: s, Style: style}}, false
		}
	}

	// Blockquote
	if len(trimmed) > 0 && trimmed[0] == '>' {
		return []Token{{Text: s, Style: theme.Blockquote}}, false
	}

	// Horizontal rule
	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		return []Token{{Text: s, Style: theme.HR}}, false
	}

	// Inline tokenization
	tokens := tokenizeInline(line, theme)
	return tokens, false
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

		current = append(current, line[i])
		i++
	}

	flushCurrent()
	return tokens
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
