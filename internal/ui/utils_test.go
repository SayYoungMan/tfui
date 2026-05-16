package ui

import (
	"testing"

	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectedResources_Empty(t *testing.T) {
	m := newTestModelEmpty()

	resources := m.selectedResources()

	assert.Empty(t, resources)
}

func TestSelectedResources_OnlyResources(t *testing.T) {
	m := newTestModelWithResources([]*terraform.Resource{
		{Address: "aws_s3.a", Action: terraform.ActionCreate},
		{Address: "aws_s3.b", Action: terraform.ActionCreate},
		{Address: "aws_s3.c", Action: terraform.ActionCreate},
	})
	m.selected = map[string]bool{"aws_s3.a": true, "aws_s3.c": true}

	resources := m.selectedResources()

	addrs := make([]string, len(resources))
	for i, r := range resources {
		addrs[i] = r.Address
	}
	assert.Len(t, resources, 2)
	assert.Contains(t, addrs, "aws_s3.a")
	assert.Contains(t, addrs, "aws_s3.c")
}

func TestSelectedResources_IndependentOfFilter(t *testing.T) {
	m := newTestModelWithResources([]*terraform.Resource{
		{Address: "aws_s3.a", Action: terraform.ActionCreate},
		{Address: "aws_lambda.b", Action: terraform.ActionCreate},
	})
	m.selected = map[string]bool{"aws_s3.a": true}

	m.filterInput.SetValue("lambda")
	m.rebuildRows()
	for _, row := range m.rows {
		require.NotEqual(t, "aws_s3.a", row.Item.Address(), "filter should hide aws_s3.a from the visible tree")
	}

	resources := m.selectedResources()

	require.Len(t, resources, 1)
	assert.Equal(t, "aws_s3.a", resources[0].Address)
}

func TestSelectedResources_IndependentOfHideUnchanged(t *testing.T) {
	m := newTestModelWithResources([]*terraform.Resource{
		{Address: "aws_s3.a", Action: terraform.ActionNoop},
	})
	m.selected = map[string]bool{"aws_s3.a": true}

	m.hideUnchanged = true
	m.rebuildRows()
	require.Empty(t, m.rows, "hideUnchanged should hide the no-op resource")

	resources := m.selectedResources()

	require.Len(t, resources, 1)
	assert.Equal(t, "aws_s3.a", resources[0].Address)
}

func TestIsAddressInSelection(t *testing.T) {
	m := newTestModelEmpty()
	m.selected = map[string]bool{
		"aws_s3.direct":          true,
		"module.a":               true,
		"module.b.module.nested": true,
	}

	tests := []struct {
		addr     string
		expected bool
	}{
		{"aws_s3.direct", true},
		{"module.a.aws_s3.x", true},
		{"module.a.module.b.aws_s3.y", true},
		{"module.b.module.nested.aws_s3.z", true},
		{"module.b.aws_s3.unrelated", false},
		{"aws_s3.unselected", false},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			assert.Equal(t, tt.expected, m.isAddressInSelection(tt.addr))
		})
	}
}

func TestSelectedResources_NestedModules(t *testing.T) {
	m := newTestModelWithResources([]*terraform.Resource{
		{Address: "module.a.module.b.aws_s3.x", Module: "module.a.module.b", Action: terraform.ActionCreate},
		{Address: "module.a.module.b.aws_s3.y", Module: "module.a.module.b", Action: terraform.ActionCreate},
		{Address: "module.a.aws_s3.z", Module: "module.a", Action: terraform.ActionCreate},
	})
	m.selected = map[string]bool{"module.a": true}

	resources := m.selectedResources()

	addrs := make([]string, len(resources))
	for i, r := range resources {
		addrs[i] = r.Address
	}
	assert.Len(t, resources, 3)
	assert.Contains(t, addrs, "module.a.module.b.aws_s3.x")
	assert.Contains(t, addrs, "module.a.module.b.aws_s3.y")
	assert.Contains(t, addrs, "module.a.aws_s3.z")
}

func TestAdjustOffset(t *testing.T) {
	visible := 48 - listViewReservedRows
	tests := []struct {
		name     string
		cursor   int
		offset   int
		expected int
	}{
		{name: "cursor_went_below_visible", cursor: visible, offset: 0, expected: 1},
		{name: "cursor_went_above_visible", cursor: 0, offset: 1, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModelEmpty()

			m.cursor = tt.cursor
			m.offset = tt.offset
			m.adjustOffset()

			assert.Equal(t, tt.expected, m.offset)
		})
	}
}

func TestIsAncestor(t *testing.T) {
	assert.True(t, isAncestor("module.a", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.b", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.a", "aws_s3.x"))
}

func TestParentModuleAddr(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{name: "normal resource", address: "module.a.aws_s3.x", expected: "module.a"},
		{name: "normal module", address: "module.a.module.b", expected: "module.a"},
		{name: "resource no parent", address: "aws_s3.x", expected: ""},
		{name: "module no parent", address: "module.a", expected: ""},
		{name: "resource with module bracket and dot", address: "module.vpc[\"a.b\"].aws_s3.x", expected: "module.vpc[\"a.b\"]"},
		{name: "data under module", address: "module.api.data.aws_region.current", expected: "module.api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parentModuleAddr(tt.address))
		})
	}
}
