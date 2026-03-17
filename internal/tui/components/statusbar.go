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
	hints   []statusHint

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
	m.message = ""
	m.isError = false
}

func (m *StatusBarModel) SetHints(hints []statusHint) {
	m.hints = hints
}

var SecretsViewHints = []statusHint{
	{"tab", "switch"},
	{"enter", "open"},
	{"n", "new"},
	{"e", "edit"},
	{"d", "delete"},
	{"c", "copy"},
	{"w", "wrap"},
	{"u", "unwrap"},
	{"W", "wrap view"},
	{"r", "refresh"},
	{"q", "quit"},
}

var WrapViewHints = []statusHint{
	{"ctrl+s", "submit"},
	{"ctrl+n", "add row"},
	{"ctrl+d", "del row"},
	{"ctrl+w", "switch pane"},
	{"c", "copy"},
	{"W", "secrets"},
	{"esc", "back"},
	{"q", "quit"},
}

func (m StatusBarModel) Height() int {
	if m.message != "" || m.width == 0 {
		return 1
	}
	hints := m.hints
	if hints == nil {
		hints = SecretsViewHints
	}
	lines := 1
	lineWidth := 0
	for _, h := range hints {
		s := m.KeyStyle.Render(h.key) + m.DescStyle.Render(h.desc)
		w := lipgloss.Width(s)
		if lineWidth > 0 && lineWidth+w > m.width {
			lines++
			lineWidth = 0
		}
		lineWidth += w
	}
	return lines
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

	hints := m.hints
	if hints == nil {
		hints = SecretsViewHints
	}

	// Build hint segments and measure their widths
	type segment struct {
		text  string
		width int
	}
	var segs []segment
	for _, h := range hints {
		s := m.KeyStyle.Render(h.key) + m.DescStyle.Render(h.desc)
		segs = append(segs, segment{s, lipgloss.Width(s)})
	}

	// Lay out hints, wrapping to next line when needed
	var lines []string
	lineWidth := 0
	var b strings.Builder
	for _, s := range segs {
		if lineWidth > 0 && lineWidth+s.width > m.width {
			pad := m.width - lineWidth
			if pad > 0 {
				b.WriteString(m.BgStyle.Width(pad).Render(""))
			}
			lines = append(lines, b.String())
			b.Reset()
			lineWidth = 0
		}
		b.WriteString(s.text)
		lineWidth += s.width
	}
	if b.Len() > 0 {
		pad := m.width - lineWidth
		if pad > 0 {
			b.WriteString(m.BgStyle.Width(pad).Render(""))
		}
		lines = append(lines, b.String())
	}

	return strings.Join(lines, "\n")
}
