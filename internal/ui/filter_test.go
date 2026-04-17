package ui

import (
	"testing"

	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
)

func TestRebuildFilter_EmptyShowsAll(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("")
	m.rebuildFilter()

	assert.Len(t, m.filteredIdx, len(testResources))
}

func TestRebuildFilter_MatchesSubset(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("s3")
	m.rebuildFilter()

	addrs := make([]string, len(m.filteredIdx))
	for i, idx := range m.filteredIdx {
		addrs[i] = m.resources[idx].Address
	}

	assert.Len(t, m.filteredIdx, 3)
	assert.Contains(t, addrs, testResources[0].Address)
	assert.Contains(t, addrs, testResources[1].Address)
	assert.Contains(t, addrs, testResources[4].Address)
}

func TestRebuildFilter_NoMatch(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("zzzzz")
	m.rebuildFilter()

	assert.Empty(t, m.filteredIdx)
}

func TestRebuildFilter_ResetsCursorAndOffset(t *testing.T) {
	m := newTestModel()
	m.cursor = 2
	m.offset = 1

	m.filterInput.SetValue("s3")
	m.rebuildFilter()

	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, 0, m.offset)
}

func TestRebuildFilter_HideUnchanged(t *testing.T) {
	m := newTestModel()

	m.hideUnchanged = true
	m.rebuildFilter()

	assert.Len(t, m.filteredIdx, 5)
}

func TestRebuildFilter_HideUnchangedFiltered(t *testing.T) {
	m := newTestModel()

	m.filterInput.SetValue("aws_vpc")
	m.hideUnchanged = true
	m.rebuildFilter()

	assert.Empty(t, m.filteredIdx)
}

func TestMatchesFilter_EmptyAlwaysTrue(t *testing.T) {
	m := newTestModelEmpty()

	r := terraform.Resource{Address: "anything"}
	assert.True(t, m.matchesFilter(r))
}

func TestMatchesFilter_MatchAndMiss(t *testing.T) {
	m := newTestModelEmpty()
	m.filterInput.SetValue("s3")

	assert.True(t, m.matchesFilter(terraform.Resource{Address: "aws_s3_bucket.a"}))
	assert.False(t, m.matchesFilter(terraform.Resource{Address: "aws_lambda_function.b"}))
}

func TestMatchesFilter_NoopIgnored(t *testing.T) {
	m := newTestModelEmpty()
	m.hideUnchanged = true

	assert.False(t, m.matchesFilter(terraform.Resource{Address: "resource", Action: terraform.ActionNoop}))
	assert.False(t, m.matchesFilter(terraform.Resource{Address: "resource", Action: terraform.ActionRead}))
}
