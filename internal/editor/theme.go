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
	CodeBlock   lipgloss.Style // code block content (fenced ```...```)
	CodeFence   lipgloss.Style // the ``` fence delimiters
	TableBorder lipgloss.Style // table pipe | and separator row
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
		CodeBlock:   lipgloss.NewStyle().Foreground(c.InlineCode).Background(midBg),
		CodeFence:   lipgloss.NewStyle().Foreground(c.Dim).Background(midBg),
		TableBorder: lipgloss.NewStyle().Foreground(c.Dim).Background(bg),
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

// CatppuccinMochaTheme returns a pastel dark palette inspired by Catppuccin Mocha.
func CatppuccinMochaTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#1E1E2E",
		ChromeBg:    "#313244",
		MidBg:       "#45475A",
		Fg:          "#CDD6F4",
		FgItalic:    "#BAC2DE",
		Bold:        "#FAB387",
		Heading1:    "#F38BA8",
		Heading2:    "#CBA6F7",
		Link:        "#89B4FA",
		InlineCode:  "#A6E3A1",
		ListMarker:  "#F5C2E7",
		FootnoteRef: "#94E2D5",
		Accent:      "#CBA6F7",
		Dim:         "#6C7086",
		Dir:         "#94E2D5",
	})
}

// CatppuccinLatteTheme returns a pastel light palette inspired by Catppuccin Latte.
func CatppuccinLatteTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#EFF1F5",
		ChromeBg:    "#CCD0DA",
		MidBg:       "#BCC0CC",
		Fg:          "#4C4F69",
		FgItalic:    "#5C5F77",
		Bold:        "#FE640B",
		Heading1:    "#D20F39",
		Heading2:    "#8839EF",
		Link:        "#1E66F5",
		InlineCode:  "#40A02B",
		ListMarker:  "#EA76CB",
		FootnoteRef: "#179299",
		Accent:      "#8839EF",
		Dim:         "#9CA0B0",
		Dir:         "#179299",
	})
}

// PalenightTheme returns a muted purple palette inspired by Material Palenight.
func PalenightTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#292D3E",
		ChromeBg:    "#32374D",
		MidBg:       "#434758",
		Fg:          "#A6ACCD",
		FgItalic:    "#959DCB",
		Bold:        "#F78C6C",
		Heading1:    "#F07178",
		Heading2:    "#C792EA",
		Link:        "#82AAFF",
		InlineCode:  "#C3E88D",
		ListMarker:  "#89DDFF",
		FootnoteRef: "#89DDFF",
		Accent:      "#C792EA",
		Dim:         "#676E95",
		Dir:         "#82AAFF",
	})
}

// SolarizedDarkTheme returns a warm-cool palette inspired by Solarized Dark.
func SolarizedDarkTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#002B36",
		ChromeBg:    "#073642",
		MidBg:       "#0A4050",
		Fg:          "#839496",
		FgItalic:    "#93A1A1",
		Bold:        "#B58900",
		Heading1:    "#CB4B16",
		Heading2:    "#268BD2",
		Link:        "#2AA198",
		InlineCode:  "#859900",
		ListMarker:  "#D33682",
		FootnoteRef: "#6C71C4",
		Accent:      "#268BD2",
		Dim:         "#586E75",
		Dir:         "#2AA198",
	})
}

// SolarizedLightTheme returns a warm-cool light palette inspired by Solarized Light.
func SolarizedLightTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#FDF6E3",
		ChromeBg:    "#EEE8D5",
		MidBg:       "#E6E0CB",
		Fg:          "#657B83",
		FgItalic:    "#586E75",
		Bold:        "#B58900",
		Heading1:    "#CB4B16",
		Heading2:    "#268BD2",
		Link:        "#2AA198",
		InlineCode:  "#859900",
		ListMarker:  "#D33682",
		FootnoteRef: "#6C71C4",
		Accent:      "#268BD2",
		Dim:         "#93A1A1",
		Dir:         "#2AA198",
	})
}

// TokyoNightTheme returns a dark blue palette inspired by Tokyo Night.
func TokyoNightTheme() Theme {
	return buildTheme(ThemeColors{
		Bg:          "#1A1B26",
		ChromeBg:    "#24283B",
		MidBg:       "#33467C",
		Fg:          "#A9B1D6",
		FgItalic:    "#9AA5CE",
		Bold:        "#FF9E64",
		Heading1:    "#F7768E",
		Heading2:    "#BB9AF7",
		Link:        "#7AA2F7",
		InlineCode:  "#9ECE6A",
		ListMarker:  "#7DCFFF",
		FootnoteRef: "#73DACA",
		Accent:      "#BB9AF7",
		Dim:         "#565F89",
		Dir:         "#73DACA",
	})
}

// ThemeNames returns the list of available theme names in display order.
func ThemeNames() []string {
	return append(DarkThemeNames(), LightThemeNames()...)
}

// DarkThemeNames returns dark theme names in display order.
func DarkThemeNames() []string {
	return []string{"gruvbox", "nord", "dracula", "catppuccin-mocha", "palenight", "solarized-dark", "tokyo-night"}
}

// LightThemeNames returns light theme names in display order.
func LightThemeNames() []string {
	return []string{"catppuccin-latte", "solarized-light"}
}

// ThemeByName returns a theme by its config name. Falls back to GruvboxTheme.
func ThemeByName(name string) Theme {
	switch strings.ToLower(name) {
	case "nord":
		return NordTheme()
	case "dracula":
		return DraculaTheme()
	case "catppuccin-mocha":
		return CatppuccinMochaTheme()
	case "catppuccin-latte":
		return CatppuccinLatteTheme()
	case "palenight":
		return PalenightTheme()
	case "solarized-dark":
		return SolarizedDarkTheme()
	case "solarized-light":
		return SolarizedLightTheme()
	case "tokyo-night":
		return TokyoNightTheme()
	default:
		return GruvboxTheme()
	}
}
