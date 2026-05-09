package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) renderActionPickerView() string {
	title := fmt.Sprintf("%d resource(s) selected", len(m.selected))
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
			fmt.Fprintln(&s, cursorStyle.Render("  > "+choice))
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

	// For viewHeight >= 20, show max 10 resource names
	// For viewHeight 12~19, show max 1~9 resource names
	// For viewHeight < 12, show just 1 resource name max
	maxResourceRows := max(min(10, m.viewHeight-10), 1)

	addrs := m.selectedAddresses()
	if len(addrs) > maxResourceRows {
		addrs = addrs[:maxResourceRows]
	}

	var resourceLines []string
	for _, addr := range addrs {
		var line string
		if r, isResource := m.resources[addr]; isResource {
			line = fmt.Sprintf("  %s %s", r.Action.Symbol(), addr)
			if style, ok := actionStyles[r.Action]; ok {
				line = style.Render(line)
			}
		} else {
			line = fmt.Sprintf("    %s", addr)
		}
		resourceLines = append(resourceLines, line)
	}

	truncated := len(m.selected) - len(addrs)
	if truncated > 0 {
		resourceLines = append(resourceLines, dimStyle.Render(fmt.Sprintf("  ... and %d more", truncated)))
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
	statusColWidth = 16
	timeColWidth   = 10
)

func (m Model) renderProgressView() string {
	action := actionChoices[m.actionCursor]
	title := fmt.Sprintf("%sing %d resources...", action, len(m.progresses))

	addrColWidth := max(1, m.viewWidth-statusColWidth-timeColWidth*3-10)
	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s",
		addrColWidth, "Resource",
		statusColWidth, "Status",
		timeColWidth, "Wait",
		timeColWidth, "Read",
		timeColWidth, "Process",
	)

	var rows strings.Builder
	fmt.Fprintln(&rows, dimStyle.Render(header))
	fmt.Fprintln(&rows, dimStyle.Render(strings.Repeat("─", m.viewWidth)))

	var offset int
	if m.viewState == viewProgress {
		offset = m.offset
	} else {
		offset = 0
	}

	resources := m.selectedResources()
	visibleRows := max(1, m.viewHeight-4)
	end := min(offset+visibleRows, len(resources))

	for _, resource := range resources[offset:end] {
		addr := resource.Address
		ar := m.progresses[addr]

		displayAddr := addr
		if lipgloss.Width(displayAddr) > addrColWidth {
			displayAddr = ansi.Truncate(displayAddr, addrColWidth, "…")
		}

		var status string
		wait := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.waitDuration(m.actionStartTime))))
		read := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.readDuration())))
		process := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		switch ar.Status {
		case progressStatusPending:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Pending"))
			read = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		case progressStatusReadingState:
			status = infoBarStyle.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" Reading"))
			read = infoBarStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.readDuration())))
		case progressStatusWaitingForAction:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Waiting"))
		case progressStatusInProgress:
			status = infoBarStyle.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" In Progress"))
			process = infoBarStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.processDuration())))
		case progressStatusSuccessful:
			status = successStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "✅ Complete"))
			process = successStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.processDuration())))
		case progressStatusFailed:
			status = errorStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "❌ Failed"))
			process = errorStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(ar.processDuration())))
		case progressStatusSkipped:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "— No change"))
			wait = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
			read = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		}

		fmt.Fprintf(&rows, "  %-*s  %s  %s  %s  %s\n", addrColWidth, displayAddr, status, wait, read, process)
	}

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, rows.String())
	fmt.Fprintln(&s)

	var help string
	if m.isRunning() {
		help = "'o' raw output | Running..."
	} else {
		help = "'o' raw output | Esc to close and re-plan"
	}
	fmt.Fprint(&s, help)

	return s.String()
}

func (m *Model) formatElapsed(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

const (
	defaultReservedOutputWidth = 6
	defaultReservedOutputRows  = 8
)

func (m Model) renderOutputLayer(background string) string {
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

	fg := strings.TrimSuffix(content.String(), "\n")
	bg := dimStyle.Render(background)

	return m.renderModalWithBackground(fg, bg, &modalOpts{width: m.viewWidth - 2, height: boxHeight})
}

func (m Model) renderDetailView() string {
	addr := m.rows[m.cursor].Item.Address()
	title := fmt.Sprintf("Detail (%s)", addr)

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

	box := borderStyle.Width(m.viewWidth - 2).Height(boxHeight).Render(strings.TrimSuffix(content.String(), "\n"))

	help := "↑/↓ scroll | Esc to close"

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
