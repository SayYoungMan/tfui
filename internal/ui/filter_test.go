package ui

import (
	"testing"

	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
)

var testResources = []terraform.Resource{
	{Address: "aws_s3_bucket.uploads", Action: terraform.ActionCreate},
	{Address: "aws_lambda_function.api", Action: terraform.ActionDelete},
	{Address: "aws_s3_bucket.logs", Action: terraform.ActionNoop},
	{Address: "aws_dynamodb_table.items", Action: terraform.ActionUpdate},
}

func TestRebuildFilter_EmptyShowsAll(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources

	m.filterInput.SetValue("")
	m.rebuildFilter()

	assert.Len(t, m.filteredIdx, 4)
}

func TestRebuildFilter_MatchesSubset(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources

	m.filterInput.SetValue("s3")
	m.rebuildFilter()

	addrs := make([]string, len(m.filteredIdx))
	for i, idx := range m.filteredIdx {
		addrs[i] = m.resources[idx].Address
	}

	assert.Len(t, m.filteredIdx, 2)
	assert.Contains(t, addrs, testResources[0].Address)
	assert.Contains(t, addrs, testResources[2].Address)
}

func TestRebuildFilter_NoMatch(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources

	m.filterInput.SetValue("zzzzz")
	m.rebuildFilter()

	assert.Empty(t, m.filteredIdx)
}

func TestRebuildFilter_ResetsCursorAndOffset(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources
	m.cursor = 2
	m.offset = 1

	m.filterInput.SetValue("s3")
	m.rebuildFilter()

	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, 0, m.offset)
}

func TestRebuildFilter_HideNoops(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources

	m.hideUnchanged = true
	m.rebuildFilter()

	assert.Len(t, m.filteredIdx, 3)
}

func TestRebuildFilter_HideNoopsFiltered(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = testResources

	m.filterInput.SetValue("s3")
	m.hideUnchanged = true
	m.rebuildFilter()

	assert.Len(t, m.filteredIdx, 1)
}

func TestMatchesFilter_EmptyAlwaysTrue(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	r := terraform.Resource{Address: "anything"}
	assert.True(t, m.matchesFilter(r))
}

func TestMatchesFilter_MatchAndMiss(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.filterInput.SetValue("s3")

	assert.True(t, m.matchesFilter(terraform.Resource{Address: "aws_s3_bucket.a"}))
	assert.False(t, m.matchesFilter(terraform.Resource{Address: "aws_lambda_function.b"}))
}

func TestMatchesFilter_NoopIgnored(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	m.hideUnchanged = true

	assert.False(t, m.matchesFilter(terraform.Resource{Address: "resource", Action: terraform.ActionNoop}))
	assert.False(t, m.matchesFilter(terraform.Resource{Address: "resource", Action: terraform.ActionRead}))
}
