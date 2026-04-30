package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
