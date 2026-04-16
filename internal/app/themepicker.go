package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tupini07/wordsmith/internal/editor"
)

type pickerEntry struct {
	name     string
	isHeader bool
}

// themePicker is a small popup that lists available themes.
type themePicker struct {
	entries []pickerEntry
	cursor  int
	visible bool

	// The theme name that was active before the picker opened,
	// used for reverting on cancel.
	originalTheme string
}

func newThemePicker() themePicker {
	var entries []pickerEntry
	entries = append(entries, pickerEntry{name: "Dark", isHeader: true})
	for _, n := range editor.DarkThemeNames() {
		entries = append(entries, pickerEntry{name: n})
	}
	entries = append(entries, pickerEntry{name: "", isHeader: true}) // blank separator
	entries = append(entries, pickerEntry{name: "Light", isHeader: true})
	for _, n := range editor.LightThemeNames() {
		entries = append(entries, pickerEntry{name: n})
	}
	return themePicker{entries: entries}
}

// Show opens the picker and snapshots the current theme for revert.
func (tp *themePicker) Show(currentTheme string) {
	tp.visible = true
	tp.originalTheme = currentTheme
	// Place cursor on the currently active theme
	tp.cursor = tp.firstSelectable()
	lower := strings.ToLower(currentTheme)
	for i, e := range tp.entries {
		if !e.isHeader && e.name == lower {
			tp.cursor = i
			break
		}
	}
}

func (tp *themePicker) Hide() {
	tp.visible = false
}

func (tp *themePicker) IsVisible() bool {
	return tp.visible
}

func (tp *themePicker) firstSelectable() int {
	for i, e := range tp.entries {
		if !e.isHeader {
			return i
		}
	}
	return 0
}

// SelectedTheme returns the theme name the cursor is on.
func (tp *themePicker) SelectedTheme() string {
	if tp.cursor >= 0 && tp.cursor < len(tp.entries) && !tp.entries[tp.cursor].isHeader {
		return tp.entries[tp.cursor].name
	}
	return "gruvbox"
}

// OriginalTheme returns the theme that was active before the picker opened.
func (tp *themePicker) OriginalTheme() string {
	return tp.originalTheme
}

type themePickerResult struct {
	confirmed bool
	theme     string
}

// HandleKey processes a key event and returns a result if the picker should close.
func (tp *themePicker) HandleKey(msg tea.KeyMsg) (result *themePickerResult, preview string) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		for i := tp.cursor - 1; i >= 0; i-- {
			if !tp.entries[i].isHeader {
				tp.cursor = i
				return nil, tp.SelectedTheme()
			}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		for i := tp.cursor + 1; i < len(tp.entries); i++ {
			if !tp.entries[i].isHeader {
				tp.cursor = i
				return nil, tp.SelectedTheme()
			}
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return &themePickerResult{confirmed: true, theme: tp.SelectedTheme()}, ""
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "f4"))):
		return &themePickerResult{confirmed: false, theme: tp.originalTheme}, ""
	}
	return nil, ""
}

// View renders the picker as a full-screen overlay with the popup centered.
func (tp *themePicker) View(theme editor.Theme, totalWidth, totalHeight int) string {
	if !tp.visible {
		return ""
	}

	bg := theme.Bg
	fg := theme.Fg
	accent := theme.AccentColor
	dim := theme.DimColor

	bgStyle := lipgloss.NewStyle().Background(bg)

	titleStyle := lipgloss.NewStyle().
		Foreground(fg).Background(bg).
		Bold(true).Padding(0, 1)
	headerStyle := lipgloss.NewStyle().
		Foreground(dim).Background(bg).
		Bold(true).Padding(0, 1)
	normalStyle := lipgloss.NewStyle().
		Foreground(fg).Background(bg).
		Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().
		Foreground(accent).Background(bg).
		Bold(true).Padding(0, 1)
	hintStyle := lipgloss.NewStyle().
		Foreground(dim).Background(bg).
		Padding(0, 1)

	var contentLines []string
	contentLines = append(contentLines, titleStyle.Render("Theme"))
	contentLines = append(contentLines, "")
	for i, e := range tp.entries {
		if e.isHeader {
			if e.name == "" {
				contentLines = append(contentLines, "")
			} else {
				contentLines = append(contentLines, headerStyle.Render("  "+e.name))
			}
			continue
		}
		display := "    " + e.name
		if i == tp.cursor {
			display = "  ▸ " + e.name
			contentLines = append(contentLines, selectedStyle.Render(display))
		} else {
			contentLines = append(contentLines, normalStyle.Render(display))
		}
	}
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, hintStyle.Render("↑↓ preview · enter confirm · esc cancel"))

	content := strings.Join(contentLines, "\n")

	popupWidth := 50
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		BorderBackground(bg).
		Background(bg).
		Width(popupWidth)

	popup := border.Render(content)
	popupLines := strings.Split(popup, "\n")
	popupHeight := len(popupLines)

	topPad := (totalHeight - popupHeight) / 3
	if topPad < 1 {
		topPad = 1
	}

	// Build full-screen view with themed background
	var lines []string
	for y := 0; y < totalHeight; y++ {
		popupIdx := y - topPad
		if popupIdx >= 0 && popupIdx < popupHeight {
			line := popupLines[popupIdx]
			lineW := lipgloss.Width(line)

			leftPad := (totalWidth - lineW) / 2
			if leftPad < 0 {
				leftPad = 0
			}
			rightPad := totalWidth - leftPad - lineW
			if rightPad < 0 {
				rightPad = 0
			}

			left := bgStyle.Render(strings.Repeat(" ", leftPad))
			right := bgStyle.Render(strings.Repeat(" ", rightPad))
			lines = append(lines, left+line+right)
		} else {
			lines = append(lines, bgStyle.Render(strings.Repeat(" ", totalWidth)))
		}
	}

	return strings.Join(lines, "\n")
}
