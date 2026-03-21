package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

func TestClassifyStatement(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected query.StatementType
	}{
		{"SELECT uppercase", "SELECT * FROM foo", query.StatementQuery},
		{"select lowercase", "select * from foo", query.StatementQuery},
		{"INSERT", "INSERT INTO foo VALUES (1)", query.StatementExec},
		{"UPDATE", "UPDATE foo SET x=1", query.StatementExec},
		{"DELETE", "DELETE FROM foo", query.StatementExec},
		{"WITH", "WITH cte AS (SELECT 1) SELECT * FROM cte", query.StatementQuery},
		{"SHOW", "SHOW TABLES", query.StatementQuery},
		{"DESCRIBE", "DESCRIBE foo", query.StatementQuery},
		{"EXPLAIN", "EXPLAIN SELECT * FROM foo", query.StatementQuery},
		{"CREATE", "CREATE TABLE foo (id INT)", query.StatementExec},
		{"ALTER", "ALTER TABLE foo ADD COLUMN x INT", query.StatementExec},
		{"DROP", "DROP TABLE foo", query.StatementExec},
		{"whitespace-prefixed SELECT", "  SELECT * FROM foo", query.StatementQuery},
		{"unknown GRANT", "GRANT ALL ON foo TO user", query.StatementExec},
		{"empty string", "", query.StatementExec},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := query.ClassifyStatement(tt.sql)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestValue_Display(t *testing.T) {
	tests := []struct {
		name     string
		value    query.Value
		expected string
	}{
		{"valid string", query.Value{String: "hello", Valid: true}, "hello"},
		{"NULL", query.Value{String: "", Valid: false}, "<NULL>"},
		{"empty valid string", query.Value{String: "", Valid: true}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.value.Display()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResult_PageCount(t *testing.T) {
	tests := []struct {
		name      string
		totalRows int
		pageSize  int
		expected  int
	}{
		{"0 rows, 100 page size", 0, 100, 0},
		{"1 row, 100 page size", 1, 100, 1},
		{"100 rows, 100 page size", 100, 100, 1},
		{"101 rows, 100 page size", 101, 100, 2},
		{"250 rows, 100 page size", 250, 100, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &query.Result{TotalRows: tt.totalRows}
			got := r.PageCount(tt.pageSize)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResult_Page(t *testing.T) {
	// Build 10 rows of 1 column each
	rows := make([][]query.Value, 10)
	for i := range rows {
		rows[i] = []query.Value{{String: "row", Valid: true}}
	}
	r := &query.Result{
		Rows:      rows,
		TotalRows: 10,
	}

	t.Run("first page", func(t *testing.T) {
		page := r.Page(0, 3)
		assert.Len(t, page, 3)
		assert.Equal(t, rows[0], page[0])
		assert.Equal(t, rows[2], page[2])
	})

	t.Run("last partial page", func(t *testing.T) {
		page := r.Page(3, 3)
		// rows 9 only (index 9)
		assert.Len(t, page, 1)
		assert.Equal(t, rows[9], page[0])
	})

	t.Run("page beyond range returns empty", func(t *testing.T) {
		page := r.Page(10, 3)
		assert.Empty(t, page)
	})
}
