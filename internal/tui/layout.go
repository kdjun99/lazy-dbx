package tui

import (
	"github.com/rivo/tview"
)

// BuildLayout creates the main application layout.
// tree is placed on the left (fixed width 30), editor and results are stacked
// vertically on the right (editor weight 3, results weight 7),
// and statusBar is pinned to the bottom (fixed height 1).
func BuildLayout(tree tview.Primitive, editor tview.Primitive, results tview.Primitive, statusBar tview.Primitive) *tview.Flex {
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(editor, 0, 3, false).
		AddItem(results, 0, 7, false)

	topFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tree, 30, 0, true).
		AddItem(rightFlex, 0, 1, false)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	return mainFlex
}
