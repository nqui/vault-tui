package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TreeKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Open   key.Binding
	Back   key.Binding
	Top    key.Binding
	Bottom key.Binding
}

type TreeStyles struct {
	Cursor      lipgloss.Style
	Directory   lipgloss.Style
	File        lipgloss.Style
	Loading     lipgloss.Style
	Engine      lipgloss.Style
	Dim         lipgloss.Style
	Denied      lipgloss.Style
	DeniedBadge lipgloss.Style
}

func DefaultTreeStyles() TreeStyles {
	return TreeStyles{
		Cursor: lipgloss.NewStyle().
			Background(lipgloss.Color("#3B4261")).
			Foreground(lipgloss.Color("#E0E0FF")).
			Bold(true),
		Directory: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DCFFF")).
			Bold(true),
		File: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0CAF5")),
		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89")).
			Italic(true),
		Engine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BB9AF7")).
			Bold(true),
		Dim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#414868")),
		Denied: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89")),
		DeniedBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F7768E")),
	}
}

type TreeModel struct {
	Roots    []*TreeNode
	flatList []*TreeNode
	cursor   int
	offset   int
	height   int
	width    int
	focused  bool
	KeyMap   TreeKeyMap
	Styles   TreeStyles
}

func NewTree() TreeModel {
	return TreeModel{
		KeyMap: TreeKeyMap{
			Up:     key.NewBinding(key.WithKeys("k", "up")),
			Down:   key.NewBinding(key.WithKeys("j", "down")),
			Open:   key.NewBinding(key.WithKeys("enter", "l", "right")),
			Back:   key.NewBinding(key.WithKeys("h", "left")),
			Top:    key.NewBinding(key.WithKeys("g")),
			Bottom: key.NewBinding(key.WithKeys("G")),
		},
		Styles: DefaultTreeStyles(),
	}
}

func (m *TreeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *TreeModel) SetFocused(f bool) {
	m.focused = f
}

func (m TreeModel) Focused() bool {
	return m.focused
}

func (m TreeModel) Selected() *TreeNode {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		return m.flatList[m.cursor]
	}
	return nil
}

func (m *TreeModel) SetRoots(roots []*TreeNode) {
	m.Roots = roots
	m.Flatten()
}

func (m *TreeModel) ExpandNode(nodeID string, children []*TreeNode) {
	node := m.findNode(nodeID)
	if node == nil {
		return
	}
	node.State = NodeExpanded
	node.Children = children
	m.Flatten()
}

func (m *TreeModel) SetNodeLoading(nodeID string) {
	node := m.findNode(nodeID)
	if node != nil {
		node.State = NodeLoading
	}
}

func (m *TreeModel) SetNodeError(nodeID string, kind NodeErrorKind) {
	node := m.findNode(nodeID)
	if node != nil {
		node.State = NodeError
		node.ErrKind = kind
		m.Flatten()
	}
}

func (m *TreeModel) CollapseNode(nodeID string) {
	node := m.findNode(nodeID)
	if node != nil {
		node.State = NodeCollapsed
		m.Flatten()
	}
}

func (m *TreeModel) RemoveLeaf(nodeID string) {
	node := m.findNode(nodeID)
	if node == nil || node.Parent == nil {
		return
	}
	parent := node.Parent
	for i, child := range parent.Children {
		if child.ID == nodeID {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}
	m.Flatten()
	if m.cursor >= len(m.flatList) {
		m.cursor = len(m.flatList) - 1
	}
}

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Up):
			m.moveUp()
		case key.Matches(msg, m.KeyMap.Down):
			m.moveDown()
		case key.Matches(msg, m.KeyMap.Top):
			m.cursor = 0
			m.offset = 0
		case key.Matches(msg, m.KeyMap.Bottom):
			m.cursor = len(m.flatList) - 1
			m.fixOffset()
		}
	}

	return m, nil
}

func (m TreeModel) View() string {
	if len(m.flatList) == 0 {
		return m.Styles.Loading.Render("  Connecting to Vault...")
	}

	var b strings.Builder
	end := m.offset + m.height
	if end > len(m.flatList) {
		end = len(m.flatList)
	}

	for i := m.offset; i < end; i++ {
		node := m.flatList[i]
		line := m.renderNode(node, i == m.cursor)
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (m TreeModel) renderNode(node *TreeNode, selected bool) string {
	var prefix string
	if node.Depth > 0 {
		prefix = m.buildGuide(node)
	}

	var icon string
	switch {
	case node.Depth == 0:
		if node.State == NodeExpanded {
			icon = " "
		} else if node.State == NodeLoading {
			icon = "◌ "
		} else {
			icon = " "
		}
	case node.State == NodeError:
		icon = " "
	case node.State == NodeLoading:
		icon = "◌ "
	case node.IsDir && node.State == NodeExpanded:
		icon = " "
	case node.IsDir:
		icon = " "
	default:
		icon = "  "
	}

	name := node.Name
	if node.IsDir {
		name = strings.TrimSuffix(name, "/")
	}

	var badge string
	if node.State == NodeError {
		switch node.ErrKind {
		case NodeErrDenied:
			badge = "  access denied"
		case NodeErrNotFound:
			badge = "  not found"
		default:
			badge = "  error"
		}
	}

	plainLine := prefix + icon + name + badge
	plainWidth := lipgloss.Width(plainLine)
	if plainWidth < m.width {
		plainLine += strings.Repeat(" ", m.width-plainWidth)
	}

	if selected && m.focused {
		return m.Styles.Cursor.Render(plainLine)
	}

	var nameStyle lipgloss.Style
	switch {
	case node.State == NodeError:
		nameStyle = m.Styles.Denied
	case node.State == NodeLoading:
		nameStyle = m.Styles.Loading
	case node.Depth == 0:
		nameStyle = m.Styles.Engine
	case node.IsDir:
		nameStyle = m.Styles.Directory
	default:
		nameStyle = m.Styles.File
	}

	guide := m.Styles.Dim.Render(prefix)
	styledLine := guide + icon + nameStyle.Render(name)
	if badge != "" {
		styledLine += m.Styles.DeniedBadge.Render(badge)
	}

	styledWidth := lipgloss.Width(styledLine)
	if styledWidth < m.width {
		styledLine += strings.Repeat(" ", m.width-styledWidth)
	}

	return styledLine
}

func (m TreeModel) buildGuide(node *TreeNode) string {
	if node.Depth == 0 {
		return ""
	}

	parts := make([]string, node.Depth)

	current := node
	for d := node.Depth - 1; d >= 0; d-- {
		if d == node.Depth-1 {
			if m.isLastChild(current) {
				parts[d] = "└─"
			} else {
				parts[d] = "├─"
			}
		} else {
			current = current.Parent
			if current != nil && !m.isLastChild(current) {
				parts[d] = "│ "
			} else {
				parts[d] = "  "
			}
		}
	}

	return strings.Join(parts, "")
}

func (m TreeModel) isLastChild(node *TreeNode) bool {
	if node.Parent == nil {
		for i, r := range m.Roots {
			if r.ID == node.ID {
				return i == len(m.Roots)-1
			}
		}
		return true
	}
	children := node.Parent.Children
	return len(children) > 0 && children[len(children)-1].ID == node.ID
}

func (m *TreeModel) Flatten() {
	m.flatList = nil
	for _, root := range m.Roots {
		m.flattenNode(root)
	}
}

func (m *TreeModel) flattenNode(node *TreeNode) {
	m.flatList = append(m.flatList, node)
	if node.State == NodeExpanded && node.Children != nil {
		for _, child := range node.Children {
			m.flattenNode(child)
		}
	}
}

func (m *TreeModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

func (m *TreeModel) moveDown() {
	if m.cursor < len(m.flatList)-1 {
		m.cursor++
		m.fixOffset()
	}
}

func (m *TreeModel) fixOffset() {
	if m.height > 0 && m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

func (m TreeModel) findNode(id string) *TreeNode {
	for _, node := range m.flatList {
		if node.ID == id {
			return node
		}
	}
	for _, root := range m.Roots {
		if n := findNodeRecursive(root, id); n != nil {
			return n
		}
	}
	return nil
}

func findNodeRecursive(node *TreeNode, id string) *TreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if n := findNodeRecursive(child, id); n != nil {
			return n
		}
	}
	return nil
}
