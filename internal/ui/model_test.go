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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

	assert.True(t, m.isScanning)

	newModel, cmd := m.Update(scanCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isScanning)
	assert.Nil(t, cmd)
}

func TestModel_CursorOperatesOnFilteredList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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

func TestModel_OutputLineMsg(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	outputCh := make(chan string, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewOutput
	m.isOutputing = true
	m.outputChannel = outputCh
	m.viewHeight = 20

	newModel, cmd := m.Update(outputLineMsg("first line"))
	m = newModel.(Model)

	require.Len(t, m.outputLines, 1)
	assert.Equal(t, "first line", m.outputLines[0])
	assert.NotNil(t, cmd)
}

func TestModel_OutputCompleteMsg(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewOutput
	m.isOutputing = true

	newModel, cmd := m.Update(outputCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isOutputing)
	assert.Nil(t, cmd)
}

func TestModel_MouseWheelScrollsList(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewList
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m.filteredIdx = []int{0, 1, 2}

	// Scroll down
	newModel, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// Scroll down again
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)

	// Clamp at bottom
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)

	// Scroll up
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// Back to top
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor)

	// Clamp at top
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor)
}

func TestModel_MouseWheelScrollsOutput(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewOutput
	m.viewHeight = defaultReservedOutputRows + 2 // 2 visible rows
	m.outputLines = []string{"line 0", "line 1", "line 2", "line 3", "line 4"}

	// Scroll down
	newModel, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 1, m.outputOffset)

	// Scroll to max
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 3, m.outputOffset) // 5 lines - 2 visible = 3

	// Clamp at bottom
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 3, m.outputOffset)

	// Scroll back up
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 2, m.outputOffset)

	// Back to top
	m.outputOffset = 0
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.outputOffset)
}
