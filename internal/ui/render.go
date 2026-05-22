package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderActionPickerView() string {
	title := fmt.Sprintf("%d resource(s) selected", len(m.selected))
	if m.selectAll {
		title = "ALL resources selected"
	}

	keyInfo := []keyInfo{
		{key: "Enter", info: "select"},
		{key: "Esc", info: "cancel"},
	}
	help := m.renderKeyInfo(keyInfo)

	width := max(lipgloss.Width(title), lipgloss.Width(help)) + 6
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)

	var s strings.Builder

	fmt.Fprintln(&s, centered.Render(title))
	fmt.Fprintln(&s, centered.Render(strings.Repeat("─", width-6)))

	for i, choice := range actionChoices {
		if i == m.actionCursor {
			fmt.Fprintln(&s, "  "+m.styles.cursor.Render("> "+choice))
		} else if m.selectAll && strings.Contains(choice, "taint") {
			fmt.Fprintln(&s, "    "+m.styles.dim.Render(choice))
		} else {
			fmt.Fprintln(&s, "    "+choice)
		}
	}

	fmt.Fprintln(&s)
	fmt.Fprintln(&s, centered.Render(help))

	return m.renderModalWithBackground(s.String(), m.renderListView(), nil)
}

func (m Model) renderConfirmView() string {
	chosenAction := actionChoices[m.actionCursor]
	title := fmt.Sprintf("⚠  %s %d resource(s)?", chosenAction, len(m.selected))
	if m.selectAll {
		title = fmt.Sprintf("⚠  %s ALL resource(s)?", chosenAction)
	}

	// For viewHeight >= 20, show max 10 resource names
	// For viewHeight 12~19, show max 1~9 resource names
	// For viewHeight < 12, show just 1 resource name max
	maxResourceRows := max(min(10, m.viewHeight-m.getReservedRows()), 1)

	addrs := m.targetedAddresses()
	if len(addrs) > maxResourceRows {
		addrs = addrs[:maxResourceRows]
	}

	var resourceLines []string
	for _, addr := range addrs {
		var line string
		if r, isResource := m.resources[addr]; isResource {
			line = fmt.Sprintf("  %s %s", r.Action.Symbol(), addr)
			if style, ok := m.styles.actions[r.Action]; ok {
				line = style.Render(line)
			}
		} else {
			line = fmt.Sprintf("    %s", addr)
		}
		resourceLines = append(resourceLines, line)
	}

	truncated := len(m.selected) - len(addrs)
	if truncated > 0 {
		resourceLines = append(resourceLines, m.styles.dim.Render(fmt.Sprintf("    ... and %d more", truncated)))
	}

	keyInfo := []keyInfo{
		{key: "Enter", info: "select"},
		{key: "Esc", info: "cancel"},
	}
	help := m.renderKeyInfo(keyInfo)

	var maxWidth int = 0
	for _, line := range resourceLines {
		maxWidth = max(maxWidth, lipgloss.Width(line))
	}
	maxWidth = max(maxWidth, lipgloss.Width(help)) + 2
	centered := lipgloss.NewStyle().Width(maxWidth).Align(lipgloss.Center)

	var s strings.Builder
	fmt.Fprintln(&s, centered.Render(title))
	fmt.Fprintln(&s)
	for _, line := range resourceLines {
		fmt.Fprintln(&s, line)
	}
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, centered.Render(m.renderConfirmCancelButtons()))
	fmt.Fprintln(&s)
	fmt.Fprint(&s, centered.Render(help))
	fmt.Fprintln(&s)

	return m.renderModalWithBackground(s.String(), m.renderListView(), nil)
}

func (m Model) renderDetailView() string {
	addr := m.rows[m.cursor].Item.Address()
	title := fmt.Sprintf(" Detail (%s)", addr)

	box := m.renderScrollableBox(m.outputLines, m.viewWidth, m.viewHeight-m.getReservedRows())

	keyInfo := []keyInfo{
		{key: "↑/↓", info: "scroll"},
		{key: "PgUp/PgDn", info: "page scroll"},
		{key: "Esc", info: "close"},
	}
	help := " " + m.renderKeyInfo(keyInfo)

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, box)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, help)

	return s.String()
}

func (m Model) renderOutputView() string {
	action := actionChoices[m.actionCursor]

	// If viewOutput, let content be the whole output lines
	// if viewResourceOutput, content is just that resource's output lines
	content := m.outputLines
	title := fmt.Sprintf(" %s output", action)
	if m.viewState == viewResourceOutput {
		progress := m.progressRows[m.cursor]
		title += fmt.Sprintf(" (%s)", progress.Address)
		content = progress.OutputLines
	}

	if len(content) == 0 {
		content = append(content, m.styles.dim.Render("No output available yet."))
	}

	box := m.renderScrollableBox(content, m.viewWidth-4, m.viewHeight-m.getReservedRows())
	keyInfos := []keyInfo{
		{key: "↑/↓", info: "scroll"},
		{key: "PgUp/PgDn", info: "page scroll"},
		{key: "o", info: "close output"},
	}
	if m.viewState == viewResourceOutput {
		keyInfos[2] = keyInfo{key: "Enter", info: "close output"}
	}
	if !m.isRunning() {
		keyInfos = append(keyInfos, keyInfo{key: "Esc", info: "close"})
	}
	help := " " + m.renderKeyInfo(keyInfos)

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, box)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, help)

	bg := lipgloss.NewLayer(m.styles.dim.Render(m.renderProgressView()))
	fg := m.renderModal(s.String(), &modalOpts{contentStyle: &lipgloss.Style{}})

	return lipgloss.NewCompositor(bg, fg).Render()
}

const quitConfirmTitle = "Do you want to quit?"

func (m Model) renderQuitConfirmLayer() *lipgloss.Layer {
	keyInfo := []keyInfo{
		{key: "Enter", info: "select"},
		{key: "Esc", info: "cancel"},
	}
	help := m.renderKeyInfo(keyInfo)

	width := lipgloss.Width(help) + 4
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)

	var s strings.Builder
	fmt.Fprintln(&s, centered.Render(quitConfirmTitle))
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, centered.Render(m.renderConfirmCancelButtons()))
	fmt.Fprintln(&s)
	fmt.Fprint(&s, centered.Render(help))
	fmt.Fprintln(&s)

	return m.renderModal(s.String(), nil)
}

func (m Model) renderShutdownLayer() *lipgloss.Layer {
	msg := "Exiting the program...\n\nWaiting for terraform to finish..."
	if m.quitState == forceQuitReadyState {
		msg += "\n\nPress q or ctrl+c again to force quit"
	}

	return m.renderModal(msg, &modalOpts{contentStyle: &m.styles.shutdownBorder})
}

func (m Model) renderErrorView() string {
	var s strings.Builder
	fmt.Fprintln(&s, m.styles.error.Render("Scanning Failed"))
	fmt.Fprintln(&s)

	for _, d := range m.diagnostics {
		if d.Severity == "error" {
			fmt.Fprintln(&s, m.styles.error.Render("Error: "+d.Summary))
		} else {
			fmt.Fprintln(&s, m.styles.warning.Render("Warning: "+d.Summary))
		}
		if d.Detail != "" {
			fmt.Fprintln(&s, "  "+d.Detail)
		}
		fmt.Fprintln(&s)
	}

	if m.err != nil {
		fmt.Fprintln(&s, m.styles.error.Render(fmt.Sprintf("Error: %v", m.err)))
		fmt.Fprintln(&s)
	}

	fmt.Fprint(&s, "Press Esc or Enter to quit")

	modalStyle := m.styles.focusedBorder.Width(m.viewWidth - 4)
	modal := modalStyle.Render(s.String())

	return lipgloss.Place(m.viewWidth, m.viewHeight, lipgloss.Center, lipgloss.Center, modal)
}
