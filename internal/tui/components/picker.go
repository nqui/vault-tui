package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nq/hv-tui/internal/tui/theme"
)

type PickerResult struct {
	Value     string
	Cancelled bool
	Context   string
}

type PickerModel struct {
	title   string
	options []string
	cursor  int
	context string
	active  bool
	width   int
}

func NewPicker() PickerModel {
	return PickerModel{}
}

func (m *PickerModel) Show(title string, options []string, context string) {
	m.title = title
	m.options = options
	m.cursor = 0
	m.context = context
	m.active = true
}

func (m *PickerModel) Close() {
	m.active = false
}

func (m PickerModel) Active() bool {
	return m.active
}

func (m *PickerModel) SetWidth(w int) {
	m.width = w
}

func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	up := key.NewBinding(key.WithKeys("k", "up"))
	down := key.NewBinding(key.WithKeys("j", "down"))
	submit := key.NewBinding(key.WithKeys("enter"))
	cancel := key.NewBinding(key.WithKeys("esc", "ctrl+c"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, down):
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case key.Matches(msg, submit):
			value := m.options[m.cursor]
			m.active = false
			return m, func() tea.Msg {
				return PickerResult{Value: value, Context: m.context}
			}
		case key.Matches(msg, cancel):
			m.active = false
			return m, func() tea.Msg {
				return PickerResult{Cancelled: true, Context: m.context}
			}
		}
	}

	return m, nil
}

func (m PickerModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

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

	content := titleStyle.Render(m.title) + "\n\n"

	for i, opt := range m.options {
		if i == m.cursor {
			content += selectedStyle.Render("  > " + opt) + "\n"
		} else {
			content += normalStyle.Render("    " + opt) + "\n"
		}
	}

	content += "\n" +
		hintKeyStyle.Render("↑/↓") + hintStyle.Render(" select") + "   " +
		hintKeyStyle.Render("enter") + hintStyle.Render(" confirm") + "   " +
		hintKeyStyle.Render("esc") + hintStyle.Render(" cancel")

	return boxStyle.Render(content)
}
