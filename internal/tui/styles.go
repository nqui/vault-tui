package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
)

var (
	titleBarStyle     lipgloss.Style
	titleBarInfoStyle lipgloss.Style
	focusedBorder     lipgloss.Style
	unfocusedBorder   lipgloss.Style
	paneHeaderStyle   lipgloss.Style
	statusBarBg       lipgloss.Style
	statusKeyStyle    lipgloss.Style
	statusDescStyle   lipgloss.Style
	statusMsgStyle    lipgloss.Style
	statusErrStyle    lipgloss.Style
)

// InitStyles rebuilds all module-level styles from the active theme.
// Must be called after theme.Set().
func InitStyles() {
	t := theme.Active

	titleBarStyle = lipgloss.NewStyle().
		Background(t.Primary).
		Foreground(t.Bg).
		Bold(true).
		PaddingLeft(2).
		PaddingRight(2)

	titleBarInfoStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Subtle).
		PaddingLeft(1).
		PaddingRight(1)

	focusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary)

	unfocusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay)

	paneHeaderStyle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		PaddingBottom(1)

	statusBarBg = lipgloss.NewStyle().
		Background(t.Surface)

	statusKeyStyle = lipgloss.NewStyle().
		Background(t.Overlay).
		Foreground(t.Bright).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	statusDescStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Subtle).
		PaddingRight(2)

	statusMsgStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Green).
		PaddingLeft(1)

	statusErrStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Red).
		Bold(true).
		PaddingLeft(1)
}
