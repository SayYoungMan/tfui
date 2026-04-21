package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
