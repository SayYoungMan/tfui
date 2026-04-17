package ui

import (
	"context"
	"sort"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m Model) normalModeKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.filteredIdx)-1 {
			m.cursor++
			m.adjustOffset()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}
	case "space":
		if len(m.filteredIdx) > 0 {
			idx := m.filteredIdx[m.cursor]
			addr := m.resources[idx].Address
			if m.selected[addr] {
				delete(m.selected, addr)
			} else {
				m.selected[addr] = true
			}
		}
	case "enter":
		if !m.isScanning && len(m.selected) > 0 {
			m.actionCursor = 0
			m.viewState = viewActionPicker
		}
	case "/":
		m.viewState = viewFilter
		m.filterInput.Focus()
		return m, textinput.Blink
	case "h":
		m.hideUnchanged = !m.hideUnchanged
		m.rebuildFilter()
	}

	return m, nil
}

func (m Model) filterModeKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.viewState = viewList
		m.filterInput.Blur()
		return m, nil
	}

	return m.updateFilter(msg)
}

func (m Model) actionPickerKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.actionCursor < len(actionChoices)-1 {
			m.actionCursor++
		}
	case "k", "up":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	case "enter":
		m.viewState = viewConfirm
		m.confirmCursor = 0
	case "esc":
		m.viewState = viewList
	}
	return m, nil
}

func (m Model) confirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		m.confirmCursor = 0
	case "l", "right":
		m.confirmCursor = 1
	case "enter":
		if m.confirmCursor == 0 {
			m.viewState = viewActionPicker
		} else {
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel

			addrs := m.selectedAddresses()
			action := actionChoices[m.actionCursor]

			actionFuncs := map[string]func(context.Context, []string) <-chan string{
				"plan":    m.runner.Plan,
				"apply":   m.runner.Apply,
				"destroy": m.runner.Destroy,
				"taint":   m.runner.Taint,
				"untaint": m.runner.Untaint,
			}
			m.outputChannel = actionFuncs[action](ctx, addrs)
			m.outputLines = nil
			m.isOutputing = false
			m.outputOffset = 0
			m.viewState = viewOutput

			return m, waitForOutput(m.outputChannel)
		}
	case "esc":
		m.viewState = viewActionPicker
	}

	return m, nil
}

// helper function to get addresses out of m.selected
// TODO: Refactor selected to be field of resource and remove this
func (m Model) selectedAddresses() []string {
	addrs := make([]string, 0, len(m.selected))
	for addr := range m.selected {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)
	return addrs
}

func (m Model) outputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "k", "up":
		if m.outputOffset > 0 {
			m.outputOffset--
		}
	case "j", "down":
		maxOffset := max(0, len(m.outputLines)-m.visibleOutputRows())
		if m.outputOffset < maxOffset {
			m.outputOffset++
		}
	case "esc":
		if !m.isOutputing {
			return m.startRescan()
		}
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	}
	return m, nil
}
