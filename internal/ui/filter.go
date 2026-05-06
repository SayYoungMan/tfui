package ui

import (
	"sort"

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

// filter box (3) + resource borders (2) + info bar (1) + blank + help bar
const defaultReservedRows = 8

func (m Model) visibleRows() int {
	reserved := defaultReservedRows
	if m.viewWidth < 90 {
		reserved++
	}

	return max(1, m.viewHeight-reserved)
}

// returns slice of resources that matches the filter result, in rank order
// if there is no filter applied, return the result in alphabetical order
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
		sort.Slice(shown, func(i, j int) bool {
			return shown[i].Address < shown[j].Address
		})
		return shown
	}

	filtered := fuzzy.FindFrom(filter, shown)
	resources := make([]*terraform.Resource, len(filtered))
	for i, r := range filtered {
		resources[i] = shown[r.Index]
	}
	return resources
}
