package finder

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// FileSelectedMsg is sent when a file is selected in the finder.
type FileSelectedMsg struct {
	Path string
}

// FileCreateMsg is sent when the user wants to create a new file.
type FileCreateMsg struct {
	Path string
}

// FileRenameMsg is sent when the user confirms a file rename.
type FileRenameMsg struct {
	OldPath string
	NewPath string
}

// FileScanCompleteMsg is sent when background file scanning completes.
type FileScanCompleteMsg struct {
	Files []string
}

// Model is the fuzzy file finder.
type Model struct {
	vaultPath   string
	files       []string // relative paths
	recentFiles []string // recently opened files, most recent first
	matches     []fuzzy.Match
	query       string
	queryCursor int // cursor position within the query string (in runes)
	cursor      int
	width       int
	height      int
	visible     bool
	scanning    bool
	showCreate  bool   // whether to show a "Create" entry
	createPath  string // normalized relative path for file creation
	renameMode  bool   // when true, finder acts as a rename prompt
	renameFrom  string // absolute path of the file being renamed
	theme       finderTheme
}

type finderTheme struct {
	border      lipgloss.Style
	normal      lipgloss.Style
	selected    lipgloss.Style
	matched     lipgloss.Style
	input       lipgloss.Style
	inputCursor lipgloss.Style
	dimmed      lipgloss.Style
	create      lipgloss.Style // style for the "Create" entry
	bg          lipgloss.Color
}

func defaultFinderTheme() finderTheme {
	cream := lipgloss.Color("#F5E6D3")
	orange := lipgloss.Color("#E8A87C")
	darkBg := lipgloss.Color("#1E1C1F")
	chromeBg := lipgloss.Color("#2D2A2E")
	midBg := lipgloss.Color("#3E3B40")
	dimText := lipgloss.Color("#8A8488")

	return makeFinderTheme(darkBg, chromeBg, midBg, cream, orange, dimText)
}

func makeFinderTheme(bg, chromeBg, midBg, fg, accent, dim lipgloss.Color) finderTheme {
	return finderTheme{
		bg: bg,
		border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			BorderBackground(bg).
			Background(chromeBg),
		normal:   lipgloss.NewStyle().Foreground(fg).Background(chromeBg),
		selected: lipgloss.NewStyle().Foreground(accent).Background(chromeBg).Bold(true),
		matched:  lipgloss.NewStyle().Foreground(accent).Background(chromeBg).Underline(true),
		input:       lipgloss.NewStyle().Foreground(fg).Background(midBg).Padding(0, 1),
		inputCursor: lipgloss.NewStyle().Foreground(midBg).Background(fg),
		dimmed:   lipgloss.NewStyle().Foreground(dim).Background(chromeBg),
		create:   lipgloss.NewStyle().Foreground(accent).Background(chromeBg).Italic(true),
	}
}

// New creates a new finder model.
func New(vaultPath string) Model {
	return Model{
		vaultPath: vaultPath,
		theme:     defaultFinderTheme(),
	}
}

// SetThemeColors updates the finder's colors from the main theme.
func (m *Model) SetThemeColors(bg, chromeBg, midBg, fg, accent, dim lipgloss.Color) {
	m.theme = makeFinderTheme(bg, chromeBg, midBg, fg, accent, dim)
}

// SetRecentFiles updates the list of recently opened files (most recent first).
func (m *Model) SetRecentFiles(recent []string) {
	m.recentFiles = recent
}

// SetSize sets the finder dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Show makes the finder visible. Loads cached file list immediately,
// then returns a Cmd to refresh the list in the background.
func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.query = ""
	m.queryCursor = 0
	m.cursor = 0

	// Load from cache for instant display
	if cached := loadFileCache(m.vaultPath); len(cached) > 0 {
		m.files = cached
		m.updateMatches()
	}

	m.scanning = true
	vaultPath := m.vaultPath
	return func() tea.Msg {
		files := scanVaultFiles(vaultPath)
		saveFileCache(vaultPath, files)
		return FileScanCompleteMsg{Files: files}
	}
}

// HandleScanComplete updates the file list after a background scan.
func (m *Model) HandleScanComplete(files []string) {
	m.files = files
	m.scanning = false
	m.updateMatches()
}

// ShowWithQuery makes the finder visible with a prefilled query.
func (m *Model) ShowWithQuery(q string) tea.Cmd {
	cmd := m.Show()
	m.query = q
	m.queryCursor = len([]rune(q))
	m.updateMatches()
	return cmd
}

// ShowRename opens the finder as a rename prompt with the current path pre-filled.
func (m *Model) ShowRename(absPath string) {
	m.visible = true
	m.renameMode = true
	m.renameFrom = absPath
	// Pre-fill with vault-relative path
	if rel, err := filepath.Rel(m.vaultPath, absPath); err == nil {
		m.query = strings.TrimSuffix(rel, ".md")
	} else {
		m.query = absPath
	}
	m.queryCursor = len([]rune(m.query))
	m.cursor = 0
	m.matches = nil
	m.showCreate = false
}

// AddFile adds a newly created file to the cached list.
func (m *Model) AddFile(relPath string) {
	m.files = append(m.files, relPath)
	sort.Strings(m.files)
	m.updateMatches()
}

// Hide hides the finder.
func (m *Model) Hide() {
	m.visible = false
	m.query = ""
	m.queryCursor = 0
	m.cursor = 0
	m.renameMode = false
	m.renameFrom = ""
}

// IsVisible returns whether the finder is visible.
func (m Model) IsVisible() bool {
	return m.visible
}

// scanVaultFiles walks the vault and returns sorted relative paths of .md files.
func scanVaultFiles(vaultPath string) []string {
	if vaultPath == "" {
		return nil
	}

	var files []string
	filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := info.Name()
		if info.IsDir() {
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}

		rel, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})

	sort.Strings(files)
	return files
}

// Cache helpers — store file lists in ~/.local/state/wordsmith/

func cachePathFor(vaultPath string) string {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, ".local", "state")
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(vaultPath)))[:12]
	return filepath.Join(stateDir, "wordsmith", "filecache-"+hash+".json")
}

func loadFileCache(vaultPath string) []string {
	data, err := os.ReadFile(cachePathFor(vaultPath))
	if err != nil {
		return nil
	}
	var files []string
	if json.Unmarshal(data, &files) != nil {
		return nil
	}
	return files
}

func saveFileCache(vaultPath string, files []string) {
	path := cachePathFor(vaultPath)
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(files)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		os.Rename(tmp, path)
	}
}

func (m *Model) updateMatches() {
	if m.query == "" {
		// Show recent files first (in recency order), then all others alphabetically
		m.matches = nil
		recentSet := make(map[string]bool, len(m.recentFiles))

		// Build a lookup from file name to index in m.files
		fileIndex := make(map[string]int, len(m.files))
		for i, f := range m.files {
			fileIndex[f] = i
		}

		// Add recent files that still exist in the vault
		for _, rf := range m.recentFiles {
			if idx, ok := fileIndex[rf]; ok {
				m.matches = append(m.matches, fuzzy.Match{
					Str:   rf,
					Index: idx,
				})
				recentSet[rf] = true
			}
		}

		// Add remaining files alphabetically
		for i, f := range m.files {
			if recentSet[f] {
				continue
			}
			m.matches = append(m.matches, fuzzy.Match{
				Str:   f,
				Index: i,
			})
			if len(m.matches) >= 100 {
				break
			}
		}
	} else {
		m.matches = fuzzy.Find(m.query, m.files)
	}

	// Determine if we should show a "Create" option
	m.showCreate = false
	m.createPath = ""
	if m.query != "" {
		candidate := normalizeCreatePath(m.query)
		if candidate != "" && !m.fileExists(candidate) {
			m.showCreate = true
			m.createPath = candidate
		}
	}

	total := m.totalItems()
	if m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) totalItems() int {
	n := len(m.matches)
	if m.showCreate {
		n++
	}
	return n
}

func (m *Model) fileExists(relPath string) bool {
	lower := strings.ToLower(relPath)
	for _, f := range m.files {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}

// wordLeft returns the cursor position after moving one word to the left.
// Mirrors the editor's moveCursorWordLeft logic.
func (m *Model) wordLeft() int {
	runes := []rune(m.query)
	pos := m.queryCursor
	// skip whitespace backwards
	for pos > 0 && unicode.IsSpace(runes[pos-1]) {
		pos--
	}
	// skip word chars backwards
	if pos > 0 && isWordChar(runes[pos-1]) {
		for pos > 0 && isWordChar(runes[pos-1]) {
			pos--
		}
	} else if pos > 0 {
		// single punctuation char
		pos--
	}
	return pos
}

// wordRight returns the cursor position after moving one word to the right.
// Mirrors the editor's moveCursorWordRight logic.
func (m *Model) wordRight() int {
	runes := []rune(m.query)
	pos := m.queryCursor
	n := len(runes)
	// skip whitespace forward
	for pos < n && unicode.IsSpace(runes[pos]) {
		pos++
	}
	// skip word chars forward
	if pos < n && isWordChar(runes[pos]) {
		for pos < n && isWordChar(runes[pos]) {
			pos++
		}
	} else if pos < n {
		// single punctuation char
		pos++
	}
	return pos
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// normalizeCreatePath cleans and validates a path for file creation.
// Returns "" if the path is invalid.
func normalizeCreatePath(query string) string {
	p := filepath.Clean(query)

	if filepath.IsAbs(p) {
		return ""
	}

	// Reject paths that try to escape the vault
	for _, part := range strings.Split(p, string(filepath.Separator)) {
		if part == ".." {
			return ""
		}
	}

	base := filepath.Base(p)
	if base == "" || base == "." || base == "/" {
		return ""
	}

	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		p += ".md"
	}

	return p
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.Hide()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Rename mode: confirm rename
			if m.renameMode {
				newPath := normalizeCreatePath(m.query)
				if newPath != "" {
					absNew := filepath.Join(m.vaultPath, newPath)
					oldPath := m.renameFrom
					m.Hide()
					return m, func() tea.Msg {
						return FileRenameMsg{OldPath: oldPath, NewPath: absNew}
					}
				}
				return m, nil
			}
			if m.showCreate && m.cursor == 0 {
				absPath := filepath.Join(m.vaultPath, m.createPath)
				m.Hide()
				return m, func() tea.Msg {
					return FileCreateMsg{Path: absPath}
				}
			}
			matchIdx := m.cursor
			if m.showCreate {
				matchIdx--
			}
			if matchIdx >= 0 && matchIdx < len(m.matches) {
				selected := m.matches[matchIdx].Str
				m.Hide()
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: filepath.Join(m.vaultPath, selected)}
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			if !m.renameMode && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if !m.renameMode {
				total := m.totalItems()
				if m.cursor < total-1 {
					m.cursor++
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			runes := []rune(m.query)
			if m.queryCursor > 0 {
				runes = append(runes[:m.queryCursor-1], runes[m.queryCursor:]...)
				m.query = string(runes)
				m.queryCursor--
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("delete"))):
			runes := []rune(m.query)
			if m.queryCursor < len(runes) {
				runes = append(runes[:m.queryCursor], runes[m.queryCursor+1:]...)
				m.query = string(runes)
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			}
			return m, nil

		case msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'd' || msg.Runes[0] == 'D'):
			// Delete word forward (alt+d), matching editor behaviour
			runes := []rune(m.query)
			end := m.wordRight()
			if end > m.queryCursor {
				runes = append(runes[:m.queryCursor], runes[end:]...)
				m.query = string(runes)
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+w", "ctrl+h", "ctrl+backspace"))):
			// Delete word backward, matching editor behaviour
			runes := []rune(m.query)
			start := m.wordLeft()
			if start < m.queryCursor {
				runes = append(runes[:start], runes[m.queryCursor:]...)
				m.query = string(runes)
				m.queryCursor = start
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("left"))):
			if m.queryCursor > 0 {
				m.queryCursor--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+left"))):
			m.queryCursor = m.wordLeft()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("right"))):
			if m.queryCursor < len([]rune(m.query)) {
				m.queryCursor++
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+right"))):
			m.queryCursor = m.wordRight()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			m.queryCursor = 0
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			m.queryCursor = len([]rune(m.query))
			return m, nil

		default:
			if msg.Type == tea.KeyRunes {
				runes := []rune(m.query)
				newRunes := make([]rune, 0, len(runes)+len(msg.Runes))
				newRunes = append(newRunes, runes[:m.queryCursor]...)
				newRunes = append(newRunes, msg.Runes...)
				newRunes = append(newRunes, runes[m.queryCursor:]...)
				m.query = string(newRunes)
				m.queryCursor += len(msg.Runes)
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			} else if msg.Type == tea.KeySpace {
				runes := []rune(m.query)
				newRunes := make([]rune, 0, len(runes)+1)
				newRunes = append(newRunes, runes[:m.queryCursor]...)
				newRunes = append(newRunes, ' ')
				newRunes = append(newRunes, runes[m.queryCursor:]...)
				m.query = string(newRunes)
				m.queryCursor++
				if !m.renameMode {
					m.cursor = 0
					m.updateMatches()
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(m.theme.bg)

	popupWidth := m.width * 2 / 3
	if popupWidth < 40 {
		popupWidth = 40
	}
	if popupWidth > m.width-4 {
		popupWidth = m.width - 4
	}

	maxResults := m.height/2 - 4
	if maxResults < 3 {
		maxResults = 3
	}

	// Input line with visible cursor
	prompt := "❯ "
	if m.renameMode {
		prompt = "Rename to: "
	}
	qRunes := []rune(m.query)
	qc := m.queryCursor
	if qc > len(qRunes) {
		qc = len(qRunes)
	}

	inputStyle := m.theme.input.Width(0) // don't constrain inner width
	// Text style for non-cursor characters (matches input background so
	// moving the cursor doesn't reveal the terminal's default background).
	textStyle := lipgloss.NewStyle().Foreground(m.theme.input.GetForeground()).Background(m.theme.input.GetBackground())
	var queryRendered string
	if qc < len(qRunes) {
		before := textStyle.Render(string(qRunes[:qc]))
		cursorCh := m.theme.inputCursor.Render(string(qRunes[qc : qc+1]))
		after := textStyle.Render(string(qRunes[qc+1:]))
		queryRendered = before + cursorCh + after
	} else {
		queryRendered = textStyle.Render(m.query) + m.theme.inputCursor.Render(" ")
	}

	inputContent := prompt + queryRendered
	inputLine := inputStyle.Width(popupWidth - 2).Render(inputContent)

	// Results — skip in rename mode
	var resultLines []string

	if m.renameMode {
		// Show a hint
		resultLines = append(resultLines, m.theme.dimmed.Render("  Enter to confirm, Esc to cancel"))
	} else {
		total := m.totalItems()
		start := 0
		if m.cursor >= maxResults {
			start = m.cursor - maxResults + 1
		}
		end := start + maxResults
		if end > total {
			end = total
		}

		for i := start; i < end; i++ {
			if m.showCreate && i == 0 {
				// Create entry
				display := "✚ Create: " + m.createPath
				if len(display) > popupWidth-4 {
					display = display[:popupWidth-5] + "…"
				}
				if i == m.cursor {
					resultLines = append(resultLines, m.theme.selected.Render("▸ "+display))
				} else {
					resultLines = append(resultLines, m.theme.create.Render("  "+display))
				}
			} else {
				matchIdx := i
				if m.showCreate {
					matchIdx--
				}
				if matchIdx >= 0 && matchIdx < len(m.matches) {
					match := m.matches[matchIdx]
					display := match.Str

					// Truncate long paths
					if len(display) > popupWidth-4 {
						display = "…" + display[len(display)-popupWidth+5:]
					}

					if i == m.cursor {
						resultLines = append(resultLines, m.theme.selected.Render("▸ "+display))
					} else {
						resultLines = append(resultLines, m.theme.normal.Render("  "+display))
					}
				}
			}
		}

		if total == 0 {
			if m.scanning {
				resultLines = append(resultLines, m.theme.dimmed.Render("  Scanning files…"))
			} else {
				resultLines = append(resultLines, m.theme.dimmed.Render("  No matches"))
			}
		}
	}

	content := inputLine + "\n" + strings.Join(resultLines, "\n")
	popup := m.theme.border.Width(popupWidth).Render(content)

	// Split popup into lines and measure actual rendered width
	popupLines := strings.Split(popup, "\n")
	popupHeight := len(popupLines)

	// Calculate popup position
	topPad := (m.height - popupHeight) / 3
	if topPad < 1 {
		topPad = 1
	}

	// Build full-screen overlay with themed background everywhere
	var lines []string
	for y := 0; y < m.height; y++ {
		popupIdx := y - topPad
		if popupIdx >= 0 && popupIdx < popupHeight {
			line := popupLines[popupIdx]
			lineW := lipgloss.Width(line)

			leftPad := (m.width - lineW) / 2
			if leftPad < 0 {
				leftPad = 0
			}
			rightPad := m.width - leftPad - lineW
			if rightPad < 0 {
				rightPad = 0
			}

			left := bgStyle.Render(strings.Repeat(" ", leftPad))
			right := bgStyle.Render(strings.Repeat(" ", rightPad))
			lines = append(lines, left+line+right)
		} else {
			// Fill entire line with background
			lines = append(lines, bgStyle.Render(strings.Repeat(" ", m.width)))
		}
	}

	return strings.Join(lines, "\n")
}
