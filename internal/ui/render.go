package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderListView() string {
	var s strings.Builder

	fmt.Fprintln(&s, m.renderFilterBox())

	end := min(m.offset+m.visibleRows(), len(m.filteredIdx))
	for i := m.offset; i < end; i++ {
		r := m.resources[m.filteredIdx[i]]
		symbol := r.Action.Symbol()
		reason := ""
		if r.Reason != "" {
			reason = fmt.Sprintf(" (%s)", r.Reason)
		}
		line := fmt.Sprintf("%s %s%s", symbol, r.Address, reason)

		switch {
		case i == m.cursor:
			line = cursorStyle.Render(line)
		case m.selected[r.Address]:
			line = selectedStyle.Render(line)
		}
		if style, ok := actionStyles[r.Action]; ok {
			line = style.Render(line)
		}

		fmt.Fprintln(&s, line)
	}

	var infoLine string
	if m.isScanning {
		infoLine = fmt.Sprintf("\n %s Scanning... (%d resources found)", m.spinner.View(), len(m.resources))
	} else {
		infoLine = fmt.Sprintf("\n Scan Complete (%d resources found)", len(m.resources))
	}
	if m.filterInput.Value() != "" {
		infoLine += fmt.Sprintf(" | showing %d", len(m.filteredIdx))
	}
	if len(m.selected) > 0 {
		infoLine += fmt.Sprintf(" | %d selected", len(m.selected))
	}
	fmt.Fprintln(&s, infoLine)

	if m.err != nil {
		fmt.Fprintf(&s, "\n error occurred: %v\n", m.err)
	}

	var hKeyInfo string
	if m.hideUnchanged {
		hKeyInfo = "h to show unchanged"
	} else {
		hKeyInfo = "h to hide unchanged"
	}
	keyInfoLine := fmt.Sprintf("\n / to filter | Space to select | Enter to perform action | %s | q or ctrl+C to quit.\n", hKeyInfo)
	s.WriteString(keyInfoLine)

	return s.String()
}

func (m Model) renderFilterBox() string {
	var s strings.Builder

	filterIcon := "⌕ "
	filterContent := filterIcon + m.filterInput.View()
	if m.viewState == viewFilter {
		fmt.Fprintln(&s, focusedBorderStyle.Render(filterContent))
	} else {
		fmt.Fprintln(&s, borderStyle.Render(filterContent))
	}

	return s.String()
}

func (m Model) renderActionPickerView() string {
	title := fmt.Sprintf("%d resource(s) selected", len(m.selected))
	help := "Enter to choose | Esc to cancel"

	width := max(lipgloss.Width(title), lipgloss.Width(help)) + 6
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)

	var s strings.Builder

	fmt.Fprintln(&s, centered.Render(title))
	fmt.Fprintln(&s, centered.Render(strings.Repeat("─", width-6)))

	for i, choice := range actionChoices {
		if i == m.actionCursor {
			fmt.Fprintln(&s, cursorStyle.Render("  > "+choice))
		} else {
			fmt.Fprintln(&s, "    "+choice)
		}
	}

	fmt.Fprintln(&s)
	fmt.Fprintln(&s, centered.Render(help))

	modal := focusedBorderStyle.Render(s.String())
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	x := max(0, (m.viewWidth-modalWidth)/2)
	y := max(0, (m.viewHeight-modalHeight)/2)

	background := lipgloss.NewLayer(m.renderListView())
	foreground := lipgloss.NewLayer(modal).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(background, foreground).Render()
}
