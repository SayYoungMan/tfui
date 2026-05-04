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

	var event *StreamEvent
	switch msg.Type {
	case "refresh_start", "refresh_complete", "apply_start", "apply_progress", "apply_complete", "apply_errored":
		if msg.Hook != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Hook.Resource, normalizeAction(msg.Hook.Action), ""),
				Type:     msg.Type,
			}
		}
	case "resource_drift":
		if msg.Change != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Change.Resource, normalizeAction(msg.Change.Action), "drift"),
				Type:     msg.Type,
			}
		}
	case "planned_change":
		if msg.Change != nil {
			event = &StreamEvent{
				Resource: extractResourceInfo(&msg.Change.Resource, normalizeAction(msg.Change.Action), msg.Change.Reason),
				Type:     msg.Type,
			}
		}
	case "diagnostic":
		if msg.Diagnostic != nil {
			event = &StreamEvent{
				Diagnostic: msg.Diagnostic,
				Type:       msg.Type,
			}
		}
	case "change_summary":
		if msg.Changes != nil {
			event = &StreamEvent{
				Summary: msg.Changes,
				Type:    msg.Type,
			}
		}
	case "outputs":
		if msg.Outputs != nil {
			event = &StreamEvent{
				Outputs: msg.Outputs,
				Type:    msg.Type,
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

func normalizeAction(raw string) Action {
	switch raw {
	case "create":
		return ActionCreate
	case "read":
		return ActionRead
	case "update":
		return ActionUpdate
	case "delete":
		return ActionDelete
	case "replace":
		return ActionReplace
	case "move":
		return ActionMove
	case "import":
		return ActionImport
	case "no-op", "noop":
		return ActionNoop
	default:
		return ActionNoop
	}
}
