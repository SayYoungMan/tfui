package ui

import (
	"testing"
	"time"

	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAddr = "aws_s3_bucket.a"

func TestHandleActionEvent_RefreshStart(t *testing.T) {
	m := newActionTestModel()

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeRefreshStart,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.actionResources[testAddr]
	assert.Equal(t, actionResourceReadingState, ar.Status)
	assert.False(t, ar.ReadStartedAt.IsZero())
	assert.NotNil(t, cmd)
}

func TestHandleActionEvent_RefreshComplete(t *testing.T) {
	tests := []struct {
		name         string
		actionCursor int
		status       actionResourceStatus
	}{
		{name: "Apply", actionCursor: 1, status: actionResourceWaitingForAction},
		{name: "Plan", actionCursor: 0, status: actionResourceSuccessful},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newActionTestModel()
			m.actionCursor = tt.actionCursor
			m.actionResources[testAddr].Status = actionResourceReadingState
			m.actionResources[testAddr].ReadStartedAt = time.Now().Add(-2 * time.Second)

			newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
				Type:     terraform.MsgTypeRefreshComplete,
				Resource: &terraform.Resource{Address: testAddr},
			}))
			m = newModel.(Model)

			ar := m.actionResources[testAddr]
			assert.Equal(t, tt.status, ar.Status)
			assert.False(t, ar.ReadCompletedAt.IsZero())
		})
	}
}

func TestHandleActionEvent_ApplyStart(t *testing.T) {
	m := newActionTestModel()
	m.actionResources[testAddr].Status = actionResourceWaitingForAction

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyStart,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.actionResources[testAddr]
	assert.Equal(t, actionResourceInProgress, ar.Status)
	assert.False(t, ar.ProcessStartedAt.IsZero())
}

func TestHandleActionEvent_ApplyComplete(t *testing.T) {
	m := newActionTestModel()
	m.actionResources[testAddr].Status = actionResourceInProgress
	m.actionResources[testAddr].ProcessStartedAt = time.Now().Add(-3 * time.Second)

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyComplete,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.actionResources[testAddr]
	assert.Equal(t, actionResourceSuccessful, ar.Status)
	assert.False(t, ar.ProcessCompletedAt.IsZero())
}

func TestHandleActionEvent_ApplyErrored(t *testing.T) {
	m := newActionTestModel()
	m.actionResources[testAddr].Status = actionResourceInProgress
	m.actionResources[testAddr].ProcessStartedAt = time.Now().Add(-3 * time.Second)

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyErrored,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.actionResources[testAddr]
	assert.Equal(t, actionResourceFailed, ar.Status)
	assert.False(t, ar.ProcessCompletedAt.IsZero())
}

func TestHandleActionEvent_UnknownAddress(t *testing.T) {
	m := newActionTestModel()

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyStart,
		Resource: &terraform.Resource{Address: "aws_iam_role.unknown"},
	}))
	m = newModel.(Model)

	assert.NotContains(t, m.actionResources, "aws_iam_role.unknown")
	assert.NotNil(t, cmd)
}

func TestHandleActionEvent_NilResource(t *testing.T) {
	m := newActionTestModel()

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:    terraform.MsgTypeChangeSummary,
		Message: "Apply complete!",
	}))
	m = newModel.(Model)

	assert.NotNil(t, cmd)
}

func TestHandleActionEvent_AppendsMessage(t *testing.T) {
	m := newActionTestModel()

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyStart,
		Resource: &terraform.Resource{Address: testAddr},
		Message:  "aws_s3_bucket.a: Modifying...",
	}))
	m = newModel.(Model)

	require.Len(t, m.outputLines, 1)
	assert.Equal(t, "aws_s3_bucket.a: Modifying...", m.outputLines[0])
}
