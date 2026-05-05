package ui

import (
	"sort"
	"strings"

	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func (m *Model) isRunning() bool {
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

// returns if it or ancestor module is selected
func (m Model) isSelectedOrAncestor(addr string) bool {
	if m.selected[addr] {
		return true
	}

	for path := parentModule(addr); path != ""; path = parentModule(path) {
		if m.selected[path] {
			return true
		}
	}
	return false
}

func isAncestor(ancestor string, child string) bool {
	for parent := parentModule(child); parent != ""; parent = parentModule(parent) {
		if parent == ancestor {
			return true
		}
	}
	return false
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
func (m *Model) currentCursorModule() string {
	cursorRow := m.rows[m.cursor]

	currentModule := cursorRow.Address
	if cursorRow.Kind == rowResource {
		currentModule = cursorRow.Parent
	}

	return currentModule
}

func (m *Model) adjustOffset() {
	visible := m.visibleRows()

	// Cursor went below visible area — scroll down
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}

	// Cursor went above visible area — scroll up
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}
