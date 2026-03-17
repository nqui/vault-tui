package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/nqui/vault-tui/internal/config"
	"github.com/nqui/vault-tui/internal/tui/components"
	"github.com/nqui/vault-tui/internal/tui/theme"
	"github.com/nqui/vault-tui/internal/vault"
)

type pane int

const (
	paneTree pane = iota
	paneDetail
)

type appMode int

const (
	modeBrowse appMode = iota
	modeEdit
	modeConfirmDelete
	modeWrapTTL
	modeWrapResult
	modeUnwrap
)

type appView int

const (
	viewLogin appView = iota
	viewSecrets
	viewWrap
)

type App struct {
	vault     *vault.Client
	tree      components.TreeModel
	detail    components.DetailModel
	editor    components.EditorModel
	confirm   components.ConfirmModel
	statusbar components.StatusBarModel
	spinner   spinner.Model

	// Wrap/unwrap components
	prompt     components.PromptModel
	picker     components.PickerModel
	wrapResult components.WrapResultModel
	wrapView   components.WrapViewModel

	// Wrap state for async boundary
	wrapEngine string
	wrapPath   string
	wrapKVVer  int

	// Login
	loginForm components.LoginFormModel
	tokenInfo *vault.TokenInfo
	cfg       *config.Config

	activePane pane
	mode       appMode
	view       appView
	width      int
	height     int
	loading    int
	lastErr    error
	vaultAddr  string
}

func NewApp(client *vault.Client, cfg *config.Config) *App {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.Active.Primary)),
	)

	sb := components.NewStatusBar()
	sb.BgStyle = statusBarBg
	sb.KeyStyle = statusKeyStyle
	sb.DescStyle = statusDescStyle
	sb.MsgStyle = statusMsgStyle
	sb.ErrStyle = statusErrStyle

	startView := viewSecrets
	if !client.HasToken() {
		startView = viewLogin
	}

	return &App{
		vault:      client,
		tree:       components.NewTree(),
		detail:     components.NewDetail(),
		editor:     components.NewEditor(),
		confirm:    components.NewConfirm(),
		prompt:     components.NewPrompt(),
		picker:     components.NewPicker(),
		wrapResult: components.NewWrapResult(),
		wrapView:   components.NewWrapView(),
		loginForm:  components.NewLoginForm(),
		statusbar:  sb,
		spinner:    sp,
		cfg:        cfg,
		activePane: paneTree,
		mode:       modeBrowse,
		view:       startView,
		vaultAddr:  client.Addr(),
	}
}

func (m *App) Init() tea.Cmd {
	if m.view == viewLogin {
		return m.loginForm.Show("")
	}

	m.tree.SetFocused(true)
	return tea.Batch(
		m.spinner.Tick,
		m.validateToken(),
	)
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case ClearErrorMsg:
		m.statusbar.ClearError()
		m.lastErr = nil

	case TokenValidatedMsg:
		if msg.Err != nil {
			m.view = viewLogin
			return m, m.loginForm.Show("Token invalid or expired")
		}
		m.tokenInfo = msg.Info
		cmds = append(cmds, m.loadEngines())
		if msg.Info.Renewable && msg.Info.TTL > 0 {
			cmds = append(cmds, m.scheduleRenewal(msg.Info.TTL))
		}
		return m, tea.Batch(cmds...)

	case components.LoginFormResult:
		if msg.Cancel {
			return m, tea.Quit
		}
		m.loading++
		return m, m.performLogin(msg)

	case LoginCompleteMsg:
		m.loading--
		if msg.Err != nil {
			m.loginForm.SetError(msg.Err.Error())
			return m, nil
		}
		m.tokenInfo = msg.Info
		m.vaultAddr = m.vault.Addr()
		if msg.Save {
			m.cfg.Token = msg.Info.Token
			if err := config.Save(m.cfg); err != nil {
				m.setError(fmt.Errorf("failed to save config: %w", err))
			}
		}
		m.loginForm.Close()
		m.view = viewSecrets
		m.tree.SetFocused(true)
		cmds = append(cmds, m.spinner.Tick, m.loadEngines())
		if msg.Info.Renewable && msg.Info.TTL > 0 {
			cmds = append(cmds, m.scheduleRenewal(msg.Info.TTL))
		}
		return m, tea.Batch(cmds...)

	case TokenRenewTickMsg:
		return m, m.renewToken()

	case TokenRenewedMsg:
		if msg.Err != nil {
			if errors.Is(msg.Err, vault.ErrPermissionDenied) {
				m.view = viewLogin
				return m, m.loginForm.Show("Session expired — please re-authenticate")
			}
			// Non-fatal: log but don't disrupt
			return m, nil
		}
		m.tokenInfo = msg.Info
		if msg.Info.Renewable && msg.Info.TTL > 0 {
			return m, m.scheduleRenewal(msg.Info.TTL)
		}
		return m, nil

	case EnginesLoadedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			return m, nil
		}
		roots := make([]*components.TreeNode, 0, len(msg.Engines))
		sort.Slice(msg.Engines, func(i, j int) bool {
			return msg.Engines[i].Path < msg.Engines[j].Path
		})
		for _, e := range msg.Engines {
			if e.Type == "kv" || e.Type == "generic" {
				roots = append(roots, components.NewEngineNode(e.Path, e.Type, e.Version))
			}
		}
		m.tree.SetRoots(roots)
		return m, nil

	case PathListedMsg:
		m.loading--
		if msg.Err != nil {
			if errors.Is(msg.Err, vault.ErrPermissionDenied) {
				m.tree.SetNodeError(msg.NodeID, components.NodeErrDenied)
			} else if errors.Is(msg.Err, vault.ErrNotFound) {
				m.tree.SetNodeError(msg.NodeID, components.NodeErrNotFound)
			} else {
				m.setError(msg.Err)
				m.tree.CollapseNode(msg.NodeID)
			}
			return m, nil
		}
		children := make([]*components.TreeNode, 0, len(msg.Entries))
		node := m.tree.Selected()
		if node == nil || node.ID != msg.NodeID {
			for _, root := range m.tree.Roots {
				if n := findNodeByID(root, msg.NodeID); n != nil {
					node = n
					break
				}
			}
		}
		if node != nil {
			for _, e := range msg.Entries {
				children = append(children, components.NewChildNode(node, e.Name, e.IsDir))
			}
		}
		m.tree.ExpandNode(msg.NodeID, children)
		return m, nil

	case SecretLoadedMsg:
		m.loading--
		if msg.Err != nil {
			if errors.Is(msg.Err, vault.ErrPermissionDenied) {
				m.detail.ShowDenied(msg.Path)
			} else {
				m.setError(msg.Err)
				m.detail.ShowError(msg.Err)
			}
			return m, nil
		}
		m.detail.ShowSecret(msg.Secret)
		return m, nil

	case SecretSavedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			return m, nil
		}
		m.editor.Close()
		m.mode = modeBrowse
		m.statusbar.SetMessage("Secret saved: " + msg.Path)
		cmds = append(cmds, m.clearErrorAfter(3*time.Second))
		node := m.tree.Selected()
		if node != nil {
			parentNode := node
			if !node.IsDir {
				parentNode = node.Parent
			}
			if parentNode != nil {
				cmds = append(cmds, m.listPath(parentNode))
			}
		}
		return m, tea.Batch(cmds...)

	case SecretDeletedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			return m, nil
		}
		m.detail.Clear()
		m.statusbar.SetMessage("Secret deleted: " + msg.Path)
		cmds = append(cmds, m.clearErrorAfter(3*time.Second))
		node := m.tree.Selected()
		if node != nil && !node.IsDir {
			m.tree.RemoveLeaf(node.ID)
		}
		return m, tea.Batch(cmds...)

	case VersionsLoadedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			return m, nil
		}
		m.detail.ShowVersions(msg.Versions, msg.Path)
		return m, nil

	case SecretWrappedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			m.mode = modeBrowse
			return m, m.clearErrorAfter(5 * time.Second)
		}
		m.mode = modeWrapResult
		m.wrapResult.SetWidth(m.width * 2 / 3)
		m.wrapResult.Show(msg.Token, msg.TTL, msg.Path)
		return m, nil

	case SecretUnwrappedMsg:
		m.loading--
		if msg.Err != nil {
			m.setError(msg.Err)
			if m.view == viewWrap {
				m.wrapView.SetUnwrapResult("Error: "+msg.Err.Error(), "")
			}
			m.mode = modeBrowse
			return m, m.clearErrorAfter(5 * time.Second)
		}
		if m.view == viewWrap {
			m.wrapView.SetUnwrapResult(formatKVDisplay(msg.Data), formatKVCopy(msg.Data))
			m.mode = modeBrowse
			return m, nil
		}
		// Quick unwrap: show in detail pane
		m.prompt.Close()
		entry := &vault.SecretEntry{
			Path: "(unwrapped)",
			Data: msg.Data,
		}
		m.detail.ShowSecret(entry)
		m.mode = modeBrowse
		m.statusbar.SetMessage("Token unwrapped successfully")
		return m, m.clearErrorAfter(3 * time.Second)

	case components.ConfirmResult:
		if msg.Confirmed {
			node := m.tree.Selected()
			if node != nil && !node.IsDir {
				m.mode = modeBrowse
				return m, m.deleteSecret(node)
			}
		}
		m.mode = modeBrowse
		return m, nil

	case components.PickerResult:
		m.mode = modeBrowse
		if msg.Cancelled {
			return m, nil
		}
		if msg.Context == "wrap_ttl" {
			return m, m.wrapSecret(m.wrapEngine, m.wrapPath, m.wrapKVVer, msg.Value)
		}
		return m, nil

	case components.PromptResult:
		if msg.Cancelled {
			m.mode = modeBrowse
			return m, nil
		}
		switch msg.Context {
		case "unwrap_token":
			token := strings.TrimSpace(msg.Value)
			if token == "" {
				m.mode = modeBrowse
				return m, nil
			}
			m.mode = modeBrowse
			return m, m.unwrapToken(token)
		}
		m.mode = modeBrowse
		return m, nil

	case components.WrapResultDismissed:
		m.mode = modeBrowse
		if msg.CopyToken {
			if err := clipboard.WriteAll(msg.Token); err != nil {
				m.statusbar.SetError(fmt.Errorf("copy failed: %v", err))
			} else {
				m.statusbar.SetMessage("Wrapping token copied to clipboard")
			}
			return m, m.clearErrorAfter(3 * time.Second)
		}
		return m, nil

	case components.WrapDataMsg:
		return m, m.wrapData(msg.Data, msg.TTL)

	case components.UnwrapTokenMsg:
		return m, m.unwrapToken(msg.Token)

	case components.WrapViewCopyMsg:
		if err := clipboard.WriteAll(msg.Content); err != nil {
			m.statusbar.SetError(fmt.Errorf("copy failed: %v", err))
		} else {
			m.statusbar.SetMessage("Unwrapped data copied to clipboard")
		}
		return m, m.clearErrorAfter(3 * time.Second)

	case components.WrapViewDuplicateKeyMsg:
		m.statusbar.SetError(fmt.Errorf("duplicate key: %q", msg.Key))
		return m, m.clearErrorAfter(3 * time.Second)

	case tea.KeyPressMsg:
		// Login view handles all its own keys
		if m.view == viewLogin {
			var cmd tea.Cmd
			m.loginForm, cmd = m.loginForm.Update(msg)
			return m, cmd
		}

		// Modal modes first
		switch m.mode {
		case modeEdit:
			return m.updateEditor(msg)
		case modeConfirmDelete:
			return m.updateConfirm(msg)
		case modeWrapTTL:
			return m.updatePicker(msg)
		case modeUnwrap:
			return m.updatePrompt(msg)
		case modeWrapResult:
			return m.updateWrapResult(msg)
		}

		// Global keys
		switch {
		case key.Matches(msg, keys.Quit):
			if m.view != viewWrap {
				return m, tea.Quit
			}
		case key.Matches(msg, keys.WrapView):
			if m.view == viewWrap {
				m.view = viewSecrets
				m.statusbar.SetHints(components.SecretsViewHints)
			} else {
				m.view = viewWrap
				m.statusbar.SetHints(components.WrapViewHints)
				m.wrapView.SetSize(m.width, m.height)
				cmds = append(cmds, m.wrapView.Open())
			}
			return m, tea.Batch(cmds...)
		}

		// View-specific handling
		if m.view == viewWrap {
			return m.updateWrapView(msg)
		}

		// Secrets view keys
		switch {
		case key.Matches(msg, keys.SwitchPane):
			m.switchPane()
			return m, nil
		case key.Matches(msg, keys.Help):
			return m, nil
		case key.Matches(msg, keys.Copy):
			return m, m.copySecret()
		case key.Matches(msg, keys.Wrap):
			return m.startQuickWrap()
		case key.Matches(msg, keys.Unwrap):
			return m.startQuickUnwrap()
		}

		if m.activePane == paneTree {
			return m.updateTree(msg)
		}
		return m.updateDetail(msg)
	}

	// Forward non-key messages to active components
	var cmd tea.Cmd
	if m.view == viewLogin {
		m.loginForm, cmd = m.loginForm.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	if m.view == viewWrap {
		m.wrapView, cmd = m.wrapView.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.mode == modeUnwrap {
		m.prompt, cmd = m.prompt.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.detail, cmd = m.detail.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *App) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading...")
	}

	if m.view == viewLogin {
		v := tea.NewView(m.loginForm.View())
		v.AltScreen = true
		return v
	}

	// Title bar with view indicator
	viewLabel := " Secrets "
	if m.view == viewWrap {
		viewLabel = " Wrap/Unwrap "
	}
	title := titleBarStyle.Render(" hv-tui ")
	viewTag := titleBarInfoStyle.Render(viewLabel)
	addr := titleBarInfoStyle.Render(m.vaultAddr)
	titlePad := m.width - lipgloss.Width(title) - lipgloss.Width(viewTag) - lipgloss.Width(addr)
	if titlePad < 0 {
		titlePad = 0
	}
	titleBar := title + viewTag + titleBarInfoStyle.Width(titlePad).Render("") + addr

	var content string

	switch m.mode {
	case modeEdit:
		content = m.renderEditorOverlay()
	case modeConfirmDelete:
		content = m.renderConfirmOverlay()
	case modeWrapTTL:
		content = m.renderPickerOverlay()
	case modeUnwrap:
		content = m.renderPromptOverlay()
	case modeWrapResult:
		content = m.renderWrapResultOverlay()
	default:
		if m.view == viewWrap {
			content = m.wrapView.View()
		} else {
			content = m.renderBrowse()
		}
	}

	statusBar := m.statusbar.View()

	full := lipgloss.JoinVertical(lipgloss.Left, titleBar, content, statusBar)

	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

func (m *App) renderBrowse() string {
	treeWidth := m.treeWidth()
	detailWidth := m.width - treeWidth - 4
	contentHeight := m.contentHeight()
	treeLabel := " Secrets"
	detailLabel := " Details"
	if m.activePane == paneTree {
		treeLabel = paneHeaderStyle.Render(treeLabel)
		detailLabel = lipgloss.NewStyle().Foreground(theme.Active.Subtle).Render(detailLabel)
	} else {
		treeLabel = lipgloss.NewStyle().Foreground(theme.Active.Subtle).Render(treeLabel)
		detailLabel = paneHeaderStyle.Render(detailLabel)
	}

	treeBorder := unfocusedBorder
	if m.activePane == paneTree {
		treeBorder = focusedBorder
	}
	treeInner := treeLabel + "\n" + m.tree.View()
	treePane := treeBorder.
		Width(treeWidth).
		Height(contentHeight).
		Render(treeInner)

	detailBorder := unfocusedBorder
	if m.activePane == paneDetail {
		detailBorder = focusedBorder
	}
	detailContent := m.detail.View()
	if m.loading > 0 && m.detail.Secret() == nil {
		detailContent = m.spinner.View() + "  " + detailContent
	}
	detailInner := detailLabel + "\n" + detailContent
	detailPane := detailBorder.
		Width(detailWidth).
		Height(contentHeight).
		Render(detailInner)

	return lipgloss.JoinHorizontal(lipgloss.Top, treePane, detailPane)
}

func (m *App) renderEditorOverlay() string {
	return m.editor.View()
}

func (m *App) renderConfirmOverlay() string {
	browse := m.renderBrowse()
	confirm := m.confirm.View()
	return lipgloss.JoinVertical(lipgloss.Left, browse, confirm)
}

func (m *App) renderPickerOverlay() string {
	contentHeight := m.contentHeight()
	picker := m.picker.View()
	return lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, picker)
}

func (m *App) renderPromptOverlay() string {
	contentHeight := m.contentHeight()
	prompt := m.prompt.View()
	return lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, prompt)
}

func (m *App) renderWrapResultOverlay() string {
	contentHeight := m.contentHeight()
	result := m.wrapResult.View()
	return lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, result)
}

func (m *App) startQuickWrap() (tea.Model, tea.Cmd) {
	node := m.tree.Selected()
	if node == nil || node.IsDir {
		m.statusbar.SetMessage("Select a secret to wrap")
		return m, m.clearErrorAfter(3 * time.Second)
	}
	secret := m.detail.Secret()
	if secret == nil || secret.Path != node.Engine+node.FullPath {
		m.statusbar.SetMessage("Load a secret first (press enter)")
		return m, m.clearErrorAfter(3 * time.Second)
	}
	m.wrapEngine = node.Engine
	m.wrapPath = strings.TrimSuffix(node.FullPath, "/")
	m.wrapKVVer = node.KVVer
	m.mode = modeWrapTTL
	m.picker.SetWidth(m.width / 3)
	m.picker.Show("Wrap TTL", []string{"5m", "15m", "30m", "1h", "6h", "24h"}, "wrap_ttl")
	return m, nil
}

func (m *App) startQuickUnwrap() (tea.Model, tea.Cmd) {
	m.mode = modeUnwrap
	m.prompt.SetWidth(m.width / 2)
	cmd := m.prompt.Show("Unwrap Token", "hvs.CAESI...", "unwrap_token")
	return m, cmd
}

func (m *App) updateTree(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, keys.Open):
		node := m.tree.Selected()
		if node == nil {
			break
		}
		if node.IsDir {
			if node.State == components.NodeExpanded {
				m.tree.CollapseNode(node.ID)
			} else if node.State == components.NodeCollapsed || node.State == components.NodeError {
				m.tree.SetNodeLoading(node.ID)
				cmds = append(cmds, m.listPath(node))
			}
		} else {
			m.detail.ShowLoading()
			cmds = append(cmds, m.loadSecret(node))
		}
		return m, tea.Batch(cmds...)

	case key.Matches(msg, keys.Back):
		node := m.tree.Selected()
		if node != nil && node.IsDir && (node.State == components.NodeExpanded || node.State == components.NodeError) {
			m.tree.CollapseNode(node.ID)
			return m, nil
		}

	case key.Matches(msg, keys.Refresh):
		node := m.tree.Selected()
		if node != nil && node.IsDir {
			m.tree.SetNodeLoading(node.ID)
			return m, m.listPath(node)
		}

	case key.Matches(msg, keys.New):
		node := m.tree.Selected()
		if node == nil {
			break
		}
		dirNode := node
		if !node.IsDir {
			dirNode = node.Parent
		}
		if dirNode != nil {
			m.mode = modeEdit
			m.editor.SetSize(m.width-4, m.height-6)
			return m, m.editor.OpenCreate(dirNode.Engine, dirNode.FullPath, dirNode.KVVer)
		}

	case key.Matches(msg, keys.Delete):
		node := m.tree.Selected()
		if node != nil && !node.IsDir {
			m.mode = modeConfirmDelete
			m.confirm.SetWidth(m.width / 2)
			m.confirm.Show(
				fmt.Sprintf("Delete secret %q?", node.FullPath),
				node.ID,
			)
			return m, nil
		}

	case key.Matches(msg, keys.Edit):
		node := m.tree.Selected()
		if node != nil && !node.IsDir {
			secret := m.detail.Secret()
			if secret != nil && secret.Path == node.Engine+node.FullPath {
				m.mode = modeEdit
				m.editor.SetSize(m.width-4, m.height-6)
				return m, m.editor.OpenEdit(node.Engine, node.FullPath, node.KVVer, secret.Data)
			}
			m.detail.ShowLoading()
			return m, m.loadSecret(node)
		}
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	cmds = append(cmds, cmd)

	node := m.tree.Selected()
	if node != nil && !node.IsDir {
		m.detail.ShowLoading()
		cmds = append(cmds, m.loadSecret(node))
	}

	return m, tea.Batch(cmds...)
}

func (m *App) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Edit):
		node := m.tree.Selected()
		secret := m.detail.Secret()
		if node != nil && !node.IsDir && secret != nil {
			m.mode = modeEdit
			m.editor.SetSize(m.width-4, m.height-6)
			return m, m.editor.OpenEdit(node.Engine, node.FullPath, node.KVVer, secret.Data)
		}

	case key.Matches(msg, keys.Versions):
		node := m.tree.Selected()
		if node != nil && !node.IsDir && node.KVVer == 2 {
			return m, m.loadVersions(node)
		}
	}

	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

func (m *App) updateEditor(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	ctrlC := key.NewBinding(key.WithKeys("ctrl+c"))
	if key.Matches(msg, m.editor.Cancel) || key.Matches(msg, ctrlC) {
		m.editor.Close()
		m.mode = modeBrowse
		return m, nil
	}

	if key.Matches(msg, m.editor.Save) {
		data := m.editor.Data()
		if len(data) == 0 {
			m.statusbar.SetError(fmt.Errorf("no data to save"))
			return m, m.clearErrorAfter(3 * time.Second)
		}
		path := m.editor.Path()
		if path == "" {
			m.statusbar.SetError(fmt.Errorf("path is required"))
			return m, m.clearErrorAfter(3 * time.Second)
		}
		path = strings.TrimSuffix(path, "/")
		return m, m.saveSecret(m.editor.Engine(), path, m.editor.KVVersion(), data)
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m *App) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m *App) updatePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m *App) updatePicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m *App) updateWrapResult(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.wrapResult, cmd = m.wrapResult.Update(msg)
	return m, cmd
}

func (m *App) updateWrapView(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Esc or ctrl+c returns to secrets view
	ctrlC := key.NewBinding(key.WithKeys("ctrl+c"))
	if key.Matches(msg, keys.Cancel) || key.Matches(msg, ctrlC) {
		m.view = viewSecrets
		m.statusbar.SetHints(components.SecretsViewHints)
		return m, nil
	}

	var cmd tea.Cmd
	m.wrapView, cmd = m.wrapView.Update(msg)
	return m, cmd
}

func (m *App) switchPane() {
	if m.activePane == paneTree {
		m.activePane = paneDetail
		m.tree.SetFocused(false)
		m.detail.SetFocused(true)
	} else {
		m.activePane = paneTree
		m.tree.SetFocused(true)
		m.detail.SetFocused(false)
	}
}

func (m *App) treeWidth() int {
	minW := m.width * 15 / 100
	maxW := m.width * 40 / 100
	w := m.tree.MaxVisibleWidth() + 4
	if w < minW {
		w = minW
	}
	if w > maxW {
		w = maxW
	}
	return w
}

func (m *App) contentHeight() int {
	// title bar (1) + status bar (variable) + border padding (2)
	return m.height - m.statusbar.Height() - 3
}

func (m *App) updateSizes() {
	tw := m.treeWidth()
	detailWidth := m.width - tw - 6
	contentHeight := m.contentHeight() - 4

	m.tree.SetSize(tw, contentHeight)
	m.detail.SetSize(detailWidth, contentHeight)
	m.statusbar.SetWidth(m.width)
	m.editor.SetSize(m.width-4, m.height-6)
	m.confirm.SetWidth(m.width / 2)
	m.prompt.SetWidth(m.width / 2)
	m.picker.SetWidth(m.width / 3)
	m.wrapResult.SetWidth(m.width * 2 / 3)
	m.wrapView.SetSize(m.width, m.height)
	m.loginForm.SetSize(m.width, m.height)
}

func (m *App) setError(err error) {
	m.lastErr = err
	m.statusbar.SetError(err)
}

func (m *App) loadEngines() tea.Cmd {
	m.loading++
	client := m.vault
	return func() tea.Msg {
		engines, err := client.ListEngines(context.Background())
		return EnginesLoadedMsg{Engines: engines, Err: err}
	}
}

func (m *App) listPath(node *components.TreeNode) tea.Cmd {
	m.loading++
	client := m.vault
	nodeID := node.ID
	engine := node.Engine
	fullPath := node.FullPath
	kvVer := node.KVVer
	return func() tea.Msg {
		entries, err := client.ListPath(context.Background(), engine, fullPath, kvVer)
		return PathListedMsg{NodeID: nodeID, Entries: entries, Err: err}
	}
}

func (m *App) loadSecret(node *components.TreeNode) tea.Cmd {
	m.loading++
	client := m.vault
	engine := node.Engine
	path := strings.TrimSuffix(node.FullPath, "/")
	kvVer := node.KVVer
	return func() tea.Msg {
		secret, err := client.GetSecret(context.Background(), engine, path, kvVer)
		return SecretLoadedMsg{Path: engine + path, Secret: secret, Err: err}
	}
}

func (m *App) saveSecret(engine, path string, kvVersion int, data map[string]interface{}) tea.Cmd {
	m.loading++
	client := m.vault
	return func() tea.Msg {
		err := client.PutSecret(context.Background(), engine, path, kvVersion, data)
		return SecretSavedMsg{Path: engine + path, Err: err}
	}
}

func (m *App) deleteSecret(node *components.TreeNode) tea.Cmd {
	m.loading++
	client := m.vault
	engine := node.Engine
	path := strings.TrimSuffix(node.FullPath, "/")
	kvVer := node.KVVer
	return func() tea.Msg {
		err := client.DeleteSecret(context.Background(), engine, path, kvVer)
		return SecretDeletedMsg{Path: engine + path, Err: err}
	}
}

func (m *App) loadVersions(node *components.TreeNode) tea.Cmd {
	m.loading++
	client := m.vault
	engine := node.Engine
	path := strings.TrimSuffix(node.FullPath, "/")
	return func() tea.Msg {
		versions, err := client.GetVersions(context.Background(), engine, path)
		return VersionsLoadedMsg{Path: engine + path, Versions: versions, Err: err}
	}
}

func (m *App) wrapSecret(engine, path string, kvVersion int, ttl string) tea.Cmd {
	m.loading++
	client := m.vault
	return func() tea.Msg {
		result, err := client.WrapSecret(context.Background(), engine, path, kvVersion, ttl)
		if err != nil {
			return SecretWrappedMsg{Path: engine + path, Err: err}
		}
		return SecretWrappedMsg{
			Path:  engine + path,
			Token: result.Token,
			TTL:   result.TTL,
		}
	}
}

func (m *App) wrapData(data map[string]interface{}, ttl string) tea.Cmd {
	m.loading++
	client := m.vault
	return func() tea.Msg {
		result, err := client.WrapData(context.Background(), data, ttl)
		if err != nil {
			return SecretWrappedMsg{Err: err}
		}
		return SecretWrappedMsg{
			Token: result.Token,
			TTL:   result.TTL,
		}
	}
}

func (m *App) unwrapToken(token string) tea.Cmd {
	m.loading++
	client := m.vault
	return func() tea.Msg {
		data, err := client.UnwrapToken(context.Background(), token)
		return SecretUnwrappedMsg{Data: data, Err: err}
	}
}

func (m *App) copySecret() tea.Cmd {
	text := m.detail.SecretAsKeyValue()
	if text == "" {
		m.statusbar.SetMessage("No secret to copy")
		return m.clearErrorAfter(3 * time.Second)
	}
	if err := clipboard.WriteAll(text); err != nil {
		m.statusbar.SetError(fmt.Errorf("copy failed: %v", err))
		return m.clearErrorAfter(3 * time.Second)
	}
	m.statusbar.SetMessage("Copied to clipboard")
	return m.clearErrorAfter(3 * time.Second)
}

func (m *App) validateToken() tea.Cmd {
	client := m.vault
	return func() tea.Msg {
		info, err := client.ValidateToken(context.Background())
		return TokenValidatedMsg{Info: info, Err: err}
	}
}

func (m *App) performLogin(result components.LoginFormResult) tea.Cmd {
	client := m.vault
	save := result.Save
	return func() tea.Msg {
		ctx := context.Background()
		var info *vault.TokenInfo
		var err error

		switch result.Method {
		case components.AuthToken:
			info, err = client.LoginToken(ctx, result.Token)
		case components.AuthUserpass:
			info, err = client.LoginUserpass(ctx, result.Username, result.Password)
		case components.AuthLDAP:
			info, err = client.LoginLDAP(ctx, result.Username, result.Password)
		}

		return LoginCompleteMsg{Info: info, Save: save, Err: err}
	}
}

func (m *App) renewToken() tea.Cmd {
	client := m.vault
	return func() tea.Msg {
		info, err := client.RenewToken(context.Background())
		return TokenRenewedMsg{Info: info, Err: err}
	}
}

func (m *App) scheduleRenewal(ttl time.Duration) tea.Cmd {
	interval := ttl / 2
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return TokenRenewTickMsg{}
	})
}

func (m *App) clearErrorAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearErrorMsg{}
	})
}

func findNodeByID(node *components.TreeNode, id string) *components.TreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if n := findNodeByID(child, id); n != nil {
			return n
		}
	}
	return nil
}

func formatKVDisplay(data map[string]interface{}) string {
	if len(data) == 0 {
		return "(empty)"
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Active.Yellow)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Active.Text)
	divStyle := lipgloss.NewStyle().Foreground(theme.Active.Overlay)

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	var b strings.Builder
	for _, k := range keys {
		v := fmt.Sprintf("%v", data[k])
		padding := strings.Repeat(" ", maxKeyLen-len(k))
		b.WriteString("  ")
		b.WriteString(keyStyle.Render(k))
		b.WriteString(padding)
		b.WriteString(divStyle.Render("  │ "))
		b.WriteString(valueStyle.Render(v))
		b.WriteString("\n")
	}
	return b.String()
}

func formatKVCopy(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	var b strings.Builder
	for k, v := range data {
		b.WriteString(fmt.Sprintf("%s=%v\n", k, v))
	}
	return strings.TrimRight(b.String(), "\n")
}
