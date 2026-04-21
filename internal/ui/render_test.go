package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderListView_ShowsResources(t *testing.T) {
	m := newTestModel()
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
				if !isUnchanged(r) {
					assert.Contains(t, line, actionAnsiString(r.Action))
				}
			}
		}
	}
}

func TestRenderListView_ShowsCursor(t *testing.T) {
	m := newTestModel()
	m.cursor = 1

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	var lineA, lineB string
	for _, line := range lines {
		if strings.Contains(line, testResources[0].Address) {
			lineA = line
		}
		if strings.Contains(line, testResources[1].Address) {
			lineB = line
		}
	}

	assert.NotContains(t, lineA, cursorAnsiString)
	assert.Contains(t, lineB, cursorAnsiString)
}

func TestRenderListView_ShowsSelected(t *testing.T) {
	m := newTestModel()
	m.selected = map[string]bool{m.resources[1].Address: true}

	view := m.View()
	for line := range strings.SplitSeq(view.Content, "\n") {
		if strings.Contains(line, m.resources[1].Address) {
			assert.Contains(t, line, ansiString)
		}
	}
}

func TestRenderListView_OnlyRendersVisibleSlice(t *testing.T) {
	var resources []terraform.Resource
	for i := range 5 {
		resources = append(resources, terraform.Resource{
			Address: fmt.Sprintf("aws_s3_bucket.bucket_%d", i),
			Action:  terraform.ActionNoop,
		})
	}
	m := newTestModelWithResources(resources)
	m.viewHeight = 3 + defaultReservedRows + 1 // Since the width is small

	view := m.View()

	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_0")
	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_1")
	assert.Contains(t, view.Content, "aws_s3_bucket.bucket_2")

	assert.NotContains(t, view.Content, "aws_s3_bucket.bucket_3")
	assert.NotContains(t, view.Content, "aws_s3_bucket.bucket_4")
}

func TestRenderListView_ViewShowsSpinner(t *testing.T) {
	m := newTestModelEmpty()
	m.isRunning = true

	view := m.View()

	assert.Contains(t, view.Content, "Scanning...")
	assert.Contains(t, view.Content, "0 resources")
}

func TestRenderListView_ViewShowsCompleteWhenDone(t *testing.T) {
	m := newTestModelWithResources(make([]terraform.Resource, 5))
	m.isRunning = false

	view := m.View()

	assert.Contains(t, view.Content, "Scan Complete")
	assert.Contains(t, view.Content, "5 resources")
	assert.NotContains(t, view.Content, "Scanning...")
}

func TestRenderListView_ViewShowsSelectedCount(t *testing.T) {
	m := newTestModel()
	m.selected = map[string]bool{m.resources[0].Address: true}

	view := m.View()

	assert.Contains(t, view.Content, "1 selected")
}

func TestRenderListView_ViewShowsFilterCount(t *testing.T) {
	m := newTestModel()
	m.filteredIdx = []int{2}
	m.filterInput.SetValue("iam")

	view := m.View()

	assert.Contains(t, view.Content, "showing 1")
	assert.Contains(t, view.Content, fmt.Sprintf("%d resources found", len(testResources)))
}

func TestRenderListView_ShowsWarningCount(t *testing.T) {
	m := newTestModel()
	m.isRunning = false
	m.diagnostics = []terraform.Diagnostic{
		{Severity: "warning", Summary: "Deprecated"},
		{Severity: "warning", Summary: "Also deprecated"},
	}

	view := m.View()

	assert.Contains(t, view.Content, "2 warnings")
}

func TestRenderListView_ShowsHideUnchangedInfo(t *testing.T) {
	m := newTestModelEmpty()

	require.Contains(t, m.View().Content, "hide unchanged")

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'h'})
	m = newModel.(Model)

	assert.Contains(t, m.View().Content, "show unchanged")
}

func TestRenderActionPickerView_ShowsActionPicker(t *testing.T) {
	m := newTestModel()
	m.viewState = viewActionPicker
	m.selected = map[string]bool{m.resources[0].Address: true, m.resources[1].Address: true}

	view := m.View()

	assert.Contains(t, view.Content, "2 resource(s) selected")
	assert.Contains(t, view.Content, "plan")
	assert.Contains(t, view.Content, "apply")
	assert.Contains(t, view.Content, "destroy")
	assert.Contains(t, view.Content, "taint")
	assert.Contains(t, view.Content, "untaint")
	assert.Contains(t, view.Content, "Esc to cancel")
}

func TestRenderConfirmView_ShowsResourcesAndButtons(t *testing.T) {
	m := newTestModel()
	m.selected = map[string]bool{m.resources[0].Address: true, m.resources[2].Address: true}
	m.viewState = viewConfirm
	m.actionCursor = 1

	view := m.View()

	assert.Contains(t, view.Content, "apply 2 resource(s)?")
	assert.Contains(t, view.Content, m.resources[0].Address)
	assert.Contains(t, view.Content, m.resources[2].Address)
	assert.Contains(t, view.Content, "Cancel")
	assert.Contains(t, view.Content, "Confirm")
}

func TestRenderConfirmView_TruncatesLongSelections(t *testing.T) {
	m := newTestModelEmpty()
	m.viewState = viewConfirm

	for i := range 15 {
		addr := fmt.Sprintf("aws_s3_bucket.b_%02d", i)
		m.resources = append(m.resources, terraform.Resource{
			Address: addr, Action: terraform.ActionDelete,
		})
		m.resourceIndexMap[addr] = i
		m.selected[addr] = true
	}

	view := m.View()

	assert.Contains(t, view.Content, "... and 5 more")
	assert.Contains(t, view.Content, "aws_s3_bucket.b_00")
	assert.Contains(t, view.Content, "aws_s3_bucket.b_09")
	assert.NotContains(t, view.Content, "aws_s3_bucket.b_10")
}

func TestRenderOutputView_ShowsContent(t *testing.T) {
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.actionCursor = 1
	m.isRunning = true
	m.outputLines = []string{
		"aws_s3_bucket.uploads: Modifying...",
		"aws_s3_bucket.uploads: Modifications complete after 2s",
		"Apply complete! Resources: 0 added, 1 changed, 0 destroyed.",
	}

	view := m.View()

	assert.Contains(t, view.Content, "terraform apply")
	assert.Contains(t, view.Content, "aws_s3_bucket.uploads: Modifying...")
	assert.Contains(t, view.Content, "Apply complete!")
}

func TestRenderOutputView_HelpTextChangesWhenDone(t *testing.T) {
	m := newTestModelEmpty()
	m.viewState = viewOutput
	m.isRunning = true

	view := m.View()
	assert.Contains(t, view.Content, "Running...")
	assert.NotContains(t, view.Content, "Esc to close")

	m.isRunning = false
	view = m.View()
	assert.Contains(t, view.Content, "Esc to close")
	assert.NotContains(t, view.Content, "Running...")
}

func TestRenderShutdownLayer_ShowsWaitingMessage(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.forceQuitReady = false

	view := m.View()

	assert.Contains(t, view.Content, "Waiting for terraform to finish...")
	assert.NotContains(t, view.Content, "Press q or ctrl+c again to force quit")
}

func TestRenderShutdownLayer_ShowsForceQuitAfterTimeout(t *testing.T) {
	m := newTestModel()
	m.isQuitting = true
	m.forceQuitReady = true

	view := m.View()

	assert.Contains(t, view.Content, "Waiting for terraform to finish...")
	assert.Contains(t, view.Content, "Press q or ctrl+c again to force quit")
}

func TestRenderErrorView_ShowsDiagnosticsAndError(t *testing.T) {
	m := newTestModelEmpty()
	m.viewState = viewError
	m.diagnostics = []terraform.Diagnostic{
		{Severity: "error", Summary: "Invalid reference", Detail: "Resource not declared"},
		{Severity: "warning", Summary: "Deprecated attribute"},
	}
	m.err = fmt.Errorf("failed to start terraform")

	view := m.View()

	assert.Contains(t, view.Content, "Scanning Failed")
	assert.Contains(t, view.Content, "Invalid reference")
	assert.Contains(t, view.Content, "Resource not declared")
	assert.Contains(t, view.Content, "Deprecated attribute")
	assert.Contains(t, view.Content, "failed to start terraform")
	assert.Contains(t, view.Content, "Esc")
}
