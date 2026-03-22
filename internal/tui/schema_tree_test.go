package tui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain/catalog"
	"github.com/kdjun99/lazy-dbx/internal/tui"
)

func TestSchemaTree_New(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	require.NotNil(t, st)
	require.NotNil(t, st.Widget())
}

func TestSchemaTree_LoadDatabases(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	dbs := []catalog.Database{
		{Name: "mydb"},
		{Name: "testdb"},
	}
	st.LoadDatabases(dbs)

	root := st.Widget().GetRoot()
	require.NotNil(t, root)

	children := root.GetChildren()
	assert.Len(t, children, 2)
	assert.Equal(t, "mydb", children[0].GetText())
	assert.Equal(t, "testdb", children[1].GetText())
}

func TestSchemaTree_LoadDatabases_Empty(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	st.LoadDatabases([]catalog.Database{})

	root := st.Widget().GetRoot()
	require.NotNil(t, root)

	children := root.GetChildren()
	require.Len(t, children, 1)
	assert.Equal(t, "No databases found", children[0].GetText())
}

func TestSchemaTree_LoadTables(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	dbs := []catalog.Database{{Name: "mydb"}}
	st.LoadDatabases(dbs)

	tables := []catalog.Table{
		{Name: "users", Schema: "mydb", Type: "BASE TABLE"},
		{Name: "orders", Schema: "mydb", Type: "BASE TABLE"},
	}
	st.LoadTables("mydb", tables)

	root := st.Widget().GetRoot()
	dbChildren := root.GetChildren()
	require.Len(t, dbChildren, 1)

	tableChildren := dbChildren[0].GetChildren()
	require.Len(t, tableChildren, 2)
	assert.Equal(t, "[T] users", tableChildren[0].GetText())
	assert.Equal(t, "[T] orders", tableChildren[1].GetText())
}

func TestSchemaTree_LoadTables_ViewDistinction(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	dbs := []catalog.Database{{Name: "mydb"}}
	st.LoadDatabases(dbs)

	tables := []catalog.Table{
		{Name: "users", Schema: "mydb", Type: "BASE TABLE"},
		{Name: "v_users", Schema: "mydb", Type: "VIEW"},
	}
	st.LoadTables("mydb", tables)

	root := st.Widget().GetRoot()
	dbChildren := root.GetChildren()
	require.Len(t, dbChildren, 1)

	tableChildren := dbChildren[0].GetChildren()
	require.Len(t, tableChildren, 2)
	assert.Equal(t, "[T] users", tableChildren[0].GetText())
	assert.Equal(t, "[V] v_users", tableChildren[1].GetText())
}

func TestSchemaTree_LoadColumns(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	dbs := []catalog.Database{{Name: "mydb"}}
	st.LoadDatabases(dbs)

	tables := []catalog.Table{
		{Name: "users", Schema: "mydb", Type: "BASE TABLE"},
	}
	st.LoadTables("mydb", tables)

	columns := []catalog.Column{
		{Name: "id", OrdinalPos: 1, DataType: "int", Nullable: false, KeyType: "PRI"},
		{Name: "name", OrdinalPos: 2, DataType: "varchar(100)", Nullable: true, KeyType: ""},
	}
	st.LoadColumns("mydb", "users", columns)

	root := st.Widget().GetRoot()
	dbChildren := root.GetChildren()
	require.Len(t, dbChildren, 1)

	tableChildren := dbChildren[0].GetChildren()
	require.Len(t, tableChildren, 1)

	colChildren := tableChildren[0].GetChildren()
	require.Len(t, colChildren, 2)
	assert.Equal(t, "id: int [PK] NOT NULL", colChildren[0].GetText())
	assert.Equal(t, "name: varchar(100) NULL", colChildren[1].GetText())
}

func TestSchemaTree_Clear(t *testing.T) {
	st := tui.NewSchemaTree(nil, nil)
	dbs := []catalog.Database{{Name: "mydb"}}
	st.LoadDatabases(dbs)

	st.Clear()

	root := st.Widget().GetRoot()
	children := root.GetChildren()
	require.Len(t, children, 1)
	assert.Equal(t, "Connect to browse schema", children[0].GetText())
}
