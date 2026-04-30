package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmptyLine(t *testing.T) {
	p := NewParser()
	event, err := p.ParseLine([]byte(""))

	require.NoError(t, err)
	assert.Nil(t, event)
}

func TestParseRefreshStart(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_s3_bucket.uploads: Refreshing state... [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"id_key":"id","id_value":"my-uploads-bucket"},"type":"refresh_start"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_s3_bucket.uploads", ActionNoop, "")
	assert.Empty(t, event.Resource.Module)
	assert.Equal(t, "aws_s3_bucket", event.Resource.ResourceType)
	assert.Equal(t, "aws", event.Resource.ImpliedProvider)
}

func TestParseApplyStart(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"data.aws_caller_identity.current: Reading...","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.133445+01:00","hook":{"resource":{"addr":"data.aws_caller_identity.current","module":"","resource":"data.aws_caller_identity.current","implied_provider":"aws","resource_type":"aws_caller_identity","resource_name":"current","resource_key":null},"action":"read","id_key":"id","id_value":"123456789012","elapsed_seconds":0},"type":"apply_start"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "data.aws_caller_identity.current", ActionRead, "")
}

func TestParseResourceDrift(t *testing.T) {
	p := NewParser()

	refreshLine := []byte(`{"@level":"info","@message":"aws_lambda_function.processor: Refreshing state... [id=processor]","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","hook":{"resource":{"addr":"aws_lambda_function.processor","module":"","resource":"aws_lambda_function.processor","implied_provider":"aws","resource_type":"aws_lambda_function","resource_name":"processor","resource_key":null},"id_key":"id","id_value":"processor"},"type":"refresh_start"}`)
	_, err := p.ParseLine(refreshLine)
	require.NoError(t, err)

	driftLine := []byte(`{"@level":"info","@message":"aws_lambda_function.processor: Drift detected (update)","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_lambda_function.processor","module":"","resource":"aws_lambda_function.processor","implied_provider":"aws","resource_type":"aws_lambda_function","resource_name":"processor","resource_key":null},"action":"update"},"type":"resource_drift"}`)
	event, err := p.ParseLine(driftLine)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_lambda_function.processor", ActionUpdate, "drift")
}

func TestParsePlannedChange_Update(t *testing.T) {
	p := NewParser()

	refreshLine := []byte(`{"@level":"info","@message":"aws_s3_bucket_server_side_encryption_configuration.state: Refreshing state... [id=my-state-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","hook":{"resource":{"addr":"aws_s3_bucket_server_side_encryption_configuration.state","module":"","resource":"aws_s3_bucket_server_side_encryption_configuration.state","implied_provider":"aws","resource_type":"aws_s3_bucket_server_side_encryption_configuration","resource_name":"state","resource_key":null},"id_key":"id","id_value":"my-state-bucket"},"type":"refresh_start"}`)
	_, err := p.ParseLine(refreshLine)
	require.NoError(t, err)

	plannedLine := []byte(`{"@level":"info","@message":"aws_s3_bucket_server_side_encryption_configuration.state: Plan to update in-place","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_s3_bucket_server_side_encryption_configuration.state","module":"","resource":"aws_s3_bucket_server_side_encryption_configuration.state","implied_provider":"aws","resource_type":"aws_s3_bucket_server_side_encryption_configuration","resource_name":"state","resource_key":null},"action":"update"},"type":"planned_change"}`)
	event, err := p.ParseLine(plannedLine)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_s3_bucket_server_side_encryption_configuration.state", ActionUpdate, "")
}

func TestParsePlannedChange_Create(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_dynamodb_table.sessions: Plan to create","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_dynamodb_table.sessions","module":"","resource":"aws_dynamodb_table.sessions","implied_provider":"aws","resource_type":"aws_dynamodb_table","resource_name":"sessions","resource_key":null},"action":"create"},"type":"planned_change"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_dynamodb_table.sessions", ActionCreate, "")
}

func TestParsePlannedChange_Replace(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_ecs_task_definition.worker: Plan to replace","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_ecs_task_definition.worker","module":"","resource":"aws_ecs_task_definition.worker","implied_provider":"aws","resource_type":"aws_ecs_task_definition","resource_name":"worker","resource_key":null},"action":"replace","reason":"cannot_update"},"type":"planned_change"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_ecs_task_definition.worker", ActionReplace, "cannot_update")
}

func TestParsePlannedChange_Delete(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_iam_role.legacy: Plan to delete","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_iam_role.legacy","module":"","resource":"aws_iam_role.legacy","implied_provider":"aws","resource_type":"aws_iam_role","resource_name":"legacy","resource_key":null},"action":"delete","reason":"delete_because_no_resource_config"},"type":"planned_change"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_iam_role.legacy", ActionDelete, "delete_because_no_resource_config")
}

func TestParsePlannedChange_Import(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_route53_zone.main: Plan to import","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_route53_zone.main","module":"","resource":"aws_route53_zone.main","implied_provider":"aws","resource_type":"aws_route53_zone","resource_name":"main","resource_key":null},"action":"import"},"type":"planned_change"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "aws_route53_zone.main", ActionImport, "")
}

func TestParsePlannedChange_Move(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"aws_security_group.api: Plan to move","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"module.networking.aws_security_group.api","module":"module.networking","resource":"aws_security_group.api","implied_provider":"aws","resource_type":"aws_security_group","resource_name":"api","resource_key":null},"action":"move","previous_resource":{"addr":"aws_security_group.api","module":"","resource":"aws_security_group.api","implied_provider":"aws","resource_type":"aws_security_group","resource_name":"api","resource_key":null}},"type":"planned_change"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	assertResourceEvent(t, event, "module.networking.aws_security_group.api", ActionMove, "")
}

func TestParseChangeSummary(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"Plan: 1 to add, 3 to change, 0 to destroy.","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","changes":{"add":1,"change":3,"remove":0,"operation":"plan"},"type":"change_summary"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	require.NotNil(t, event.Summary)
	assert.Equal(t, 1, event.Summary.Add)
	assert.Equal(t, 3, event.Summary.Change)
	assert.Equal(t, 0, event.Summary.Remove)
	assert.Equal(t, "plan", event.Summary.Operation)
}

func TestParseDiagnostic_Error(t *testing.T) {
	line := []byte(`{"@level":"error","@message":"Error: Invalid reference","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","diagnostic":{"severity":"error","summary":"Invalid reference","detail":"A managed resource \"aws_s3_bucket.missing\" has not been declared in the root module."},"type":"diagnostic"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	require.NotNil(t, event.Diagnostic)
	assert.Equal(t, "error", event.Diagnostic.Severity)
	assert.Equal(t, "Invalid reference", event.Diagnostic.Summary)

	_, errors, _ := p.Stats()
	assert.Equal(t, 1, errors)
}

func TestParseDiagnostic_Warning(t *testing.T) {
	line := []byte(`{"@level":"warn","@message":"Warning: Deprecated attribute","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","diagnostic":{"severity":"warning","summary":"Deprecated attribute","detail":"The attribute \"arn\" is deprecated. Use \"id\" instead."},"type":"diagnostic"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	require.NotNil(t, event.Diagnostic)

	_, _, warnings := p.Stats()
	assert.Equal(t, 1, warnings)
}

func TestParseOutputs(t *testing.T) {
	line := []byte(`{"@level":"info","@message":"Outputs: 2","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","outputs":{"api_url":{"sensitive":false,"type":"string","value":"https://api.example.com","action":"noop"},"db_password":{"sensitive":true,"type":"string","value":"hunter2","action":"noop"}},"type":"outputs"}`)

	p := NewParser()
	event, err := p.ParseLine(line)

	require.NoError(t, err)
	require.NotNil(t, event.Outputs)
	assert.Len(t, event.Outputs, 2)
	assert.True(t, event.Outputs["db_password"].Sensitive)
	assert.False(t, event.Outputs["api_url"].Sensitive)
}

func TestParseIgnoredTypes(t *testing.T) {
	lines := [][]byte{
		[]byte(`{"@level":"info","@message":"Terraform 1.14.8","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:38.279544+01:00","terraform":"1.14.8","type":"version","ui":"1.2"}`),
		[]byte(`{"@level":"info","@message":"aws_s3_bucket.uploads: Refreshing state... [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.111262+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"id_key":"id","id_value":"my-uploads-bucket"},"type":"refresh_complete"}`),
	}

	p := NewParser()
	for _, line := range lines {
		event, err := p.ParseLine(line)
		require.NoError(t, err)
		assert.Nil(t, event)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	p := NewParser()
	_, err := p.ParseLine([]byte("not json at all"))

	assert.Error(t, err)
}

// Helper function to easily compare created event from the input data
func assertResourceEvent(t *testing.T, event *StreamEvent, addr string, action Action, reason string) {
	t.Helper()
	require.NotNil(t, event)
	require.NotNil(t, event.Resource)
	assert.Equal(t, addr, event.Resource.Address)
	assert.Equal(t, action, event.Resource.Action)
	assert.Equal(t, reason, event.Resource.Reason)
}
