package jsondiff

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_NestedFieldChange(t *testing.T) {
	before := json.RawMessage(`{"bucket":"uploads","tags":{"env":"dev"}}`)
	after := json.RawMessage(`{"bucket":"uploads","tags":{"env":"prod"}}`)

	out, err := Render(before, after, Options{})

	require.NoError(t, err)
	assert.Equal(t, strings.Join([]string{
		"  {",
		`    "bucket": "uploads",`,
		`    "tags": {`,
		`-     "env": "dev"`,
		`+     "env": "prod"`,
		"    }",
		"  }",
		"",
	}, "\n"), out)
}

func TestRender_AddedAndRemovedFields(t *testing.T) {
	before := json.RawMessage(`{"a":1,"b":2}`)
	after := json.RawMessage(`{"a":1,"c":3}`)

	out, err := Render(before, after, Options{})

	require.NoError(t, err)
	assert.Contains(t, out, `-   "b": 2,`)
	assert.Contains(t, out, `+   "c": 3`)
}

func TestRender_AppliesLineStyles(t *testing.T) {
	before := json.RawMessage(`{"env":"dev"}`)
	after := json.RawMessage(`{"env":"prod"}`)

	out, err := Render(before, after, Options{
		AddedStyle: func(s string) string {
			return "[add]" + s + "[/add]"
		},
		RemovedStyle: func(s string) string {
			return "[remove]" + s + "[/remove]"
		},
	})

	require.NoError(t, err)
	assert.Contains(t, out, `[remove]  "env": "dev"[/remove]`)
	assert.Contains(t, out, `[add]  "env": "prod"[/add]`)
	assert.NotContains(t, out, `- [remove]`)
	assert.NotContains(t, out, `+ [add]`)
}

func TestRender_RootCreateDoesNotShowRemovedNull(t *testing.T) {
	before := json.RawMessage(`null`)
	after := json.RawMessage(`{"bucket":"b"}`)

	out, err := Render(before, after, Options{})

	require.NoError(t, err)
	assert.Contains(t, out, "+ {")
	assert.Contains(t, out, `+   "bucket": "b"`)
	assert.NotContains(t, out, "- null")
}

func TestRender_RootDeleteDoesNotShowAddedNull(t *testing.T) {
	before := json.RawMessage(`{"bucket":"b"}`)
	after := json.RawMessage(`null`)

	out, err := Render(before, after, Options{})

	require.NoError(t, err)
	assert.Contains(t, out, "- {")
	assert.Contains(t, out, `-   "bucket": "b"`)
	assert.NotContains(t, out, "+ null")
}

func TestRenderDocument_ReturnsLineKinds(t *testing.T) {
	before := json.RawMessage(`{"env":"dev"}`)
	after := json.RawMessage(`{"env":"prod"}`)

	result, err := RenderResult(before, after, Options{})

	require.NoError(t, err)

	var kinds []LineKind
	for _, line := range result.Lines {
		kinds = append(kinds, line.Kind)
	}

	assert.Contains(t, kinds, LineRemoved)
	assert.Contains(t, kinds, LineAdded)
}
