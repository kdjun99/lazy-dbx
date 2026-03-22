package tui

import (
	"github.com/rivo/tview"
)

// BuildLayout creates the main application layout with 5 panels.
// connTree and schemaTree are stacked vertically on the left (fixed width 30),
// editor and results are stacked vertically on the right,
// and statusBar is pinned to the bottom (fixed height 1).
func BuildLayout(connTree, schemaTree, editor, results tview.Primitive, statusBar tview.Primitive) *tview.Flex {
	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(connTree, 0, 4, true).
		AddItem(schemaTree, 0, 6, false)

	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(editor, 0, 3, false).
		AddItem(results, 0, 7, false)

	topFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 30, 0, true).
		AddItem(rightFlex, 0, 1, false)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	return mainFlex
}
