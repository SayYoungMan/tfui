package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	end := min(m.offset+m.visibleRows(), len(m.rows))
	for i := m.offset; i < end; i++ {
		row := m.rows[i]

		var line string
		switch row.Kind {
		case rowModule:
			line = m.renderModuleLine(i)
		case rowResource:
			line = m.renderResourceLine(i)
		}

		// Truncate the end to fit to screen
		if maxLineWidth := m.viewWidth - 4; lipgloss.Width(line) > maxLineWidth {
			line = ansi.Truncate(line, maxLineWidth, "…")
		}

		fmt.Fprintln(&resources, line)
	}

	rendered := end - m.offset
	for range m.visibleRows() - rendered {
		fmt.Fprintln(&resources)
	}

	renderString := strings.TrimSuffix(resources.String(), "\n")
	return resourceBorderStyle.Width(m.viewWidth).Render(renderString)
}

func (m Model) renderResourceLine(idx int) string {
	row := m.rows[idx]

	address := row.Address
	r := m.resources[m.resourceIndexMap[address]]
	if r.Reason != "" {
		address += fmt.Sprintf(" (%s)", r.Reason)
	}
	adornment := r.Action.Symbol()

	currentModule := m.currentCursorModule()
	prefix := row.TreePrefix
	if currentModule == row.Parent {
		prefix = treePrefixCurrentStyle.Render(prefix)
	} else {
		prefix = treePrefixDefaultStyle.Render(prefix)
	}

	line := fmt.Sprintf("%s %s", adornment, address)
	switch {
	case idx == m.cursor:
		line = cursorStyle.Render(line)
	case m.isSelectedOrAncestor(row.Address):
		line = selectedStyle.Render(line)
	}
	if style, ok := actionStyles[r.Action]; ok {
		line = style.Render(line)
	}

	return prefix + line
}

func (m Model) renderModuleLine(idx int) string {
	row := m.rows[idx]

	currentModule := m.currentCursorModule()
	prefix := row.TreePrefix
	if currentModule == row.Address {
		prefix = treePrefixCurrentStyle.Render(prefix)
	} else {
		prefix = treePrefixDefaultStyle.Render(prefix)
	}

	symbol := "▾"
	if m.collapsed[row.Address] {
		symbol = "▸"
	}
	line := fmt.Sprintf("%s %s", symbol, row.Address)

	switch {
	case idx == m.cursor:
		line = cursorStyle.Render(line)
	case m.isSelectedOrAncestor(row.Address):
		line = selectedStyle.Render(line)
	}

	if currentModule == row.Address {
		line = treePrefixCurrentStyle.Render(line)
	} else {
		line = moduleStyle.Render(line)
	}

	return prefix + line
}

// return the most direct module from current cursor position
func (m *Model) currentCursorModule() string {
	cursorRow := m.rows[m.cursor]

	currentModule := cursorRow.Address
	if cursorRow.Kind == rowResource {
		currentModule = cursorRow.Parent
	}

	return currentModule
}

// returns if it or ancestor module is selected
func (m Model) isSelectedOrAncestor(addr string) bool {
	if m.selected[addr] {
		return true
	}

	for path := parentModule(addr); path != ""; path = parentModule(path) {
		if m.selected[path] {
			return true
		}
	}
	return false
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
		info += fmt.Sprintf(" | showing %d", len(m.rows))
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
	var HKeyInfo string
	if m.hideUnchanged {
		HKeyInfo = "show unchanged"
	} else {
		HKeyInfo = "hide unchanged"
	}

	hints := []string{
		renderKeyHint("/", "filter"),
		renderKeyHint("Space", "select"),
		renderKeyHint("Tab", "action"),
		renderKeyHint("H", HKeyInfo),
		renderKeyHint("Ctrl+r", "refresh"),
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
		var line string
		if idx, isResource := m.resourceIndexMap[addr]; isResource {
			r := m.resources[idx]
			line = fmt.Sprintf("  %s %s", r.Action.Symbol(), addr)
			if style, ok := actionStyles[r.Action]; ok {
				line = style.Render(line)
			}
		} else {
			line = fmt.Sprintf("  ▾ %s", addr)
			line = dimStyle.Render(line)
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

	boxHeight := max(1, m.viewHeight-defaultReservedOutputRows)
	contentWidth := max(1, m.viewWidth-defaultReservedOutputWidth)
	innerHeight := boxHeight - 2 // subtract top and bottom border rows

	var content strings.Builder
	visualRows := 0
	for i := m.offset; i < len(m.outputLines); i++ {
		lineRows := max(1, (lipgloss.Width(m.outputLines[i])+contentWidth-1)/contentWidth)
		if visualRows+lineRows > innerHeight {
			break
		}
		fmt.Fprintln(&content, m.outputLines[i])
		visualRows += lineRows
	}

	box := resourceBorderStyle.
		Width(m.viewWidth - 2).
		Height(boxHeight).
		Render(strings.TrimSuffix(content.String(), "\n"))

	var help string
	if m.isRunning {
		help = "↑/↓ scroll | Running..."
	} else {
		help = "↑/↓ scroll | Esc to close and re-plan"
	}

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, box)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, help)

	return s.String()
}

const quitConfirmTitle = "Do you want to quit?"

func (m Model) renderQuitConfirmLayer() *lipgloss.Layer {
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
	fmt.Fprintln(&s, quitConfirmTitle)
	fmt.Fprintln(&s)
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

	return lipgloss.NewLayer(modal).X(x).Y(y).Z(1)
}

func (m Model) renderShutdownLayer() *lipgloss.Layer {
	msg := "Exiting the program...\n\nWaiting for terraform to finish..."
	if m.quitState == forceQuitReadyState {
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
