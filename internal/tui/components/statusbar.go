package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type statusHint struct {
	key  string
	desc string
}

type StatusBarModel struct {
	width   int
	message string
	isError bool

	BgStyle   lipgloss.Style
	KeyStyle  lipgloss.Style
	DescStyle lipgloss.Style
	MsgStyle  lipgloss.Style
	ErrStyle  lipgloss.Style
}

func NewStatusBar() StatusBarModel {
	return StatusBarModel{}
}

func (m *StatusBarModel) SetWidth(w int) {
	m.width = w
}

func (m *StatusBarModel) SetMessage(msg string) {
	m.message = msg
	m.isError = false
}

func (m *StatusBarModel) SetError(err error) {
	if err != nil {
		m.message = err.Error()
		m.isError = true
	}
}

func (m *StatusBarModel) ClearError() {
	if m.isError {
		m.message = ""
		m.isError = false
	}
}

var defaultHints = []statusHint{
	{"tab", "switch"},
	{"enter", "open"},
	{"n", "new"},
	{"e", "edit"},
	{"d", "delete"},
	{"c", "copy"},
	{"r", "refresh"},
	{"q", "quit"},
}

func (m StatusBarModel) View() string {
	if m.message != "" {
		var style lipgloss.Style
		if m.isError {
			style = m.ErrStyle
		} else {
			style = m.MsgStyle
		}
		msg := style.Render(m.message)
		pad := m.width - lipgloss.Width(msg)
		if pad > 0 {
			msg += m.BgStyle.Width(pad).Render("")
		}
		return msg
	}

	var b strings.Builder
	for _, h := range defaultHints {
		b.WriteString(m.KeyStyle.Render(h.key))
		b.WriteString(m.DescStyle.Render(h.desc))
	}
	line := b.String()

	pad := m.width - lipgloss.Width(line)
	if pad > 0 {
		line += m.BgStyle.Width(pad).Render("")
	}

	return line
}
