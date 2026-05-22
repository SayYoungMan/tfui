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

	addrColWidth := max(1, m.viewWidth-statusColWidth-timeColWidth*3-10)
	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s",
		addrColWidth, "Resource",
		statusColWidth, "Status",
		timeColWidth, "Wait",
		timeColWidth, "Read",
		timeColWidth, "Process",
	)

	var rows strings.Builder
	fmt.Fprintln(&rows, m.styles.dim.Render(header))
	fmt.Fprintln(&rows, m.styles.dim.Render(strings.Repeat("─", m.viewWidth)))

	offset := 0
	if m.viewState == viewProgress {
		offset = m.offset
	}

	visibleRows := max(1, m.viewHeight-m.getReservedRows())
	end := min(offset+visibleRows, len(m.progressRows))

	for i, p := range m.progressRows[offset:end] {
		displayAddr := p.Address
		if lipgloss.Width(displayAddr) > addrColWidth {
			displayAddr = ansi.Truncate(displayAddr, addrColWidth, "…")
		}

		var status string
		wait := m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.waitDuration(m.actionStartTime))))
		read := m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.readDuration())))
		process := m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		switch p.Status {
		case progressStatusPending:
			status = m.styles.dim.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Pending"))
			read = m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		case progressStatusReadingState:
			status = m.styles.infoBar.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" Reading"))
			read = m.styles.infoBar.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.readDuration())))
		case progressStatusWaitingForAction:
			status = m.styles.dim.Render(fmt.Sprintf("%-*s", statusColWidth, "⏳ Waiting"))
		case progressStatusInProgress:
			status = m.styles.infoBar.Render(fmt.Sprintf("%-*s", statusColWidth, m.spinner.View()+" In Progress"))
			process = m.styles.infoBar.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusSuccessful:
			status = m.styles.success.Render(fmt.Sprintf("%-*s", statusColWidth, "✅ Complete"))
			process = m.styles.success.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusFailed:
			status = m.styles.error.Render(fmt.Sprintf("%-*s", statusColWidth, "❌ Failed"))
			process = m.styles.error.Render(fmt.Sprintf("%-*s", timeColWidth, m.formatElapsed(p.processDuration())))
		case progressStatusSkipped:
			status = m.styles.dim.Render(fmt.Sprintf("%-*s", statusColWidth, "— No change"))
			wait = m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
			read = m.styles.dim.Render(fmt.Sprintf("%-*s", timeColWidth, "-"))
		}

		line := fmt.Sprintf("  %-*s  %s  %s  %s  %s", addrColWidth, displayAddr, status, wait, read, process)
		if m.viewState == viewProgress && m.offset+i == m.cursor {
			line = m.styles.cursor.Render(line)
		}
		fmt.Fprintln(&rows, line)
	}

	status := "  Running..."
	if !m.isRunning() {
		status = fmt.Sprintf("  ✅ %s Completed", action)
	}

	keyInfos := []keyInfo{
		{key: "↑/↓", info: "scroll"},
		{key: "Ctrl+↑/↓", info: "page scroll"},
		{key: "Enter", info: "resource output"},
		{key: "o", info: "raw output"},
	}
	if !m.isRunning() {
		keyInfos = append(keyInfos, keyInfo{key: "Esc", info: "close"})
	}

	var s strings.Builder
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
