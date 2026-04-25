package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// top-level struct of terraform state pull
type StateFile struct {
	Version          int             `json:"version"`
	TerraformVersion string          `json:"terraform_version"`
	Serial           int             `json:"serial"`
	Lineage          string          `json:"lineage"`
	Resources        []StateResource `json:"resources"`
}

// single resource block
type StateResource struct {
	Module    string          `json:"module"` // "" for root
	Mode      string          `json:"mode"`   // "managed" or "data"
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Provider  string          `json:"provider"`
	Instances []StateInstance `json:"instances"`
}

// single instance of resource
type StateInstance struct {
	SchemaVersion int             `json:"schema_version"`
	IndexKey      any             `json:"index_key"`
	Status        string          `json:"status,omitempty"` // "tainted" or absent
	Attributes    json.RawMessage `json:"attributes"`
	Dependencies  []string        `json:"dependencies,omitempty"`
}

func (tr *TerraformRunner) StatePull(ctx context.Context) ([]Resource, error) {
	cmd := tr.cmdFactory(ctx, tr.binary, "state", "pull")
	cmd.Dir = tr.workdir
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}

	var stdErr strings.Builder
	cmd.Stderr = &stdErr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s state pull: %w", tr.binary, err)
	}

	data, readErr := io.ReadAll(stdout)
	waitErr := cmd.Wait()

	if readErr != nil {
		return nil, fmt.Errorf("failed to read state pull output: %w", readErr)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("%s state pull failed: %w: %s", tr.binary, waitErr, stdErr.String())
	}

	return ParseState(data)
}

// Parses state pull to generate list of internal resources
func ParseState(data []byte) ([]Resource, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}
	if sf.Version != 4 {
		return nil, fmt.Errorf("only state file version 4 is supported but got %d", sf.Version)
	}

	var resources []Resource
	for _, sr := range sf.Resources {
		for _, si := range sr.Instances {
			r, err := instanceToResource(sr, si)
			if err != nil {
				return nil, err
			}
			resources = append(resources, r)
		}
	}

	return resources, nil
}

func instanceToResource(sr StateResource, si StateInstance) (Resource, error) {
	resourceAddr := sr.Type + "." + sr.Name
	if sr.Mode == "data" {
		resourceAddr = "data." + resourceAddr
	}

	switch si.IndexKey.(type) {
	case string:
		resourceAddr += fmt.Sprintf("[%q]", si.IndexKey)
	case float64:
		resourceAddr += fmt.Sprintf("[%d]", int(si.IndexKey.(float64)))
	}

	addr := resourceAddr
	if sr.Module != "" {
		addr = sr.Module + "." + addr
	}

	action := ActionUncertain
	reason := ""
	if si.Status == "tainted" {
		reason = "tainted"
	}

	return Resource{
		Address:         addr,
		Module:          sr.Module,
		ResourceAddr:    resourceAddr,
		ResourceType:    sr.Type,
		ResourceName:    sr.Name,
		ResourceKey:     si.IndexKey,
		ImpliedProvider: sr.Provider,
		Action:          action,
		Reason:          reason,
		Attributes:      si.Attributes,
	}, nil
}
