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
		}
	case "esc":
		m.viewState = viewActionPicker
	}

	return m, nil
}
