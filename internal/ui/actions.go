package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/alecthomas/chroma/v2/quick"
)

type actionResourceStatus int

const (
	actionResourcePending          actionResourceStatus = iota // Before 'refresh_start' arrives
	actionResourceReadingState                                 // While refreshing
	actionResourceWaitingForAction                             // After 'refresh_complete' but before 'apply_start'
	actionResourceInProgress                                   // During apply
	actionResourceSuccessful
	actionResourceFailed
	actionResourceSkipped // This happens when you want to apply change to a resource with no change
)

type ActionResource struct {
	Address            string
	Status             actionResourceStatus
	ReadStartedAt      time.Time
	ReadCompletedAt    time.Time
	ProcessStartedAt   time.Time
	ProcessCompletedAt time.Time
}

// duration of how long it waited to be picked up for refresh
func (ar *ActionResource) waitDuration(startTime time.Time) time.Duration {
	if ar.ReadStartedAt.IsZero() {
		return time.Since(startTime)
	}
	return ar.ReadStartedAt.Sub(startTime)
}

// duration of how long the refresh took place
func (ar *ActionResource) readDuration() time.Duration {
	// For taint, there is no refreshing state
	if ar.ReadStartedAt.IsZero() {
		return 0
	}

	if ar.ReadCompletedAt.IsZero() {
		return time.Since(ar.ReadStartedAt)
	}
	return ar.ReadCompletedAt.Sub(ar.ReadStartedAt)
}

// duration of how long the action took place
func (ar *ActionResource) processDuration() time.Duration {
	if ar.ProcessStartedAt.IsZero() {
		return 0
	}

	if ar.ProcessCompletedAt.IsZero() {
		return time.Since(ar.ProcessStartedAt)
	}
	return ar.ProcessCompletedAt.Sub(ar.ProcessStartedAt)
}

func (m Model) gracefulQuit() (tea.Model, tea.Cmd) {
	m.quitState = quittingState
	if m.cancel.fn != nil {
		m.cancel.fn()
	}
	if !m.isRunning() {
		return m, tea.Quit
	}
	return m, waitForForceQuit()
}

func (m Model) startRescan() (tea.Model, tea.Cmd) {
	// initialize
	m.resources = m.resources[:0]
	m.resourceIndexMap = make(map[string]int)
	m.rows = m.rows[:0]
	m.collapsed = make(map[string]bool)
	m.selected = make(map[string]bool)
	m.actionResources = nil
	m.cursor = 0
	m.offset = 0
	m.err = nil
	m.diagnostics = nil
	m.workState = workStatePull
	m.outputLines = nil
	m.outputCh = nil
	m.viewState = viewList

	return m, tea.Batch(
		m.spinner.Tick,
		m.waitForState(),
	)
}

func (m Model) startAction() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel.fn = cancel

	addrs := m.selectedAddresses()
	m.outputLines = nil
	m.workState = workAction
	m.viewState = viewActionResources
	m.offset = 0

	m.actionStartTime = time.Now()
	m.actionResources = make(map[string]*ActionResource, len(addrs))
	for _, addr := range addrs {
		m.actionResources[addr] = &ActionResource{
			Address: addr,
			Status:  actionResourcePending,
		}
	}

	action := actionChoices[m.actionCursor]
	switch action {
	case "plan", "apply", "destroy":
		actionFuncs := map[string]func(context.Context, []string) <-chan terraform.StreamEvent{
			"plan":    m.runner.Plan,
			"apply":   m.runner.Apply,
			"destroy": m.runner.Destroy,
		}
		ch := actionFuncs[action](ctx, addrs)
		m.eventCh = ch
		return m, tea.Batch(waitForEvent(ch), tickEverySecond())
	default:
		actionFuncs := map[string]func(context.Context, []string) <-chan string{
			"taint":   m.runner.Taint,
			"untaint": m.runner.Untaint,
		}
		m.outputCh = actionFuncs[action](ctx, addrs)

		return m, waitForOutput(m.outputCh)
	}
}

func (m *Model) openDetail() {
	addr := m.rows[m.cursor].Address
	r := m.resources[m.resourceIndexMap[addr]]

	m.offset = 0
	m.viewState = viewDetail

	if len(r.Attributes) == 0 {
		m.outputLines = []string{"No details available."}
		return
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, r.Attributes, "", "  "); err != nil {
		m.outputLines = strings.Split(string(r.Attributes), "\n")
		return
	}

	var highlighted bytes.Buffer
	if err := quick.Highlight(&highlighted, indented.String(), "json", "terminal256", "catppuccin-mocha"); err != nil {
		m.outputLines = strings.Split(indented.String(), "\n")
		return
	}

	m.outputLines = strings.Split(strings.TrimRight(highlighted.String(), "\n"), "\n")
}
