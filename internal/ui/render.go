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
	if m.isRunning {
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

const (
	maxConfirmResources        = 10
	defaultConfirmReservedRows = 10 // borders + title + blanks + buttons + help
)

func (m Model) renderConfirmView() string {
	chosenAction := actionChoices[m.actionCursor]
	title := fmt.Sprintf("⚠  %s %d resource(s)?", chosenAction, len(m.selected))

	maxResourceRows := max(min(maxConfirmResources, m.viewHeight-defaultConfirmReservedRows), 1)

	addrs := m.selectedAddresses()
	if len(addrs) > maxResourceRows {
		addrs = addrs[:maxResourceRows]
	}

	var resourceLines []string
	for _, addr := range addrs {
		r := m.resources[m.indexMap[addr]]
		line := fmt.Sprintf("  %s %s", r.Action.Symbol(), addr)
		if style, ok := actionStyles[r.Action]; ok {
			line = style.Render(line)
		}
		resourceLines = append(resourceLines, line)
	}

	truncated := len(m.selected) - len(addrs)
	if truncated > 0 {
		dim := lipgloss.NewStyle().Foreground(colorDimGrey)
		resourceLines = append(resourceLines, dim.Render(fmt.Sprintf("  ... and %d more", truncated)))
	}

	cancelButton := buttonStyle.Render("Cancel")
	confirmButton := buttonStyle.Render("Confirm")
	if m.confirmCursor == 0 {
		cancelButton = focusedButtonStyle.Render("Cancel")
	} else {
		confirmButton = focusedButtonStyle.Render("Confirm")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, "  ", confirmButton)

	help := "Enter to select | Esc to cancel"

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	for _, line := range resourceLines {
		fmt.Fprintln(&s, line)
	}
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, buttons)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, help)
	fmt.Fprintln(&s)

	modal := focusedBorderStyle.Render(s.String())
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	x := max(0, (m.viewWidth-modalWidth)/2)
	y := max(0, (m.viewHeight-modalHeight)/2)

	background := lipgloss.NewLayer(m.renderListView())
	foreground := lipgloss.NewLayer(modal).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(background, foreground).Render()
}

func (m Model) renderOutputView() string {
	action := actionChoices[m.actionCursor]
	title := fmt.Sprintf("terraform %s", action)

	visible := m.visibleOutputRows()
	start := m.offset
	end := min(m.offset+visible, len(m.outputLines))

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)

	for i := start; i < end; i++ {
		fmt.Fprintln(&s, m.outputLines[i])
	}

	// Padding empty lines so modal looks same size
	rendered := end - start
	for range visible - rendered {
		fmt.Fprintln(&s)
	}

	var help string
	if m.isRunning {
		help = "↑/↓ scroll | Running..."
	} else {
		help = "↑/↓ scroll | Esc to close and re-plan"
	}
	fmt.Fprint(&s, help)

	modalStyle := focusedBorderStyle.Width(m.viewWidth - 4)
	modal := modalStyle.Render(s.String())

	return lipgloss.Place(m.viewWidth, m.viewHeight, lipgloss.Center, lipgloss.Center, modal)
}
