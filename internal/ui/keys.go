package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m Model) listKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}

	item := m.rows[m.cursor].Item
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
	case "h", "left":
		if item.IsModule() && !m.collapsed[item.Address()] {
			m.collapsed[item.Module.Address] = true
			m.rebuildRows()
		} else if item.IsResource() && item.Parent != m.rootItem {
			// Collapse resource's parent module
			m.collapsed[item.Parent.Address()] = true
			m.rebuildRows()

			// Set cursor on its parent after collapse
			for i, r := range m.rows {
				if r.Item.Address() == item.Parent.Address() {
					m.cursor = i
					m.adjustOffset()
					break
				}
			}
		}
	case "l", "right":
		if item.IsModule() && m.collapsed[item.Address()] {
			delete(m.collapsed, item.Module.Address)
			m.rebuildRows()
		}
	case "enter":
		if item.IsModule() {
			if m.collapsed[item.Module.Address] {
				delete(m.collapsed, item.Module.Address)
			} else {
				m.collapsed[item.Module.Address] = true
			}
			m.rebuildRows()
		}
		if item.IsResource() {
			m.openDetail()
		}
	case "space":
		row := m.rows[m.cursor]
		addr := row.Item.Address()

		// If selectAll is true, turn it off and select that item only
		if m.selectAll {
			m.selectAll = false
		}

		if m.selected[addr] {
			delete(m.selected, addr)
		} else {
			// This is case where parent module is selected but this resource was not so skip
			if m.isSelectedOrAncestor(row.Item) {
				break
			}
			// Remove from selected map if there is a child row that was selected
			for selectedAddr := range m.selected {
				if isAncestor(addr, selectedAddr) {
					delete(m.selected, selectedAddr)
				}
			}
			m.selected[addr] = true
		}

	case "tab":
		if m.isRunning() {
			break
		}
		if len(m.selected) == 0 && !m.selectAll {
			m.selected[item.Address()] = true
		}
		m.actionCursor = 0
		m.viewState = viewActionPicker
	case "/":
		m.viewState = viewFilter
		m.filterInput.Focus()
		return m, textinput.Blink
	case "H":
		m.hideUnchanged = !m.hideUnchanged
		m.rebuildRows()
		m.cursor = 0
		m.offset = 0
	case "ctrl+r":
		if !m.isRunning() {
			return m.startRescan()
		}
	case "ctrl+a":
		m.selectAll = !m.selectAll
		m.selected = make(map[string]bool)
	}

	return m, nil
}

func (m Model) filterKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "tab":
		m.actionCursor++
		m.actionCursor %= len(actionChoices)
	case "enter":
		// Ignore if selectAll is on and action is 'taint' or 'untaint'
		if m.selectAll && m.actionCursor > 2 {
			break
		}
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
	case "tab":
		m.confirmCursor = 1 - m.confirmCursor
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

func (m Model) quitConfirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		m.confirmCursor = 0
	case "l", "right":
		m.confirmCursor = 1
	case "tab":
		m.confirmCursor = 1 - m.confirmCursor
	case "enter":
		if m.confirmCursor == 0 {
			m.quitState = noneQuitState
		} else {
			return m.gracefulQuit()
		}
	case "esc":
		m.quitState = noneQuitState
	}

	return m, nil
}

func (m Model) progressKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}
	case "j", "down":
		if m.cursor < len(m.progresses)-1 {
			m.cursor++
			m.adjustOffset()
		}
	case "enter":
		m.offset = 0
		m.viewState = viewResourceOutput
	case "esc":
		if !m.isRunning() {
			return m.startRescan()
		}
	case "o":
		m.offset = 0
		m.viewState = viewOutput
	case "q", "ctrl+c":
		m.cancel.fn()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) outputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "k", "up":
		if m.offset > 0 {
			m.offset--
		}
	case "j", "down":
		contentLen := len(m.outputLines)
		if m.viewState == viewResourceOutput {
			contentLen = len(m.currentProgress().OutputLines)
		}

		if m.offset < contentLen-1 {
			m.offset++
		}
	case "esc":
		if !m.isRunning() {
			return m.startRescan()
		}
	case "o", "enter":
		m.offset = 0
		m.viewState = viewProgress
	case "q", "ctrl+c":
		m.cancel.fn()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) errorKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.cancel.fn()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) detailKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.viewState = viewList
		m.outputLines = nil
		m.offset = 0
	case "k", "up":
		if m.offset > 0 {
			m.offset--
		}
	case "j", "down":
		if m.offset < len(m.outputLines)-1 {
			m.offset++
		}
	}
	return m, nil
}
