package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderActionPickerView() string {
	var s strings.Builder

	fmt.Fprintf(&s, "  %d resource(s) selected\n", len(m.selected))
	fmt.Fprintln(&s, "  "+strings.Repeat("─", 24))

	for i, choice := range actionChoices {
		if i == m.actionCursor {
			fmt.Fprintln(&s, cursorStyle.Render("  > "+choice))
		} else {
			fmt.Fprintln(&s, "    "+choice)
		}
	}

	fmt.Fprintln(&s)
	fmt.Fprintln(&s, "  Enter to choose | Esc to cancel")

	box := focusedBorderStyle.Render(s.String())
	return lipgloss.Place(m.viewHeight, m.viewHeight, lipgloss.Center, lipgloss.Center, box)
}
