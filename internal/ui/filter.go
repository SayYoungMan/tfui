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
		m.filteredIdx = make([]int, len(m.resources))
		for i := range m.resources {
			m.filteredIdx[i] = i
		}
	} else {
		results := fuzzy.FindFrom(filter, m.resources)
		m.filteredIdx = make([]int, len(results))
		for i, result := range results {
			m.filteredIdx[i] = result.Index
		}
	}

	// Reset cursor
	m.cursor = 0
	m.offset = 0
}

func (m Model) matchesFilter(r terraform.Resource) bool {
	filter := m.filterInput.Value()
	if filter == "" {
		return true
	}

	results := fuzzy.FindNoSort(filter, []string{r.Address})
	return len(results) > 0
}
