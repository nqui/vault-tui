package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/nqui/vault-tui/internal/config"
	"github.com/nqui/vault-tui/internal/tui"
	"github.com/nqui/vault-tui/internal/tui/theme"
	"github.com/nqui/vault-tui/internal/vault"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("hvt %s (%s)\n", version, commit)
		os.Exit(0)
	}

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

	p := tea.NewProgram(tui.NewApp(client, cfg))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
