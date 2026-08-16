package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/editor"
)

type settingKind int

const (
	settingText settingKind = iota
	settingDuration
	settingPositiveInt
	settingNonNegativeInt
	settingBool
	settingTheme
)

type settingField struct {
	key   string
	label string
	help  string
	kind  settingKind
}

var settingFields = []settingField{
	{key: "vault_path", label: "Vault path", help: "Markdown vault root (restart required to change)", kind: settingText},
	{key: "journal_folder", label: "Journal folder", help: "Empty disables journaling; relative paths use the vault root", kind: settingText},
	{key: "journal_date_format", label: "Journal date format", help: "Supported tokens: YYYY, YY, MM, DD", kind: settingText},
	{key: "autosave_delay", label: "Autosave delay", help: "Go duration such as 2s or 500ms", kind: settingDuration},
	{key: "tab_width", label: "Tab width", help: "Number of spaces inserted by Tab", kind: settingPositiveInt},
	{key: "content_width", label: "Content width", help: "Centered writing width; 0 uses the full terminal", kind: settingNonNegativeInt},
	{key: "show_line_numbers", label: "Show line numbers", help: "Display line numbers in the editor", kind: settingBool},
	{key: "theme", label: "Theme", help: "Use Left/Right to choose", kind: settingTheme},
}

type settingsResult struct {
	cfg      config.Config
	canceled bool
}

type settingsModal struct {
	visible bool
	draft   config.Config
	cursor  int
	editing bool
	input   textinput.Model
	err     string
}

func newSettingsModal() settingsModal {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 1024
	return settingsModal{input: input}
}

func (sm *settingsModal) Show(cfg config.Config) {
	sm.visible = true
	sm.draft = cfg
	sm.cursor = 0
	sm.editing = false
	sm.err = ""
	sm.input.Blur()
}

func (sm *settingsModal) Hide() {
	sm.visible = false
	sm.editing = false
	sm.input.Blur()
}

func (sm settingsModal) IsVisible() bool {
	return sm.visible
}

func (sm *settingsModal) SetError(err error) {
	if err == nil {
		sm.err = ""
		return
	}
	sm.err = err.Error()
}

func (sm *settingsModal) HandleKey(msg tea.KeyMsg) (*settingsResult, tea.Cmd) {
	if sm.editing {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			sm.editing = false
			sm.err = ""
			sm.input.Blur()
			return nil, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if err := sm.commitEdit(); err != nil {
				sm.err = err.Error()
				return nil, nil
			}
			sm.editing = false
			sm.err = ""
			sm.input.Blur()
			return nil, nil
		default:
			var cmd tea.Cmd
			sm.input, cmd = sm.input.Update(msg)
			return nil, cmd
		}
	}

	field := settingFields[sm.cursor]
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "f2"))):
		sm.Hide()
		return &settingsResult{canceled: true}, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
		if err := validateSettings(sm.draft); err != nil {
			sm.err = err.Error()
			return nil, nil
		}
		return &settingsResult{cfg: sm.draft}, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "shift+tab"))):
		if sm.cursor > 0 {
			sm.cursor--
		}
		sm.err = ""
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "tab"))):
		if sm.cursor < len(settingFields)-1 {
			sm.cursor++
		}
		sm.err = ""
	case key.Matches(msg, key.NewBinding(key.WithKeys("left"))):
		sm.adjustChoice(field, -1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("right"))):
		sm.adjustChoice(field, 1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		switch field.kind {
		case settingBool:
			sm.draft.ShowLineNumbers = !sm.draft.ShowLineNumbers
		case settingTheme:
			sm.adjustChoice(field, 1)
		default:
			return nil, sm.beginEdit(field)
		}
	}
	return nil, nil
}

func (sm *settingsModal) Update(msg tea.Msg) tea.Cmd {
	if !sm.visible || !sm.editing {
		return nil
	}
	var cmd tea.Cmd
	sm.input, cmd = sm.input.Update(msg)
	return cmd
}

func (sm *settingsModal) beginEdit(field settingField) tea.Cmd {
	sm.editing = true
	sm.err = ""
	sm.input.SetValue(sm.value(field))
	sm.input.CursorEnd()
	return sm.input.Focus()
}

func (sm *settingsModal) commitEdit() error {
	field := settingFields[sm.cursor]
	value := sm.input.Value()

	switch field.key {
	case "vault_path":
		sm.draft.VaultPath = value
	case "journal_folder":
		sm.draft.JournalFolder = value
	case "journal_date_format":
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("journal date format cannot be empty")
		}
		sm.draft.JournalDateFormat = value
	case "autosave_delay":
		duration, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || duration <= 0 {
			return fmt.Errorf("autosave delay must be a positive duration")
		}
		sm.draft.AutosaveDelay = duration
	case "tab_width":
		width, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || width <= 0 {
			return fmt.Errorf("tab width must be greater than zero")
		}
		sm.draft.TabWidth = width
	case "content_width":
		width, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || width < 0 {
			return fmt.Errorf("content width cannot be negative")
		}
		sm.draft.ContentWidth = width
	}
	return nil
}

func (sm *settingsModal) adjustChoice(field settingField, delta int) {
	switch field.kind {
	case settingBool:
		sm.draft.ShowLineNumbers = !sm.draft.ShowLineNumbers
	case settingTheme:
		names := editor.ThemeNames()
		index := 0
		for i, name := range names {
			if name == sm.draft.Theme {
				index = i
				break
			}
		}
		index = (index + delta + len(names)) % len(names)
		sm.draft.Theme = names[index]
	}
	sm.err = ""
}

func (sm settingsModal) value(field settingField) string {
	switch field.key {
	case "vault_path":
		return sm.draft.VaultPath
	case "journal_folder":
		return sm.draft.JournalFolder
	case "journal_date_format":
		return sm.draft.JournalDateFormat
	case "autosave_delay":
		return sm.draft.AutosaveDelay.String()
	case "tab_width":
		return strconv.Itoa(sm.draft.TabWidth)
	case "content_width":
		return strconv.Itoa(sm.draft.ContentWidth)
	case "show_line_numbers":
		if sm.draft.ShowLineNumbers {
			return "on"
		}
		return "off"
	case "theme":
		return sm.draft.Theme
	default:
		return ""
	}
}

func validateSettings(cfg config.Config) error {
	if cfg.AutosaveDelay <= 0 {
		return fmt.Errorf("autosave delay must be positive")
	}
	if cfg.TabWidth <= 0 {
		return fmt.Errorf("tab width must be greater than zero")
	}
	if cfg.ContentWidth < 0 {
		return fmt.Errorf("content width cannot be negative")
	}
	if strings.TrimSpace(cfg.JournalDateFormat) == "" {
		return fmt.Errorf("journal date format cannot be empty")
	}
	if strings.TrimSpace(cfg.JournalFolder) != "" {
		if _, err := cfg.JournalFilePath(time.Now()); err != nil {
			return err
		}
	}
	for _, name := range editor.ThemeNames() {
		if name == cfg.Theme {
			return nil
		}
	}
	return fmt.Errorf("unknown theme %q", cfg.Theme)
}

func (sm *settingsModal) View(theme editor.Theme, totalWidth, totalHeight int) string {
	if !sm.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(theme.Bg)
	titleStyle := lipgloss.NewStyle().Foreground(theme.Fg).Background(theme.Bg).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.DimColor).Background(theme.Bg)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Fg).Background(theme.Bg)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.AccentColor).Background(theme.Bg).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(theme.DimColor).Background(theme.Bg)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b")).Background(theme.Bg)

	popupWidth := 76
	if popupWidth > totalWidth-4 {
		popupWidth = totalWidth - 4
	}
	if popupWidth < 40 {
		popupWidth = 40
	}
	valueWidth := popupWidth - 28
	if valueWidth < 12 {
		valueWidth = 12
	}

	sm.input.Width = valueWidth
	sm.input.TextStyle = valueStyle
	sm.input.PromptStyle = valueStyle
	sm.input.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Bg).Background(theme.AccentColor)

	lines := []string{titleStyle.Render("Settings"), ""}
	for i, field := range settingFields {
		prefix := "  "
		label := labelStyle.Render(fmt.Sprintf("%-20s", field.label))
		var renderedValue string
		if sm.editing && i == sm.cursor {
			renderedValue = sm.input.View()
		} else {
			value := sm.value(field)
			if value == "" {
				value = "—"
			}
			value = runewidth.Truncate(value, valueWidth, "…")
			renderedValue = valueStyle.Render(value)
		}
		if i == sm.cursor {
			prefix = selectedStyle.Render("▸ ")
			label = selectedStyle.Render(fmt.Sprintf("%-20s", field.label))
		}
		lines = append(lines, prefix+label+"  "+renderedValue)
	}

	lines = append(lines, "", helpStyle.Render(settingFields[sm.cursor].help))
	if sm.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+sm.err))
	} else {
		lines = append(lines, "")
	}
	hint := "↑↓ select · enter edit/toggle · ←→ choose · ctrl+s save · esc cancel"
	if sm.editing {
		hint = "enter apply value · esc cancel edit"
	}
	lines = append(lines, helpStyle.Render(hint))

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.AccentColor).
		BorderBackground(theme.Bg).
		Background(theme.Bg).
		Padding(0, 1).
		Width(popupWidth)
	popup := border.Render(strings.Join(lines, "\n"))
	popupLines := strings.Split(popup, "\n")
	topPad := (totalHeight - len(popupLines)) / 3
	if topPad < 1 {
		topPad = 1
	}

	result := make([]string, 0, totalHeight)
	for y := 0; y < totalHeight; y++ {
		popupIndex := y - topPad
		if popupIndex >= 0 && popupIndex < len(popupLines) {
			line := popupLines[popupIndex]
			lineWidth := lipgloss.Width(line)
			leftPad := (totalWidth - lineWidth) / 2
			if leftPad < 0 {
				leftPad = 0
			}
			rightPad := totalWidth - leftPad - lineWidth
			if rightPad < 0 {
				rightPad = 0
			}
			result = append(result,
				bgStyle.Render(strings.Repeat(" ", leftPad))+
					line+
					bgStyle.Render(strings.Repeat(" ", rightPad)),
			)
		} else {
			result = append(result, bgStyle.Render(strings.Repeat(" ", totalWidth)))
		}
	}
	return strings.Join(result, "\n")
}
