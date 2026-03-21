package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

func makeResult(cols []string, rows [][]string) *query.Result {
	colInfos := make([]query.ColumnInfo, len(cols))
	for i, c := range cols {
		colInfos[i] = query.ColumnInfo{Name: c, TypeName: "VARCHAR"}
	}
	valueRows := make([][]query.Value, len(rows))
	for i, row := range rows {
		vrow := make([]query.Value, len(row))
		for j, cell := range row {
			vrow[j] = query.Value{String: cell, Valid: true}
		}
		valueRows[i] = vrow
	}
	return &query.Result{
		Columns:   colInfos,
		Rows:      valueRows,
		TotalRows: len(rows),
		Type:      query.StatementQuery,
		Duration:  10 * time.Millisecond,
	}
}

func makeNullResult() *query.Result {
	return &query.Result{
		Columns: []query.ColumnInfo{{Name: "col1", TypeName: "VARCHAR"}},
		Rows: [][]query.Value{
			{{String: "", Valid: false}},
		},
		TotalRows: 1,
		Type:      query.StatementQuery,
		Duration:  5 * time.Millisecond,
	}
}

func TestResultsTable_SetData(t *testing.T) {
	rt := NewResultsTable()
	require.NotNil(t, rt)
	require.NotNil(t, rt.Widget())

	result := makeResult(
		[]string{"id", "name"},
		[][]string{{"1", "alice"}, {"2", "bob"}},
	)
	rt.SetData(result, 100)

	// Header row + 2 data rows = 3 rows total
	assert.Equal(t, 3, rt.Widget().GetRowCount())
	// 2 columns
	assert.Equal(t, 2, rt.Widget().GetColumnCount())
}

func TestResultsTable_NullDisplay(t *testing.T) {
	rt := NewResultsTable()
	result := makeNullResult()
	rt.SetData(result, 100)

	// Row 1 (index 1) is data row, column 0
	cell := rt.Widget().GetCell(1, 0)
	assert.Equal(t, "<NULL>", cell.Text)
}

func TestResultsTable_Pagination(t *testing.T) {
	// Create 150 rows with pageSize=100 → 2 pages
	rows := make([][]string, 150)
	for i := range rows {
		rows[i] = []string{"val"}
	}
	result := makeResult([]string{"col"}, rows)
	rt := NewResultsTable()
	rt.SetData(result, 100)

	// Page 1: 100 data rows + 1 header = 101
	assert.Equal(t, 101, rt.Widget().GetRowCount())

	// Next page: 50 data rows + 1 header = 51
	rt.NextPage()
	assert.Equal(t, 51, rt.Widget().GetRowCount())

	// Prev back to page 1: 100 data rows + 1 header = 101
	rt.PrevPage()
	assert.Equal(t, 101, rt.Widget().GetRowCount())
}

func TestResultsTable_ShowError(t *testing.T) {
	rt := NewResultsTable()
	rt.ShowError("connection refused")

	// Should have at least 1 row with error text
	assert.GreaterOrEqual(t, rt.Widget().GetRowCount(), 1)
	cell := rt.Widget().GetCell(0, 0)
	assert.Contains(t, cell.Text, "connection refused")
}

func TestResultsTable_ShowExecResult(t *testing.T) {
	rt := NewResultsTable()
	rt.ShowExecResult(42, 15*time.Millisecond)

	assert.GreaterOrEqual(t, rt.Widget().GetRowCount(), 1)
	cell := rt.Widget().GetCell(0, 0)
	assert.Contains(t, cell.Text, "42")
}

func TestResultsTable_EmptyResult(t *testing.T) {
	rt := NewResultsTable()
	result := makeResult([]string{"id", "name"}, [][]string{})
	rt.SetData(result, 100)

	// Header row only
	assert.Equal(t, 1, rt.Widget().GetRowCount())
	assert.Equal(t, 2, rt.Widget().GetColumnCount())
}
