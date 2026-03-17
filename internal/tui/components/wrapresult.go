package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nq/hv-tui/internal/tui/theme"
)

type WrapResultDismissed struct {
	CopyToken bool
	Token     string
}

type WrapResultModel struct {
	token  string
	ttl    string
	path   string
	active bool
	width  int
}

func NewWrapResult() WrapResultModel {
	return WrapResultModel{}
}

func (m *WrapResultModel) Show(token, ttl, path string) {
	m.token = token
	m.ttl = ttl
	m.path = path
	m.active = true
}

func (m *WrapResultModel) Close() {
	m.active = false
}

func (m WrapResultModel) Active() bool {
	return m.active
}

func (m *WrapResultModel) SetWidth(w int) {
	m.width = w
}

func (m WrapResultModel) Update(msg tea.Msg) (WrapResultModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	copyKey := key.NewBinding(key.WithKeys("c"))
	dismiss := key.NewBinding(key.WithKeys("esc", "enter", "ctrl+c"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, copyKey):
			m.active = false
			return m, func() tea.Msg {
				return WrapResultDismissed{CopyToken: true, Token: m.token}
			}
		case key.Matches(msg, dismiss):
			m.active = false
			return m, func() tea.Msg {
				return WrapResultDismissed{CopyToken: false, Token: m.token}
			}
		}
	}

	return m, nil
}

func (m WrapResultModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Green).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	tokenStyle := lipgloss.NewStyle().
		Foreground(t.Yellow).
		Bold(true)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Green).
		Padding(1, 3).
		Width(m.width - 4)

	content := titleStyle.Render("Wrapping Token Created") + "\n\n"

	if m.path != "" {
		content += labelStyle.Render("Path: ") + valueStyle.Render(m.path) + "\n"
	}
	content += labelStyle.Render("TTL:  ") + valueStyle.Render(m.ttl) + "\n\n"
	content += labelStyle.Render("Token:") + "\n"
	content += tokenStyle.Render(m.token) + "\n\n"
	content += hintKeyStyle.Render("c") + hintStyle.Render(" copy token") + "   " +
		hintKeyStyle.Render("esc") + hintStyle.Render("/") +
		hintKeyStyle.Render("enter") + hintStyle.Render(" dismiss")

	return boxStyle.Render(content)
}
