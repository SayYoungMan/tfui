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

func TestModel_Handle_RefreshComplete(t *testing.T) {
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

func TestModel_Handle_DataSourceRead(t *testing.T) {
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

	newModel, cmd = m.Update(scanCompleteMsg{})
	m = newModel.(Model)

	assert.Equal(t, viewError, m.viewState)
	assert.False(t, m.isRunning)
}

func TestModel_ScanComplete_WarningsOnly(t *testing.T) {
	m := newTestModelEmpty()
	m.diagnostics = []terraform.Diagnostic{
		{Severity: "warning", Summary: "Deprecated attribute"},
	}

	newModel, _ := m.Update(scanCompleteMsg{})
	m = newModel.(Model)

	assert.Equal(t, viewList, m.viewState)
}

func TestModel_ScanComplete(t *testing.T) {
	m := newTestModelEmpty()

	assert.True(t, m.isRunning)

	newModel, cmd := m.Update(scanCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isRunning)
	assert.Nil(t, cmd)
}

func TestModel_CursorOperatesOnFilteredList(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_lambda_function.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m := newTestModelWithResources(resources)
	m.filteredIdx = []int{0, 2}
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
	assert.Contains(t, theLine, ansiString)
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
	outputCh := make(chan string, 1)
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.isRunning = true
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
	m.isRunning = true

	newModel, cmd := m.Update(outputCompleteMsg{})
	m = newModel.(Model)

	assert.False(t, m.isRunning)
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
	m.outputLines = []string{"line 0", "line 1", "line 2", "line 3", "line 4"}

	// Scroll down
	newModel, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 1, m.offset)

	// Scroll to max
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 3, m.offset) // 5 lines - 2 visible = 3

	// Clamp at bottom
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = newModel.(Model)
	assert.Equal(t, 3, m.offset)

	// Scroll back up
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 2, m.offset)

	// Back to top
	m.offset = 0
	newModel, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = newModel.(Model)
	assert.Equal(t, 0, m.offset)
}

func TestGracefulQuit_QuitsImmediatelyWhenIdle(t *testing.T) {
	m := newTestModel()
	m.isRunning = false

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = newModel.(Model)

	assert.True(t, m.isQuitting)
	assert.NotNil(t, cmd)
}

func TestGracefulQuit_WaitsWhenRunning(t *testing.T) {
	m := newTestModel()
	m.isRunning = true
	cancelled := false
	m.cancel = func() { cancelled = true }

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = newModel.(Model)

	assert.True(t, m.isQuitting)
	assert.True(t, cancelled)
	assert.NotNil(t, cmd)

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'q'})

	assert.Nil(t, cmd)
}

func TestGracefulQuit_ForceQuitsAfterTimeout(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.forceQuitReady = true

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_QuitsOnScanComplete(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.isRunning = true

	_, cmd := m.Update(scanCompleteMsg{})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_QuitsOnOutputComplete(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.isRunning = true

	_, cmd := m.Update(outputCompleteMsg{})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_BlocksKeysWhileQuitting(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.viewState = viewList

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)

	assert.Equal(t, 0, m.cursor)
}

func TestGracefulQuit_ForceQuitReadyMsg(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true

	newModel, _ := m.Update(forceQuitReadyMsg{})
	m = newModel.(Model)

	assert.True(t, m.forceQuitReady)
}
