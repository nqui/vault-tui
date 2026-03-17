package components

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
)

type PromptResult struct {
	Value     string
	Cancelled bool
	Context   string
}

type PromptModel struct {
	input   textinput.Model
	title   string
	context string
	active  bool
	width   int
}

func NewPrompt() PromptModel {
	ti := textinput.New()
	ti.Prompt = "> "
	return PromptModel{input: ti}
}

func (m *PromptModel) Show(title, placeholder, context string) tea.Cmd {
	m.title = title
	m.context = context
	m.active = true
	m.input.SetValue("")
	m.input.Placeholder = placeholder
	return m.input.Focus()
}

func (m *PromptModel) Close() {
	m.active = false
	m.input.Blur()
}

func (m PromptModel) Active() bool {
	return m.active
}

func (m *PromptModel) SetWidth(w int) {
	m.width = w
	m.input.SetWidth(w - 12)
}

func (m PromptModel) Update(msg tea.Msg) (PromptModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	submit := key.NewBinding(key.WithKeys("enter"))
	cancel := key.NewBinding(key.WithKeys("esc", "ctrl+c"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, submit):
			value := m.input.Value()
			m.active = false
			m.input.Blur()
			return m, func() tea.Msg {
				return PromptResult{Value: value, Context: m.context}
			}
		case key.Matches(msg, cancel):
			m.active = false
			m.input.Blur()
			return m, func() tea.Msg {
				return PromptResult{Cancelled: true, Context: m.context}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m PromptModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(m.width - 4)

	content := titleStyle.Render(m.title) +
		"\n\n" +
		m.input.View() +
		"\n\n" +
		hintKeyStyle.Render("enter") + hintStyle.Render(" confirm") + "   " +
		hintKeyStyle.Render("esc") + hintStyle.Render(" cancel")

	return boxStyle.Render(content)
}
