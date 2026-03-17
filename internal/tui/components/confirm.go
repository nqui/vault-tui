package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
)

type ConfirmResult struct {
	Confirmed bool
	Context   string
}

type ConfirmModel struct {
	message string
	context string
	active  bool
	width   int
}

func NewConfirm() ConfirmModel {
	return ConfirmModel{}
}

func (m *ConfirmModel) Show(message, context string) {
	m.message = message
	m.context = context
	m.active = true
}

func (m *ConfirmModel) Close() {
	m.active = false
}

func (m ConfirmModel) Active() bool {
	return m.active
}

func (m *ConfirmModel) SetWidth(w int) {
	m.width = w
}

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	yes := key.NewBinding(key.WithKeys("y"))
	no := key.NewBinding(key.WithKeys("n", "esc", "ctrl+c"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, yes):
			m.active = false
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: true, Context: m.context}
			}
		case key.Matches(msg, no):
			m.active = false
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: false, Context: m.context}
			}
		}
	}

	return m, nil
}

func (m ConfirmModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	iconStyle := lipgloss.NewStyle().
		Foreground(t.Red)

	msgStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Red).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Red).
		Padding(1, 3).
		Width(m.width - 4)

	content := iconStyle.Render("⚠  ") + msgStyle.Render(m.message) +
		"\n\n" +
		hintKeyStyle.Render("y") + hintStyle.Render(" confirm") + "   " +
		hintKeyStyle.Render("n") + hintStyle.Render("/") +
		hintKeyStyle.Render("esc") + hintStyle.Render(" cancel")

	return boxStyle.Render(content)
}
