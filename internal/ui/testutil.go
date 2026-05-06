package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func newTestModelEmpty() Model {
	return newTestModelWithResources([]terraform.Resource{})
}

// Sorted by address since we sort them stably now in rendering rows
var testResources = []terraform.Resource{
	{Address: "aws_db_instance.d", Action: terraform.ActionReplace, Reason: "cannot_update"},
	{Address: "aws_iam_role.c", Action: terraform.ActionDelete, Reason: "delete_because_no_resource_config"},
	{Address: "aws_s3_bucket.a", Action: terraform.ActionCreate},
	{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate, Reason: "drift"},
	{Address: "aws_s3_bucket.c", Action: terraform.ActionImport},
	{Address: "aws_vpc.main", Action: terraform.ActionNoop},
	{Address: "data.aws_region.current", Action: terraform.ActionRead},
}

func newTestModel() Model {
	return newTestModelWithResources(testResources)
}

func newTestModelWithResources(resources []terraform.Resource) Model {
	m := NewModel(&terraform.TerraformRunner{})
	for i, r := range resources {
		m.resources[r.Address] = &resources[i]
	}
	m.selected = make(map[string]bool)
	m.rebuildRows()
	m.viewHeight = 48
	m.viewWidth = 80
	m.cursor = 0
	m.workState = workIdle
	m.cancel = &cancelWrapper{}

	return m
}

// Test model currently applying resources
func newActionTestModel() Model {
	m := newTestModel()
	m.workState = workAction
	m.viewState = viewActionResources
	m.actionStartTime = time.Now().Add(-5 * time.Second)
	m.actionResources = map[string]*ActionResource{
		"aws_s3_bucket.a": {Address: "aws_s3_bucket.a", Status: actionResourcePending},
		"aws_s3_bucket.b": {Address: "aws_s3_bucket.b", Status: actionResourcePending},
	}
	m.actionCursor = 1 // apply
	m.eventCh = make(chan terraform.StreamEvent, 1)
	return m
}

const (
	cursorAnsiString = "\x1b[38;5;234;48;5;230"
	ansiString       = "\x1b["
)

var actionAnsiStrings = map[terraform.Action]string{
	terraform.ActionCreate:  "\x1b[38;5;114",
	terraform.ActionDelete:  "\x1b[38;5;167",
	terraform.ActionUpdate:  "\x1b[38;5;178",
	terraform.ActionReplace: "\x1b[38;5;178",
	terraform.ActionMove:    "\x1b[38;5;111",
	terraform.ActionImport:  "\x1b[38;5;111",
}

func actionAnsiString(a terraform.Action) string {
	return actionAnsiStrings[a]
}

func (m Model) keyPresses(keys []rune) (Model, tea.Cmd) {
	var newModel tea.Model = m
	var cmd tea.Cmd
	for _, k := range keys {
		newModel, cmd = newModel.Update(tea.KeyPressMsg{Code: k})
	}

	return newModel.(Model), cmd
}
