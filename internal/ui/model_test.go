package ui

import (
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

func TestModel_HandleRefreshComplete(t *testing.T) {
	m := newTestModelEmpty()

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

func TestModel_HandleDataSourceRead(t *testing.T) {
	m := newTestModelEmpty()

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
	m := newTestModelEmpty()

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
	m := newTestModelEmpty()

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
	m := newTestModelEmpty()
	m.hideUnchanged = true

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	require.Len(t, m.resources, 1)
	assert.Empty(t, m.rows)

	newModel, _ = m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: testResourceAddr,
			Action:  terraform.ActionUpdate,
		},
	}))
	m = newModel.(Model)

	assert.Equal(t, terraform.ActionUpdate, m.resources[0].Action)
	require.Len(t, m.rows, 1)
}

func TestModel_HandleErrorDiagnostic(t *testing.T) {
	m := newTestModelEmpty()

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Diagnostic: &terraform.Diagnostic{
			Severity: "error",
			Summary:  "Invalid reference",
			Detail:   "Resource not declared",
		},
	}))
	m = newModel.(Model)

	require.Len(t, m.diagnostics, 1)
	assert.Equal(t, "error", m.diagnostics[0].Severity)
	assert.Equal(t, "Invalid reference", m.diagnostics[0].Summary)
	assert.NotNil(t, cmd)

	newModel, cmd = m.Update(streamCompleteMsg{})
	m = newModel.(Model)

	assert.Equal(t, viewError, m.viewState)
	assert.False(t, m.isRunning())
}

func TestModel_ScanComplete_WarningsOnly(t *testing.T) {
	m := newTestModelEmpty()
	m.diagnostics = []terraform.Diagnostic{
		{Severity: "warning", Summary: "Deprecated attribute"},
	}

	newModel, _ := m.Update(streamCompleteMsg{})
	m = newModel.(Model)

	assert.Equal(t, viewList, m.viewState)
}

func TestModel_ScanComplete(t *testing.T) {
	m := newTestModelEmpty()
	m.workState = workPlan

	assert.True(t, m.isRunning())

	newModel, cmd := m.Update(streamCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isRunning())
	assert.Nil(t, cmd)
}

func TestModel_CursorOperatesOnFilteredList(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_lambda_function.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m := newTestModelWithResources(resources)
	m.filterInput.SetValue("s3")
	m.rebuildRows()
	m.cursor = 0

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)

	var theLine string
	for line := range strings.SplitSeq(m.View().Content, "\n") {
		if strings.Contains(line, m.resources[2].Address) {
			theLine = line
		}
	}
	assert.Equal(t, 1, m.cursor)
	assert.Contains(t, theLine, cursorAnsiString)
}

func TestModel_NewResourcesFilterMatch(t *testing.T) {
	m := newTestModelWithResources([]terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
	})
	m.filterInput.SetValue("s3")

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: "aws_s3_bucket.b",
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	assert.Len(t, m.rows, 2)

	newModel, _ = m.Update(streamEventMsg(terraform.StreamEvent{
		Resource: &terraform.Resource{
			Address: "aws_lambda_function.api",
			Action:  terraform.ActionNoop,
		},
	}))
	m = newModel.(Model)

	assert.Len(t, m.rows, 2)
	assert.Len(t, m.resources, 3)
}

func TestModel_OutputLineMsg(t *testing.T) {
	outputCh := make(chan string, 1)
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.workState = workAction
	m.outputCh = outputCh

	newModel, cmd := m.Update(outputLineMsg("first line"))
	m = newModel.(Model)

	require.Len(t, m.outputLines, 1)
	assert.Equal(t, "first line", m.outputLines[0])
	assert.NotNil(t, cmd)
}

func TestModel_OutputCompleteMsg(t *testing.T) {
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.workState = workAction

	newModel, cmd := m.Update(outputCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isRunning())
	assert.Nil(t, cmd)
}

func TestModel_MouseWheelScrollsList(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m := newTestModelWithResources(resources)

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
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.viewHeight = defaultReservedOutputRows + 2 // 2 visible rows
	m.outputLines = []string{"line 0", "line 1", "line 2"}

	// Scroll down
	newModel, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 1, m.offset)

	// Clamp at bottom
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 2, m.offset)

	// Scroll back up
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 1, m.offset)

	// Back to top and check clamp top
	m.offset = 0
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.offset)
}
