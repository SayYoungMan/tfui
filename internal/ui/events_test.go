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

	ar := m.progresses[testAddr]
	assert.Equal(t, progressStatusReadingState, ar.Status)
	assert.False(t, ar.ReadStartedAt.IsZero())
	assert.NotNil(t, cmd)
}

func TestHandleActionEvent_RefreshComplete(t *testing.T) {
	tests := []struct {
		name         string
		actionCursor int
		status       progressStatus
	}{
		{name: "Apply", actionCursor: 1, status: progressStatusWaitingForAction},
		{name: "Plan", actionCursor: 0, status: progressStatusSuccessful},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newActionTestModel()
			m.actionCursor = tt.actionCursor
			m.progresses[testAddr].Status = progressStatusReadingState
			m.progresses[testAddr].ReadStartedAt = time.Now().Add(-2 * time.Second)

			newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
				Type:     terraform.MsgTypeRefreshComplete,
				Resource: &terraform.Resource{Address: testAddr},
			}))
			m = newModel.(Model)

			ar := m.progresses[testAddr]
			assert.Equal(t, tt.status, ar.Status)
			assert.False(t, ar.ReadCompletedAt.IsZero())
		})
	}
}

func TestHandleActionEvent_ApplyStart(t *testing.T) {
	m := newActionTestModel()
	m.progresses[testAddr].Status = progressStatusWaitingForAction

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyStart,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.progresses[testAddr]
	assert.Equal(t, progressStatusInProgress, ar.Status)
	assert.False(t, ar.ProcessStartedAt.IsZero())
}

func TestHandleActionEvent_ApplyComplete(t *testing.T) {
	m := newActionTestModel()
	m.progresses[testAddr].Status = progressStatusInProgress
	m.progresses[testAddr].ProcessStartedAt = time.Now().Add(-3 * time.Second)

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyComplete,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.progresses[testAddr]
	assert.Equal(t, progressStatusSuccessful, ar.Status)
	assert.False(t, ar.ProcessCompletedAt.IsZero())
}

func TestHandleActionEvent_ApplyErrored(t *testing.T) {
	m := newActionTestModel()
	m.progresses[testAddr].Status = progressStatusInProgress
	m.progresses[testAddr].ProcessStartedAt = time.Now().Add(-3 * time.Second)

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyErrored,
		Resource: &terraform.Resource{Address: testAddr},
	}))
	m = newModel.(Model)

	ar := m.progresses[testAddr]
	assert.Equal(t, progressStatusFailed, ar.Status)
	assert.False(t, ar.ProcessCompletedAt.IsZero())
}

func TestHandleActionEvent_UnknownAddress(t *testing.T) {
	m := newActionTestModel()

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Type:     terraform.MsgTypeApplyStart,
		Resource: &terraform.Resource{Address: "aws_iam_role.unknown"},
	}))
	m = newModel.(Model)

	assert.NotContains(t, m.progresses, "aws_iam_role.unknown")
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

func TestHandleActionEvent_PlannedChangeUpdatesResources(t *testing.T) {
	m := newActionTestModel()
	m.resources[testAddr] = &terraform.Resource{
		Address: testAddr,
		Action:  terraform.ActionNoop,
	}

	newModel, cmd := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypePlannedChange,
		Resource: &terraform.Resource{
			Address: testAddr,
			Action:  terraform.ActionUpdate,
			Reason:  "drift",
		},
	}))
	m = newModel.(Model)

	require.NotNil(t, cmd)
	assert.Equal(t, terraform.ActionUpdate, m.resources[testAddr].Action)
	assert.Equal(t, "drift", m.resources[testAddr].Reason)
}

func TestHandleActionEvent_ResourceDriftUpdatesResources(t *testing.T) {
	m := newActionTestModel()
	m.resources[testAddr] = &terraform.Resource{
		Address: testAddr,
		Action:  terraform.ActionNoop,
	}

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypeResourceDrift,
		Resource: &terraform.Resource{
			Address: testAddr,
			Action:  terraform.ActionUpdate,
		},
	}))
	m = newModel.(Model)

	assert.Equal(t, terraform.ActionUpdate, m.resources[testAddr].Action)
}

func TestHandleActionEvent_PlannedChangeCreatesProgressForSelectedResource(t *testing.T) {
	m := newActionTestModel()
	newAddr := "aws_s3_bucket.brand_new"
	m.selected = map[string]bool{newAddr: true}

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypePlannedChange,
		Resource: &terraform.Resource{
			Address: newAddr,
			Action:  terraform.ActionCreate,
		},
	}))
	m = newModel.(Model)

	require.Contains(t, m.progresses, newAddr)
	assert.Equal(t, progressStatusPending, m.progresses[newAddr].Status)
	assert.Equal(t, newAddr, m.progresses[newAddr].Address)
}

func TestHandleActionEvent_PlannedChangeCreatesProgressUnderSelectedModule(t *testing.T) {
	m := newActionTestModel()
	newAddr := "module.foo.aws_s3.brand_new"
	m.selected = map[string]bool{"module.foo": true}

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypePlannedChange,
		Resource: &terraform.Resource{
			Address: newAddr,
			Module:  "module.foo",
			Action:  terraform.ActionCreate,
		},
	}))
	m = newModel.(Model)

	require.Contains(t, m.progresses, newAddr)
	assert.Equal(t, progressStatusPending, m.progresses[newAddr].Status)
}

func TestHandleActionEvent_PlannedChangeNoProgressForUnselectedResource(t *testing.T) {
	m := newActionTestModel()
	unrelated := "aws_s3_bucket.unrelated"

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypePlannedChange,
		Resource: &terraform.Resource{
			Address: unrelated,
			Action:  terraform.ActionCreate,
		},
	}))
	m = newModel.(Model)

	// m.resources still updated so list view reflects the new plan after the action.
	assert.Equal(t, terraform.ActionCreate, m.resources[unrelated].Action)
	// No progress entry — resource isn't under the active selection.
	assert.NotContains(t, m.progresses, unrelated)
}

func TestHandleActionEvent_PlannedChangePreservesAttributes(t *testing.T) {
	m := newActionTestModel()
	m.resources[testAddr] = &terraform.Resource{
		Address:    testAddr,
		Action:     terraform.ActionNoop,
		Attributes: []byte(`{"id":"existing"}`),
	}

	newModel, _ := m.Update(streamEventMsg(terraform.StreamEvent{
		Type: terraform.MsgTypePlannedChange,
		Resource: &terraform.Resource{
			Address: testAddr,
			Action:  terraform.ActionUpdate,
			// Attributes intentionally nil — should be preserved from existing
		},
	}))
	m = newModel.(Model)

	assert.Equal(t, terraform.ActionUpdate, m.resources[testAddr].Action)
	assert.JSONEq(t, `{"id":"existing"}`, string(m.resources[testAddr].Attributes))
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
