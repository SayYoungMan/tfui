package terraform

import (
	"encoding/json"

	"charm.land/log/v2"
)

// ParseLine parses single line of JSON into a StreamEvent
func ParseLine(line []byte) *StreamEvent {
	if len(line) == 0 {
		return nil
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		log.Debug("failed to parse plan JSON", "error", err, "line", string(line))
		return nil
	}

	msgType := MessageType(msg.Type)
	var event *StreamEvent
	switch msgType {
	case MsgTypeRefreshStart, MsgTypeRefreshComplete, MsgTypeApplyStart, MsgTypeApplyProgress, MsgTypeApplyComplete, MsgTypeApplyErrored:
		if msg.Hook != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Hook.Resource, normalizeAction(msg.Hook.Action), ""),
				Type:     msgType,
			}
		}
	case MsgTypeResourceDrift:
		if msg.Change != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Change.Resource, normalizeAction(msg.Change.Action), "drift"),
				Type:     msgType,
			}
		}
	case MsgTypePlannedChange:
		if msg.Change != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Change.Resource, normalizeAction(msg.Change.Action), msg.Change.Reason),
				Type:     msgType,
			}
		}
	case MsgTypeDiagnostic:
		if msg.Diagnostic != nil {
			event = &StreamEvent{
				Diagnostic: msg.Diagnostic,
				Type:       msgType,
			}
		}
	case MsgTypeChangeSummary:
		if msg.Changes != nil {
			event = &StreamEvent{
				Summary: msg.Changes,
				Type:    msgType,
			}
		}
	case MsgTypeOutputs:
		if msg.Outputs != nil {
			event = &StreamEvent{
				Outputs: msg.Outputs,
				Type:    msgType,
			}
		}
	default:
		return nil
	}

	if event == nil {
		return nil
	}
	event.Message = msg.Message

	return event
}

func extractResourceInfo(info *ResourceInfo, action Action, reason string) *Resource {
	return &Resource{
		Address:         info.Addr,
		Module:          info.Module,
		ResourceAddr:    info.Resource,
		ResourceType:    info.ResourceType,
		ResourceName:    info.ResourceName,
		ResourceKey:     info.ResourceKey,
		ImpliedProvider: info.ImpliedProvider,
		Action:          action,
		Reason:          reason,
	}
}
