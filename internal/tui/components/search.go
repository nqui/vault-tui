package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
	"github.com/nqui/vault-tui/internal/vault"
)

// SearchResult is emitted when the search overlay is dismissed. Name is the raw
// entry name (with a trailing slash for directories) so that Engine+Name equals
// the target tree node ID (see NewChildNode).
type SearchResult struct {
	Engine    string
	Name      string
	IsDir     bool
	Cancelled bool
}

type SearchModel struct {
	input    textinput.Model
	engine   string
	kvVer    int
	all      []vault.PathEntry
	filtered []vault.PathEntry
	cursor   int
	offset   int
	loading  bool
	active   bool
	width    int
	height   int
}

func NewSearch() SearchModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "type to filter…"
	return SearchModel{input: ti}
}

// Show activates the overlay in a loading state for the given engine and returns
// the command that focuses the text input.
func (m *SearchModel) Show(engine string, kvVer int) tea.Cmd {
	m.engine = engine
	m.kvVer = kvVer
	m.all = nil
	m.filtered = nil
	m.cursor = 0
	m.offset = 0
	m.loading = true
	m.active = true
	m.input.SetValue("")
	return m.input.Focus()
}

// SetResults populates the overlay with freshly listed engine-root entries.
func (m *SearchModel) SetResults(entries []vault.PathEntry) {
	m.all = entries
	m.loading = false
	m.applyFilter()
}

func (m *SearchModel) Close() {
	m.active = false
	m.input.Blur()
}

func (m SearchModel) Active() bool {
	return m.active
}

func (m *SearchModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.SetWidth(w - 12)
}

func (m *SearchModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	m.filtered = m.filtered[:0]
	for _, e := range m.all {
		name := strings.ToLower(strings.TrimSuffix(e.Name, "/"))
		if q == "" || strings.HasPrefix(name, q) {
			m.filtered = append(m.filtered, e)
		}
	}
	m.cursor = 0
	m.offset = 0
}

// visibleRows is the number of result rows the box can show at once.
func (m SearchModel) visibleRows() int {
	// height minus title, input, blank lines and hint footer.
	rows := m.height - 8
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	up := key.NewBinding(key.WithKeys("up", "ctrl+p"))
	down := key.NewBinding(key.WithKeys("down", "ctrl+n"))
	submit := key.NewBinding(key.WithKeys("enter"))
	cancel := key.NewBinding(key.WithKeys("esc", "ctrl+c"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, up):
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
			return m, nil
		case key.Matches(msg, down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				rows := m.visibleRows()
				if m.cursor >= m.offset+rows {
					m.offset = m.cursor - rows + 1
				}
			}
			return m, nil
		case key.Matches(msg, submit):
			if m.loading || len(m.filtered) == 0 {
				return m, nil
			}
			sel := m.filtered[m.cursor]
			engine := m.engine
			m.active = false
			m.input.Blur()
			return m, func() tea.Msg {
				return SearchResult{Engine: engine, Name: sel.Name, IsDir: sel.IsDir}
			}
		case key.Matches(msg, cancel):
			engine := m.engine
			m.active = false
			m.input.Blur()
			return m, func() tea.Msg {
				return SearchResult{Engine: engine, Cancelled: true}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m SearchModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	dirStyle := lipgloss.NewStyle().Foreground(t.Cyan)
	dimStyle := lipgloss.NewStyle().Foreground(t.Subtle).Italic(true)
	hintKeyStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(t.Subtle)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(m.width - 4)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Search " + m.engine))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(dimStyle.Render("  loading…"))
		b.WriteString("\n")
	case len(m.filtered) == 0:
		b.WriteString(dimStyle.Render("  no matches"))
		b.WriteString("\n")
	default:
		rows := m.visibleRows()
		end := m.offset + rows
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := m.offset; i < end; i++ {
			e := m.filtered[i]
			name := strings.TrimSuffix(e.Name, "/")
			if e.IsDir {
				name += "/"
			}
			if i == m.cursor {
				b.WriteString(selectedStyle.Render("  > " + name))
			} else if e.IsDir {
				b.WriteString(dirStyle.Render("    " + name))
			} else {
				b.WriteString(normalStyle.Render("    " + name))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(hintKeyStyle.Render("↑/↓") + hintStyle.Render(" select") + "   " +
		hintKeyStyle.Render("enter") + hintStyle.Render(" jump") + "   " +
		hintKeyStyle.Render("esc") + hintStyle.Render(" cancel"))

	return boxStyle.Render(b.String())
}
