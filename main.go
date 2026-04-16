package main

import (
	"fmt"
	"os"

	"github.com/tupini07/wordsmith/internal/app"
	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/state"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func main() {
	// Force true color rendering — our themes use 24-bit hex colors that
	// degrade badly when quantized to 16/256-color palettes.
	lipgloss.SetColorProfile(termenv.TrueColor)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wordsmith: config error: %v\n", err)
		os.Exit(1)
	}

	st, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wordsmith: state error: %v\n", err)
		os.Exit(1)
	}

	// Determine which file to open
	var filePath string
	if len(os.Args) > 1 {
		filePath = os.Args[1]
	} else if st.LastFile != "" && cfg.VaultPath != "" {
		filePath = cfg.AbsFilePath(st.LastFile)
	}

	model := app.New(cfg, st, filePath)

	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wordsmith: %v\n", err)
		os.Exit(1)
	}

	// Save state on exit
	if m, ok := finalModel.(app.Model); ok {
		if err := m.SaveState(); err != nil {
			fmt.Fprintf(os.Stderr, "wordsmith: failed to save state: %v\n", err)
		}
	}
}
