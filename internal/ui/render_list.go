package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) renderListView() string {
	var s strings.Builder

	fmt.Fprint(&s, m.renderFilterBox())
	fmt.Fprintln(&s, m.renderResourcesBox())
	fmt.Fprintln(&s, m.renderInfoBar())
	fmt.Fprintln(&s, m.renderHelpBar())

	return s.String()
}

func (m *Model) renderFilterBox() string {
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

// filter box (3) + resource borders (2) + info bar (1) + help bar with margin (4)
const listViewReservedRows = 10

func (m *Model) visibleRows() int {
	return max(1, m.viewHeight-listViewReservedRows)
}

func (m *Model) renderResourcesBox() string {
	var resources strings.Builder
	end := min(m.offset+m.visibleRows(), len(m.rows))
	for i := m.offset; i < end; i++ {
		row := m.rows[i]

		var line string
		if row.Item.IsResource() {
			line = m.renderResourceLine(i)
		} else {
			line = m.renderModuleLine(i)
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
	return borderStyle.Width(m.viewWidth).Render(renderString)
}

func (m *Model) renderResourceLine(idx int) string {
	row := m.rows[idx]
	addr := row.Item.Address()
	r := m.resources[addr]

	if r.Reason != "" {
		addr += fmt.Sprintf(" (%s)", r.Reason)
	}
	adornment := r.Action.Symbol()

	currentModule := m.currentCursorModule()
	prefix := row.TreePrefix
	if currentModule == row.Item.Parent.Module {
		prefix = treePrefixCurrentStyle.Render(prefix)
	} else {
		prefix = treePrefixDefaultStyle.Render(prefix)
	}

	line := fmt.Sprintf("%s %s", adornment, addr)
	switch {
	case idx == m.cursor:
		line = cursorStyle.Render(line)
	case m.isSelectedOrAncestor(row.Item):
		line = selectedStyle.Render(line)
	}
	if style, ok := actionStyles[r.Action]; ok {
		line = style.Render(line)
	}

	return prefix + line
}

func (m *Model) renderModuleLine(idx int) string {
	row := m.rows[idx]

	symbol := "▾"
	if m.collapsed[row.Item.Address()] {
		symbol = "▸"
	}
	line := fmt.Sprintf("%s %s", symbol, row.Item.Address())

	switch {
	case idx == m.cursor:
		line = cursorStyle.Render(line)
	case m.isSelectedOrAncestor(row.Item):
		line = selectedStyle.Render(line)
	}

	prefix := row.TreePrefix
	if m.currentCursorModule() == row.Item.Module {
		prefix = treePrefixCurrentStyle.Render(prefix)
		line = treePrefixCurrentStyle.Render(line)
	} else {
		prefix = treePrefixDefaultStyle.Render(prefix)
		line = moduleStyle.Render(line)
	}

	return prefix + line
}

func (m Model) renderInfoBar() string {
	var adornment, info string

	switch m.workState {
	case workStatePull:
		adornment = infoBarStyle.Render(m.spinner.View())
		info = " Scanning..."
	case workPlan:
		adornment = infoBarStyle.Render(m.spinner.View())
		var count int
		for _, r := range m.resources {
			if r.Action != terraform.ActionUncertain {
				count++
			}
		}
		info = fmt.Sprintf(" Scanning... (%d/%d resources scanned)", count, len(m.resources))
	default:
		adornment = lipgloss.NewStyle().Foreground(colorGreen).Render("✓")
		info = fmt.Sprintf("  Scan Complete (%d resources scanned)", len(m.resources))
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

func (m *Model) renderHelpBar() string {
	var HKeyInfo string
	if m.hideUnchanged {
		HKeyInfo = "show unchanged"
	} else {
		HKeyInfo = "hide unchanged"
	}
	keyInfos := []keyInfo{
		{key: "/", info: "filter"},
		{key: "Space", info: "select"},
		{key: "Enter", info: "detail"},
		{key: "Tab", info: "action"},
		{key: "H", info: HKeyInfo},
		{key: "Ctrl+r", info: "refresh"},
		{key: "q", info: "quit"},
	}
	return "\n" + m.renderKeyInfo(keyInfos)
}
