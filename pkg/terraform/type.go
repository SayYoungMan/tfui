package terraform

import "encoding/json"

// StreamEvent is sent over the channel to be consumed.
// Exactly one field will be non-nil per event.
type StreamEvent struct {
	Resource   *Resource
	Diagnostic *Diagnostic
	Summary    *ChangeSummary
	Outputs    map[string]OutputValue
	Message    string
	Type       string
	Error      error
}

// Action describes what terraform is planning to do with the resource
// List taken from https://developer.hashicorp.com/terraform/internals/machine-readable-ui#action-1
type Action string

const (
	ActionCreate    Action = "create"
	ActionRead      Action = "read"
	ActionUpdate    Action = "update"
	ActionDelete    Action = "delete"
	ActionReplace   Action = "replace"
	ActionMove      Action = "move"
	ActionImport    Action = "import"
	ActionNoop      Action = "no-op"
	ActionUncertain Action = "uncertain" // Not an actual terraform action but set when we do state pull so we don't know its actual state
)

// Returns what symbol to be shown as pre-adornment
func (a Action) Symbol() string {
	switch a {
	case ActionCreate:
		return "+"
	case ActionUpdate:
		return "~"
	case ActionDelete:
		return "-"
	case ActionReplace:
		return "+/-"
	case ActionMove:
		return "→"
	case ActionImport:
		return "↓"
	case ActionRead, ActionNoop, ActionUncertain:
		return " "
	default:
		return "?"
	}
}

// https://developer.hashicorp.com/terraform/internals/machine-readable-ui#resource-object
type Resource struct {
	Address         string // Full unique address, e.g. "module.vpc.aws_subnet.private[0]"
	Module          string // Module path, e.g. "module.vpc", or "" for root
	ResourceAddr    string // Module-relative address, e.g. "aws_subnet.private[0]"
	ResourceType    string // Type of resource e.g. "aws_subnet"
	ResourceName    string // Name label of resource e.g. "private"
	ResourceKey     any    // Address key such as, count index (float64) or for_each key (string), or nil
	ImpliedProvider string // e.g. "aws"
	Action          Action
	Reason          string          // Why this change is happening, e.g. "tainted", "cannot_update"
	Attributes      json.RawMessage // JSON detail about this resource populated by state pull
}

// Implement these methods to satisfy interface of fuzzy matching
type Resources []Resource

func (r Resources) String(i int) string {
	return r[i].Address
}

func (r Resources) Len() int {
	return len(r)
}

// --- Below are raw JSON structures from `terraform plan/apply -json` ---

// struct for every line of plan/apply -json output.
type Message struct {
	Level      string                 `json:"@level"`
	Message    string                 `json:"@message"`
	Module     string                 `json:"@module"`
	Timestamp  string                 `json:"@timestamp"`
	Type       string                 `json:"type"`
	Change     *ChangePayload         `json:"change,omitempty"`
	Hook       *HookPayload           `json:"hook,omitempty"`
	Diagnostic *Diagnostic            `json:"diagnostic,omitempty"`
	Changes    *ChangeSummary         `json:"changes,omitempty"`
	Outputs    map[string]OutputValue `json:"outputs,omitempty"`
}

// ChangePayload is the body of a "planned_change" or "resource_drift" message.
type ChangePayload struct {
	Resource         ResourceInfo  `json:"resource"`
	PreviousResource *ResourceInfo `json:"previous_resource,omitempty"`
	Action           string        `json:"action"`
	Reason           string        `json:"reason,omitempty"`
}

// ResourceInfo identifies a resource within a change or hook payload.
type ResourceInfo struct {
	Addr            string `json:"addr"`
	Module          string `json:"module"`
	Resource        string `json:"resource"`
	ResourceType    string `json:"resource_type"`
	ResourceName    string `json:"resource_name"`
	ResourceKey     any    `json:"resource_key"`
	ImpliedProvider string `json:"implied_provider"`
}

// HookPayload is the body of apply_start, apply_progress, apply_complete,
// apply_errored, refresh_start, refresh_complete, provision_*, and
// ephemeral_op_* messages.
type HookPayload struct {
	Resource       ResourceInfo `json:"resource"`
	Action         string       `json:"action,omitempty"`
	IDKey          string       `json:"id_key,omitempty"`
	IDValue        string       `json:"id_value,omitempty"`
	ElapsedSeconds int          `json:"elapsed_seconds,omitempty"`
	Provisioner    string       `json:"provisioner,omitempty"`
	Output         string       `json:"output,omitempty"`
}

// Diagnostic carries error/warning diagnostics.
type Diagnostic struct {
	Severity string `json:"severity"` // "error" or "warning"
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
}

// ChangeSummary is the body of a "change_summary" message.
type ChangeSummary struct {
	Add       int    `json:"add"`
	Change    int    `json:"change"`
	Remove    int    `json:"remove"`
	Operation string `json:"operation"` // "plan", "apply", or "destroy"
}

// OutputValue represents a single root module output.
type OutputValue struct {
	Action    string `json:"action,omitempty"`
	Value     any    `json:"value,omitempty"`
	Type      any    `json:"type,omitempty"`
	Sensitive bool   `json:"sensitive"`
}
