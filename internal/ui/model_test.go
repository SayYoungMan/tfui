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
	ansiString         = "\x1b["
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

func TestModel_ViewShowsResources(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})

	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionCreate},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate, Reason: "drift"},
		{Address: "aws_iam_role.c", Action: terraform.ActionDelete, Reason: "delete_because_no_resource_config"},
		{Address: "aws_db_instance.d", Action: terraform.ActionReplace, Reason: "cannot_update"},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionImport},
		{Address: "data.aws_region.current", Action: terraform.ActionRead},
		{Address: "aws_vpc.main", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 1, 2, 3, 4, 5, 6}
	m.viewHeight = len(m.resources) + defaultReservedRows
	m.cursor = -1 // Remove cursor so that it doesn't color any of the lines

	view := m.View()

	assert.Contains(t, view.Content, "+ aws_s3_bucket.a")
	assert.Contains(t, view.Content, "~ aws_s3_bucket.b (drift)")
	assert.Contains(t, view.Content, "- aws_iam_role.c (delete_because_no_resource_config)")
	assert.Contains(t, view.Content, "+/- aws_db_instance.d (cannot_update)")
	assert.Contains(t, view.Content, "↓ aws_s3_bucket.c")
	assert.Contains(t, view.Content, "data.aws_region.current")
	assert.Contains(t, view.Content, "aws_vpc.main")

	// Check for ANSI string for color
	for _, r := range m.resources {
		for line := range strings.SplitSeq(view.Content, "\n") {
			if strings.Contains(line, r.Address) {
				if r.Action == terraform.ActionNoop || r.Action == terraform.ActionRead {
					assert.NotContains(t, line, ansiString)
				} else {
					assert.Contains(t, line, ansiString)
				}
			}
		}
	}
}

func TestModel_ViewShowsCursor(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate},
	}
	m.filteredIdx = []int{0, 1}
	m.viewHeight = len(m.resources) + defaultReservedRows
	m.cursor = 1

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	var lineA, lineB string
	for _, line := range lines {
		if strings.Contains(line, "aws_s3_bucket.a") {
			lineA = line
		}
		if strings.Contains(line, "aws_s3_bucket.b") {
			lineB = line
		}
	}

	assert.NotContains(t, lineA, ansiString)
	assert.Contains(t, lineB, ansiString)
}

func TestModel_CursorOperatesOnFilteredList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_lambda_function.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 2}
	m.cursor = 0

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)

	var theLine string
	for line := range strings.SplitSeq(m.View().Content, "\n") {
		if strings.Contains(line, "aws_s3_bucket.c") {
			theLine = line
		}
	}
	assert.Equal(t, 1, m.cursor)
	assert.Contains(t, theLine, ansiString)
}

func TestModel_SelectOnFilteredList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_lambda_function.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 2}
	m.cursor = 1

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	assert.True(t, m.selected["aws_s3_bucket.c"])
	assert.False(t, m.selected["aws_lambda_function.b"])
}

func TestModel_NewResourcesFilterMatch(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 3 + defaultReservedRows
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
	}
	m.indexMap = map[string]int{"aws_s3_bucket.a": 0}
	m.filterInput.SetValue("s3")
	m.filteredIdx = []int{0}

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: "aws_s3_bucket.b",
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	assert.Len(t, m.filteredIdx, 2)

	newModel, _ = m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: "aws_lambda_function.api",
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	assert.Len(t, m.filteredIdx, 2)
	assert.Len(t, m.resources, 3)
}

func TestModel_ViewShowsSelected(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate},
	}
	m.filteredIdx = []int{0, 1}
	m.viewHeight = len(m.resources) + defaultReservedRows
	m.cursor = 0
	m.selected = map[string]bool{m.resources[1].Address: true}

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	var lineB string
	for _, line := range lines {
		if strings.Contains(line, "aws_s3_bucket.b") {
			lineB = line
		}
	}

	assert.Contains(t, lineB, ansiString)
}

func TestModel_ViewOnlyRendersVisibleSlice(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 3 + defaultReservedRows
	m.filteredIdx = []int{0, 1, 2, 3, 4}

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

func TestModel_ViewShowsFilterCount(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.viewHeight = 2 + defaultReservedRows
	m.isScanning = false
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_lambda_function.b", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0}
	m.filterInput.SetValue("s3")

	view := m.View()

	assert.Contains(t, view.Content, "showing 1")
	assert.Contains(t, view.Content, "2 resources found")
}

func TestModel_ViewShowsError(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.err = fmt.Errorf("something broke")

	view := m.View()

	assert.Contains(t, view.Content, "error occurred: something broke")
}
