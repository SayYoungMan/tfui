package ui

import "github.com/SayYoungMan/tfui/pkg/terraform"

func newTestModelEmpty() Model {
	return newTestModelWithResources([]terraform.Resource{})
}

var testResources = []terraform.Resource{
	{Address: "aws_s3_bucket.a", Action: terraform.ActionCreate},
	{Address: "aws_s3_bucket.b", Action: terraform.ActionUpdate, Reason: "drift"},
	{Address: "aws_iam_role.c", Action: terraform.ActionDelete, Reason: "delete_because_no_resource_config"},
	{Address: "aws_db_instance.d", Action: terraform.ActionReplace, Reason: "cannot_update"},
	{Address: "aws_s3_bucket.c", Action: terraform.ActionImport},
	{Address: "data.aws_region.current", Action: terraform.ActionRead},
	{Address: "aws_vpc.main", Action: terraform.ActionNoop},
}

func newTestModel() Model {
	return newTestModelWithResources(testResources)
}

func newTestModelWithResources(resources []terraform.Resource) Model {
	ch := make(chan terraform.StreamEvent, 1)
	m := NewModel(&terraform.TerraformRunner{}, ch, func() {})
	m.resources = resources
	m.filteredIdx = make([]int, len(m.resources))
	m.selected = make(map[string]bool)
	m.indexMap = make(map[string]int)
	for i, r := range m.resources {
		m.indexMap[r.Address] = i
		m.filteredIdx[i] = i
	}
	m.viewHeight = 48
	m.viewWidth = 80
	m.cursor = 0

	return m
}

const (
	cursorAnsiString = "\x1b[38;5;234;48;5;230"
	ansiString       = "\x1b["
)
