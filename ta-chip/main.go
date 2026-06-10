package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"ta-chip/internal/config"
	"ta-chip/internal/ui"
	"ta-chip/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			// Print bare version string for PowerShell to parse
			fmt.Println(version.Version)
			os.Exit(0)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ta-chip: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(cfg)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ta-chip: %v\n", err)
		os.Exit(1)
	}
}
