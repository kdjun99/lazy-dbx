package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

const (
	maxColumnWidth = 50
)

// ResultsTable displays query results with pagination support.
type ResultsTable struct {
	table      *tview.Table
	pagination PaginationState
	data       *query.Result
	pageSize   int
}

// NewResultsTable creates a ResultsTable with borders, fixed header, and row selection.
func NewResultsTable() *ResultsTable {
	table := tview.NewTable().
		SetFixed(1, 0).
		SetSelectable(true, false)
	table.SetTitle("Results").SetBorder(true)

	rt := &ResultsTable{
		table:    table,
		pageSize: 100,
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case 'h':
			row, col := table.GetOffset()
			if col > 0 {
				table.SetOffset(row, col-1)
			}
			return nil
		case 'l':
			row, col := table.GetOffset()
			table.SetOffset(row, col+1)
			return nil
		}
		switch event.Key() {
		case tcell.KeyCtrlF:
			rt.NextPage()
			return nil
		case tcell.KeyCtrlB:
			rt.PrevPage()
			return nil
		}
		return event
	})

	return rt
}

// SetData stores the result and renders the first page.
func (rt *ResultsTable) SetData(result *query.Result, pageSize int) {
	rt.data = result
	rt.pageSize = pageSize
	rt.pagination = NewPagination(result.TotalRows, pageSize)
	rt.render()
}

// render clears the table and fills it with the current page of data.
func (rt *ResultsTable) render() {
	rt.table.Clear()

	if rt.data == nil {
		return
	}

	cols := rt.data.Columns

	// Compute column widths: max(headerLen, maxCellLen) capped at maxColumnWidth.
	colWidths := make([]int, len(cols))
	for i, col := range cols {
		colWidths[i] = len(col.Name)
	}

	pageRows := rt.data.Page(rt.pagination.CurrentPage, rt.pagination.PageSize)
	for _, row := range pageRows {
		for j, val := range row {
			if j >= len(colWidths) {
				break
			}
			cellLen := len(val.Display())
			if cellLen > colWidths[j] {
				colWidths[j] = cellLen
			}
		}
	}
	for i := range colWidths {
		if colWidths[i] > maxColumnWidth {
			colWidths[i] = maxColumnWidth
		}
	}

	// Header row (row 0).
	for i, col := range cols {
		cell := tview.NewTableCell(col.Name).
			SetTextColor(tcell.ColorWhite).
			SetAttributes(tcell.AttrBold).
			SetMaxWidth(colWidths[i]).
			SetExpansion(1)
		rt.table.SetCell(0, i, cell)
	}

	// Data rows.
	for rowIdx, row := range pageRows {
		for colIdx, val := range row {
			if colIdx >= len(cols) {
				break
			}
			text := val.Display()
			cell := tview.NewTableCell(text).
				SetMaxWidth(colWidths[colIdx]).
				SetExpansion(1)
			rt.table.SetCell(rowIdx+1, colIdx, cell)
		}
	}

	// Update title with pagination info.
	title := fmt.Sprintf("Results (%d rows, %dms)", rt.data.TotalRows, rt.data.Duration.Milliseconds())
	if rt.data.Truncated {
		title += " [truncated]"
	}
	if rt.pagination.TotalPages > 1 {
		title += " " + rt.pagination.FormatStatus()
	}
	rt.table.SetTitle(title)
}

// NextPage advances to the next page and re-renders.
func (rt *ResultsTable) NextPage() {
	if rt.pagination.NextPage() {
		rt.render()
	}
}

// PrevPage moves to the previous page and re-renders.
func (rt *ResultsTable) PrevPage() {
	if rt.pagination.PrevPage() {
		rt.render()
	}
}

// ShowError clears the table and displays an error message.
func (rt *ResultsTable) ShowError(msg string) {
	rt.table.Clear()
	rt.table.SetTitle("Results")
	cell := tview.NewTableCell(msg).
		SetTextColor(tcell.ColorRed)
	rt.table.SetCell(0, 0, cell)
}

// ShowExecResult displays a DML result message (rows affected + duration).
func (rt *ResultsTable) ShowExecResult(rowsAffected int64, duration time.Duration) {
	rt.table.Clear()
	rt.table.SetTitle("Results")
	msg := fmt.Sprintf("%d rows affected (%dms)", rowsAffected, duration.Milliseconds())
	cell := tview.NewTableCell(msg).
		SetTextColor(tcell.ColorGreen)
	rt.table.SetCell(0, 0, cell)
}

// Clear removes all data from the table.
func (rt *ResultsTable) Clear() {
	rt.data = nil
	rt.table.Clear()
	rt.table.SetTitle("Results")
}

// Widget returns the underlying tview.Table.
func (rt *ResultsTable) Widget() *tview.Table {
	return rt.table
}

// PaginationStatus returns the formatted page status string.
func (rt *ResultsTable) PaginationStatus() string {
	return rt.pagination.FormatStatus()
}
