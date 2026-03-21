package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nqui/vault-tui/internal/tui/theme"
)

type AuthMethod int

const (
	AuthToken AuthMethod = iota
	AuthUserpass
	AuthLDAP
)

var authMethodLabels = []string{"Token", "Userpass", "LDAP"}

type LoginFormResult struct {
	Method   AuthMethod
	Addr     string
	Token    string
	Username string
	Password string
	Save     bool
	Cancel   bool
}

// login form field indices
const (
	fieldAddr = iota
	fieldMethod
	fieldFirst  // token input or username input
	fieldSecond // password input (userpass/ldap only)
	fieldSave
)

type LoginFormModel struct {
	method        AuthMethod
	addrInput     textinput.Model
	tokenInput    textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model
	save          bool
	focusedField  int
	errorMsg      string
	active        bool
	width         int
	height        int
}

func NewLoginForm() LoginFormModel {
	ai := textinput.New()
	ai.Prompt = "> "
	ai.Placeholder = "https://vault.example.com:8200"

	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "hvs.CAESI..."

	ui := textinput.New()
	ui.Prompt = "> "
	ui.Placeholder = "username"

	pi := textinput.New()
	pi.Prompt = "> "
	pi.Placeholder = "password"
	pi.EchoMode = textinput.EchoPassword

	return LoginFormModel{
		addrInput:     ai,
		tokenInput:    ti,
		usernameInput: ui,
		passwordInput: pi,
		focusedField:  fieldAddr,
	}
}

// SetAddr pre-fills the address field (e.g. from config).
func (m *LoginFormModel) SetAddr(addr string) {
	m.addrInput.SetValue(addr)
}

func (m *LoginFormModel) Show(errMsg string) tea.Cmd {
	m.active = true
	m.errorMsg = errMsg
	m.tokenInput.SetValue("")
	m.usernameInput.SetValue("")
	m.passwordInput.SetValue("")
	if m.addrInput.Value() == "" {
		m.focusedField = fieldAddr
	} else {
		m.focusedField = fieldFirst
	}
	return m.updateFocus()
}

func (m *LoginFormModel) Close() {
	m.active = false
	m.blurAll()
}

func (m LoginFormModel) Active() bool { return m.active }

func (m *LoginFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputW := min(w-16, 60)
	m.addrInput.SetWidth(inputW)
	m.tokenInput.SetWidth(inputW)
	m.usernameInput.SetWidth(inputW)
	m.passwordInput.SetWidth(inputW)
}

func (m *LoginFormModel) SetError(msg string) {
	m.errorMsg = msg
}

func (m LoginFormModel) Update(msg tea.Msg) (LoginFormModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	next := key.NewBinding(key.WithKeys("tab"))
	prev := key.NewBinding(key.WithKeys("shift+tab"))
	submit := key.NewBinding(key.WithKeys("enter"))
	cancel := key.NewBinding(key.WithKeys("esc", "ctrl+c"))
	left := key.NewBinding(key.WithKeys("left"))
	right := key.NewBinding(key.WithKeys("right"))
	space := key.NewBinding(key.WithKeys("space"))

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, cancel):
			m.active = false
			m.blurAll()
			return m, func() tea.Msg {
				return LoginFormResult{Cancel: true}
			}

		case key.Matches(msg, submit):
			m.errorMsg = ""
			result := LoginFormResult{
				Method: m.method,
				Addr:   strings.TrimSpace(m.addrInput.Value()),
				Save:   m.save,
			}
			if result.Addr == "" {
				m.errorMsg = "Vault address is required"
				return m, nil
			}
			if m.method == AuthToken {
				result.Token = m.tokenInput.Value()
				if result.Token == "" {
					m.errorMsg = "Token is required"
					return m, nil
				}
			} else {
				result.Username = m.usernameInput.Value()
				result.Password = m.passwordInput.Value()
				if result.Username == "" || result.Password == "" {
					m.errorMsg = "Username and password are required"
					return m, nil
				}
			}
			return m, func() tea.Msg { return result }

		case m.focusedField == fieldMethod && key.Matches(msg, left):
			if m.method > AuthToken {
				m.method--
				m.errorMsg = ""
			}
			return m, nil

		case m.focusedField == fieldMethod && key.Matches(msg, right):
			if m.method < AuthLDAP {
				m.method++
				m.errorMsg = ""
			}
			return m, nil

		case m.focusedField == fieldSave && key.Matches(msg, space):
			m.save = !m.save
			return m, nil

		case key.Matches(msg, next):
			m.focusedField++
			// skip fieldSecond for token method
			if m.method == AuthToken && m.focusedField == fieldSecond {
				m.focusedField++
			}
			if m.focusedField > fieldSave {
				m.focusedField = fieldAddr
			}
			return m, m.updateFocus()

		case key.Matches(msg, prev):
			m.focusedField--
			if m.method == AuthToken && m.focusedField == fieldSecond {
				m.focusedField--
			}
			if m.focusedField < fieldAddr {
				m.focusedField = fieldSave
			}
			return m, m.updateFocus()
		}
	}

	// Forward to focused text input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldAddr:
		m.addrInput, cmd = m.addrInput.Update(msg)
	case fieldFirst:
		if m.method == AuthToken {
			m.tokenInput, cmd = m.tokenInput.Update(msg)
		} else {
			m.usernameInput, cmd = m.usernameInput.Update(msg)
		}
	case fieldSecond:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m LoginFormModel) View() string {
	if !m.active {
		return ""
	}

	t := theme.Active

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.Text).
		Width(12)

	hintKeyStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Subtle)

	errorStyle := lipgloss.NewStyle().
		Foreground(t.Red).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(t.Primary).
		Foreground(t.Bg).
		PaddingLeft(1).
		PaddingRight(1)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(t.Subtle).
		PaddingLeft(1).
		PaddingRight(1)

	focusLabelStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Width(12)

	checkStyle := lipgloss.NewStyle().
		Foreground(t.Green)

	var b strings.Builder

	b.WriteString(titleStyle.Render("Login to Vault"))
	b.WriteString("\n\n")

	// Vault address
	lbl := labelStyle
	if m.focusedField == fieldAddr {
		lbl = focusLabelStyle
	}
	b.WriteString(lbl.Render("Address"))
	b.WriteString(m.addrInput.View())
	b.WriteString("\n\n")

	// Method selector
	lbl = labelStyle
	if m.focusedField == fieldMethod {
		lbl = focusLabelStyle
	}
	b.WriteString(lbl.Render("Method"))
	for i, label := range authMethodLabels {
		if AuthMethod(i) == m.method {
			b.WriteString(selectedStyle.Render(label))
		} else {
			b.WriteString(unselectedStyle.Render(label))
		}
	}
	b.WriteString("\n\n")

	// Input fields
	if m.method == AuthToken {
		lbl = labelStyle
		if m.focusedField == fieldFirst {
			lbl = focusLabelStyle
		}
		b.WriteString(lbl.Render("Token"))
		b.WriteString(m.tokenInput.View())
		b.WriteString("\n\n")
	} else {
		lbl = labelStyle
		if m.focusedField == fieldFirst {
			lbl = focusLabelStyle
		}
		b.WriteString(lbl.Render("Username"))
		b.WriteString(m.usernameInput.View())
		b.WriteString("\n\n")

		lbl = labelStyle
		if m.focusedField == fieldSecond {
			lbl = focusLabelStyle
		}
		b.WriteString(lbl.Render("Password"))
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n\n")
	}

	// Save toggle
	lbl = labelStyle
	if m.focusedField == fieldSave {
		lbl = focusLabelStyle
	}
	check := "[ ]"
	if m.save {
		check = checkStyle.Render("[x]")
	}
	b.WriteString(lbl.Render("Save"))
	b.WriteString(check + " Save token to config")
	b.WriteString("\n\n")

	// Error message
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render(m.errorMsg))
		b.WriteString("\n\n")
	}

	// Hints
	b.WriteString(
		hintKeyStyle.Render("tab") + hintStyle.Render(" next") + "   " +
			hintKeyStyle.Render("\u2190/\u2192") + hintStyle.Render(" method") + "   " +
			hintKeyStyle.Render("enter") + hintStyle.Render(" submit") + "   " +
			hintKeyStyle.Render("esc") + hintStyle.Render(" quit"),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(min(m.width-4, 70))

	box := boxStyle.Render(b.String())

	// Center vertically and horizontally
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *LoginFormModel) blurAll() {
	m.addrInput.Blur()
	m.tokenInput.Blur()
	m.usernameInput.Blur()
	m.passwordInput.Blur()
}

func (m *LoginFormModel) updateFocus() tea.Cmd {
	m.blurAll()
	switch m.focusedField {
	case fieldAddr:
		return m.addrInput.Focus()
	case fieldFirst:
		if m.method == AuthToken {
			return m.tokenInput.Focus()
		}
		return m.usernameInput.Focus()
	case fieldSecond:
		return m.passwordInput.Focus()
	}
	return nil
}
