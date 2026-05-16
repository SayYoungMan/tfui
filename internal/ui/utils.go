package ui

import (
	"sort"
	"strings"

	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func (m Model) isRunning() bool {
	return m.workState != workIdle
}

func (m Model) hasError() bool {
	for _, d := range m.diagnostics {
		if d.Severity == "error" {
			return true
		}
	}
	return m.err != nil
}

func isUnchanged(r *terraform.Resource) bool {
	return r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead || r.Action == terraform.ActionUncertain
}

func (m Model) selectedAddresses() []string {
	addrs := make([]string, 0, len(m.selected))
	for addr := range m.selected {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)
	return addrs
}

// selectedResources returns the resources that would actually be acted on,
// independent of filter and "hide unchanged". Selection is resolved against
// m.resources (the full known set), so the picker / confirm / progress views
// and the terraform target list agree regardless of which rows are currently
// visible.
func (m Model) selectedResources() []*terraform.Resource {
	resources := make([]*terraform.Resource, 0, len(m.selected))
	for addr, r := range m.resources {
		if !m.isAddressInSelection(addr) {
			continue
		}
		resources = append(resources, r)
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Address < resources[j].Address
	})
	return resources
}

// isAddressInSelection reports whether the resource at addr is selected
// directly or sits under a selected ancestor module.
func (m Model) isAddressInSelection(addr string) bool {
	if m.selected[addr] {
		return true
	}
	for parent := parentModuleAddr(addr); parent != ""; parent = parentModuleAddr(parent) {
		if m.selected[parent] {
			return true
		}
	}
	return false
}

// returns if it or ancestor module is selected
func (m Model) isSelectedOrAncestor(item *Item) bool {
	if m.selected[item.Address()] {
		return true
	}

	for parent := item.Parent; parent != m.rootItem; parent = parent.Parent {
		if m.selected[parent.Address()] {
			return true
		}
	}
	return false
}

func isAncestor(ancestor string, child string) bool {
	for parent := parentModuleAddr(child); parent != ""; parent = parentModuleAddr(parent) {
		if parent == ancestor {
			return true
		}
	}
	return false
}

// Takes the address and check for last occurring module.x and returns it
func parentModuleAddr(address string) string {
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

	// We need to take 3 trailing segments for data and ephemeral resource otherwise 2
	trailing := 2
	if len(segments) >= 3 && (segments[len(segments)-3] == "data" || segments[len(segments)-3] == "ephemeral") {
		trailing = 3
	}

	if len(segments) < trailing {
		return ""
	}
	return strings.Join(segments[:len(segments)-trailing], ".")
}

// return the most direct module from current cursor position
func (m *Model) currentCursorModule() *Module {
	cursorItem := m.rows[m.cursor].Item

	if cursorItem.IsResource() {
		return cursorItem.Parent.Module
	}

	return cursorItem.Module
}

func (m *Model) adjustOffset() {
	visible := max(1, m.viewHeight-listViewReservedRows)

	// Cursor went below visible area — scroll down
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}

	// Cursor went above visible area — scroll up
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}
