package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/nq/hv-tui/internal/config"
	"github.com/nq/hv-tui/internal/tui"
	"github.com/nq/hv-tui/internal/tui/theme"
	"github.com/nq/hv-tui/internal/vault"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := theme.Set(cfg.Theme); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	tui.InitStyles()

	client, err := vault.New(cfg.Addr, cfg.Token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Vault: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewApp(client))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
