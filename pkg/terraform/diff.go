package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type planFile struct {
	ResourceChanges []planResourceChange `json:"resource_changes"`
}

type planResourceChange struct {
	Address string        `json:"address"`
	Change  PlannedChange `json:"change"`
}

func (tr *TerraformRunner) PlannedChanges(ctx context.Context) (map[string]PlannedChange, error) {
	cmd := tr.cmdFactory(ctx, tr.binary, "show", "-json", planFilePath)
	cmd.Dir = tr.workdir
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}

	var stdErr strings.Builder
	cmd.Stderr = &stdErr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform show plan failed: %w: %s", err, strings.TrimSpace(stdErr.String()))
	}

	var plan planFile
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	changes := make(map[string]PlannedChange, len(plan.ResourceChanges))
	for _, rc := range plan.ResourceChanges {
		if isMeaningfulChange(rc.Change) {
			changes[rc.Address] = rc.Change
		}
	}

	return changes, nil
}

func isMeaningfulChange(change PlannedChange) bool {
	for _, action := range change.Actions {
		switch action {
		case "", "noop", "no-op", "read":
			continue
		default:
			return true
		}
	}
	return false
}
