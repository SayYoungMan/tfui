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
			rowMap[addr] = Row{Kind: rowModule, Address: addr}
			parent := parentModule(addr)
			children[parent] = append(children[parent], addr)
		}

		rowMap[r.Address] = Row{Kind: rowResource, Address: r.Address}
		children[r.Module] = append(children[r.Module], r.Address)
	}

	// Sort children list for stable output
	for parent := range children {
		sort.Strings(children[parent])
	}

	m.rows = m.rows[:0]
	m.addRowsDFS(rowMap, children, "", 0)

	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
		m.adjustOffset()
	}
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
	resources := make([]terraform.Resource, len(filtered))
	for i, r := range filtered {
		resources[i] = shown[r.Index]
	}
	return resources
}

func (m *Model) addRowsDFS(rowMap map[string]Row, children map[string][]string, parent string, depth int) {
	for _, addr := range children[parent] {
		row := rowMap[addr]
		row.Depth = depth
		m.rows = append(m.rows, row)
		if row.Kind == rowModule && !m.collapsed[addr] {
			m.addRowsDFS(rowMap, children, addr, depth+1)
		}
	}
}

func parentModule(address string) string {
	if !strings.HasPrefix(address, "module.") {
		return ""
	}

	raw := strings.Split(address, ".")
	segments := make([]string, 0, len(raw))
	// Go through all segments with splitted by . to find if it contains any unmatched " and match it
	// which means that there is a case like module.vpc["a.b"] that is edge case
	for i := 0; i < len(raw); i++ {
		seg := raw[i]
		for strings.Count(seg, "\"")%2 == 1 && i+1 < len(raw) {
			i++
			seg += "." + raw[i]
		}
		segments = append(segments, seg)
	}

	if len(segments) < 2 {
		return ""
	}
	return strings.Join(segments[:len(segments)-2], ".")
}

func isUnchanged(r terraform.Resource) bool {
	return r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead
}
