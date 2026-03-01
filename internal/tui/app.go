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
	"github.com/nq/hv-tui/internal/tui/components"
	"github.com/nq/hv-tui/internal/vault"
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
)

type App struct {
	vault     *vault.Client
	tree      components.TreeModel
	detail    components.DetailModel
	editor    components.EditorModel
	confirm   components.ConfirmModel
	statusbar components.StatusBarModel
	spinner   spinner.Model

	activePane pane
	mode       appMode
	width      int
	height     int
	loading    int
	lastErr    error
	vaultAddr  string
}

func NewApp(client *vault.Client) *App {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorPurple)),
	)

	sb := components.NewStatusBar()
	sb.BgStyle = statusBarBg
	sb.KeyStyle = statusKeyStyle
	sb.DescStyle = statusDescStyle
	sb.MsgStyle = statusMsgStyle
	sb.ErrStyle = statusErrStyle

	return &App{
		vault:      client,
		tree:       components.NewTree(),
		detail:     components.NewDetail(),
		editor:     components.NewEditor(),
		confirm:    components.NewConfirm(),
		statusbar:  sb,
		spinner:    sp,
		activePane: paneTree,
		mode:       modeBrowse,
		vaultAddr:  client.Addr(),
	}
}

func (m *App) Init() tea.Cmd {
	m.tree.SetFocused(true)
	return tea.Batch(
		m.spinner.Tick,
		m.loadEngines(),
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

	case tea.KeyPressMsg:
		switch m.mode {
		case modeEdit:
			return m.updateEditor(msg)
		case modeConfirmDelete:
			return m.updateConfirm(msg)
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.SwitchPane):
			m.switchPane()
			return m, nil
		case key.Matches(msg, keys.Help):
			// TODO: help overlay
			return m, nil
		case key.Matches(msg, keys.Copy):
			return m, m.copySecret()
		}

		if m.activePane == paneTree {
			return m.updateTree(msg)
		}
		return m.updateDetail(msg)
	}

	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *App) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading...")
	}

	title := titleBarStyle.Render(" hv-tui ")
	addr := titleBarInfoStyle.Render(m.vaultAddr)
	titlePad := m.width - lipgloss.Width(title) - lipgloss.Width(addr)
	if titlePad < 0 {
		titlePad = 0
	}
	titleBar := title + titleBarInfoStyle.Width(titlePad).Render("") + addr

	var content string

	switch m.mode {
	case modeEdit:
		content = m.renderEditorOverlay()
	case modeConfirmDelete:
		content = m.renderConfirmOverlay()
	default:
		content = m.renderBrowse()
	}

	statusBar := m.statusbar.View()

	full := lipgloss.JoinVertical(lipgloss.Left, titleBar, content, statusBar)

	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

func (m *App) renderBrowse() string {
	treeWidth := m.width*35/100 - 2
	detailWidth := m.width - treeWidth - 4
	contentHeight := m.height - 4
	treeLabel := " Secrets"
	detailLabel := " Details"
	if m.activePane == paneTree {
		treeLabel = paneHeaderStyle.Render(treeLabel)
		detailLabel = lipgloss.NewStyle().Foreground(colorSubtle).Render(detailLabel)
	} else {
		treeLabel = lipgloss.NewStyle().Foreground(colorSubtle).Render(treeLabel)
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
				// Allow retry on errored nodes
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
	if key.Matches(msg, m.editor.Cancel) {
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

func (m *App) updateSizes() {
	treeWidth := m.width*35/100 - 2
	detailWidth := m.width - treeWidth - 6
	contentHeight := m.height - 8

	m.tree.SetSize(treeWidth, contentHeight)
	m.detail.SetSize(detailWidth, contentHeight)
	m.statusbar.SetWidth(m.width)
	m.editor.SetSize(m.width-4, m.height-6)
	m.confirm.SetWidth(m.width / 2)
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
