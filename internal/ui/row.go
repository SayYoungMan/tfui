package ui

import (
	"strings"

	"github.com/SayYoungMan/tfui/pkg/terraform"
)

// Row is the row shown in list view
type Row struct {
	Item       *Item
	TreePrefix string
}

func (m *Model) rebuildRows() {
	resources := m.visibleResources()
	m.buildItemTree(resources)

	m.recursiveBuildRow(m.rootItem, []bool{})

	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
	m.adjustOffset()
}

// build rows recursively via DFS traversal from m.rootItem
func (m *Model) recursiveBuildRow(item *Item, isLast []bool) {
	if item != m.rootItem {
		m.rows = append(m.rows, Row{Item: item, TreePrefix: treePrefix(isLast)})
	}

	// No need to go deeper into resource or collapsed module
	if item.IsResource() {
		return
	}
	if m.collapsed[item.Address()] && m.filterInput.Value() == "" {
		return
	}

	childIsLast := append([]bool{}, isLast...)
	if item != m.rootItem {
		childIsLast = append(childIsLast, false)
	}
	for i, child := range item.Module.Children {
		if i == len(item.Module.Children)-1 && len(childIsLast) > 0 {
			childIsLast[len(childIsLast)-1] = true
		}
		m.recursiveBuildRow(child, childIsLast)
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

func (i *Item) IsModule() bool {
	return i.Module != nil
}

func (i *Item) Address() string {
	if i.Resource != nil {
		return i.Resource.Address
	}
	return i.Module.Address
}

// Builds tree of *Item with module hierarchy
func (m *Model) buildItemTree(resources []*terraform.Resource) {
	itemMap := map[string]*Item{"": m.rootItem}

	for i, r := range resources {
		parentItem := m.buildModuleItem(r.Module, itemMap)
		item := &Item{
			Resource: resources[i],
			Parent:   parentItem.Module,
		}
		parentItem.Module.Children = append(parentItem.Module.Children, item)
	}
}

type Module struct {
	Address  string
	Children []*Item
}

// Given module address, recursively build modules and add to child
func (m *Model) buildModuleItem(addr string, itemMap map[string]*Item) *Item {
	if addr == "" {
		return m.rootItem
	}
	if existing, ok := itemMap[addr]; ok {
		return existing
	}

	parentAddr := parentModuleAddr(addr)
	parentItem := m.buildModuleItem(parentAddr, itemMap)

	item := &Item{
		Module: &Module{Address: addr},
		Parent: parentItem.Module,
	}
	parentItem.Module.Children = append(parentItem.Module.Children, item)
	itemMap[addr] = item

	return item
}
