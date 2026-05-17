package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	statusColWidth = 16
	timeColWidth   = 10
)

func (m Model) renderProgressView() string {
	action := actionChoices[m.actionCursor]
	title := fmt.Sprintf("  %sing %d resources...", action, len(m.progresses))
	if !m.isRunning() {
		title = fmt.Sprintf("  Finished %sing %d resources", action, len(m.progresses))
	}

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

	offset := 0
	if m.viewState == viewProgress {
		offset = m.offset
	}

	resources := m.selectedResources()
	visibleRows := max(1, m.viewHeight-9)
	end := min(offset+visibleRows, len(resources))

	for _, resource := range resources[offset:end] {
		addr := resource.Address
		p := m.progresses[addr]

		displayAddr := addr
		if lipgloss.Width(displayAddr) > addrColWidth {
			displayAddr = ansi.Truncate(displayAddr, addrColWidth, "…")
		}

		var status string
		wait := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.waitDuration(m.actionStartTime))))
		read := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.readDuration())))
		process := dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		switch p.Status {
		case progressStatusPending:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Pending"))
			read = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		case progressStatusReadingState:
			status = infoBarStyle.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" Reading"))
			read = infoBarStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.readDuration())))
		case progressStatusWaitingForAction:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Waiting"))
		case progressStatusInProgress:
			status = infoBarStyle.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" In Progress"))
			process = infoBarStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusSuccessful:
			status = successStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "✅ Complete"))
			process = successStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusFailed:
			status = errorStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "❌ Failed"))
			process = errorStyle.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusSkipped:
			status = dimStyle.Render(fmt.Sprintf("%-*s", statusColWidth, "— No change"))
			wait = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
			read = dimStyle.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		}

		fmt.Fprintf(&rows, "  %-*s  %s  %s  %s  %s\n", addrColWidth, displayAddr, status, wait, read, process)
	}

	status := "  Running..."
	if !m.isRunning() {
		status = fmt.Sprintf("  ✅ Completed %sing", action)
	}

	keyInfos := []keyInfo{
		{key: "↑/↓", info: "scroll"},
		{key: "o", info: "open output"},
	}
	if !m.isRunning() {
		keyInfos = append(keyInfos, keyInfo{key: "Enter/Esc", info: "close"})
	}

	var s strings.Builder
	fmt.Fprintln(&s, title)
	fmt.Fprintln(&s)
	fmt.Fprint(&s, rows.String())
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, status)
	fmt.Fprintln(&s)
	fmt.Fprintln(&s, "  "+m.renderKeyInfo(keyInfos))
	fmt.Fprintln(&s)

	return s.String()
}

func (m *Model) formatElapsed(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}
