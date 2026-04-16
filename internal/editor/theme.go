package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	FootnoteRef lipgloss.Style
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

// ThemeColors defines the color palette for building a theme.
// Each field maps to a semantic role; buildTheme turns them into styles.
type ThemeColors struct {
	Bg       lipgloss.Color // main background
	ChromeBg lipgloss.Color // chrome (title, status, active line) background
	MidBg    lipgloss.Color // elevated surface (inline code, selection)

	Fg          lipgloss.Color // primary text
	FgItalic    lipgloss.Color // italic text (often slightly muted)
	Bold        lipgloss.Color // bold text
	Heading1    lipgloss.Color // h1 headings
	Heading2    lipgloss.Color // h2+ headings
	Link        lipgloss.Color // link text
	InlineCode  lipgloss.Color // inline code
	ListMarker  lipgloss.Color // bullet / number
	FootnoteRef lipgloss.Color // [^N] references
	Accent      lipgloss.Color // accent (finder highlight, file tree selection, etc.)
	Dim         lipgloss.Color // muted text (URLs, frontmatter, blockquotes, HR)
	Dir         lipgloss.Color // directory names in file tree
}

// buildTheme constructs a full Theme from a color palette.
func buildTheme(c ThemeColors) Theme {
	bg := c.Bg
	chromeBg := c.ChromeBg
	midBg := c.MidBg

	return Theme{
		Bg:          bg,
		ChromeBg:    chromeBg,
		Fg:          c.Fg,
		AccentColor: c.Accent,
		DimColor:    c.Dim,
		DirColor:    c.Dir,

		Normal:      lipgloss.NewStyle().Foreground(c.Fg).Background(bg),
		Bold:        lipgloss.NewStyle().Foreground(c.Bold).Background(bg).Bold(true),
		Italic:      lipgloss.NewStyle().Foreground(c.FgItalic).Background(bg).Italic(true),
		BoldItalic:  lipgloss.NewStyle().Foreground(c.Bold).Background(bg).Bold(true).Italic(true),
		Heading:     lipgloss.NewStyle().Foreground(c.Heading1).Background(bg).Bold(true),
		Heading2:    lipgloss.NewStyle().Foreground(c.Heading2).Background(bg).Bold(true),
		Heading3:    lipgloss.NewStyle().Foreground(c.Heading2).Background(bg),
		Link:        lipgloss.NewStyle().Foreground(c.Link).Background(bg).Underline(true),
		LinkURL:     lipgloss.NewStyle().Foreground(c.Dim).Background(bg),
		InlineCode:  lipgloss.NewStyle().Foreground(c.InlineCode).Background(midBg),
		Blockquote:  lipgloss.NewStyle().Foreground(c.Dim).Background(bg).Italic(true),
		ListMarker:  lipgloss.NewStyle().Foreground(c.ListMarker).Background(bg),
		Frontmatter: lipgloss.NewStyle().Foreground(c.Dim).Background(bg),
		HR:          lipgloss.NewStyle().Foreground(c.Dim).Background(bg),
		FootnoteRef: lipgloss.NewStyle().Foreground(c.FootnoteRef).Background(bg),
		Cursor:      lipgloss.NewStyle().Foreground(bg).Background(c.Fg),
		Selection:   lipgloss.NewStyle().Background(midBg),
		StatusBar:   lipgloss.NewStyle().Background(chromeBg).Foreground(c.Fg).Padding(0, 1),
		TitleBar:    lipgloss.NewStyle().Background(chromeBg).Foreground(c.Fg).Padding(0, 1).Bold(true),
		FileTree:    lipgloss.NewStyle().Foreground(c.Fg).Background(bg),
		FileTreeSel: lipgloss.NewStyle().Foreground(c.Accent).Background(bg).Bold(true),
		FileTreeDir: lipgloss.NewStyle().Foreground(c.Dir).Background(bg).Bold(true),
		Margin:      lipgloss.NewStyle().Background(bg),
		Dimmed:      lipgloss.NewStyle().Foreground(c.Dim).Background(bg),
		ActiveLine:  lipgloss.NewStyle().Background(chromeBg),
	}
}

// NordTheme returns a cool blue-gray palette inspired by Nord.
func NordTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#2E3440",
		ChromeBg:    "#3B4252",
		MidBg:       "#434C5E",
		Fg:          "#ECEFF4",
		FgItalic:    "#D8DEE9",
		Bold:        "#88C0D0",
		Heading1:    "#BF616A",
		Heading2:    "#81A1C1",
		Link:        "#8FBCBB",
		InlineCode:  "#B48EAD",
		ListMarker:  "#8FBCBB",
		FootnoteRef: "#8FBCBB",
		Accent:      "#88C0D0",
		Dim:         "#7B88A1",
		Dir:         "#8FBCBB",
	})
}

// DraculaTheme returns a dark purple-accented palette inspired by Dracula.
func DraculaTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#282A36",
		ChromeBg:    "#343746",
		MidBg:       "#44475A",
		Fg:          "#F8F8F2",
		FgItalic:    "#E8E4DE",
		Bold:        "#FFB86C",
		Heading1:    "#FF79C6",
		Heading2:    "#BD93F9",
		Link:        "#8BE9FD",
		InlineCode:  "#50FA7B",
		ListMarker:  "#FF79C6",
		FootnoteRef: "#8BE9FD",
		Accent:      "#BD93F9",
		Dim:         "#6272A4",
		Dir:         "#8BE9FD",
	})
}

// GruvboxTheme returns a warm retro palette inspired by Gruvbox.
func GruvboxTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#282828",
		ChromeBg:    "#3C3836",
		MidBg:       "#504945",
		Fg:          "#EBDBB2",
		FgItalic:    "#D5C4A1",
		Bold:        "#FE8019",
		Heading1:    "#FB4934",
		Heading2:    "#FABD2F",
		Link:        "#83A598",
		InlineCode:  "#D3869B",
		ListMarker:  "#8EC07C",
		FootnoteRef: "#83A598",
		Accent:      "#FE8019",
		Dim:         "#928374",
		Dir:         "#8EC07C",
	})
}

// ThemeNames returns the list of available theme names in display order.
func ThemeNames() []string {
	return []string{"gruvbox", "nord", "dracula"}
}

// ThemeByName returns a theme by its config name. Falls back to GruvboxTheme.
func ThemeByName(name string) Theme {
	switch strings.ToLower(name) {
	case "nord":
		return NordTheme()
	case "dracula":
		return DraculaTheme()
	default:
		return GruvboxTheme()
	}
}
