// Package query defines domain models and interfaces for SQL query execution.
package query

import (
	"strings"
	"time"
)

// StatementType classifies a SQL statement.
type StatementType int

const (
	// StatementQuery covers SELECT, SHOW, DESCRIBE, EXPLAIN, WITH.
	StatementQuery StatementType = iota
	// StatementExec covers INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, and others.
	StatementExec
)

// ColumnInfo holds metadata for a result column.
type ColumnInfo struct {
	Name     string
	TypeName string
}

// Value represents a single cell value that may be NULL.
type Value struct {
	String string
	Valid  bool // false means NULL
}

// Display returns the human-readable representation of the value.
func (v Value) Display() string {
	if !v.Valid {
		return "<NULL>"
	}
	return v.String
}

// Result holds the result of executing a SQL statement.
type Result struct {
	Columns      []ColumnInfo
	Rows         [][]Value
	TotalRows    int
	Truncated    bool
	RowsAffected int64
	Type         StatementType
	Duration     time.Duration
}

// ClassifyStatement returns StatementQuery for SELECT/SHOW/DESCRIBE/EXPLAIN/WITH,
// and StatementExec for all other statements (INSERT, UPDATE, DELETE, CREATE, etc.).
func ClassifyStatement(sql string) StatementType {
	trimmed := strings.TrimSpace(sql)
	upper := strings.ToUpper(trimmed)

	queryKeywords := []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN", "WITH"}
	for _, kw := range queryKeywords {
		if strings.HasPrefix(upper, kw) {
			return StatementQuery
		}
	}
	return StatementExec
}

// PageCount returns the number of pages for the given page size.
func (r *Result) PageCount(pageSize int) int {
	if pageSize <= 0 || r.TotalRows == 0 {
		return 0
	}
	return (r.TotalRows + pageSize - 1) / pageSize
}

// Page returns the rows for the given 0-indexed page number.
func (r *Result) Page(pageNum, pageSize int) [][]Value {
	if pageSize <= 0 {
		return nil
	}
	start := pageNum * pageSize
	if start >= len(r.Rows) {
		return nil
	}
	end := start + pageSize
	if end > len(r.Rows) {
		end = len(r.Rows)
	}
	return r.Rows[start:end]
}
