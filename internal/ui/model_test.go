package ui

import (
	"errors"
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

func TestModel_HideUnchanged_ResourceBecomesChanged(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(ch, func() {})
	m.hideUnchanged = true

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Empty(t, m.filteredIdx)

	newModel, _ = m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionUpdate,
		},
	}))
	m = newModel.(Model)

	assert.Equal(t, terraform.ActionUpdate, m.resources[0].Action)
	require.Len(t, m.filteredIdx, 1)
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
