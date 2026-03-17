package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
)

type WrapDataMsg struct {
	Data map[string]interface{}
	TTL  string
}

type UnwrapTokenMsg struct {
	Token string
}

type WrapViewCopyMsg struct {
	Content string
}

type WrapViewDuplicateKeyMsg struct {
	Key string
}

type wrapSection int

const (
	sectionWrap wrapSection = iota
	sectionUnwrap
)

var ttlOptions = []string{"5m", "15m", "30m", "1h", "6h", "24h"}

type WrapViewModel struct {
	// Wrap side
	rows           []kvRow
	ttlIdx         int
	rowOffset      int
	maxVisibleRows int

	// Unwrap side
	tokenInput     textinput.Model
	resultViewport viewport.Model
	resultContent  string
	resultCopyText string

	activeSection wrapSection
	focusIdx      int
	width         int
	height        int

	Submit    key.Binding
	AddRow    key.Binding
	RemoveRow key.Binding
	NextField key.Binding
	PrevField key.Binding
	SwitchSec key.Binding
	Cancel    key.Binding
}

func NewWrapView() WrapViewModel {
	token := textinput.New()
	token.Prompt = ""
	token.Placeholder = "hvs.CAESI..."

	return WrapViewModel{
		rows:       nil,
		ttlIdx:     0,
		tokenInput: token,
		Submit:     key.NewBinding(key.WithKeys("ctrl+s")),
		AddRow:     key.NewBinding(key.WithKeys("ctrl+n")),
		RemoveRow:  key.NewBinding(key.WithKeys("ctrl+d")),
		NextField:  key.NewBinding(key.WithKeys("tab")),
		PrevField:  key.NewBinding(key.WithKeys("shift+tab")),
		SwitchSec:  key.NewBinding(key.WithKeys("ctrl+w")),
		Cancel:     key.NewBinding(key.WithKeys("esc", "ctrl+c")),
	}
}

func (m *WrapViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	halfW := w/2 - 6
	if halfW < 20 {
		halfW = 20
	}
	m.tokenInput.SetWidth(halfW - 4)

	inputW := halfW - 8
	if inputW < 16 {
		inputW = 16
	}
	for i := range m.rows {
		m.rows[i].key.SetWidth(inputW / 3)
		m.rows[i].value.SetWidth(inputW * 2 / 3)
	}

	// Wrap pane chrome: title(1) + sep(1) + blank(1) + "Data"(1) + blank(1) + [rows] + blank(1) + TTL(1) + blank(1) + sep(1) + hints(1) + border/padding(4) = 14
	m.maxVisibleRows = h - 14
	if m.maxVisibleRows < 3 {
		m.maxVisibleRows = 3
	}

	vpHeight := h - 18
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.resultViewport = viewport.New()
	m.resultViewport.SetWidth(halfW - 2)
	m.resultViewport.SetHeight(vpHeight)
	if m.resultContent != "" {
		m.resultViewport.SetContent(m.resultContent)
	}
}

func (m *WrapViewModel) Open() tea.Cmd {
	if len(m.rows) == 0 {
		m.rows = []kvRow{m.newRow()}
	}
	m.activeSection = sectionWrap
	m.focusIdx = 0
	m.resultContent = ""
	return m.updateFocus()
}

func (m *WrapViewModel) SetUnwrapResult(display, copyText string) {
	m.resultContent = display
	m.resultCopyText = copyText
	m.resultViewport.SetContent(display)
	m.resultViewport.GotoTop()
	m.tokenInput.Blur()
}

func (m WrapViewModel) Update(msg tea.Msg) (WrapViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.SwitchSec):
			if m.activeSection == sectionWrap {
				m.activeSection = sectionUnwrap
				m.focusIdx = 0
			} else {
				m.activeSection = sectionWrap
				m.focusIdx = 0
			}
			return m, m.updateFocus()

		case key.Matches(msg, m.Submit):
			if m.activeSection == sectionWrap {
				dupKey := m.findDuplicateKey()
				if dupKey != "" {
					return m, func() tea.Msg {
						return WrapViewDuplicateKeyMsg{Key: dupKey}
					}
				}
				data := m.wrapData()
				if len(data) == 0 {
					return m, nil
				}
				ttl := ttlOptions[m.ttlIdx]
				return m, func() tea.Msg {
					return WrapDataMsg{Data: data, TTL: ttl}
				}
			}
			// In unwrap section, ctrl+s also submits
			token := strings.TrimSpace(m.tokenInput.Value())
			if token == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				return UnwrapTokenMsg{Token: token}
			}
		}

		if m.activeSection == sectionWrap {
			return m.updateWrapSection(msg)
		}
		return m.updateUnwrapSection(msg)
	}

	// Pass non-key messages to focused input
	return m.updateFocusedInput(msg)
}

func (m WrapViewModel) updateWrapSection(msg tea.KeyPressMsg) (WrapViewModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.AddRow):
		m.rows = append(m.rows, m.newRow())
		m.focusIdx = len(m.rows)*2 - 2
		return m, m.updateFocus()

	case key.Matches(msg, m.RemoveRow):
		if len(m.rows) > 1 {
			rowIdx := m.wrapRowIdx()
			if rowIdx >= 0 && rowIdx < len(m.rows) {
				m.rows = append(m.rows[:rowIdx], m.rows[rowIdx+1:]...)
				// Focus the key of the next row, or previous row if we deleted the last
				if rowIdx >= len(m.rows) {
					rowIdx = len(m.rows) - 1
				}
				m.focusIdx = rowIdx * 2
				return m, m.updateFocus()
			}
		}

	case key.Matches(msg, m.NextField):
		maxIdx := len(m.rows) * 2 // rows * 2 fields + TTL
		m.focusIdx++
		if m.focusIdx > maxIdx {
			m.focusIdx = 0
		}
		return m, m.updateFocus()

	case key.Matches(msg, m.PrevField):
		maxIdx := len(m.rows) * 2
		m.focusIdx--
		if m.focusIdx < 0 {
			m.focusIdx = maxIdx
		}
		return m, m.updateFocus()
	}

	// When focused on TTL selector, handle left/right to cycle
	if m.focusIdx == len(m.rows)*2 {
		leftKey := key.NewBinding(key.WithKeys("left", "h"))
		rightKey := key.NewBinding(key.WithKeys("right", "l"))
		switch {
		case key.Matches(msg, leftKey):
			m.ttlIdx--
			if m.ttlIdx < 0 {
				m.ttlIdx = len(ttlOptions) - 1
			}
			return m, nil
		case key.Matches(msg, rightKey):
			m.ttlIdx++
			if m.ttlIdx >= len(ttlOptions) {
				m.ttlIdx = 0
			}
			return m, nil
		}
		return m, nil
	}

	return m.updateFocusedInput(msg)
}

func (m WrapViewModel) updateUnwrapSection(msg tea.KeyPressMsg) (WrapViewModel, tea.Cmd) {
	enterKey := key.NewBinding(key.WithKeys("enter"))
	copyKey := key.NewBinding(key.WithKeys("c"))
	scrollKeys := key.NewBinding(key.WithKeys("j", "k", "up", "down"))

	switch {
	case key.Matches(msg, enterKey):
		token := strings.TrimSpace(m.tokenInput.Value())
		if token == "" {
			return m, nil
		}
		return m, func() tea.Msg {
			return UnwrapTokenMsg{Token: token}
		}
	case key.Matches(msg, copyKey):
		// Only copy when token input is not focused (after unwrap result)
		if m.resultContent != "" && !m.tokenInput.Focused() {
			content := m.resultCopyText
			return m, func() tea.Msg {
				return WrapViewCopyMsg{Content: content}
			}
		}
	case key.Matches(msg, scrollKeys):
		// Scroll viewport when result is shown and token is blurred
		if m.resultContent != "" && !m.tokenInput.Focused() {
			var cmd tea.Cmd
			m.resultViewport, cmd = m.resultViewport.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.tokenInput, cmd = m.tokenInput.Update(msg)
	return m, cmd
}

func (m WrapViewModel) updateFocusedInput(msg tea.Msg) (WrapViewModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.activeSection == sectionUnwrap {
		if m.resultContent != "" && !m.tokenInput.Focused() {
			m.resultViewport, cmd = m.resultViewport.Update(msg)
			return m, cmd
		}
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		return m, cmd
	}

	// Wrap section: TTL is last field (no text input, skip)
	if m.focusIdx == len(m.rows)*2 {
		return m, nil
	}

	rowIdx := m.focusIdx / 2
	isValue := m.focusIdx%2 == 1
	if rowIdx >= 0 && rowIdx < len(m.rows) {
		if isValue {
			m.rows[rowIdx].value, cmd = m.rows[rowIdx].value.Update(msg)
		} else {
			m.rows[rowIdx].key, cmd = m.rows[rowIdx].key.Update(msg)
		}
	}
	return m, cmd
}

func (m WrapViewModel) View() string {
	halfW := m.width/2 - 3
	if halfW < 20 {
		halfW = 20
	}

	leftPane := m.renderWrapPane(halfW)
	rightPane := m.renderUnwrapPane(halfW)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (m WrapViewModel) renderWrapPane(w int) string {
	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true)

	rowNumStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	eqStyle := lipgloss.NewStyle().
		Foreground(t.Overlay)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Overlay)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	selectedStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	var b strings.Builder
	b.WriteString(titleStyle.Render("  Wrap Data"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", w-8)))
	b.WriteString("\n\n")

	scrollIndicator := lipgloss.NewStyle().
		Foreground(t.Subtle).
		Italic(true)

	b.WriteString(labelStyle.Render("  Data"))
	if len(m.rows) > m.maxVisibleRows {
		b.WriteString(scrollIndicator.Render(fmt.Sprintf("  (%d/%d)", m.wrapRowIdx()+1, len(m.rows))))
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
		b.WriteString(eqStyle.Render(" = "))
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
	b.WriteString(labelStyle.Render("  TTL  "))
	isTTLFocused := m.activeSection == sectionWrap && m.focusIdx == len(m.rows)*2
	for i, opt := range ttlOptions {
		if i == m.ttlIdx {
			if isTTLFocused {
				b.WriteString(selectedStyle.Render(" [" + opt + "] "))
			} else {
				b.WriteString(selectedStyle.Render(" " + opt + " "))
			}
		} else {
			b.WriteString(unselectedStyle.Render(" " + opt + " "))
		}
	}
	if isTTLFocused {
		b.WriteString("  ")
		b.WriteString(hintStyle.Render("←/→ change"))
	}

	b.WriteString("\n\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", w-8)))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(hintKeyStyle.Render("ctrl+s") + hintStyle.Render(" wrap  "))
	b.WriteString(hintKeyStyle.Render("ctrl+n") + hintStyle.Render(" add  "))
	b.WriteString(hintKeyStyle.Render("ctrl+d") + hintStyle.Render(" remove  "))
	b.WriteString(hintKeyStyle.Render("tab") + hintStyle.Render("/"))
	b.WriteString(hintKeyStyle.Render("s-tab") + hintStyle.Render(" navigate"))

	borderColor := t.Overlay
	if m.activeSection == sectionWrap {
		borderColor = t.Primary
	}

	contentHeight := m.height - 4
	if contentHeight < 6 {
		contentHeight = 6
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(w).
		Height(contentHeight)

	return boxStyle.Render(b.String())
}

func (m WrapViewModel) renderUnwrapPane(w int) string {
	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(t.Overlay)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	var b strings.Builder
	b.WriteString(titleStyle.Render("  Unwrap Token"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", w-8)))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("  Token"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.tokenInput.View())

	b.WriteString("\n\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", w-8)))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(hintKeyStyle.Render("enter") + hintStyle.Render(" unwrap  "))
	if m.resultContent != "" && !m.tokenInput.Focused() {
		b.WriteString(hintKeyStyle.Render("↑/↓") + hintStyle.Render(" scroll  "))
		b.WriteString(hintKeyStyle.Render("c") + hintStyle.Render(" copy  "))
	}

	if m.resultContent != "" {
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("  Result"))
		b.WriteString("\n")
		b.WriteString(m.resultViewport.View())
	}

	borderColor := t.Overlay
	if m.activeSection == sectionUnwrap {
		borderColor = t.Primary
	}

	contentHeight := m.height - 4
	if contentHeight < 6 {
		contentHeight = 6
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(w).
		Height(contentHeight)

	return boxStyle.Render(b.String())
}

func (m WrapViewModel) wrapData() map[string]interface{} {
	data := make(map[string]interface{})
	for _, row := range m.rows {
		k := strings.TrimSpace(row.key.Value())
		if k != "" {
			data[k] = row.value.Value()
		}
	}
	return data
}

func (m WrapViewModel) findDuplicateKey() string {
	seen := make(map[string]bool)
	for _, row := range m.rows {
		k := strings.TrimSpace(row.key.Value())
		if k == "" {
			continue
		}
		if seen[k] {
			return k
		}
		seen[k] = true
	}
	return ""
}

func (m WrapViewModel) wrapRowIdx() int {
	return m.focusIdx / 2
}

func (m *WrapViewModel) fixRowOffset() {
	rowIdx := m.wrapRowIdx()
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

func (m *WrapViewModel) newRow() kvRow {
	k := textinput.New()
	k.Prompt = ""
	k.Placeholder = "key"

	v := textinput.New()
	v.Prompt = ""
	v.Placeholder = "value"

	halfW := m.width/2 - 6
	inputW := halfW - 8
	if inputW < 16 {
		inputW = 16
	}
	k.SetWidth(inputW / 3)
	v.SetWidth(inputW * 2 / 3)

	return kvRow{key: k, value: v}
}

func (m *WrapViewModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd

	// Blur everything
	m.tokenInput.Blur()
	for i := range m.rows {
		m.rows[i].key.Blur()
		m.rows[i].value.Blur()
	}

	if m.activeSection == sectionUnwrap {
		if m.resultContent == "" {
			cmds = append(cmds, m.tokenInput.Focus())
		}
		// If there's a result, keep token blurred so c works
		return tea.Batch(cmds...)
	}

	// Keep row offset in sync with focus
	m.fixRowOffset()

	// Wrap section: fields are row keys/values then TTL (no text input for TTL)
	if m.focusIdx == len(m.rows)*2 {
		// TTL selector - no text input to focus
		return tea.Batch(cmds...)
	}

	rowIdx := m.focusIdx / 2
	isValue := m.focusIdx%2 == 1
	if rowIdx >= 0 && rowIdx < len(m.rows) {
		if isValue {
			cmds = append(cmds, m.rows[rowIdx].value.Focus())
		} else {
			cmds = append(cmds, m.rows[rowIdx].key.Focus())
		}
	}

	return tea.Batch(cmds...)
}
