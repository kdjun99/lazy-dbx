package tui

import "fmt"

// PaginationState tracks the current page position within a result set.
type PaginationState struct {
	CurrentPage int
	TotalPages  int
	PageSize    int
	TotalRows   int
}

// NewPagination creates a PaginationState for the given total rows and page size.
func NewPagination(totalRows, pageSize int) PaginationState {
	totalPages := 0
	if pageSize > 0 && totalRows > 0 {
		totalPages = (totalRows + pageSize - 1) / pageSize
	}
	return PaginationState{
		CurrentPage: 0,
		TotalPages:  totalPages,
		PageSize:    pageSize,
		TotalRows:   totalRows,
	}
}

// NextPage advances to the next page. Returns false if already on the last page.
func (p *PaginationState) NextPage() bool {
	if p.TotalPages == 0 || p.CurrentPage >= p.TotalPages-1 {
		return false
	}
	p.CurrentPage++
	return true
}

// PrevPage moves to the previous page. Returns false if already on the first page.
func (p *PaginationState) PrevPage() bool {
	if p.CurrentPage <= 0 {
		return false
	}
	p.CurrentPage--
	return true
}

// GoToFirst jumps to the first page.
func (p *PaginationState) GoToFirst() {
	p.CurrentPage = 0
}

// GoToLast jumps to the last page.
func (p *PaginationState) GoToLast() {
	if p.TotalPages > 0 {
		p.CurrentPage = p.TotalPages - 1
	}
}

// StartRow returns the 0-indexed first row of the current page.
func (p *PaginationState) StartRow() int {
	return p.CurrentPage * p.PageSize
}

// EndRow returns the exclusive end row index (min of start+pageSize and totalRows).
func (p *PaginationState) EndRow() int {
	end := p.StartRow() + p.PageSize
	if end > p.TotalRows {
		end = p.TotalRows
	}
	return end
}

// FormatStatus returns a human-readable page status string.
func (p *PaginationState) FormatStatus() string {
	if p.TotalPages == 0 {
		return "Page 0/0 (0 rows)"
	}
	startDisplay := p.StartRow() + 1
	endDisplay := p.EndRow()
	return fmt.Sprintf("Page %d/%d (rows %d-%d of %d)",
		p.CurrentPage+1, p.TotalPages, startDisplay, endDisplay, p.TotalRows)
}
