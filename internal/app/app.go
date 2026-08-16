package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/editor"
	"github.com/tupini07/wordsmith/internal/filetree"
	"github.com/tupini07/wordsmith/internal/finder"
	"github.com/tupini07/wordsmith/internal/state"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// AppMode represents the current application mode.
type AppMode int

const (
	ModeEditor AppMode = iota
	ModeFileTree
	ModeFuzzyFinder
	ModeThemePicker
	ModeSettings
)

const fileTreeWidth = 30

// Model is the top-level application model.
type Model struct {
	mode        AppMode
	editor      editor.Model
	tree        filetree.Model
	finder      finder.Model
	picker      themePicker
	settings    settingsModal
	cfg         config.Config
	state       state.State
	width       int
	height      int
	initFile    string
	loaded      bool
	activeTheme string // runtime theme name (may differ from config)
	now         func() time.Time
}

// New creates a new app model.
func New(cfg config.Config, st state.State, filePath string) Model {
	theme := editor.ThemeByName(cfg.Theme)
	ed := editor.New(cfg.TabWidth, cfg.ContentWidth, theme)
	ed.SetAutosaveDelay(cfg.AutosaveDelay)
	ed.SetVaultPath(cfg.VaultPath)
	ed.SetShowLineNumbers(cfg.ShowLineNumbers)

	tree := filetree.New(cfg.VaultPath)
	tree.SetThemeColors(theme.Bg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DirColor)

	fnd := finder.New(cfg.VaultPath)
	fnd.SetThemeColors(theme.Bg, theme.ChromeBg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DimColor)

	return Model{
		mode:        ModeEditor,
		editor:      ed,
		tree:        tree,
		finder:      fnd,
		picker:      newThemePicker(),
		settings:    newSettingsModal(),
		cfg:         cfg,
		state:       st,
		initFile:    filePath,
		activeTheme: cfg.Theme,
		now:         time.Now,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return editor.FileWatchCmd()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

		// Load initial file on first resize (we need dimensions first)
		if !m.loaded {
			m.loaded = true
			if m.initFile != "" {
				if err := m.editor.LoadFile(m.initFile); err == nil {
					rel := m.cfg.RelFilePath(m.initFile)
					m.state.SetLastFile(rel)
					// Restore cursor position
					line, col := m.state.GetCursorPos(rel)
					if line > 0 || col > 0 {
						m.editor.SetCursorPos(line, col)
					}
				}
			} else if m.cfg.VaultPath != "" {
				// No file specified — show fuzzy finder
				m.mode = ModeFuzzyFinder
				m.finder.SetRecentFiles(m.state.RecentFiles)
				cmd := m.finder.Show()
				m.editor.SetFocused(false)
				return m, cmd
			}
		}
		return m, nil

	case finder.FileScanCompleteMsg:
		m.finder.HandleScanComplete(msg.Files)
		return m, nil

	case finder.FileCreateMsg:
		return m.createAndOpenFile(msg.Path)

	case finder.FileRenameMsg:
		return m.renameFile(msg.OldPath, msg.NewPath)

	case filetree.CreateInDirMsg:
		// Open finder pre-filled with the directory path
		m.mode = ModeFuzzyFinder
		m.finder.SetRecentFiles(m.state.RecentFiles)
		cmd := m.finder.ShowWithQuery(msg.Dir)
		m.tree.Hide()
		m.editor.SetFocused(false)
		m.updateSizes()
		return m, cmd

	case finder.FileSelectedMsg:
		return m.openFile(msg.Path)

	case editor.FileWatchTickMsg:
		m.editor, _ = m.editor.Update(msg)
		return m, editor.FileWatchCmd()

	case filetree.FileSelectedMsg:
		return m.openFile(msg.Path)

	case tea.KeyMsg:
		if m.mode == ModeSettings {
			result, cmd := m.settings.HandleKey(msg)
			if result == nil {
				return m, cmd
			}
			if result.canceled {
				m.mode = ModeEditor
				m.editor.SetFocused(true)
				return m, nil
			}
			if err := config.Save(result.cfg); err != nil {
				m.settings.SetError(fmt.Errorf("save config: %w", err))
				return m, nil
			}

			m.settings.Hide()
			m.mode = ModeEditor
			m.editor.SetFocused(true)
			restartRequired := m.applySettings(result.cfg)
			if restartRequired {
				m.editor.SetStatus("Settings saved; restart to apply vault path")
			} else {
				m.editor.SetStatus("Settings saved")
			}
			return m, nil
		}

		// Alt-based bindings are reliably distinguishable in Windows Terminal,
		// unlike several Ctrl combinations that map to control characters.
		if key.Matches(msg, key.NewBinding(key.WithKeys("alt+j"))) {
			return m.openTodayJournal()
		}

		// Theme picker gets first-priority key handling
		if m.mode == ModeThemePicker {
			result, preview := m.picker.HandleKey(msg)
			if preview != "" {
				m.applyTheme(preview)
			}
			if result != nil {
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
				m.applyTheme(result.theme)
				if result.confirmed {
					m.editor.SetStatus("Theme: " + result.theme)
				}
			}
			return m, nil
		}

		// Global keybindings that work in any mode
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+q"))):
			// Save before quitting
			if m.editor.IsDirty() {
				m.editor.SaveFile()
			}
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+e"))):
			if m.mode == ModeFileTree {
				m.mode = ModeEditor
				m.tree.Hide()
				m.editor.SetFocused(true)
			} else {
				m.mode = ModeFileTree
				m.tree.Show()
				m.editor.SetFocused(false)
			}
			m.updateSizes()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
			if m.mode == ModeFuzzyFinder {
				m.mode = ModeEditor
				m.finder.Hide()
				m.editor.SetFocused(true)
				return m, nil
			} else {
				m.mode = ModeFuzzyFinder
				m.finder.SetRecentFiles(m.state.RecentFiles)
				cmd := m.finder.Show()
				m.editor.SetFocused(false)
				return m, cmd
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("f2"))):
			cfg, err := config.Load()
			if err != nil {
				m.editor.SetStatus("Config error: " + err.Error())
				return m, nil
			}
			m.mode = ModeSettings
			m.settings.Show(cfg)
			m.tree.Hide()
			m.finder.Hide()
			m.editor.SetFocused(false)
			m.updateSizes()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("f3"))):
			// Rename current file
			if fp := m.editor.FilePath(); fp != "" {
				if m.editor.IsDirty() {
					m.editor.SaveFile()
				}
				m.mode = ModeFuzzyFinder
				m.finder.ShowRename(fp)
				m.editor.SetFocused(false)
				m.updateSizes()
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("f4"))):
			// Toggle theme picker
			if m.mode == ModeThemePicker {
				// Close and revert (same as Esc)
				m.applyTheme(m.picker.OriginalTheme())
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
			} else {
				m.mode = ModeThemePicker
				m.picker.Show(m.activeTheme)
				m.editor.SetFocused(false)
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.mode == ModeThemePicker {
				m.applyTheme(m.picker.OriginalTheme())
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
				return m, nil
			}
			if m.mode != ModeEditor {
				m.mode = ModeEditor
				m.tree.Hide()
				m.finder.Hide()
				m.editor.SetFocused(true)
				m.updateSizes()
				return m, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			// Route to editor for copy (don't quit)
			if m.mode == ModeEditor {
				m.editor, _ = m.editor.Update(msg)
			}
			return m, nil
		}
	case tea.MouseMsg:
		if m.mode == ModeSettings {
			return m, nil
		}
		// When in file tree mode, clicks in the editor area switch to editor
		if m.mode == ModeFileTree && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			treeW := fileTreeWidth
			if msg.X >= treeW {
				m.mode = ModeEditor
				m.tree.Hide()
				m.editor.SetFocused(true)
				m.updateSizes()
				// Forward the click to the editor
				m.editor, _ = m.editor.Update(msg)
				return m, nil
			}
		}
		// When in file tree mode, scroll wheel in editor area scrolls the editor
		if m.mode == ModeFileTree &&
			(msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			treeW := fileTreeWidth
			if msg.X >= treeW {
				m.editor, _ = m.editor.Update(msg)
				return m, nil
			}
		}
	}

	// Route to active component
	var cmd tea.Cmd

	switch m.mode {
	case ModeEditor:
		m.editor, cmd = m.editor.Update(msg)
	case ModeFileTree:
		// Intercept 'l' to reveal current file in tree
		if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "l" {
			if fp := m.editor.FilePath(); fp != "" {
				m.tree.RevealFile(fp)
			}
			return m, nil
		}
		m.tree, cmd = m.tree.Update(msg)
	case ModeFuzzyFinder:
		m.finder, cmd = m.finder.Update(msg)
	case ModeSettings:
		cmd = m.settings.Update(msg)
	}

	return m, cmd
}

func (m *Model) openFile(path string) (Model, tea.Cmd) {
	prevPath := m.editor.FilePath()

	// Save cursor position for the file we're leaving
	if prevPath != "" {
		line, col := m.editor.CursorPos()
		rel := m.cfg.RelFilePath(prevPath)
		m.state.SetCursorPos(rel, line, col)
	}

	if err := m.editor.LoadFile(path); err != nil {
		m.editor.SetStatus("Open error: " + err.Error())
		return *m, nil
	}

	rel := m.cfg.RelFilePath(path)
	m.state.SetLastFile(rel)

	// Restore cursor position
	line, col := m.state.GetCursorPos(rel)
	if line > 0 || col > 0 {
		m.editor.SetCursorPos(line, col)
	}

	// Switch back to editor mode
	m.mode = ModeEditor
	m.tree.Hide()
	m.finder.Hide()
	m.editor.SetFocused(true)
	m.updateSizes()

	return *m, nil
}

func (m *Model) createAndOpenFile(absPath string) (tea.Model, tea.Cmd) {
	// Validate the path stays inside the vault
	absPath = filepath.Clean(absPath)
	if !m.cfg.IsPathInVault(absPath) {
		m.editor.SetStatus("Cannot create file outside vault")
		return *m, nil
	}

	// Create parent directories
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.editor.SetStatus("Error: " + err.Error())
		return *m, nil
	}

	// Create file exclusively — never truncate an existing file
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// File was created between the finder check and now — just open it
			return m.openFile(absPath)
		}
		m.editor.SetStatus("Error: " + err.Error())
		return *m, nil
	}
	if err := f.Close(); err != nil {
		os.Remove(absPath)
		m.editor.SetStatus("Error: " + err.Error())
		return *m, nil
	}

	// Add to finder's cached file list
	rel := m.cfg.RelFilePath(absPath)
	m.finder.AddFile(rel)

	return m.openFile(absPath)
}

func (m *Model) renameFile(oldPath, newPath string) (tea.Model, tea.Cmd) {
	// Validate new path stays inside the vault
	newPath = filepath.Clean(newPath)
	if !m.cfg.IsPathInVault(newPath) {
		m.editor.SetStatus("Cannot rename file outside vault")
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return *m, nil
	}

	// Don't rename to same path
	if oldPath == newPath {
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return *m, nil
	}

	// Create parent directories for the new path
	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.editor.SetStatus("Error: " + err.Error())
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return *m, nil
	}

	// Check the destination doesn't already exist
	if _, err := os.Stat(newPath); err == nil {
		m.editor.SetStatus("File already exists")
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return *m, nil
	}

	// Perform the rename
	if err := os.Rename(oldPath, newPath); err != nil {
		m.editor.SetStatus("Rename error: " + err.Error())
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return *m, nil
	}

	// Update the editor to point at the new path
	m.editor.SetFilePath(newPath)
	m.editor.SetStatus("Renamed to " + m.cfg.RelFilePath(newPath))

	// Update recent files state
	rel := m.cfg.RelFilePath(newPath)
	m.state.SetLastFile(rel)

	m.mode = ModeEditor
	m.editor.SetFocused(true)
	m.updateSizes()
	return *m, nil
}

func (m *Model) openTodayJournal() (tea.Model, tea.Cmd) {
	m.closeOverlays()

	if m.editor.IsDirty() {
		if err := m.editor.SaveFile(); err != nil {
			m.editor.SetStatus("Cannot open journal: " + err.Error())
			return *m, nil
		}
	}

	now := time.Now
	if m.now != nil {
		now = m.now
	}
	path, err := m.cfg.JournalFilePath(now())
	if err != nil {
		m.editor.SetStatus("Journal: " + err.Error())
		return *m, nil
	}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			m.editor.SetStatus("Journal path is a directory")
			return *m, nil
		}
		return m.openFile(path)
	}
	if !os.IsNotExist(err) {
		m.editor.SetStatus("Journal error: " + err.Error())
		return *m, nil
	}

	return m.createAndOpenFile(path)
}

func (m *Model) closeOverlays() {
	if m.picker.IsVisible() {
		m.applyTheme(m.picker.OriginalTheme())
		m.picker.Hide()
	}
	m.mode = ModeEditor
	m.tree.Hide()
	m.finder.Hide()
	m.settings.Hide()
	m.editor.SetFocused(true)
	m.updateSizes()
}

func (m *Model) updateSizes() {
	// Reserve 1 line for title, 1 for status, 2 for padding
	editorHeight := m.height - 4
	if editorHeight < 1 {
		editorHeight = 1
	}

	treeW := 0
	if m.tree.IsVisible() {
		treeW = fileTreeWidth
		m.tree.SetSize(treeW, editorHeight)
	}

	editorWidth := m.width - treeW
	if editorWidth < 10 {
		editorWidth = 10
	}

	m.editor.SetSize(editorWidth, editorHeight)
	m.editor.SetFullWidth(m.width)
	m.editor.SetEditorOffset(treeW, 2) // Y=2 for title bar + padding row
	m.finder.SetSize(m.width, m.height)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Title bar
	title := m.editor.TitleView()
	pad := m.editor.PaddingLine()

	// Editor content
	editorView := m.editor.View()

	// File tree (if visible)
	if m.tree.IsVisible() {
		treeView := m.tree.View()
		// Merge tree and editor side by side (editor height excludes padding)
		editorView = mergeSideBySide(treeView, editorView, m.height-4, m.width, m.editor.ThemeMarginStyle())
	}

	// Status bar
	status := m.editor.StatusView()

	composed := title + "\n" + pad + "\n" + editorView + "\n" + pad + "\n" + status

	// Emoji picker overlay
	if m.editor.EmojiActive() {
		emojiView := m.renderEmojiPicker()
		if emojiView != "" {
			composed = overlayAt(composed, emojiView, m.emojiPickerX(), m.emojiPickerY(), m.width, m.height)
		}
	}

	// Fuzzy finder overlay
	if m.finder.IsVisible() {
		finderView := m.finder.View()
		if finderView != "" {
			return overlayCenter(composed, finderView, m.width, m.height)
		}
	}

	// Theme picker overlay (renders its own full-screen view)
	if m.picker.IsVisible() {
		theme := editor.ThemeByName(m.activeTheme)
		return m.picker.View(theme, m.width, m.height)
	}

	if m.settings.IsVisible() {
		theme := editor.ThemeByName(m.activeTheme)
		return m.settings.View(theme, m.width, m.height)
	}

	return composed
}

// SaveState saves the current session state.
func (m Model) SaveState() error {
	// Save current cursor position
	if fp := m.editor.FilePath(); fp != "" {
		line, col := m.editor.CursorPos()
		rel := m.cfg.RelFilePath(fp)
		m.state.SetCursorPos(rel, line, col)
	}
	return m.state.Save()
}

// applyTheme applies a theme by name to all components.
func (m *Model) applyTheme(name string) {
	theme := editor.ThemeByName(name)
	m.activeTheme = name
	m.editor.SetTheme(theme)
	m.tree.SetThemeColors(theme.Bg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DirColor)
	m.finder.SetThemeColors(theme.Bg, theme.ChromeBg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DimColor)
}

func (m *Model) applySettings(newCfg config.Config) bool {
	vaultChanged := newCfg.VaultPath != m.cfg.VaultPath
	if vaultChanged {
		// Keep path-sensitive settings aligned with the active vault until the
		// process restarts and rebuilds the finder and file tree.
		newCfg.VaultPath = m.cfg.VaultPath
		newCfg.JournalFolder = m.cfg.JournalFolder
		newCfg.JournalDateFormat = m.cfg.JournalDateFormat
	}

	if newCfg.Theme != m.activeTheme {
		m.applyTheme(newCfg.Theme)
	}
	m.editor.SetTabWidth(newCfg.TabWidth)
	m.editor.SetContentWidth(newCfg.ContentWidth)
	m.editor.SetAutosaveDelay(newCfg.AutosaveDelay)
	m.editor.SetShowLineNumbers(newCfg.ShowLineNumbers)
	m.cfg = newCfg
	m.updateSizes()
	return vaultChanged
}

// mergeSideBySide renders two views side by side, filling to fullWidth.
func mergeSideBySide(left, right string, height, fullWidth int, fillStyle lipgloss.Style) string {
	leftLines := splitLines(left, height)
	rightLines := splitLines(right, height)

	var result string
	for i := 0; i < height; i++ {
		if i > 0 {
			result += "\n"
		}
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		line := l + r
		// Fill any remaining space to the right edge
		lineWidth := lipgloss.Width(line)
		if lineWidth < fullWidth {
			line += fillStyle.Render(strings.Repeat(" ", fullWidth-lineWidth))
		}
		result += line
	}
	return result
}

// overlayCenter places an overlay on top of a background.
func overlayCenter(bg, overlay string, width, height int) string {
	_ = width
	_ = height
	// Simple overlay: just return the background with the overlay rendered on top
	// The finder's View already handles positioning
	bgLines := splitLines(bg, height)
	overlayLines := splitLines(overlay, height)

	var result string
	for i := 0; i < height; i++ {
		if i > 0 {
			result += "\n"
		}
		if i < len(overlayLines) && overlayLines[i] != "" {
			result += overlayLines[i]
		} else if i < len(bgLines) {
			result += bgLines[i]
		}
	}
	return result
}

func splitLines(s string, minLen int) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	for len(lines) < minLen {
		lines = append(lines, "")
	}
	return lines
}

func (m Model) emojiPickerX() int {
	row, _ := m.editor.CursorScreenPos()
	if row < 0 {
		return 0
	}

	// Estimate picker height: header + matches + border (top+bottom)
	matches := m.editor.EmojiMatches()
	itemCount := len(matches)
	if itemCount == 0 {
		itemCount = 1 // "no matches" line
	}
	pickerHeight := itemCount + 1 + 2 // items + header + border

	belowY := row + 3 // below cursor (title + padding = 2 extra lines)

	if belowY+pickerHeight > m.height {
		// Show above cursor instead
		aboveY := row + 2 - pickerHeight
		if aboveY < 0 {
			aboveY = 0
		}
		return aboveY
	}
	return belowY
}

func (m Model) emojiPickerY() int {
	// Left-align with some margin
	leftMargin := 0
	if m.editor.ContentWidth() > 0 && m.width > m.editor.ContentWidth() {
		leftMargin = (m.width - m.editor.ContentWidth()) / 2
	}
	return leftMargin + 2
}

func (m Model) renderEmojiPicker() string {
	theme := editor.ThemeByName(m.activeTheme)
	matches := m.editor.EmojiMatches()
	query := m.editor.EmojiQuery()
	cursor := m.editor.EmojiCursor()

	innerWidth := 24

	borderStyle := lipgloss.NewStyle().
		Foreground(theme.Fg).
		Background(theme.ChromeBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.AccentColor).
		BorderBackground(theme.ChromeBg)

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.DimColor).
		Background(theme.ChromeBg)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.Fg).
		Background(theme.ChromeBg)

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Bg).
		Background(theme.AccentColor)

	padLine := func(line string, style lipgloss.Style) string {
		w := runewidth.StringWidth(line)
		if w > innerWidth {
			line = runewidth.Truncate(line, innerWidth, "")
			w = innerWidth
		}
		if w < innerWidth {
			line += strings.Repeat(" ", innerWidth-w)
		}
		return style.Render(line)
	}

	var lines []string
	header := fmt.Sprintf(" ::%s", query)
	lines = append(lines, padLine(header, headerStyle))

	if len(matches) == 0 {
		lines = append(lines, padLine(" no matches", normalStyle))
	} else {
		for i, entry := range matches {
			line := fmt.Sprintf(" %s  :%s:", entry.Emoji, entry.Name)
			if i == cursor {
				lines = append(lines, padLine(line, selectedStyle))
			} else {
				lines = append(lines, padLine(line, normalStyle))
			}
		}
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Width(innerWidth).Render(content)
}

// overlayAt places an overlay at position (row, col) on the background.
func overlayAt(bg, overlay string, row, col, width, height int) string {
	bgLines := splitLines(bg, height)
	overlayLines := strings.Split(overlay, "\n")

	for i, ol := range overlayLines {
		targetRow := row + i
		if targetRow < 0 || targetRow >= len(bgLines) {
			continue
		}
		bgLine := bgLines[targetRow]

		// Use charmbracelet/x/ansi for proper ANSI-aware string slicing
		prefix := ansi.Truncate(bgLine, col, "")
		overlayWidth := lipgloss.Width(ol)
		suffixStart := col + overlayWidth
		suffix := ""
		bgWidth := lipgloss.Width(bgLine)
		if suffixStart < bgWidth {
			suffix = ansi.TruncateLeft(bgLine, suffixStart, "")
		}

		bgLines[targetRow] = prefix + ol + suffix
	}

	return strings.Join(bgLines[:height], "\n")
}
