package terraform

import (
	"encoding/json"
	"fmt"
)

// Parser converts raw JSON line from `terraform plan -json` into StreamEvents.
type Parser struct {
	seen          map[string]bool // map of addresses already parsed
	resourceCount int
	warningCount  int
	errorCount    int
}

func NewParser() *Parser {
	return &Parser{
		seen: make(map[string]bool),
	}
}

func (p *Parser) Stats() (resources, errors, warnings int) {
	return p.resourceCount, p.errorCount, p.warningCount
}

// ParseLine parses single line of JSON into a StreamEvent
func (p *Parser) ParseLine(line []byte) (*StreamEvent, error) {
	if len(line) == 0 {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	var event *StreamEvent
	var err error
	switch msg.Type {
	case "refresh_start":
		event, err = p.parseRefreshStart(msg.Hook)
	case "apply_start", "apply_progress", "apply_complete", "apply_errored":
		event, err = &StreamEvent{
			Resource: extractResourceInfo(&msg.Hook.Resource, normalizeAction(msg.Hook.Action), ""),
			Hook:     msg.Hook,
			Type:     msg.Type,
		}, nil
	case "resource_drift":
		event, err = p.parseResourceDrift(msg.Change)
	case "planned_change":
		event, err = p.parsePlannedChange(msg.Change)
	case "diagnostic":
		event, err = p.parseDiagnostic(msg.Diagnostic)
	case "change_summary":
		event, err = p.parseChangeSummary(msg.Changes)
	case "outputs":
		event, err = p.parseOutputs(msg.Outputs)
	default:
		return nil, nil
	}

	if event != nil {
		event.Message = msg.Message
	}

	return event, err
}

func (p *Parser) parseRefreshStart(hook *HookPayload) (*StreamEvent, error) {
	addr := hook.Resource.Addr
	if !p.seen[addr] {
		p.seen[addr] = true
		p.resourceCount++
	}

	return &StreamEvent{Resource: extractResourceInfo(&hook.Resource, ActionNoop, "")}, nil
}

func (p *Parser) parseResourceDrift(change *ChangePayload) (*StreamEvent, error) {
	addr := change.Resource.Addr
	if !p.seen[addr] {
		p.seen[addr] = true
		p.resourceCount++
	}

	action := normalizeAction(change.Action)

	return &StreamEvent{Resource: extractResourceInfo(&change.Resource, action, "drift")}, nil
}

func (p *Parser) parsePlannedChange(change *ChangePayload) (*StreamEvent, error) {
	addr := change.Resource.Addr
	if !p.seen[addr] {
		p.seen[addr] = true
		p.resourceCount++
	}

	action := normalizeAction(change.Action)

	return &StreamEvent{Resource: extractResourceInfo(&change.Resource, action, change.Reason)}, nil
}

func (p *Parser) parseDiagnostic(diag *Diagnostic) (*StreamEvent, error) {
	switch diag.Severity {
	case "error":
		p.errorCount++
	case "warning":
		p.warningCount++
	}

	return &StreamEvent{Diagnostic: diag}, nil
}

func (p *Parser) parseChangeSummary(changes *ChangeSummary) (*StreamEvent, error) {
	return &StreamEvent{Summary: changes}, nil
}

func (p *Parser) parseOutputs(outputs map[string]OutputValue) (*StreamEvent, error) {
	return &StreamEvent{Outputs: outputs}, nil
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
