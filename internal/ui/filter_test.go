package ui

import (
	"testing"

	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRebuildRows_EmptyShowsAll(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("")
	m.rebuildRows()

	assert.Len(t, m.rows, len(testResources))
}

func TestRebuildRows_MatchesSubset(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("s3")
	m.rebuildRows()

	addrs := make([]string, len(m.rows))
	for i, row := range m.rows {
		addrs[i] = row.Address
	}

	assert.Len(t, m.rows, 3)
	assert.Contains(t, addrs, testResources[2].Address)
	assert.Contains(t, addrs, testResources[3].Address)
	assert.Contains(t, addrs, testResources[4].Address)
}

func TestRebuildRows_NoMatch(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("zzzzz")
	m.rebuildRows()

	assert.Empty(t, m.rows)
}

func TestRebuildRows_HideUnchanged(t *testing.T) {
	m := newTestModel()

	m.hideUnchanged = true
	m.rebuildRows()

	assert.Len(t, m.rows, 5)
}

func TestRebuildRows_HideUnchangedFiltered(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("aws_vpc")
	m.hideUnchanged = true
	m.rebuildRows()

	assert.Empty(t, m.rows)
}

func TestRebuildRows_CursorLastWhenRowsShrink(t *testing.T) {
	m := newTestModel()
	m.cursor = len(m.rows) - 1

	m.filterInput.SetValue("s3")
	m.rebuildRows()

	assert.Equal(t, len(m.rows)-1, m.cursor)
}

func TestRebuildRows_Collapse(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "module.a.aws_s3.x", Module: "module.a", Action: terraform.ActionCreate},
		{Address: "module.a.aws_s3.y", Module: "module.a", Action: terraform.ActionCreate},
	}
	m := newTestModelWithResources(resources)
	require.Len(t, m.rows, 3)

	m.collapsed["module.a"] = true
	m.rebuildRows()

	assert.Len(t, m.rows, 1)
	assert.Equal(t, "module.a", m.rows[0].Address)
}

func TestRebuildRows_FilterIncludesParent(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "module.a.aws_s3.x", Module: "module.a", Action: terraform.ActionCreate},
	}
	m := newTestModelWithResources(resources)

	m.filterInput.SetValue("s3")
	m.rebuildRows()

	assert.Len(t, m.rows, 2)
	assert.Equal(t, "module.a", m.rows[0].Address)
	assert.Equal(t, "module.a.aws_s3.x", m.rows[1].Address)
}

func TestTreePrefix(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "module.a.module.b.aws_s3.x", Module: "module.a.module.b", Action: terraform.ActionCreate},
		{Address: "module.a.module.c.aws_s3.y", Module: "module.a.module.c", Action: terraform.ActionCreate},
	}
	m := newTestModelWithResources(resources)

	//   module.a              prefix: ""
	//   ├─ module.b           prefix: "├─ "
	//   │  └─ aws_s3.x        prefix: "│  └─ "
	//   └─ module.c           prefix: "└─ "
	//      └─ aws_s3.y        prefix: "   └─ "
	expected := []string{"", "├─ ", "│  └─ ", "└─ ", "   └─ "}
	for i, exp := range expected {
		assert.Equal(t, exp, m.rows[i].TreePrefix)
	}
}

func TestParentModule(t *testing.T) {
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
			assert.Equal(t, tt.expected, parentModule(tt.address))
		})
	}
}

func TestIsAncestor(t *testing.T) {
	assert.True(t, isAncestor("module.a", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.b", "module.a.module.b.aws_s3.x"))
	assert.False(t, isAncestor("module.a", "aws_s3.x"))
}
