package filetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileSelectedMsg is sent when a file is selected in the tree.
type FileSelectedMsg struct {
	Path string
}

// CreateInDirMsg is sent when the user wants to create a file in a directory.
type CreateInDirMsg struct {
	Dir string // relative directory path within the vault
}

// Node represents a file or directory in the tree.
type Node struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*Node
	Expanded bool
	Depth    int
}

// Model is the file tree sidebar.
type Model struct {
	root      *Node
	vaultPath string
	flat      []*Node // flattened visible nodes
	cursor    int
	width     int
	height    int
	visible   bool
	scroll    int
	theme     treeTheme
}

type treeTheme struct {
	normal   lipgloss.Style
	selected lipgloss.Style
	dir      lipgloss.Style
	file     lipgloss.Style
	border   lipgloss.Style
	bg       lipgloss.Color
}

func defaultTreeTheme() treeTheme {
	bg := lipgloss.Color("#1E1C1F")
	cream := lipgloss.Color("#F5E6D3")
	orange := lipgloss.Color("#E8A87C")
	teal := lipgloss.Color("#41B3A3")
	chromeBg := lipgloss.Color("#2D2A2E")

	return makeTreeTheme(bg, chromeBg, cream, orange, teal)
}

func makeTreeTheme(bg, chromeBg, fg, accent, dirColor lipgloss.Color) treeTheme {
	return treeTheme{
		bg:       bg,
		normal:   lipgloss.NewStyle().Foreground(fg).Background(bg),
		selected: lipgloss.NewStyle().Foreground(accent).Background(bg).Bold(true),
		dir:      lipgloss.NewStyle().Foreground(dirColor).Background(bg).Bold(true),
		file:     lipgloss.NewStyle().Foreground(fg).Background(bg),
		border:   lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(chromeBg).Background(bg),
	}
}

// New creates a new file tree model.
func New(vaultPath string) Model {
	return Model{
		vaultPath: vaultPath,
		theme:     defaultTreeTheme(),
	}
}

// SetThemeColors updates the tree's colors from the main theme.
func (m *Model) SetThemeColors(bg, chromeBg, fg, accent, dirColor lipgloss.Color) {
	m.theme = makeTreeTheme(bg, chromeBg, fg, accent, dirColor)
}

// SetSize sets the tree dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Show makes the tree visible and scans files.
func (m *Model) Show() {
	m.visible = true
	m.scan()
	m.flatten()
}

// Hide hides the tree.
func (m *Model) Hide() {
	m.visible = false
}

// Toggle toggles visibility.
func (m *Model) Toggle() {
	if m.visible {
		m.Hide()
	} else {
		m.Show()
	}
}

// IsVisible returns whether the tree is visible.
func (m Model) IsVisible() bool {
	return m.visible
}

// Width returns the sidebar width.
func (m Model) Width() int {
	if !m.visible {
		return 0
	}
	return m.width
}

func (m *Model) scan() {
	if m.vaultPath == "" {
		return
	}
	m.root = &Node{
		Name:     filepath.Base(m.vaultPath),
		Path:     m.vaultPath,
		IsDir:    true,
		Expanded: true,
		Depth:    0,
	}
	m.scanDir(m.root)
}

func (m *Model) scanDir(parent *Node) {
	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return
	}

	var dirs, files []*Node

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		path := filepath.Join(parent.Path, name)
		node := &Node{
			Name:  name,
			Path:  path,
			IsDir: e.IsDir(),
			Depth: parent.Depth + 1,
		}

		if e.IsDir() {
			dirs = append(dirs, node)
		} else if strings.HasSuffix(strings.ToLower(name), ".md") {
			files = append(files, node)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	parent.Children = append(dirs, files...)
}

func (m *Model) flatten() {
	m.flat = nil
	if m.root != nil {
		m.flattenNode(m.root)
	}
}

func (m *Model) flattenNode(node *Node) {
	m.flat = append(m.flat, node)
	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child)
		}
	}
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if m.cursor < len(m.flat)-1 {
				m.cursor++
				m.ensureVisible()
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.flat) {
				node := m.flat[m.cursor]
				if node.IsDir {
					if !node.Expanded {
						node.Expanded = true
						if len(node.Children) == 0 {
							m.scanDir(node)
						}
					} else {
						node.Expanded = false
					}
					m.flatten()
				} else {
					path := node.Path
					return m, func() tea.Msg {
						return FileSelectedMsg{Path: path}
					}
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			// Create a new file — determine the target directory
			dir := m.vaultPath
			if m.cursor < len(m.flat) {
				node := m.flat[m.cursor]
				if node.IsDir {
					dir = node.Path
				} else {
					dir = filepath.Dir(node.Path)
				}
			}
			rel, err := filepath.Rel(m.vaultPath, dir)
			if err != nil {
				rel = ""
			}
			if rel == "." {
				rel = ""
			}
			if rel != "" {
				rel += "/"
			}
			return m, func() tea.Msg {
				return CreateInDirMsg{Dir: rel}
			}
		}
	}

	return m, nil
}

func (m *Model) ensureVisible() {
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+m.height {
		m.scroll = m.cursor - m.height + 1
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.visible || m.width <= 0 || m.height <= 0 {
		return ""
	}

	var lines []string
	end := m.scroll + m.height
	if end > len(m.flat) {
		end = len(m.flat)
	}

	for i := m.scroll; i < end; i++ {
		node := m.flat[i]
		indent := strings.Repeat("  ", node.Depth)

		var prefix, name string
		if node.IsDir {
			if node.Expanded {
				prefix = "▾ "
			} else {
				prefix = "▸ "
			}
			name = node.Name + "/"
		} else {
			prefix = "  "
			// Strip .md extension for cleaner display
			name = strings.TrimSuffix(node.Name, ".md")
		}

		display := indent + prefix + name

		// Truncate if too long
		if len(display) > m.width-1 {
			display = display[:m.width-4] + "…"
		}

		// Pad to width
		for len(display) < m.width-1 {
			display += " "
		}

		if i == m.cursor {
			lines = append(lines, m.theme.selected.Render(display))
		} else if node.IsDir {
			lines = append(lines, m.theme.dir.Render(display))
		} else {
			lines = append(lines, m.theme.file.Render(display))
		}
	}

	// Fill remaining height with background
	bgFill := m.theme.normal.Render(strings.Repeat(" ", m.width-1))
	for len(lines) < m.height {
		lines = append(lines, bgFill)
	}

	content := strings.Join(lines, "\n")
	return m.theme.border.Render(content)
}
