package ui

import (
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

func (m Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prev := m.filterInput.Value()
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)

	if m.filterInput.Value() != prev {
		m.rebuildFilter()
	}

	return m, cmd
}

func (m *Model) rebuildFilter() {
	filter := m.filterInput.Value()

	if filter == "" {
		m.filteredIdx = make([]int, 0, len(m.resources))
		for i, r := range m.resources {
			if m.hideNoops && (r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead) {
				continue
			}
			m.filteredIdx = append(m.filteredIdx, i)
		}
	} else {
		results := fuzzy.FindFrom(filter, m.resources)
		m.filteredIdx = make([]int, 0, len(results))
		for _, result := range results {
			idx := result.Index
			r := m.resources[idx]
			if m.hideNoops && (r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead) {
				continue
			}
			m.filteredIdx = append(m.filteredIdx, idx)
		}
	}

	// Reset cursor
	m.cursor = 0
	m.offset = 0
}

func (m Model) matchesFilter(r terraform.Resource) bool {
	if m.hideNoops && (r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead) {
		return false
	}

	filter := m.filterInput.Value()
	if filter == "" {
		return true
	}

	results := fuzzy.FindNoSort(filter, []string{r.Address})
	return len(results) > 0
}
