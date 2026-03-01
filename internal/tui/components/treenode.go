package components

type NodeState int

const (
	NodeCollapsed NodeState = iota
	NodeExpanded
	NodeLoading
	NodeError
)

type NodeErrorKind int

const (
	NodeErrGeneric    NodeErrorKind = iota
	NodeErrDenied
	NodeErrNotFound
)

type TreeNode struct {
	ID       string
	Name     string
	FullPath string
	Engine   string
	IsDir    bool
	State    NodeState
	ErrKind  NodeErrorKind
	Children []*TreeNode
	Parent   *TreeNode
	Depth    int
	KVVer    int
}

func NewEngineNode(engine string, engineType string, kvVersion int) *TreeNode {
	return &TreeNode{
		ID:     engine,
		Name:   engine,
		Engine: engine,
		IsDir:  true,
		State:  NodeCollapsed,
		KVVer:  kvVersion,
	}
}

func NewChildNode(parent *TreeNode, name string, isDir bool) *TreeNode {
	fullPath := parent.FullPath + name
	return &TreeNode{
		ID:       parent.Engine + fullPath,
		Name:     name,
		FullPath: fullPath,
		Engine:   parent.Engine,
		IsDir:    isDir,
		State:    NodeCollapsed,
		Parent:   parent,
		Depth:    parent.Depth + 1,
		KVVer:    parent.KVVer,
	}
}
