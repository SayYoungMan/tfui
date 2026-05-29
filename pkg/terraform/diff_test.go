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

func TestDiffHelperProcess(t *testing.T) {
	if os.Getenv(GO_TEST_HELPER_PROCESS) != "1" {
		return
	}

	fmt.Fprint(os.Stdout, os.Getenv(MOCK_OUTPUT))
	fmt.Fprint(os.Stderr, os.Getenv(MOCK_STDERR))

	exitCode := 0
	if code := os.Getenv(MOCK_EXIT_CODE); code != "" {
		fmt.Sscanf(code, "%d", &exitCode)
	}

	os.Exit(exitCode)
}

func mockShowCmdFactory(output string, stderr string, exitCode int, calls *[]string) CommandFactory {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		*calls = append(*calls, name+" "+strings.Join(args, " "))

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestDiffHelperProcess", "--")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("%s=1", GO_TEST_HELPER_PROCESS),
			fmt.Sprintf("%s=%s", MOCK_OUTPUT, output),
			fmt.Sprintf("%s=%s", MOCK_STDERR, stderr),
			fmt.Sprintf("%s=%d", MOCK_EXIT_CODE, exitCode),
		)
		return cmd
	}
}

func TestPlannedChanges_Success(t *testing.T) {
	output := `{
		"resource_changes": [
			{
				"address": "aws_s3_bucket.uploads",
				"change": {
					"actions": ["update"],
					"before": {"acl": "private"},
					"after": {"acl": "public-read"},
					"after_unknown": {"id": true},
					"before_sensitive": {"secret": true},
					"after_sensitive": {"secret": true}
				}
			},
			{
				"address": "module.vpc.aws_subnet.private[0]",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"cidr_block": "10.0.1.0/24"}
				}
			}
		]
	}`

	var calls []string
	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockShowCmdFactory(output, "", 0, &calls),
	}

	changes, err := runner.PlannedChanges(context.Background())

	require.NoError(t, err)
	require.Len(t, changes, 2)

	uploadChange := changes["aws_s3_bucket.uploads"]
	assert.Equal(t, []string{"update"}, uploadChange.Actions)
	assert.JSONEq(t, `{"acl":"private"}`, string(uploadChange.Before))
	assert.JSONEq(t, `{"acl":"public-read"}`, string(uploadChange.After))
	assert.JSONEq(t, `{"id":true}`, string(uploadChange.AfterUnknown))
	assert.JSONEq(t, `{"secret":true}`, string(uploadChange.BeforeSensitive))
	assert.JSONEq(t, `{"secret":true}`, string(uploadChange.AfterSensitive))

	subnetChange := changes["module.vpc.aws_subnet.private[0]"]
	assert.Equal(t, []string{"create"}, subnetChange.Actions)
	assert.Equal(t, "null", string(subnetChange.Before))
	assert.JSONEq(t, `{"cidr_block":"10.0.1.0/24"}`, string(subnetChange.After))

	require.Len(t, calls, 1)
	assert.Equal(t, "terraform show -json .tfui/latest.tfplan", calls[0])
}

func TestPlannedChanges_InvalidJSON(t *testing.T) {
	var calls []string
	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockShowCmdFactory("not-json", "", 0, &calls),
	}

	changes, err := runner.PlannedChanges(context.Background())

	assert.Nil(t, changes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse plan JSON")
}

func TestPlannedChanges_ShowErrorIncludesStderr(t *testing.T) {
	var calls []string
	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockShowCmdFactory("", "saved plan is stale", 1, &calls),
	}

	changes, err := runner.PlannedChanges(context.Background())

	assert.Nil(t, changes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terraform show plan failed")
	assert.Contains(t, err.Error(), "saved plan is stale")
}
