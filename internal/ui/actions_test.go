package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitDuration_Pending(t *testing.T) {
	start := time.Now().Add(-5 * time.Second)
	ar := &ActionResource{Status: actionResourcePending}

	d := ar.waitDuration(start)

	assert.InDelta(t, 5, d.Seconds(), 0.5)
}

func TestWaitDuration_ReadStarted(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	ar := &ActionResource{
		ReadStartedAt: time.Now().Add(-7 * time.Second),
	}

	d := ar.waitDuration(start)

	assert.InDelta(t, 3, d.Seconds(), 0.5)
}

func TestWaitDuration_NoRead_ProcessStarted(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	ar := &ActionResource{
		ProcessStartedAt: time.Now().Add(-6 * time.Second),
	}

	d := ar.waitDuration(start)

	assert.InDelta(t, 4, d.Seconds(), 0.5)
}

func TestReadDuration_NoReadPhase(t *testing.T) {
	ar := &ActionResource{}

	assert.Equal(t, time.Duration(0), ar.readDuration())
}

func TestReadDuration_Reading(t *testing.T) {
	ar := &ActionResource{
		ReadStartedAt: time.Now().Add(-3 * time.Second),
	}

	d := ar.readDuration()

	assert.InDelta(t, 3, d.Seconds(), 0.5)
}

func TestReadDuration_ReadCompleted(t *testing.T) {
	ar := &ActionResource{
		ReadStartedAt:   time.Now().Add(-5 * time.Second),
		ReadCompletedAt: time.Now().Add(-2 * time.Second),
	}

	d := ar.readDuration()

	assert.InDelta(t, 3, d.Seconds(), 0.5)
}

func TestProcessDuration_NotStarted(t *testing.T) {
	ar := &ActionResource{}

	assert.Equal(t, time.Duration(0), ar.processDuration())
}

func TestProcessDuration_InProgress(t *testing.T) {
	ar := &ActionResource{
		ProcessStartedAt: time.Now().Add(-4 * time.Second),
	}

	d := ar.processDuration()

	assert.InDelta(t, 4, d.Seconds(), 0.5)
}

func TestProcessDuration_Completed(t *testing.T) {
	ar := &ActionResource{
		ProcessStartedAt:   time.Now().Add(-6 * time.Second),
		ProcessCompletedAt: time.Now().Add(-1 * time.Second),
	}

	d := ar.processDuration()

	assert.InDelta(t, 5, d.Seconds(), 0.5)
}

func TestGracefulQuit_QuitsImmediatelyWhenIdle(t *testing.T) {
	m := newTestModel()
	m.workState = workIdle

	m, cmd := m.keyPresses([]rune{'q', tea.KeyTab, tea.KeyEnter})

	assert.Equal(t, quittingState, m.quitState)
	assert.NotNil(t, cmd)
}

func TestGracefulQuit_WaitsWhenRunning(t *testing.T) {
	m := newTestModel()
	m.workState = workAction
	cancelled := false
	m.cancel.fn = func() { cancelled = true }

	m, cmd := m.keyPresses([]rune{'q', tea.KeyTab, tea.KeyEnter})

	assert.Equal(t, quittingState, m.quitState)
	assert.True(t, cancelled)
	assert.NotNil(t, cmd)

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'q'})

	assert.Nil(t, cmd)
}

func TestGracefulQuit_ForceQuitsAfterTimeout(t *testing.T) {
	m := newTestModel()
	m.quitState = forceQuitReadyState

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_QuitsOnScanComplete(t *testing.T) {
	m := newTestModel()
	m.quitState = quittingState
	m.workState = workPlan

	_, cmd := m.Update(streamCompleteMsg{})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_QuitsOnOutputComplete(t *testing.T) {
	m := newTestModel()
	m.quitState = quittingState
	m.workState = workAction

	_, cmd := m.Update(outputCompleteMsg{})

	assert.NotNil(t, cmd)
}

func TestGracefulQuit_BlocksKeysWhileQuitting(t *testing.T) {
	m := newTestModel()
	m.quitState = quittingState
	m.viewState = viewList

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)

	assert.Equal(t, 0, m.cursor)
}

func TestGracefulQuit_ForceQuitReadyMsg(t *testing.T) {
	m := newTestModel()
	m.quitState = quittingState

	newModel, _ := m.Update(forceQuitReadyMsg{})
	m = newModel.(Model)

	assert.Equal(t, forceQuitReadyState, m.quitState)
}

func TestGracefulQuit_CanCancelInitPull(t *testing.T) {
	runner := terraform.NewTerraformRunner(t.TempDir(), "sleep")
	m := NewModel(runner)

	cmd := m.Init()
	require.NotNil(t, cmd)

	assert.NotNil(t, m.cancel.fn)
}

func TestStartAction_PopulatesActionResources(t *testing.T) {
	m := NewModel(terraform.NewTerraformRunner(t.TempDir(), "true"))
	m.workState = workIdle
	m.selected = map[string]bool{
		"aws_s3_bucket.a": true,
		"aws_s3_bucket.b": true,
	}

	m.actionCursor = 1 // apply
	m.viewState = viewConfirm
	m.confirmCursor = 1
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	assert.Equal(t, viewActionResources, m.viewState)
	assert.Equal(t, workAction, m.workState)
	assert.Len(t, m.actionResources, 2)
	assert.Contains(t, m.actionResources, "aws_s3_bucket.a")
	assert.Contains(t, m.actionResources, "aws_s3_bucket.b")
	assert.Equal(t, actionResourcePending, m.actionResources["aws_s3_bucket.a"].Status)
	assert.Equal(t, actionResourcePending, m.actionResources["aws_s3_bucket.b"].Status)
	assert.False(t, m.actionStartTime.IsZero())
	assert.Nil(t, m.outputLines)
	assert.Equal(t, 0, m.offset)
}
