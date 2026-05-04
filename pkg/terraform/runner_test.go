package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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

func TestStreamJsonEvents_ExitWithError(t *testing.T) {
	output := `{"@level":"info","@message":"Terraform 1.14.8","@module":"terraform.ui","@timestamp":"2026-04-11T15:46:38.279544+01:00","terraform":"1.14.8","type":"version","ui":"1.2"}`

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 1),
	}

	ctx := context.Background()
	ch := runner.Plan(ctx, nil)

	var hasError bool
	for event := range ch {
		if event.Error != nil {
			hasError = true
		}
	}

	assert.True(t, hasError, "exit code 1 should produce an error event")
}

func TestStreamJsonEvents_ContextCancellation(t *testing.T) {
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

	ch := runner.Plan(ctx, nil)

	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after cancel")
	}
}

func TestStreamJsonEvents_Plan(t *testing.T) {
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

	assert.Equal(t, MsgTypeRefreshStart, events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)

	assert.Equal(t, MsgTypeRefreshComplete, events[1].Type)

	assert.Equal(t, ActionUpdate, events[2].Resource.Action)

	assert.NotNil(t, events[3].Summary)
	assert.Equal(t, 1, events[3].Summary.Change)
	assert.Equal(t, "plan", events[3].Summary.Operation)
}

func TestStreamJsonEvents_Apply(t *testing.T) {
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

	assert.Equal(t, MsgTypeApplyStart, events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)
	assert.Equal(t, ActionUpdate, events[0].Resource.Action)

	assert.Equal(t, MsgTypeApplyComplete, events[1].Type)

	assert.NotNil(t, events[2].Summary)
	assert.Equal(t, 1, events[2].Summary.Change)
	assert.Equal(t, "apply", events[2].Summary.Operation)
}

func TestStreamJsonEvents_Destroy(t *testing.T) {
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

	assert.Equal(t, MsgTypeApplyStart, events[0].Type)
	assert.Equal(t, ActionDelete, events[0].Resource.Action)

	assert.Equal(t, MsgTypeApplyComplete, events[1].Type)

	assert.NotNil(t, events[2].Summary)
	assert.Equal(t, 1, events[2].Summary.Remove)
	assert.Equal(t, "destroy", events[2].Summary.Operation)
}

func TestStreamPerResource_Taint(t *testing.T) {
	output := "Resource instance aws_s3_bucket.uploads has been marked as tainted."

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	targets := []string{"aws_s3_bucket.a", "aws_s3_bucket.b"}
	ch := runner.Taint(ctx, targets)

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 6)

	assert.Equal(t, MsgTypeApplyStart, events[0].Type)
	assert.Equal(t, "aws_s3_bucket.a", events[0].Resource.Address)
	assert.Contains(t, events[1].Message, "tainted")
	assert.Equal(t, MsgTypeApplyComplete, events[2].Type)
	assert.Equal(t, "aws_s3_bucket.a", events[2].Resource.Address)

	assert.Equal(t, MsgTypeApplyStart, events[3].Type)
	assert.Equal(t, "aws_s3_bucket.b", events[3].Resource.Address)
	assert.Contains(t, events[4].Message, "tainted")
	assert.Equal(t, MsgTypeApplyComplete, events[5].Type)
	assert.Equal(t, "aws_s3_bucket.b", events[5].Resource.Address)
}

func TestStreamPerResource_Untaint(t *testing.T) {
	output := "Resource instance aws_s3_bucket.uploads has been successfully untainted."

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	ctx := context.Background()
	ch := runner.Untaint(ctx, []string{"aws_s3_bucket.uploads"})

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 3)

	assert.Equal(t, MsgTypeApplyStart, events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)

	assert.Contains(t, events[1].Message, "untainted")

	assert.Equal(t, MsgTypeApplyComplete, events[2].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[2].Resource.Address)
}

func TestStreamPerResource_Error(t *testing.T) {
	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory("Error: No such resource", 1),
	}

	ch := runner.Taint(context.Background(), []string{"aws_s3_bucket.uploads"})

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	require.Len(t, events, 3)

	assert.Equal(t, MsgTypeApplyStart, events[0].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[0].Resource.Address)

	assert.Contains(t, events[1].Message, "Error")

	assert.Equal(t, MsgTypeApplyErrored, events[2].Type)
	assert.Equal(t, "aws_s3_bucket.uploads", events[2].Resource.Address)
}
