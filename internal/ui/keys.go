package ui

import (
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
	case "/":
		m.filterFocused = true
		m.filterInput.Focus()
		return m, textinput.Blink
	case "h":
		m.hideNoops = !m.hideNoops
		m.rebuildFilter()
	}

	return m, nil
}

func (m Model) filterModeKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.filterFocused = false
		m.filterInput.Blur()
		return m, nil
	}

	return m.updateFilter(msg)
}
