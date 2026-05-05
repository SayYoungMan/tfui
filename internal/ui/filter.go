package ui

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/sahilm/fuzzy"
)

func newFilterInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Press '/' to filter..."
	ti.Prompt = ""

	return ti
}

func (m *Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prev := m.filterInput.Value()
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)

	if m.filterInput.Value() != prev {
		m.rebuildRows()
		m.cursor = 0
		m.offset = 0
	}

	return m, cmd
}

func (m *Model) rebuildRows() {
	resources := m.visibleResources()

	rowMap := make(map[string]Row, len(resources)*2)
	children := make(map[string][]string)

	for _, r := range resources {
		for addr := r.Module; addr != ""; addr = parentModule(addr) {
			if _, exists := rowMap[addr]; exists {
				break
			}
			parent := parentModule(addr)
			rowMap[addr] = Row{Kind: rowModule, Address: addr, Parent: parent}
			children[parent] = append(children[parent], addr)
		}

		rowMap[r.Address] = Row{Kind: rowResource, Address: r.Address, Parent: r.Module}
		children[r.Module] = append(children[r.Module], r.Address)
	}

	// Sort children list for stable output
	for parent := range children {
		sort.Strings(children[parent])
	}

	m.rows = m.rows[:0]
	m.addRowsDFS(rowMap, children, "", []bool{})

	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
	m.adjustOffset()
}

// filter box (3) + resource borders (2) + info bar (1) + blank + help bar
const defaultReservedRows = 8

func (m Model) visibleRows() int {
	reserved := defaultReservedRows
	if m.viewWidth < 90 {
		reserved++
	}

	return max(1, m.viewHeight-reserved)
}

func (m *Model) visibleResources() terraform.Resources {
	var shown terraform.Resources
	for _, r := range m.resources {
		if m.hideUnchanged && isUnchanged(r) {
			continue
		}
		shown = append(shown, r)
	}

	filter := m.filterInput.Value()
	if filter == "" {
		return shown
	}

	filtered := fuzzy.FindFrom(filter, shown)
	resources := make([]*terraform.Resource, len(filtered))
	for i, r := range filtered {
		resources[i] = shown[r.Index]
	}
	return resources
}

func (m *Model) addRowsDFS(rowMap map[string]Row, children map[string][]string, parent string, isLast []bool) {
	if parent != "" {
		isLast = append(isLast, false)
	}
	for i, addr := range children[parent] {
		if i == len(children[parent])-1 && len(isLast) > 0 {
			isLast[len(isLast)-1] = true
		}

		row := rowMap[addr]
		row.TreePrefix = treePrefix(isLast)
		m.rows = append(m.rows, row)
		if row.Kind == rowModule && !m.collapsed[addr] {
			m.addRowsDFS(rowMap, children, addr, isLast)
		}
	}
}

func treePrefix(isLast []bool) string {
	if len(isLast) == 0 {
		return ""
	}

	var s strings.Builder
	for i := range len(isLast) - 1 {
		if isLast[i] {
			s.WriteString("   ")
		} else {
			s.WriteString("│  ")
		}
	}

	if isLast[len(isLast)-1] {
		s.WriteString("└─ ")
	} else {
		s.WriteString("├─ ")
	}

	return s.String()
}
