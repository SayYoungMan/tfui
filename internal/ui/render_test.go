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
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

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

func TestRenderListView_ShowsCursor(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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

func TestRenderListView_ShowsSelected(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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

func TestRenderListView_OnlyRendersVisibleSlice(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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

func TestRenderListView_ViewShowsSpinner(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.isScanning = true

	view := m.View()

	assert.Contains(t, view.Content, "Scanning...")
	assert.Contains(t, view.Content, "0 resources")
}

func TestRenderListView_ViewShowsCompleteWhenDone(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.isScanning = false
	m.resources = make([]terraform.Resource, 5)

	view := m.View()

	assert.Contains(t, view.Content, "Scan Complete")
	assert.Contains(t, view.Content, "5 resources")
	assert.NotContains(t, view.Content, "Scanning...")
}

func TestRenderListView_ViewShowsSelectedCount(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewHeight = 5 + defaultReservedRows
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionNoop},
	}
	m.selected = map[string]bool{"aws_s3_bucket.a": true}

	view := m.View()

	assert.Contains(t, view.Content, "1 selected")
}

func TestRenderListView_ViewShowsFilterCount(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
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

func TestRenderListView_ViewShowsHideNoopMessage(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})

	require.Contains(t, m.View().Content, "h to hide unchanged")

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'h'})
	m = newModel.(Model)

	assert.Contains(t, m.View().Content, "h to show unchanged")
}

func TestRenderListView_ViewShowsError(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.err = fmt.Errorf("something broke")

	view := m.View()

	assert.Contains(t, view.Content, "error occurred: something broke")
}

func TestRenderActionPickerView_ShowsActionPicker(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewActionPicker
	m.selected = map[string]bool{"aws_s3_bucket.a": true, "aws_lambda_function.api": true}
	m.viewWidth = 80
	m.viewHeight = 24

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
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewConfirm
	m.actionCursor = 1
	m.viewWidth = 80
	m.viewHeight = 24
	m.resources = []terraform.Resource{
		{Address: "aws_s3_bucket.a", Action: terraform.ActionCreate},
		{Address: "aws_lambda.b", Action: terraform.ActionUpdate},
	}
	m.indexMap = map[string]int{"aws_s3_bucket.a": 0, "aws_lambda.b": 1}
	m.selected = map[string]bool{"aws_s3_bucket.a": true, "aws_lambda.b": true}

	view := m.View()

	assert.Contains(t, view.Content, "apply 2 resource(s)?")
	assert.Contains(t, view.Content, "aws_s3_bucket.a")
	assert.Contains(t, view.Content, "aws_lambda.b")
	assert.Contains(t, view.Content, "Cancel")
	assert.Contains(t, view.Content, "Confirm")
}

func TestRenderConfirmView_TruncatesLongSelections(t *testing.T) {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.viewState = viewConfirm
	m.actionCursor = 2
	m.viewWidth = 80
	m.viewHeight = 100
	m.selected = map[string]bool{}
	m.indexMap = map[string]int{}

	for i := range 15 {
		addr := fmt.Sprintf("aws_s3_bucket.b_%02d", i)
		m.resources = append(m.resources, terraform.Resource{
			Address: addr, Action: terraform.ActionDelete,
		})
		m.indexMap[addr] = i
		m.selected[addr] = true
	}

	view := m.View()

	assert.Contains(t, view.Content, "... and 5 more")
	assert.Contains(t, view.Content, "aws_s3_bucket.b_00")
	assert.Contains(t, view.Content, "aws_s3_bucket.b_09")
	assert.NotContains(t, view.Content, "aws_s3_bucket.b_10")
}
