package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPagination(t *testing.T) {
	tests := []struct {
		name          string
		totalRows     int
		pageSize      int
		wantPage      int
		wantTotPages  int
		wantPageSize  int
		wantTotalRows int
	}{
		{"normal", 487, 100, 0, 5, 100, 487},
		{"zero rows", 0, 100, 0, 0, 100, 0},
		{"exact fit", 100, 100, 0, 1, 100, 100},
		{"pageSize > totalRows", 10, 100, 0, 1, 100, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(tt.totalRows, tt.pageSize)
			assert.Equal(t, tt.wantPage, p.CurrentPage)
			assert.Equal(t, tt.wantTotPages, p.TotalPages)
			assert.Equal(t, tt.wantPageSize, p.PageSize)
			assert.Equal(t, tt.wantTotalRows, p.TotalRows)
		})
	}
}

func TestPagination_NextPrev(t *testing.T) {
	p := NewPagination(487, 100) // 5 pages, 0-indexed

	// Start at page 0
	assert.Equal(t, 0, p.CurrentPage)

	// Next moves forward
	ok := p.NextPage()
	assert.True(t, ok)
	assert.Equal(t, 1, p.CurrentPage)

	// Next to last page (page 4)
	p.NextPage()
	p.NextPage()
	p.NextPage()
	assert.Equal(t, 4, p.CurrentPage)

	// Next at last returns false
	ok = p.NextPage()
	assert.False(t, ok)
	assert.Equal(t, 4, p.CurrentPage)

	// Prev moves back
	ok = p.PrevPage()
	assert.True(t, ok)
	assert.Equal(t, 3, p.CurrentPage)

	// Back to first
	p.PrevPage()
	p.PrevPage()
	p.PrevPage()
	assert.Equal(t, 0, p.CurrentPage)

	// Prev at first returns false
	ok = p.PrevPage()
	assert.False(t, ok)
	assert.Equal(t, 0, p.CurrentPage)
}

func TestPagination_Bounds(t *testing.T) {
	p := NewPagination(487, 100) // 5 pages

	// Page 0: rows 0-100
	assert.Equal(t, 0, p.StartRow())
	assert.Equal(t, 100, p.EndRow())

	// Jump to last page (page 4): rows 400-487
	p.GoToLast()
	assert.Equal(t, 4, p.CurrentPage)
	assert.Equal(t, 400, p.StartRow())
	assert.Equal(t, 487, p.EndRow())

	// Go to first
	p.GoToFirst()
	assert.Equal(t, 0, p.CurrentPage)
	assert.Equal(t, 0, p.StartRow())
	assert.Equal(t, 100, p.EndRow())
}

func TestPagination_FormatStatus(t *testing.T) {
	tests := []struct {
		name      string
		totalRows int
		pageSize  int
		page      int
		want      string
	}{
		{"page 1 of 5", 487, 100, 0, "Page 1/5 (rows 1-100 of 487)"},
		{"page 5 of 5", 487, 100, 4, "Page 5/5 (rows 401-487 of 487)"},
		{"single page", 50, 100, 0, "Page 1/1 (rows 1-50 of 50)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination(tt.totalRows, tt.pageSize)
			p.CurrentPage = tt.page
			assert.Equal(t, tt.want, p.FormatStatus())
		})
	}
}

func TestPagination_EdgeCases(t *testing.T) {
	// 0 rows
	p0 := NewPagination(0, 100)
	assert.Equal(t, 0, p0.TotalPages)
	assert.Equal(t, 0, p0.StartRow())
	assert.Equal(t, 0, p0.EndRow())

	// 1 row
	p1 := NewPagination(1, 100)
	assert.Equal(t, 1, p1.TotalPages)
	assert.Equal(t, 0, p1.StartRow())
	assert.Equal(t, 1, p1.EndRow())

	// pageSize > totalRows
	p2 := NewPagination(10, 100)
	assert.Equal(t, 1, p2.TotalPages)
	assert.Equal(t, 0, p2.StartRow())
	assert.Equal(t, 10, p2.EndRow())
}
