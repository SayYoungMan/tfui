package ui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalModeKeys_Quit(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"q key", tea.KeyPressMsg{Code: 'q'}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan terraform.StreamEvent, 1)
			cancelled := false
			cancel := func() { cancelled = true }
			m := NewModel(ch, cancel)

			_, cmd := m.Update(tt.msg)

			assert.True(t, cancelled)
			assert.NotNil(t, cmd)
		})
	}
}

func TestNormalModeKeys_CursorNavigation(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 1, 2}

	assert.Equal(t, 0, m.cursor)

	// j moves down
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// down arrow moves down
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)

	// Clamps at bottom
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)

	// k moves up
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// up arrow moves up
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor)

	// Clamps at top
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor)
}

func TestNormalModeKeys_ScrollsUpWithCursor(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 3 + defaultReservedRows

	for i := range 10 {
		m.resources = append(m.resources, terraform.Resource{
			Address: fmt.Sprintf("aws_s3_bucket.bucket_%d", i),
			Action:  terraform.ActionNoop,
		})
	}

	m.cursor = 5
	m.offset = 3

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 4, m.cursor)
	assert.Equal(t, 3, m.offset)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 3, m.cursor)
	assert.Equal(t, 3, m.offset)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)
	assert.Equal(t, 2, m.offset) // offset changes -> it scrolled up
}

func TestNormalModeKeys_ToggleSelect(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 1}

	// Select first resource
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.True(t, m.selected["aws_s3_bucket.a"])

	// Deselect it
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.False(t, m.selected["aws_s3_bucket.a"])
}

func TestNormalModeKeys_SelectEmptyList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	assert.Empty(t, m.selected)
}

func TestFilterModeKeys_FilterFocusAndUnfocus(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	require.False(t, m.filterFocused)

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: '/'})
	m = newModel.(Model)
	assert.True(t, m.filterFocused)
	assert.NotNil(t, cmd)

	m.filterInput.SetValue("s3")
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(Model)
	assert.False(t, m.filterFocused)
	assert.Equal(t, "s3", m.filterInput.Value())

	newModel, cmd = m.Update(tea.KeyPressMsg{Code: '/'})
	m = newModel.(Model)
	assert.True(t, m.filterFocused)
	assert.NotNil(t, cmd)

	m.filterInput.SetValue("s3")
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	assert.False(t, m.filterFocused)
	assert.Equal(t, "s3", m.filterInput.Value())
}
