package ui

import (
	"errors"
	"fmt"
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

	view := m.View()

	assert.Contains(t, view.Content, "+ aws_s3_bucket.a")
	assert.Contains(t, view.Content, "~ aws_s3_bucket.b (drift)")
	assert.Contains(t, view.Content, "- aws_iam_role.c (delete_because_no_resource_config)")
	assert.Contains(t, view.Content, "+/- aws_db_instance.d (cannot_update)")
	assert.Contains(t, view.Content, "data.aws_region.current")
	assert.Contains(t, view.Content, "aws_vpc.main")
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

func TestModel_ViewShowsError(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.err = fmt.Errorf("something broke")

	view := m.View()

	assert.Contains(t, view.Content, "error occurred: something broke")
}
