package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderActionPickerView() string {
	resolved := m.selectedResources()
	title := fmt.Sprintf("%d resource(s) selected", len(resolved))
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
			fmt.Fprintln(&s, "  "+cursorStyle.Render("> "+choice))
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
	resolved := m.selectedResources()
	title := fmt.Sprintf("⚠  %s %d resource(s)?", chosenAction, len(resolved))

	// For viewHeight >= 20, show max 10 resource names
	// For viewHeight 12~19, show max 1~9 resource names
	// For viewHeight < 12, show just 1 resource name max
	maxResourceRows := max(min(10, m.viewHeight-10), 1)

	shown := resolved
	if len(shown) > maxResourceRows {
		shown = shown[:maxResourceRows]
	}

	var resourceLines []string
	for _, r := range shown {
		line := fmt.Sprintf("  %s %s", r.Action.Symbol(), r.Address)
		if style, ok := actionStyles[r.Action]; ok {
			line = style.Render(line)
		}
		resourceLines = append(resourceLines, line)
	}

	truncated := len(resolved) - len(shown)
	if truncated > 0 {
		resourceLines = append(resourceLines, dimStyle.Render(fmt.Sprintf("    ... and %d more", truncated)))
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

const (
	defaultReservedOutputRows = 8
)

func (m Model) renderDetailView() string {
	addr := m.rows[m.cursor].Item.Address()
	title := fmt.Sprintf(" Detail (%s)", addr)

	box := m.renderScrollableBox(m.outputLines, m.viewWidth, m.viewHeight-6)

	keyInfo := []keyInfo{
		{key: "↑/↓", info: "scroll"},
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
	title := fmt.Sprintf(" %s output", action)

	box := m.renderScrollableBox(m.outputLines, m.viewWidth-4, m.viewHeight-10)
	keyInfo := []keyInfo{
		{key: "↑/↓", info: "scroll"},
		{key: "o", info: "close output"},
		{key: "Esc", info: "close"},
	}
	help := " " + m.renderKeyInfo(keyInfo)

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, box)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, help)

	bg := lipgloss.NewLayer(dimStyle.Render(m.renderProgressView()))
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

	return m.renderModal(msg, &modalOpts{contentStyle: &shutdownBorderStyle})
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
