package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testResourceAddr   = "aws_s3_bucket.uploads"
	testDataSourceAddr = "data.aws_caller_identity.current"
)

func TestModel_Handle_RefreshComplete(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	event := terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionNoop,
		},
	}

	newModel, cmd := m.Update(streamEventMsg(event))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Equal(t, testResourceAddr, m.resources[0].Address)
	assert.Equal(t, terraform.ActionNoop, m.resources[0].Action)
	assert.NotNil(t, cmd)
}

func TestModel_Handle_DataSourceRead(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	event := terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testDataSourceAddr,
			Action:  terraform.ActionRead,
		},
	}

	newModel, cmd := m.Update(streamEventMsg(event))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Equal(t, testDataSourceAddr, m.resources[0].Address)
	assert.Equal(t, terraform.ActionRead, m.resources[0].Action)
	assert.NotNil(t, cmd)
}

func TestModel_UpdateExistingResource(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionUpdate,
		},
	}))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Equal(t, testResourceAddr, m.resources[0].Address)
	assert.Equal(t, terraform.ActionUpdate, m.resources[0].Action)
	assert.NotNil(t, cmd)
}

func TestModel_DriftExistingResource(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionUpdate,
			Reason:  "drift",
		},
	}))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Equal(t, testResourceAddr, m.resources[0].Address)
	assert.Equal(t, terraform.ActionUpdate, m.resources[0].Action)
	assert.Equal(t, "drift", m.resources[0].Reason)
	assert.NotNil(t, cmd)
}

func TestModel_HandleError(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	errMsg := "terraform plan failed"
	event := terraform.StreamEvent{
		Error: errors.New(errMsg),
	}

	newModel, cmd := m.Update(streamEventMsg(event))
	m = newModel.(Model)

	assert.EqualError(t, m.err, errMsg)
	assert.NotNil(t, cmd) // should continue listening for events
}

func TestModel_ScanComplete(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	assert.True(t, m.isScanning)

	newModel, cmd := m.Update(scanCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isScanning)
	assert.Nil(t, cmd)
}

func TestModel_Quit(t *testing.T) {
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

func TestModel_CursorNavigation(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}

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

func TestModel_ScrollsUpWithCursor(t *testing.T) {
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

func TestModel_ToggleSelect(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
	}

	// Select first resource
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.True(t, m.selected["aws_s3_bucket.a"])

	// Deselect it
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.False(t, m.selected["aws_s3_bucket.a"])
}

func TestModel_SelectEmptyList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	assert.Empty(t, m.selected)
}

func TestModel_ViewShowsResources(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionCreate},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate, Reason: "drift"},
		{Address: "aws_iam_role.c", Action: terraform.ActionDelete, Reason: "delete_because_no_resource_config"},
		{Address: "aws_db_instance.d", Action: terraform.ActionReplace, Reason: "cannot_update"},
		{Address: "data.aws_region.current", Action: terraform.ActionRead},
		{Address: "aws_vpc.main", Action: terraform.ActionNoop},
	}
	m.viewHeight = len(m.resources) + defaultReservedRows

	view := m.View()

	assert.Contains(t, view.Content, "+ aws_s3_bucket.a")
	assert.Contains(t, view.Content, "~ aws_s3_bucket.b (drift)")
	assert.Contains(t, view.Content, "- aws_iam_role.c (delete_because_no_resource_config)")
	assert.Contains(t, view.Content, "+/- aws_db_instance.d (cannot_update)")
	assert.Contains(t, view.Content, "data.aws_region.current")
	assert.Contains(t, view.Content, "aws_vpc.main")
}

func TestModel_ViewShowsCursor(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate},
	}
	m.viewHeight = len(m.resources) + defaultReservedRows
	m.cursor = 1

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	ansiString := "\x1b["

	assert.NotContains(t, lines[0], ansiString)
	assert.Contains(t, lines[1], ansiString)
}

func TestModel_ViewShowsSelected(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate},
	}
	m.viewHeight = len(m.resources) + defaultReservedRows
	m.cursor = 0
	m.selected = map[string]bool{m.resources[1].Address: true}

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	ansiString := "\x1b["

	assert.Contains(t, lines[1], ansiString)
}

func TestModel_ViewOnlyRendersVisibleSlice(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 3 + defaultReservedRows

	for i := range 5 {
		m.resources = append(m.resources, terraform.Resource{
			Address: fmt.Sprintf("aws_s3_bucket.bucket_%d", i),
			Action:  terraform.ActionNoop,
		})
	}

	view := m.View()

	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_0")
	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_1")
	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_2")

	assert.NotContains(t, view.Content, "aws_s3_bucket.bucket_3")
	assert.NotContains(t, view.Content, "aws_s3_bucket.bucket_4")
}

func TestModel_ViewShowsSpinner(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.isScanning = true

	view := m.View()

	assert.Contains(t, view.Content, "Scanning...")
	assert.Contains(t, view.Content, "0 resources")
}

func TestModel_ViewShowsCompleteWhenDone(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.isScanning = false
	m.resources = make([]terraform.Resource, 5)

	view := m.View()

	assert.Contains(t, view.Content, "Scan Complete")
	assert.Contains(t, view.Content, "5 resources")
	assert.NotContains(t, view.Content, "Scanning...")
}

func TestModel_ViewShowsSelectedCount(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 5 + defaultReservedRows
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
	}
	m.selected = map[string]bool{"aws_s3_bucket.a": true}

	view := m.View()

	assert.Contains(t, view.Content, "1 selected")
}

func TestModel_ViewShowsError(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.err = fmt.Errorf("something broke")

	view := m.View()

	assert.Contains(t, view.Content, "error occurred: something broke")
}
