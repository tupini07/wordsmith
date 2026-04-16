package finder

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	cursor      int
	width       int
	height      int
	visible     bool
	scanning    bool
	showCreate  bool   // whether to show a "Create" entry
	createPath  string // normalized relative path for file creation
	theme       finderTheme
}

type finderTheme struct {
	border   lipgloss.Style
	normal   lipgloss.Style
	selected lipgloss.Style
	matched  lipgloss.Style
	input    lipgloss.Style
	dimmed   lipgloss.Style
	create   lipgloss.Style // style for the "Create" entry
	bg       lipgloss.Color
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
		input:    lipgloss.NewStyle().Foreground(fg).Background(midBg).Padding(0, 1),
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
	m.updateMatches()
	return cmd
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
	m.cursor = 0
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
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			total := m.totalItems()
			if m.cursor < total-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.cursor = 0
				m.updateMatches()
			}
			return m, nil

		default:
			if msg.Type == tea.KeyRunes {
				m.query += string(msg.Runes)
				m.cursor = 0
				m.updateMatches()
			} else if msg.Type == tea.KeySpace {
				m.query += " "
				m.cursor = 0
				m.updateMatches()
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

	// Input line
	prompt := "❯ "
	inputLine := m.theme.input.Width(popupWidth - 2).Render(prompt + m.query)

	// Results — virtual list includes optional create entry at index 0, then matches
	var resultLines []string
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
