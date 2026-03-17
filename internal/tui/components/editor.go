package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nq/hv-tui/internal/tui/theme"
)

type editorMode int

const (
	EditorCreate editorMode = iota
	EditorEdit
)

type kvRow struct {
	key   textinput.Model
	value textinput.Model
}

type EditorModel struct {
	pathInput      textinput.Model
	rows           []kvRow
	rowOffset      int
	maxVisibleRows int
	mode           editorMode
	active         bool
	focusIdx       int
	width          int
	height         int
	engine         string
	kvVersion      int

	Save      key.Binding
	AddRow    key.Binding
	RemoveRow key.Binding
	Cancel    key.Binding
	NextField key.Binding
	PrevField key.Binding
}

func NewEditor() EditorModel {
	pathInput := textinput.New()
	pathInput.Prompt = "Path: "
	pathInput.Placeholder = "path/to/secret"

	return EditorModel{
		pathInput: pathInput,
		Save:      key.NewBinding(key.WithKeys("ctrl+s")),
		AddRow:    key.NewBinding(key.WithKeys("ctrl+n")),
		RemoveRow: key.NewBinding(key.WithKeys("ctrl+d")),
		Cancel:    key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		NextField: key.NewBinding(key.WithKeys("tab")),
		PrevField: key.NewBinding(key.WithKeys("shift+tab")),
	}
}

func (m *EditorModel) OpenCreate(engine string, basePath string, kvVersion int) tea.Cmd {
	m.active = true
	m.mode = EditorCreate
	m.engine = engine
	m.kvVersion = kvVersion
	m.pathInput.SetValue(basePath)
	m.rows = []kvRow{m.newRow()}
	m.focusIdx = 0
	return m.updateFocus()
}

func (m *EditorModel) OpenEdit(engine string, path string, kvVersion int, data map[string]interface{}) tea.Cmd {
	m.active = true
	m.mode = EditorEdit
	m.engine = engine
	m.kvVersion = kvVersion
	m.pathInput.SetValue(path)

	m.rows = nil
	for k, v := range data {
		row := m.newRow()
		row.key.SetValue(k)
		row.value.SetValue(fmt.Sprintf("%v", v))
		m.rows = append(m.rows, row)
	}
	if len(m.rows) == 0 {
		m.rows = []kvRow{m.newRow()}
	}
	m.focusIdx = 1 // focus first key
	return m.updateFocus()
}

func (m *EditorModel) Close() {
	m.active = false
}

func (m EditorModel) Active() bool {
	return m.active
}

func (m EditorModel) Engine() string {
	return m.engine
}

func (m EditorModel) KVVersion() int {
	return m.kvVersion
}

func (m EditorModel) Path() string {
	return m.pathInput.Value()
}

func (m EditorModel) Data() map[string]interface{} {
	data := make(map[string]interface{})
	for _, row := range m.rows {
		k := strings.TrimSpace(row.key.Value())
		if k != "" {
			data[k] = row.value.Value()
		}
	}
	return data
}

func (m EditorModel) Mode() editorMode {
	return m.mode
}

func (m *EditorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := w - 12
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.pathInput.SetWidth(inputWidth)
	for i := range m.rows {
		m.rows[i].key.SetWidth(inputWidth / 3)
		m.rows[i].value.SetWidth(inputWidth * 2 / 3)
	}

	// Chrome: title(1) + sep(1) + blank(1) + path(2-3) + blank(1) + "Data"(1) + blank(1) + [rows] + blank(1) + sep(1) + hints(1) + border/padding(4) = ~16
	m.maxVisibleRows = h - 16
	if m.maxVisibleRows < 3 {
		m.maxVisibleRows = 3
	}
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.AddRow):
			m.rows = append(m.rows, m.newRow())
			m.focusIdx = 1 + (len(m.rows)-1)*2
			return m, m.updateFocus()
		case key.Matches(msg, m.RemoveRow):
			if len(m.rows) > 1 {
				rowIdx := m.currentRowIdx()
				if rowIdx >= 0 && rowIdx < len(m.rows) {
					m.rows = append(m.rows[:rowIdx], m.rows[rowIdx+1:]...)
					if m.focusIdx > len(m.rows)*2 {
						m.focusIdx = len(m.rows) * 2
					}
					return m, m.updateFocus()
				}
			}
		case key.Matches(msg, m.NextField):
			maxIdx := len(m.rows) * 2
			if m.mode == EditorCreate {
				maxIdx++ // include path field
			}
			m.focusIdx++
			if m.focusIdx > maxIdx {
				m.focusIdx = 0
			}
			return m, m.updateFocus()
		case key.Matches(msg, m.PrevField):
			maxIdx := len(m.rows) * 2
			if m.mode == EditorCreate {
				maxIdx++
			}
			m.focusIdx--
			if m.focusIdx < 0 {
				m.focusIdx = maxIdx
			}
			return m, m.updateFocus()
		}
	}

	var cmd tea.Cmd
	if m.focusIdx == 0 {
		m.pathInput, cmd = m.pathInput.Update(msg)
	} else {
		rowIdx := (m.focusIdx - 1) / 2
		isValue := (m.focusIdx-1)%2 == 1
		if rowIdx >= 0 && rowIdx < len(m.rows) {
			if isValue {
				m.rows[rowIdx].value, cmd = m.rows[rowIdx].value.Update(msg)
			} else {
				m.rows[rowIdx].key, cmd = m.rows[rowIdx].key.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m EditorModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true)

	pathDisplayStyle := lipgloss.NewStyle().
		Foreground(t.Text)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Overlay)

	rowNumStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	eqStyle := lipgloss.NewStyle().
		Foreground(t.Overlay)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	var b strings.Builder

	title := "  Create Secret"
	if m.mode == EditorEdit {
		title = "  Edit Secret"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	editorWidth := m.width - 10
	if editorWidth < 30 {
		editorWidth = 30
	}
	b.WriteString(separatorStyle.Render(strings.Repeat("─", editorWidth)))
	b.WriteString("\n\n")

	if m.mode == EditorCreate {
		b.WriteString(labelStyle.Render("  Path"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(m.pathInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString(labelStyle.Render("  Path  "))
		b.WriteString(pathDisplayStyle.Render(m.pathInput.Value()))
		b.WriteString("\n\n")
	}

	scrollIndicator := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Italic(true)

	b.WriteString(labelStyle.Render("  Data"))
	if len(m.rows) > m.maxVisibleRows {
		b.WriteString(scrollIndicator.Render(fmt.Sprintf("  (%d/%d)", m.currentRowIdx()+1, len(m.rows))))
	}
	b.WriteString("\n\n")

	end := m.rowOffset + m.maxVisibleRows
	if end > len(m.rows) {
		end = len(m.rows)
	}
	if m.rowOffset > 0 {
		b.WriteString(scrollIndicator.Render("     ↑ more"))
		b.WriteByte('\n')
	}
	for i := m.rowOffset; i < end; i++ {
		row := m.rows[i]
		num := fmt.Sprintf("  %2d ", i+1)
		b.WriteString(rowNumStyle.Render(num))
		b.WriteString(row.key.View())
		b.WriteString(eqStyle.Render("  =  "))
		b.WriteString(row.value.View())
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	if end < len(m.rows) {
		b.WriteByte('\n')
		b.WriteString(scrollIndicator.Render("     ↓ more"))
	}

	b.WriteString("\n\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", editorWidth)))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(hintKeyStyle.Render("ctrl+s") + hintStyle.Render(" save  "))
	b.WriteString(hintKeyStyle.Render("ctrl+n") + hintStyle.Render(" add row  "))
	b.WriteString(hintKeyStyle.Render("ctrl+d") + hintStyle.Render(" remove row  "))
	b.WriteString(hintKeyStyle.Render("esc") + hintStyle.Render(" cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}

func (m *EditorModel) newRow() kvRow {
	k := textinput.New()
	k.Prompt = ""
	k.Placeholder = "key"

	v := textinput.New()
	v.Prompt = ""
	v.Placeholder = "value"

	inputWidth := m.width - 12
	if inputWidth < 20 {
		inputWidth = 20
	}
	k.SetWidth(inputWidth / 3)
	v.SetWidth(inputWidth * 2 / 3)

	return kvRow{key: k, value: v}
}

func (m EditorModel) currentRowIdx() int {
	if m.focusIdx <= 0 {
		return -1
	}
	return (m.focusIdx - 1) / 2
}

func (m *EditorModel) fixRowOffset() {
	rowIdx := m.currentRowIdx()
	if rowIdx < 0 {
		return
	}
	if rowIdx < m.rowOffset {
		m.rowOffset = rowIdx
	}
	if rowIdx >= m.rowOffset+m.maxVisibleRows {
		m.rowOffset = rowIdx - m.maxVisibleRows + 1
	}
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}
}

func (m *EditorModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd

	m.pathInput.Blur()
	for i := range m.rows {
		m.rows[i].key.Blur()
		m.rows[i].value.Blur()
	}

	m.fixRowOffset()

	if m.focusIdx == 0 {
		cmds = append(cmds, m.pathInput.Focus())
	} else {
		rowIdx := (m.focusIdx - 1) / 2
		isValue := (m.focusIdx-1)%2 == 1
		if rowIdx >= 0 && rowIdx < len(m.rows) {
			if isValue {
				cmds = append(cmds, m.rows[rowIdx].value.Focus())
			} else {
				cmds = append(cmds, m.rows[rowIdx].key.Focus())
			}
		}
	}

	return tea.Batch(cmds...)
}
