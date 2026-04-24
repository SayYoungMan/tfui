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
			m := newTestModelEmpty()

			newModel, _ := m.Update(tt.msg)
			m = newModel.(Model)
			assert.Equal(t, confirmQuitState, m.quitState)
		})
	}
}

func TestNormalModeKeys_CursorNavigation(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.b", Action: terraform.ActionNoop},
		{Address: "aws_s3_bucket.c", Action: terraform.ActionNoop},
	}
	m := newTestModelWithResources(resources)

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
	m := newTestModelEmpty()
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

func TestNormalModeKeys_ToggleHideUnchanged(t *testing.T) {
	m := newTestModel()

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'H'})
	m = newModel.(Model)
	assert.True(t, m.hideUnchanged)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'H'})
	m = newModel.(Model)
	assert.False(t, m.hideUnchanged)
}

func TestNormalModeKeys_ToggleSelect(t *testing.T) {
	m := newTestModel()

	// Select first resource
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.True(t, m.selected[m.resources[0].Address])

	// Deselect it
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)
	assert.False(t, m.selected[m.resources[0].Address])
}

func TestNormalModeKeys_SelectEmptyList(t *testing.T) {
	m := newTestModelEmpty()

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	assert.Empty(t, m.selected)
}

func TestNormalModeKeys_RemoveSelectionIfParentSelected(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "module.a.aws_s3.x", Module: "module.a", Action: terraform.ActionCreate},
	}
	m := newTestModelWithResources(resources)
	m.selected["module.a.aws_s3.x"] = true

	// Cursor is already on module m.cursor = 0
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	assert.Contains(t, m.selected, "module.a")
	assert.NotContains(t, m.selected, "module.a.aws_s3.x")

	m.cursor = 1
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = newModel.(Model)

	// Ignore child selection if parent selected
	assert.Contains(t, m.selected, "module.a")
	assert.NotContains(t, m.selected, "module.a.aws_s3.x")
}

func TestNormalModeKeys_ActionBlockedWhileScanning(t *testing.T) {
	m := newTestModel()
	m.isRunning = true
	m.selected = map[string]bool{m.resources[0].Address: true}

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)

	assert.Equal(t, viewList, m.viewState)
}

func TestNormalModeKeys_ActionBlockedWithNoSelection(t *testing.T) {
	m := newTestModel()
	m.isRunning = false

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)

	assert.Equal(t, viewList, m.viewState)
}

func TestNormalModeKeys_RefreshRescan(t *testing.T) {
	m := newTestModel()
	m.isRunning = false
	m.runner = terraform.NewTerraformRunner(t.TempDir(), "true")

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = newModel.(Model)

	assert.True(t, m.isRunning)
	assert.Equal(t, viewList, m.viewState)
	assert.Empty(t, m.selected)
	assert.NotNil(t, cmd)
}

func TestNormalModeKeys_RefreshBlockedWhileScanning(t *testing.T) {
	m := newTestModel()
	m.isRunning = true

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	m = newModel.(Model)

	assert.Nil(t, cmd)
}

func TestNormalModeKeys_TabOpensActionPicker(t *testing.T) {
	m := newTestModel()
	m.isRunning = false
	m.selected = map[string]bool{m.resources[0].Address: true}

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)

	assert.Equal(t, viewActionPicker, m.viewState)
	assert.Equal(t, 0, m.actionCursor)
}

func TestNormalModeKeys_ModuleExpandCollapse(t *testing.T) {
	resources := []terraform.Resource{
		{Address: "module.a.aws_s3.x", Module: "module.a", Action: terraform.ActionCreate},
	}
	m := newTestModelWithResources(resources)

	require.Empty(t, m.collapsed)

	// The cursor on resource not module
	m.cursor = 1
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'h'})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 1)
	require.Equal(t, 0, m.cursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'l'})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 0)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 1)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 0)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 1)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	require.Len(t, m.collapsed, 0)
}

func TestFilterModeKeys_FilterFocusAndUnfocus(t *testing.T) {
	m := newTestModel()

	require.Equal(t, viewList, m.viewState)

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: '/'})
	m = newModel.(Model)
	require.Equal(t, viewFilter, m.viewState)
	assert.NotNil(t, cmd)

	m.filterInput.SetValue("s3")
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(Model)
	require.Equal(t, viewList, m.viewState)
	assert.Equal(t, "s3", m.filterInput.Value())

	newModel, cmd = m.Update(tea.KeyPressMsg{Code: '/'})
	m = newModel.(Model)
	require.Equal(t, viewFilter, m.viewState)
	assert.NotNil(t, cmd)

	m.filterInput.SetValue("s3")
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	require.Equal(t, viewList, m.viewState)
	assert.Equal(t, "s3", m.filterInput.Value())
}

func TestActionPickerKeys_Navigation(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.actionCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.actionCursor)

	// Clamp at top
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.actionCursor)

	// Clamp at bottom
	for range len(actionChoices) {
		newModel, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
		m = newModel.(Model)
	}
	assert.Equal(t, len(actionChoices)-1, m.actionCursor)
}

func TestActionPickerKeys_TabNext(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	require.Equal(t, 1, m.actionCursor)

	m.actionCursor = 4
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	assert.Equal(t, 0, m.actionCursor)
}

func TestActionPickerKeys_EscReturnsToList(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(Model)

	assert.Equal(t, viewList, m.viewState)
}

func TestActionPickerKeys_CursorResetsOnEntry(t *testing.T) {
	m := newTestModel()
	m.isRunning = false
	m.selected = map[string]bool{m.resources[0].Address: true}
	m.actionCursor = 3

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)

	assert.Equal(t, 0, m.actionCursor)
}

func TestActionPickerKeys_EnterGoesConfirmView(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker
	m.actionCursor = 2

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	assert.Equal(t, "destroy", actionChoices[m.actionCursor])
	assert.Equal(t, 0, m.confirmCursor)
	assert.Equal(t, viewConfirm, m.viewState)
}

func TestConfirmKeys_DefaultsToCancel(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker
	m.selected = map[string]bool{m.resources[0].Address: true}

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	assert.Equal(t, 0, m.confirmCursor)
}

func TestConfirmKeys_Navigation(t *testing.T) {
	m := newTestModel()
	m.viewState = viewConfirm

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = newModel.(Model)
	assert.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'h'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = newModel.(Model)
	assert.Equal(t, 0, m.confirmCursor)
}

func TestConfirmKeys_TabAlternate(t *testing.T) {
	m := newTestModel()
	m.viewState = viewConfirm

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	require.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	require.Equal(t, 0, m.confirmCursor)
}

func TestConfirmKeys_CancelToPicker(t *testing.T) {
	m := newTestModel()
	m.viewState = viewConfirm
	m.confirmCursor = 0

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	assert.Equal(t, viewActionPicker, m.viewState)
}

func TestConfirmKeys_ConfirmToOutput(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(terraform.NewTerraformRunner(t.TempDir(), "true"), ch, func() {})
	m.viewState = viewConfirm
	m.confirmCursor = 1

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	assert.Equal(t, viewOutput, m.viewState)
}

func TestConfirmKeys_EscToPicker(t *testing.T) {
	m := newTestModel()
	m.viewState = viewConfirm

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(Model)
	assert.Equal(t, viewActionPicker, m.viewState)
}

func TestQuitConfirmKeys_Navigation(t *testing.T) {
	m := newTestModel()
	m.quitState = confirmQuitState

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = newModel.(Model)
	assert.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'h'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = newModel.(Model)
	assert.Equal(t, 0, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	require.Equal(t, 1, m.confirmCursor)

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = newModel.(Model)
	require.Equal(t, 0, m.confirmCursor)
}

func TestQuitConfirmKeys_Cancel(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "cancel enter", key: tea.KeyPressMsg{Code: tea.KeyEnter}},
		{name: "esc", key: tea.KeyPressMsg{Code: tea.KeyEsc}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			m.quitState = confirmQuitState
			m.confirmCursor = 0

			require.Contains(t, m.View().Content, quitConfirmTitle)

			newModel, _ := m.Update(tt.key)
			m = newModel.(Model)
			assert.NotContains(t, m.View().Content, quitConfirmTitle)
		})
	}
}

func TestQuitConfirmKeys_Confirm(t *testing.T) {
	m := newTestModel()
	m.quitState = confirmQuitState
	m.confirmCursor = 1

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.NotNil(t, cmd)
}

func TestOutputKeys_EscBlockedWhileOutputing(t *testing.T) {
	m := newTestModel()
	m.viewState = viewOutput
	m.isRunning = true

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = newModel.(Model)

	assert.Equal(t, viewOutput, m.viewState)
}

func TestOutputKeys_Navigation(t *testing.T) {
	m := newTestModel()
	m.viewState = viewOutput
	m.viewHeight = defaultReservedOutputRows + 2 // 2 visible rows
	m.outputLines = []string{"line 0", "line 1", "line 2", "line 3"}

	// j scrolls down
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)
	assert.Equal(t, 1, m.offset)

	// k scrolls up
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.offset)

	// Clamp at top
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = newModel.(Model)
	assert.Equal(t, 0, m.offset)

	// Clamp at bottom
	m.offset = 2 // max = 4 lines - 2 visible
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = newModel.(Model)
	assert.Equal(t, 2, m.offset)
}

func TestErrorViewKeys_Quit(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			m.viewState = viewError
			cancelled := false
			m.cancel = func() { cancelled = true }

			_, cmd := m.Update(tt.msg)

			assert.True(t, cancelled)
			assert.NotNil(t, cmd)
		})
	}
}
