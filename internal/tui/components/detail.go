package components

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/nq/hv-tui/internal/tui/theme"
	"github.com/nq/hv-tui/internal/vault"
)

type detailState int

const (
	detailEmpty detailState = iota
	detailLoading
	detailSecret
	detailVersions
	detailError
)

type DetailModel struct {
	viewport viewport.Model
	state    detailState
	secret   *vault.SecretEntry
	versions []vault.VersionInfo
	errMsg   string
	width    int
	height   int
	focused  bool
}

func NewDetail() DetailModel {
	vp := viewport.New()
	return DetailModel{
		viewport: vp,
		state:    detailEmpty,
	}
}

func (m *DetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m *DetailModel) SetFocused(f bool) {
	m.focused = f
}

func (m DetailModel) Focused() bool {
	return m.focused
}

func (m *DetailModel) ShowSecret(secret *vault.SecretEntry) {
	m.state = detailSecret
	m.secret = secret
	m.viewport.SetContent(RenderSecret(secret, m.width))
}

func (m *DetailModel) ShowVersions(versions []vault.VersionInfo, path string) {
	m.state = detailVersions
	m.versions = versions
	m.viewport.SetContent(RenderVersions(versions, path))
}

func (m *DetailModel) ShowLoading() {
	m.state = detailLoading
	m.viewport.SetContent("Loading...")
}

func (m *DetailModel) ShowError(err error) {
	m.state = detailError
	m.errMsg = err.Error()

	t := theme.Active
	errStyle := lipgloss.NewStyle().Foreground(t.Red)
	msgStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	m.viewport.SetContent(errStyle.Render("  Error") + "\n\n  " + msgStyle.Render(m.errMsg))
}

func (m *DetailModel) ShowDenied(path string) {
	m.state = detailError
	m.secret = nil

	t := theme.Active
	pathStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	iconStyle := lipgloss.NewStyle().Foreground(t.Red)
	titleStyle := lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	msgStyle := lipgloss.NewStyle().Foreground(t.Subtle)
	dividerStyle := lipgloss.NewStyle().Foreground(t.Overlay)

	content := "  " + pathStyle.Render(path) + "\n" +
		"  " + dividerStyle.Render(strings.Repeat("─", 38)) + "\n\n" +
		"  " + iconStyle.Render("") + titleStyle.Render(" Access Denied") + "\n\n" +
		"  " + msgStyle.Render("Your token does not have permission") + "\n" +
		"  " + msgStyle.Render("to read this secret.") + "\n\n" +
		"  " + msgStyle.Render("Check your Vault policies or contact") + "\n" +
		"  " + msgStyle.Render("your Vault administrator.")

	m.viewport.SetContent(content)
}

func (m *DetailModel) Clear() {
	m.state = detailEmpty
	m.secret = nil
	m.versions = nil
	m.viewport.SetContent("")
}

func (m DetailModel) Secret() *vault.SecretEntry {
	return m.secret
}

func (m DetailModel) SecretAsKeyValue() string {
	if m.secret == nil || len(m.secret.Data) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m.secret.Data))
	for k := range m.secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		b.WriteString(fmt.Sprintf("%s=%v", k, m.secret.Data[k]))
		if i < len(keys)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DetailModel) View() string {
	if m.state == detailEmpty {
		t := theme.Active
		hint := lipgloss.NewStyle().
			Foreground(t.Subtle).
			Italic(true)
		icon := lipgloss.NewStyle().
			Foreground(t.Overlay)

		content := icon.Render("  ◇") + "  " + hint.Render("Navigate the tree and select a secret")
		return content
	}
	return m.viewport.View()
}
