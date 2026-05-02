package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	GO_TEST_HELPER_PROCESS = "GO_TEST_HELPER_PROCESS"
	MOCK_OUTPUT            = "MOCK_OUTPUT"
	MOCK_EXIT_CODE         = "MOCK_EXIT_CODE"
	MOCK_ARGS              = "MOCK_ARGS"
)

// Invoked as a subprocess by mockCmdFactory to simulate Terraform output
func TestHelperProcess(t *testing.T) {
	// Check if it's called by the mockCmdFactory
	if os.Getenv(GO_TEST_HELPER_PROCESS) != "1" {
		return
	}

	output := os.Getenv(MOCK_OUTPUT)
	fmt.Fprint(os.Stdout, output)

	exitCode := 0
	if code := os.Getenv(MOCK_EXIT_CODE); code != "" {
		fmt.Sscanf(code, "%d", &exitCode)
	}

	os.Exit(exitCode)
}

func mockCmdFactory(output string, exitCode int) CommandFactory {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// os.Args[0] is the test binary so we are running the test again in subprocess
		// but only the TestHelperProcess function
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("%s=1", GO_TEST_HELPER_PROCESS),
			fmt.Sprintf("%s=%s", MOCK_OUTPUT, output),
			fmt.Sprintf("%s=%d", MOCK_EXIT_CODE, exitCode),
			fmt.Sprintf("%s=%s %s", MOCK_ARGS, name, strings.Join(args, " ")),
		)
		return cmd
	}
}

func TestStreamPlan_ParsesEvents(t *testing.T) {
	output := strings.Join([]string{
		`{"@level":"info","@message":"Terraform 1.14.8","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:38.279544+01:00","terraform":"1.14.8","type":"version","ui":"1.2"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Refreshing state... [id=uploads]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"id_key":"id","id_value":"uploads"},"type":"refresh_start"}`,
		`{"@level":"info","@message":"data.aws_region.current: Reading...","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.133445+01:00","hook":{"resource":{"addr":"data.aws_region.current","module":"","resource":"data.aws_region.current","implied_provider":"aws","resource_type":"aws_region","resource_name":"current","resource_key":null},"action":"read","id_key":"id","id_value":"eu-west-2","elapsed_seconds":0},"type":"apply_start"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Plan to update in-place","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"update"},"type":"planned_change"}`,
		`{"@level":"info","@message":"Plan: 0 to add, 1 to change, 0 to destroy.","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","changes":{"add":0,"change":1,"remove":0,"operation":"plan"},"type":"change_summary"}`,
	}, "\n")

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.StreamPlan(ctx)

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 4)

	assert.NotNil(t, events[0].Resource)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)
	assert.Equal(t, ActionNoop, events[0].Resource.Action)

	assert.NotNil(t, events[1].Resource)
	assert.Equal(t, "data.aws_region.current", events[1].Resource.Address)
	assert.Equal(t, ActionRead, events[1].Resource.Action)

	assert.NotNil(t, events[2].Resource)
	assert.Equal(t, ActionUpdate, events[2].Resource.Action)

	assert.NotNil(t, events[3].Summary)
	assert.Equal(t, 1, events[3].Summary.Change)
}

func TestStreamPlan_ExitWithError(t *testing.T) {
	output := `{"@level":"info","@message":"Terraform 1.14.8","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:38.279544+01:00","terraform":"1.14.8","type":"version","ui":"1.2"}`

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 1),
	}

	ctx := context.Background()
	ch := runner.StreamPlan(ctx)

	var hasError bool
	for event := range ch {
		if event.Error != nil {
			hasError = true
		}
	}

	assert.True(t, hasError, "exit code 1 should produce an error event")
}

func TestStreamPlan_ContextCancellation(t *testing.T) {
	output := strings.Join([]string{
		`{"@level":"info","@message":"aws_s3_bucket.a: Refreshing state... [id=a]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.a","module":"","resource":"aws_s3_bucket.a","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"a","resource_key":null},"id_key":"id","id_value":"a"},"type":"refresh_start"}`,
	}, "\n")

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := runner.StreamPlan(ctx)

	// Should drain and close without hanging
	for range ch {
	}
}

func TestStreamOutput_Plan(t *testing.T) {
	output := strings.Join([]string{
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Refreshing state... [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"id_key":"id","id_value":"my-uploads-bucket"},"type":"refresh_start"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Refresh complete [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"id_key":"id","id_value":"my-uploads-bucket"},"type":"refresh_complete"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Plan to update in-place","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","change":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"update"},"type":"planned_change"}`,
		`{"@level":"info","@message":"Plan: 0 to add, 1 to change, 0 to destroy.","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:47.040866+01:00","changes":{"add":0,"change":1,"remove":0,"operation":"plan"},"type":"change_summary"}`,
	}, "\n")

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Plan(ctx, []string{"aws_s3_bucket.uploads"})

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 4)

	assert.Equal(t, "refresh_start", events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)

	assert.Equal(t, "refresh_complete", events[1].Type)

	assert.Equal(t, "planned_change", events[2].Type)
	assert.Equal(t, ActionUpdate, events[2].Resource.Action)

	assert.NotNil(t, events[3].Summary)
	assert.Equal(t, 1, events[3].Summary.Change)
	assert.Equal(t, "plan", events[3].Summary.Operation)
}

func TestStreamOutput_Apply(t *testing.T) {
	output := strings.Join([]string{
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Modifying...","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"update","elapsed_seconds":0},"type":"apply_start"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Modifications complete after 2s [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:48.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"update","id_key":"id","id_value":"my-uploads-bucket","elapsed_seconds":2},"type":"apply_complete"}`,
		`{"@level":"info","@message":"Apply complete! Resources: 0 added, 1 changed, 0 destroyed.","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:48.108644+01:00","changes":{"add":0,"change":1,"remove":0,"operation":"apply"},"type":"change_summary"}`,
	}, "\n")

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Apply(ctx, []string{"aws_s3_bucket.uploads"})

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 3)

	assert.Equal(t, "apply_start", events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)
	assert.Equal(t, ActionUpdate, events[0].Resource.Action)

	assert.Equal(t, "apply_complete", events[1].Type)
	assert.Equal(t, 2, events[1].Hook.ElapsedSeconds)

	assert.NotNil(t, events[2].Summary)
	assert.Equal(t, 1, events[2].Summary.Change)
	assert.Equal(t, "apply", events[2].Summary.Operation)
}

func TestStreamOutput_Destroy(t *testing.T) {
	output := strings.Join([]string{
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Destroying... [id=my-uploads-bucket]","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:46.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"delete","elapsed_seconds":0},"type":"apply_start"}`,
		`{"@level":"info","@message":"aws_s3_bucket.uploads: Destruction complete after 1s","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:47.108644+01:00","hook":{"resource":{"addr":"aws_s3_bucket.uploads","module":"","resource":"aws_s3_bucket.uploads","implied_provider":"aws","resource_type":"aws_s3_bucket","resource_name":"uploads","resource_key":null},"action":"delete","elapsed_seconds":1},"type":"apply_complete"}`,
		`{"@level":"info","@message":"Destroy complete! Resources: 1 destroyed.","@module":"terraform.ui","@timestamp":"2026-04-11T09:14:47.108644+01:00","changes":{"add":0,"change":0,"remove":1,"operation":"destroy"},"type":"change_summary"}`,
	}, "\n")

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Destroy(ctx, []string{"aws_s3_bucket.uploads"})

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 3)

	assert.Equal(t, "apply_start", events[0].Type)
	assert.Equal(t, ActionDelete, events[0].Resource.Action)

	assert.Equal(t, "apply_complete", events[1].Type)
	assert.Equal(t, 1, events[1].Hook.ElapsedSeconds)

	assert.NotNil(t, events[2].Summary)
	assert.Equal(t, 1, events[2].Summary.Remove)
	assert.Equal(t, "destroy", events[2].Summary.Operation)
}

func TestStreamOutput_Taint(t *testing.T) {
	output := "Resource instance aws_s3_bucket.uploads has been marked as tainted."

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Taint(ctx, []string{"aws_s3_bucket.uploads"})

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "tainted")
}

func TestStreamOutput_Untaint(t *testing.T) {
	output := "Resource instance aws_s3_bucket.uploads has been successfully untainted."

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Untaint(ctx, []string{"aws_s3_bucket.uploads"})

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}

	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "untainted")
}
