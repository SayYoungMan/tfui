package ui

import (
	"sort"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m Model) normalModeKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			m.adjustOffset()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}
	case "space":
		if len(m.rows) > 0 {
			row := m.rows[m.cursor]
			// TODO: Make modules also selectable
			if row.Kind == rowResource {
				addr := row.Address
				if m.selected[addr] {
					delete(m.selected, addr)
				} else {
					m.selected[addr] = true
				}
			}
		}
	case "enter":
		if !m.isRunning && len(m.selected) > 0 {
			m.actionCursor = 0
			m.viewState = viewActionPicker
		}
	case "/":
		m.viewState = viewFilter
		m.filterInput.Focus()
		return m, textinput.Blink
	case "h":
		m.hideUnchanged = !m.hideUnchanged
		m.rebuildRows()
		m.cursor = 0
		m.offset = 0
	case "ctrl+r":
		if !m.isRunning {
			return m.startRescan()
		}
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
			return m.startAction()
		}
	case "esc":
		m.viewState = viewActionPicker
	}

	return m, nil
}

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
		if m.offset > 0 {
			m.offset--
		}
	case "j", "down":
		if m.offset < m.maxOutputOffset() {
			m.offset++
		}
	case "esc":
		if !m.isRunning {
			return m.startRescan()
		}
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) errorKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.cancel()
		return m, tea.Quit
	}
	return m, nil
}
