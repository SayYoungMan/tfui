package terraform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseState_Empty(t *testing.T) {
	resources, err := ParseState(nil)

	assert.Nil(t, resources)
	assert.Nil(t, err)
}

func TestParseState_InvalidJSON(t *testing.T) {
	resources, err := ParseState([]byte("invalid"))

	assert.Nil(t, resources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse state JSON")
}

func TestParseState_WrongVersion(t *testing.T) {
	wrongVersionStateJSON := `{"version": 5, "resources": []}`

	resources, err := ParseState([]byte(wrongVersionStateJSON))

	assert.Nil(t, resources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only state file version 4 is supported but got 5")
}

func TestParseState_AllResources(t *testing.T) {
	stateJSON := `{
  "version": 4,
  "terraform_version": "1.7.0",
  "serial": 20,
  "lineage": "11111111-1111-1111-1111-111111111111",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "uploads",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "attributes": {"id": "my-uploads-bucket", "region": "eu-west-2"}}
      ]
    },
    {
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "envs",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "index_key": "prod", "attributes": {"id": "envs-prod"}},
        {"schema_version": 0, "index_key": "staging", "attributes": {"id": "envs-staging"}}
      ]
    },
    {
      "mode": "managed",
      "type": "aws_subnet",
      "name": "private",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "index_key": 0, "attributes": {"id": "subnet-0"}},
        {"schema_version": 0, "index_key": 1, "attributes": {"id": "subnet-1"}}
      ]
    },
    {
      "module": "module.networking",
      "mode": "managed",
      "type": "aws_vpc",
      "name": "main",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 1, "attributes": {"id": "vpc-0abc123"}}
      ]
    },
    {
      "module": "module.networking.module.subnets",
      "mode": "managed",
      "type": "aws_subnet",
      "name": "app",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "attributes": {"id": "subnet-app"}}
      ]
    },
    {
      "mode": "data",
      "type": "aws_caller_identity",
      "name": "current",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 0, "attributes": {"account_id": "123456789012"}}
      ]
    },
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "broken",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {"schema_version": 1, "status": "tainted", "attributes": {"id": "i-0deadbeef"}}
      ]
    }
  ]
}`

	resources, err := ParseState([]byte(stateJSON))
	require.NoError(t, err)
	require.Len(t, resources, 9)

	resourcesMap := make(map[string]Resource, len(resources))
	for _, r := range resources {
		resourcesMap[r.Address] = r
	}

	tests := []struct {
		name         string
		address      string
		module       string
		resourceAddr string
		resourceType string
		resourceName string
		key          any
		reason       string
	}{
		{
			name:         "simple managed",
			address:      "aws_s3_bucket.uploads",
			resourceAddr: "aws_s3_bucket.uploads",
			resourceType: "aws_s3_bucket",
			resourceName: "uploads",
			key:          nil,
		},
		{
			name:         "for_each prod",
			address:      `aws_s3_bucket.envs["prod"]`,
			resourceAddr: `aws_s3_bucket.envs["prod"]`,
			resourceType: "aws_s3_bucket",
			resourceName: "envs",
			key:          "prod",
		},
		{
			name:         "for_each staging",
			address:      `aws_s3_bucket.envs["staging"]`,
			resourceAddr: `aws_s3_bucket.envs["staging"]`,
			resourceType: "aws_s3_bucket",
			resourceName: "envs",
			key:          "staging",
		},
		{
			name:         "count 0",
			address:      "aws_subnet.private[0]",
			resourceAddr: "aws_subnet.private[0]",
			resourceType: "aws_subnet",
			resourceName: "private",
			key:          float64(0),
		},
		{
			name:         "count 1",
			address:      "aws_subnet.private[1]",
			resourceAddr: "aws_subnet.private[1]",
			resourceType: "aws_subnet",
			resourceName: "private",
			key:          float64(1),
		},
		{
			name:         "single nested module",
			address:      "module.networking.aws_vpc.main",
			module:       "module.networking",
			resourceAddr: "aws_vpc.main",
			resourceType: "aws_vpc",
			resourceName: "main",
			key:          nil,
		},
		{
			name:         "double nested module",
			address:      "module.networking.module.subnets.aws_subnet.app",
			module:       "module.networking.module.subnets",
			resourceAddr: "aws_subnet.app",
			resourceType: "aws_subnet",
			resourceName: "app",
			key:          nil,
		},
		{
			name:         "data source",
			address:      "data.aws_caller_identity.current",
			resourceAddr: "data.aws_caller_identity.current",
			resourceType: "aws_caller_identity",
			resourceName: "current",
			key:          nil,
		},
		{
			name:         "tainted",
			address:      "aws_instance.broken",
			resourceAddr: "aws_instance.broken",
			resourceType: "aws_instance",
			resourceName: "broken",
			key:          nil,
			reason:       "tainted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ok := resourcesMap[tt.address]
			require.True(t, ok)

			assert.Equal(t, tt.module, r.Module)
			assert.Equal(t, tt.resourceAddr, r.ResourceAddr)
			assert.Equal(t, tt.resourceType, r.ResourceType)
			assert.Equal(t, tt.resourceName, r.ResourceName)
			assert.Equal(t, tt.key, r.ResourceKey)
			assert.Equal(t, tt.reason, r.Reason)

			assert.Equal(t, ActionUncertain, r.Action)
			assert.Equal(t, "provider[\"registry.terraform.io/hashicorp/aws\"]", r.ImpliedProvider)
			assert.NotEmpty(t, r.Attributes)
		})
	}
}

func TestPullState_Success(t *testing.T) {
	output := `{"version":4,"terraform_version":"1.7.0","serial":1,"lineage":"abc","outputs":{},"resources":[{"mode":"managed","type":"aws_s3_bucket","name":"uploads","provider":"provider[\"registry.terraform.io/hashicorp/aws\"]","instances":[{"schema_version":0,"attributes":{"id":"my-uploads-bucket"}}]}]}`

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	resources, err := runner.StatePull(context.Background())

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "aws_s3_bucket.uploads", resources[0].Address)
	assert.Equal(t, ActionUncertain, resources[0].Action)
	assert.NotEmpty(t, resources[0].Attributes)
}

func TestPullState_EmptyState(t *testing.T) {
	output := `{"version":4,"terraform_version":"1.7.0","serial":1,"lineage":"abc","outputs":{},"resources":[]}`

	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory(output, 0),
	}

	resources, err := runner.StatePull(context.Background())

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func TestPullState_CommandFailure(t *testing.T) {
	runner := &TerraformRunner{
		binary:     "terraform",
		workdir:    t.TempDir(),
		cmdFactory: mockCmdFactory("", 1),
	}

	_, err := runner.StatePull(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "state pull failed")
}
