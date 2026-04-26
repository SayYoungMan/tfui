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
	case "h", "left":
		if len(m.rows) == 0 {
			break
		}
		row := m.rows[m.cursor]
		if row.Kind == rowModule && !m.collapsed[row.Address] {
			m.collapsed[row.Address] = true
			m.rebuildRows()
		} else if row.Kind == rowResource {
			parent := parentModule(row.Address)
			if parent == "" {
				break
			}

			m.collapsed[parent] = true
			m.rebuildRows()

			// Set cursor on its parent after collapse
			for i, r := range m.rows {
				if r.Address != parent {
					continue
				}
				m.cursor = i
				m.adjustOffset()
				break
			}
		}
	case "l", "right":
		if len(m.rows) == 0 {
			break
		}
		row := m.rows[m.cursor]
		if row.Kind == rowModule && m.collapsed[row.Address] {
			delete(m.collapsed, row.Address)
			m.rebuildRows()
		}
	case "enter":
		if len(m.rows) == 0 {
			break
		}
		row := m.rows[m.cursor]
		if row.Kind == rowModule {
			if m.collapsed[row.Address] {
				delete(m.collapsed, row.Address)
			} else {
				m.collapsed[row.Address] = true
			}
			m.rebuildRows()
		}
		if row.Kind == rowResource {
			m.openDetail()
		}
	case "space":
		m.toggleSelected()
	case "tab":
		if !m.isRunning() && len(m.selected) > 0 {
			m.actionCursor = 0
			m.viewState = viewActionPicker
		}
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
	}

	return m, nil
}

func (m *Model) toggleSelected() {
	if len(m.rows) <= 0 {
		return
	}
	row := m.rows[m.cursor]
	addr := row.Address
	if m.selected[addr] {
		delete(m.selected, addr)
	} else {
		// This is case where parent module is selected but this resource was not so skip
		if m.isSelectedOrAncestor(addr) {
			return
		}
		// Remove from selected map if there is a child row that was selected
		for selectedAddr := range m.selected {
			if isAncestor(addr, selectedAddr) {
				delete(m.selected, selectedAddr)
			}
		}
		m.selected[addr] = true
	}
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
	case "tab":
		m.actionCursor++
		m.actionCursor %= len(actionChoices)
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

func (m Model) quitViewConfirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		if m.offset < len(m.outputLines)-1 {
			m.offset++
		}
	case "esc":
		if !m.isRunning() {
			return m.startRescan()
		}
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
