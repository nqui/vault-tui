package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	colorBg      = lipgloss.Color("#1A1B26")
	colorSurface = lipgloss.Color("#24283B")
	colorOverlay = lipgloss.Color("#414868")
	colorSubtle  = lipgloss.Color("#565F89")
	colorText    = lipgloss.Color("#C0CAF5")
	colorBright  = lipgloss.Color("#E0E0FF")
	colorPurple  = lipgloss.Color("#BB9AF7")
	colorBlue    = lipgloss.Color("#7AA2F7")
	colorCyan    = lipgloss.Color("#7DCFFF")
	colorGreen   = lipgloss.Color("#9ECE6A")
	colorYellow  = lipgloss.Color("#E0AF68")
	colorRed     = lipgloss.Color("#F7768E")
	colorOrange  = lipgloss.Color("#FF9E64")

	titleBarStyle = lipgloss.NewStyle().
			Background(colorPurple).
			Foreground(lipgloss.Color("#1A1B26")).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(2)

	titleBarInfoStyle = lipgloss.NewStyle().
				Background(colorSurface).
				Foreground(colorSubtle).
				PaddingLeft(1).
				PaddingRight(1)

	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple)

	unfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorOverlay)

	paneHeaderStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true).
			PaddingBottom(1)

	statusBarBg = lipgloss.NewStyle().
			Background(colorSurface)

	statusKeyStyle = lipgloss.NewStyle().
			Background(colorOverlay).
			Foreground(colorBright).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	statusDescStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorSubtle).
			PaddingRight(2)

	statusMsgStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorGreen).
			PaddingLeft(1)

	statusErrStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorRed).
			Bold(true).
			PaddingLeft(1)
)
