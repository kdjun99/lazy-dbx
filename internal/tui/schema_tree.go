package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kdjun99/lazy-dbx/internal/domain/catalog"
)

// schemaNodeRef holds metadata for each node in the schema tree.
type schemaNodeRef struct {
	NodeType string // "database", "table", "column"
	Database string
	Table    string
}

// SchemaTree wraps a tview.TreeView for browsing database schema lazily.
type SchemaTree struct {
	view          *tview.TreeView
	root          *tview.TreeNode
	onTableSelect func(database, table string)
	onExpand      func(nodeType string, database string, table string)
	loadedNodes   map[string]bool
}

// NewSchemaTree creates a SchemaTree with lazy-loading callbacks.
// onTableSelect is called when user selects a table node.
// onExpand is called when user expands a database or table node.
func NewSchemaTree(onTableSelect func(database, table string), onExpand func(nodeType, database, table string)) *SchemaTree {
	st := &SchemaTree{
		onTableSelect: onTableSelect,
		onExpand:      onExpand,
		loadedNodes:   make(map[string]bool),
	}

	st.root = tview.NewTreeNode("Schema").
		SetColor(tcell.ColorWhite).
		SetSelectable(false)

	st.view = tview.NewTreeView().
		SetRoot(st.root).
		SetCurrentNode(st.root)

	st.view.SetBorder(true).SetTitle("Schema")

	placeholder := tview.NewTreeNode("Connect to browse schema").
		SetSelectable(false).
		SetColor(tcell.ColorGray)
	st.root.AddChild(placeholder)

	st.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		if event.Key() != tcell.KeyEnter {
			return event
		}
		st.handleEnter()
		return nil
	})

	return st
}

// handleEnter processes the Enter key on the currently selected node.
func (st *SchemaTree) handleEnter() {
	node := st.view.GetCurrentNode()
	if node == nil {
		return
	}
	ref, ok := node.GetReference().(schemaNodeRef)
	if !ok {
		return
	}
	switch ref.NodeType {
	case "database":
		if !st.loadedNodes[ref.Database] && st.onExpand != nil {
			st.onExpand("database", ref.Database, "")
			node.SetExpanded(true)
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
	case "table":
		tableKey := ref.Database + "." + ref.Table
		if !st.loadedNodes[tableKey] && st.onExpand != nil {
			// First Enter: load columns via lazy fetch.
			st.onExpand("table", ref.Database, ref.Table)
			st.loadedNodes[tableKey] = true
			node.SetExpanded(true)
		} else if node.IsExpanded() {
			// Columns already visible: collapse.
			node.SetExpanded(false)
		} else if len(node.GetChildren()) > 0 {
			// Columns loaded but collapsed: expand.
			node.SetExpanded(true)
		} else if st.onTableSelect != nil {
			// No children (empty table?): trigger preview.
			st.onTableSelect(ref.Database, ref.Table)
		}
	}
}

// LoadDatabases replaces root children with database nodes.
func (st *SchemaTree) LoadDatabases(databases []catalog.Database) {
	st.root.ClearChildren()
	if len(databases) == 0 {
		placeholder := tview.NewTreeNode("No databases found").
			SetSelectable(false).
			SetColor(tcell.ColorGray)
		st.root.AddChild(placeholder)
		return
	}
	for i, db := range databases {
		dbNode := tview.NewTreeNode(db.Name).
			SetColor(tcell.ColorYellow).
			SetReference(schemaNodeRef{NodeType: "database", Database: db.Name})
		st.root.AddChild(dbNode)
		if i == 0 {
			st.view.SetCurrentNode(dbNode)
		}
	}
}

// findDatabaseNode returns the tree node for the given database name, or nil.
func (st *SchemaTree) findDatabaseNode(database string) *tview.TreeNode {
	for _, child := range st.root.GetChildren() {
		ref, ok := child.GetReference().(schemaNodeRef)
		if ok && ref.NodeType == "database" && ref.Database == database {
			return child
		}
	}
	return nil
}

// findTableNode returns the tree node for the given table under a database node, or nil.
func findTableNode(dbNode *tview.TreeNode, table string) *tview.TreeNode {
	for _, child := range dbNode.GetChildren() {
		ref, ok := child.GetReference().(schemaNodeRef)
		if ok && ref.NodeType == "table" && ref.Table == table {
			return child
		}
	}
	return nil
}

// removeLoadingNode removes any "Loading..." placeholder child from a node.
func removeLoadingNode(parent *tview.TreeNode) {
	children := parent.GetChildren()
	filtered := make([]*tview.TreeNode, 0, len(children))
	for _, c := range children {
		if c.GetText() != "Loading..." {
			filtered = append(filtered, c)
		}
	}
	parent.ClearChildren()
	for _, c := range filtered {
		parent.AddChild(c)
	}
}

// LoadTables adds table children under the given database node.
func (st *SchemaTree) LoadTables(database string, tables []catalog.Table) {
	dbNode := st.findDatabaseNode(database)
	if dbNode == nil {
		return
	}
	removeLoadingNode(dbNode)
	st.loadedNodes[database] = true

	for _, tbl := range tables {
		prefix := "[T]"
		if tbl.IsView() {
			prefix = "[V]"
		}
		label := prefix + " " + tbl.Name
		tableNode := tview.NewTreeNode(label).
			SetColor(tcell.ColorWhite).
			SetReference(schemaNodeRef{NodeType: "table", Database: database, Table: tbl.Name})
		dbNode.AddChild(tableNode)
	}
}

// LoadColumns adds column children under the given table node.
func (st *SchemaTree) LoadColumns(database, table string, columns []catalog.Column) {
	dbNode := st.findDatabaseNode(database)
	if dbNode == nil {
		return
	}
	tableNode := findTableNode(dbNode, table)
	if tableNode == nil {
		return
	}
	removeLoadingNode(tableNode)

	for _, col := range columns {
		label := col.Name + ": " + col.FormatType()
		colNode := tview.NewTreeNode(label).
			SetSelectable(false).
			SetColor(tcell.ColorGray)
		tableNode.AddChild(colNode)
	}
}

// Clear resets the tree to its initial state with a placeholder message.
func (st *SchemaTree) Clear() {
	st.root.ClearChildren()
	st.loadedNodes = make(map[string]bool)
	placeholder := tview.NewTreeNode("Connect to browse schema").
		SetSelectable(false).
		SetColor(tcell.ColorGray)
	st.root.AddChild(placeholder)
}

// SetLoading adds a "Loading..." placeholder child to the node with the given label.
func (st *SchemaTree) SetLoading(parentLabel string) {
	var parent *tview.TreeNode
	if st.root.GetText() == parentLabel {
		parent = st.root
	} else {
		for _, child := range st.root.GetChildren() {
			if child.GetText() == parentLabel {
				parent = child
				break
			}
		}
	}
	if parent == nil {
		return
	}
	loading := tview.NewTreeNode("Loading...").
		SetSelectable(false).
		SetColor(tcell.ColorGray)
	parent.AddChild(loading)
}

// ShowError adds a red error message child to the node with the given label.
func (st *SchemaTree) ShowError(parentLabel string, msg string) {
	var parent *tview.TreeNode
	if st.root.GetText() == parentLabel {
		parent = st.root
	} else {
		for _, child := range st.root.GetChildren() {
			if child.GetText() == parentLabel {
				parent = child
				break
			}
		}
	}
	if parent == nil {
		return
	}
	errNode := tview.NewTreeNode(msg).
		SetSelectable(false).
		SetColor(tcell.ColorRed)
	parent.AddChild(errNode)
}

// Widget returns the underlying tview.TreeView for embedding in layouts.
func (st *SchemaTree) Widget() *tview.TreeView {
	return st.view
}
