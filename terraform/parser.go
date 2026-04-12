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
	var msg PlanMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	switch msg.Type {
	case "refresh_complete":
		return p.parseRefreshComplete(msg.Hook)
	case "apply_complete":
		return p.parseApplyComplete(msg.Hook)
	case "resource_drift":
		return p.parseResourceDrift(msg.Change)
	case "planned_change":
		return p.parsePlannedChange(msg.Change)
	case "diagnostic":
		return p.parseDiagnostic(msg.Diagnostic)
	case "change_summary":
		return p.parseChangeSummary(msg.Changes)
	case "outputs":
		return p.parseOutputs(msg.Outputs)
	default:
		return nil, nil
	}
}

func (p *Parser) parseRefreshComplete(hook *HookPayload) (*StreamEvent, error) {
	addr := hook.Resource.Addr
	if !p.seen[addr] {
		p.seen[addr] = true
		p.resourceCount++
	}

	return &StreamEvent{Resource: extractResourceInfo(&hook.Resource, ActionNoop, "")}, nil
}

func (p *Parser) parseApplyComplete(hook *HookPayload) (*StreamEvent, error) {
	addr := hook.Resource.Addr
	if !p.seen[addr] {
		p.seen[addr] = true
		p.resourceCount++
	}

	// We want to record events of apply_complete only when it's finished reading data block
	if hook.Action != "read" {
		return nil, nil
	}

	return &StreamEvent{Resource: extractResourceInfo(&hook.Resource, ActionRead, "")}, nil
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
