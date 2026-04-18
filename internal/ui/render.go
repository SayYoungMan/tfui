package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderListView() string {
	var s strings.Builder

	fmt.Fprint(&s, m.renderFilterBox())
	fmt.Fprintln(&s, m.renderResourcesBox())
	fmt.Fprintln(&s, m.renderInfoBar())
	s.WriteString("\n" + m.renderHelpBar() + "\n")

	return s.String()
}

func (m Model) renderFilterBox() string {
	var s strings.Builder

	filterIcon := "⌕ "
	filterContent := filterIcon + m.filterInput.View()
	if m.viewState == viewFilter {
		fmt.Fprintln(&s, focusedBorderStyle.Width(m.viewWidth).Render(filterContent))
	} else {
		fmt.Fprintln(&s, borderStyle.Width(m.viewWidth).Render(filterContent))
	}

	return s.String()
}

func (m Model) renderResourcesBox() string {
	var resources strings.Builder
	end := min(m.offset+m.visibleRows(), len(m.filteredIdx))
	for i := m.offset; i < end; i++ {
		r := m.resources[m.filteredIdx[i]]
		symbol := r.Action.Symbol()
		reason := ""
		if r.Reason != "" {
			reason = fmt.Sprintf(" (%s)", r.Reason)
		}
		line := fmt.Sprintf("%s %s%s", symbol, r.Address, reason)

		// Truncate the end to fit to screen
		maxLineWidth := m.viewWidth - 4
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth-1] + "…"
		}

		switch {
		case i == m.cursor:
			line = cursorStyle.Render(line)
		case m.selected[r.Address]:
			line = selectedStyle.Render(line)
		}
		if style, ok := actionStyles[r.Action]; ok {
			line = style.Render(line)
		}

		fmt.Fprintln(&resources, line)
	}

	// Padding for visual consistency
	rendered := end - m.offset
	for range m.visibleRows() - rendered {
		fmt.Fprintln(&resources)
	}

	renderString := strings.TrimRight(resources.String(), "\n")
	return resourceBorderStyle.Width(m.viewWidth).Height(m.visibleRows()).Render(renderString)
}

func (m Model) renderInfoBar() string {
	var adornment, info string
	if m.isRunning {
		adornment = infoBarStyle.Render(m.spinner.View())
		info = fmt.Sprintf(" Scanning... (%d resources found)", len(m.resources))
	} else {
		adornment = lipgloss.NewStyle().Foreground(colorGreen).Render("✓")
		info = fmt.Sprintf("  Scan Complete (%d resources found)", len(m.resources))
	}
	if m.filterInput.Value() != "" {
		info += fmt.Sprintf(" | showing %d", len(m.filteredIdx))
	}
	if len(m.selected) > 0 {
		info += fmt.Sprintf(" | %d selected", len(m.selected))
	}
	if len(m.diagnostics) > 0 {
		info += fmt.Sprintf(" | %d warnings", len(m.diagnostics))
	}
	return " " + adornment + infoBarStyle.Render(info)
}

func renderKeyHint(key, desc string) string {
	key = "'" + key + "'"
	return helpKeyStyle.Render(key) + helpDescStyle.Render(" "+desc)
}

func (m Model) renderHelpBar() string {
	var hKeyInfo string
	if m.hideUnchanged {
		hKeyInfo = "show unchanged"
	} else {
		hKeyInfo = "hide unchanged"
	}

	hints := []string{
		renderKeyHint("/", "filter"),
		renderKeyHint("Space", "select"),
		renderKeyHint("Enter", "action"),
		renderKeyHint("h", hKeyInfo),
		renderKeyHint("q", "quit"),
	}

	if m.viewWidth >= 90 {
		return " " + strings.Join(hints, "  ")
	}

	mid := (len(hints) + 1) / 2
	line1 := " " + strings.Join(hints[:mid], "  ")
	line2 := " " + strings.Join(hints[mid:], "  ")
	return line1 + "\n" + line2
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

func (m Model) renderShutdownLayer() *lipgloss.Layer {
	msg := "Exiting the program...\n\nWaiting for terraform to finish..."
	if m.forceQuitReady {
		msg += "\n\nPress q or ctrl+c again to force quit"
	}

	modal := shutdownBorderStyle.Render(msg)
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	x := max(0, (m.viewWidth-modalWidth)/2)
	y := max(0, (m.viewHeight-modalHeight)/2)

	return lipgloss.NewLayer(modal).X(x).Y(y).Z(2)
}

func (m Model) renderErrorView() string {
	var s strings.Builder
	fmt.Fprintln(&s, errorStyle.Render("Scanning Failed"))
	fmt.Fprintln(&s)

	for _, d := range m.diagnostics {
		if d.Severity == "error" {
			fmt.Fprintln(&s, errorStyle.Render("Error: "+d.Summary))
		} else {
			fmt.Fprintln(&s, warningStyle.Render("Warning: "+d.Summary))
		}
		if d.Detail != "" {
			fmt.Fprintln(&s, "  "+d.Detail)
		}
		fmt.Fprintln(&s)
	}

	if m.err != nil {
		fmt.Fprintln(&s, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		fmt.Fprintln(&s)
	}

	fmt.Fprint(&s, "Press Esc or Enter to quit")

	modalStyle := focusedBorderStyle.Width(m.viewWidth - 4)
	modal := modalStyle.Render(s.String())

	return lipgloss.Place(m.viewWidth, m.viewHeight, lipgloss.Center, lipgloss.Center, modal)
}
