package ui

import (
	"sort"

	"github.com/SayYoungMan/tfui/pkg/terraform"
)

// Row is the row shown in list view
type Row struct {
	Item       *Item
	TreePrefix string
}

func (m *Model) rebuildRows() {
	resources := m.visibleResources()

	rowMap := make(map[string]Row, len(resources)*2)
	children := make(map[string][]string)

	for _, r := range resources {
		for addr := r.Module; addr != ""; addr = parentModuleAddr(addr) {
			if _, exists := rowMap[addr]; exists {
				break
			}
			parent := parentModuleAddr(addr)
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

// Item has exactly one Resource or Module. Indicates which resource UI's row points to
type Item struct {
	Resource *terraform.Resource
	Module   *Module
	Parent   *Module
}

// True for resource item and false for module item
func (i *Item) IsResource() bool {
	return i.Resource != nil
}

func (i *Item) Address() string {
	if i.Resource != nil {
		return i.Resource.Address
	}
	return i.Module.Address
}

type Module struct {
	Address   string
	Children  []*Item
	Collapsed bool
}
