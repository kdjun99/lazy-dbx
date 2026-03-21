package tui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
)

// TreeNode holds data for a single node in the connection tree.
type TreeNode struct {
	Label    string
	Path     string
	IsLeaf   bool
	Children []TreeNode
	Env      config.Environment
}

// BuildTreeData converts a ConnectionsConfig into a sorted slice of TreeNodes.
// Groups, subgroups, and connections are sorted alphabetically for deterministic order.
// connectedPaths maps connection path → true for connected connections.
// readonly controls the mode label displayed on leaf nodes.
func BuildTreeData(cfg *config.ConnectionsConfig, readonly bool, connectedPaths map[string]bool) []TreeNode {
	groupNames := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	nodes := make([]TreeNode, 0, len(groupNames))
	for _, groupName := range groupNames {
		group := cfg.Groups[groupName]
		groupNode := TreeNode{
			Label:  groupName,
			IsLeaf: false,
		}

		subgroupNames := make([]string, 0, len(group.Subgroups))
		for name := range group.Subgroups {
			subgroupNames = append(subgroupNames, name)
		}
		sort.Strings(subgroupNames)

		for _, subgroupName := range subgroupNames {
			subgroup := group.Subgroups[subgroupName]
			subgroupNode := TreeNode{
				Label:  subgroupName,
				IsLeaf: false,
			}

			connNames := make([]string, 0, len(subgroup.Connections))
			for name := range subgroup.Connections {
				connNames = append(connNames, name)
			}
			sort.Strings(connNames)

			for _, connName := range connNames {
				entry := subgroup.Connections[connName]
				path := groupName + "." + subgroupName + "." + connName
				connStatus := ConnectionStatusDisconnected
				if connectedPaths[path] {
					connStatus = ConnectionStatusConnected
				}
				icon := StatusIcon(connStatus)
				label := fmt.Sprintf("%s %s [%s] %s", icon, entry.Name, string(entry.Env), ModeLabel(readonly))
				leafNode := TreeNode{
					Label:  label,
					Path:   path,
					IsLeaf: true,
					Env:    entry.Env,
				}
				subgroupNode.Children = append(subgroupNode.Children, leafNode)
			}

			groupNode.Children = append(groupNode.Children, subgroupNode)
		}

		nodes = append(nodes, groupNode)
	}

	return nodes
}

// ConnectionTree wraps a tview.TreeView for displaying database connections.
type ConnectionTree struct {
	view    *tview.TreeView
	nodeMap map[string]*tview.TreeNode
}

// NewConnectionTree creates a ConnectionTree from BuildTreeData output.
// onSelect is called with the connection path when the user selects a leaf node.
func NewConnectionTree(data []TreeNode, onSelect func(path string)) *ConnectionTree {
	ct := &ConnectionTree{
		nodeMap: make(map[string]*tview.TreeNode),
	}

	root := tview.NewTreeNode("Connections").
		SetColor(tcell.ColorWhite).
		SetSelectable(false)

	ct.view = tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	for _, node := range data {
		tNode := ct.buildTviewNode(node)
		root.AddChild(tNode)
	}

	ct.view.SetSelectedFunc(func(tNode *tview.TreeNode) {
		ref := tNode.GetReference()
		if ref == nil {
			// non-leaf: toggle expand/collapse
			tNode.SetExpanded(!tNode.IsExpanded())
			return
		}
		path, ok := ref.(string)
		if !ok {
			return
		}
		if onSelect != nil {
			onSelect(path)
		}
	})

	ct.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		return event
	})

	return ct
}

// buildTviewNode recursively converts a TreeNode into a tview.TreeNode.
func (ct *ConnectionTree) buildTviewNode(data TreeNode) *tview.TreeNode {
	tNode := tview.NewTreeNode(data.Label)

	if data.IsLeaf {
		tNode.SetReference(data.Path)
		tNode.SetColor(EnvColor(data.Env))
		ct.nodeMap[data.Path] = tNode
	} else {
		tNode.SetColor(tcell.ColorWhite)
		tNode.SetExpanded(true)
	}

	for _, child := range data.Children {
		childNode := ct.buildTviewNode(child)
		tNode.AddChild(childNode)
	}

	return tNode
}

// UpdateNodeStatus updates the icon prefix of a leaf node to reflect connection status.
func (ct *ConnectionTree) UpdateNodeStatus(path string, status ConnectionStatus) {
	tNode, ok := ct.nodeMap[path]
	if !ok {
		return
	}
	text := tNode.GetText()
	if len([]rune(text)) > 0 {
		icon := StatusIcon(status)
		runes := []rune(text)
		runes[0] = []rune(icon)[0]
		tNode.SetText(string(runes))
	}
}

// SetEmptyMessage adds a placeholder child to the root node for empty configs.
func (ct *ConnectionTree) SetEmptyMessage(msg string) {
	root := ct.view.GetRoot()
	placeholder := tview.NewTreeNode(msg).
		SetSelectable(false).
		SetColor(tcell.ColorGray)
	root.AddChild(placeholder)
}

// Widget returns the underlying tview.TreeView for embedding in layouts.
func (ct *ConnectionTree) Widget() *tview.TreeView {
	return ct.view
}
