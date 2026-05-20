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

type progressStatus int

const (
	progressStatusPending          progressStatus = iota // Before 'refresh_start' arrives
	progressStatusReadingState                           // While refreshing
	progressStatusWaitingForAction                       // After 'refresh_complete' but before 'apply_start'
	progressStatusInProgress                             // During apply
	progressStatusSuccessful
	progressStatusFailed
	progressStatusSkipped // This happens when you want to apply change to a resource with no change
)

type Progress struct {
	Address            string
	Status             progressStatus
	OutputLines        []string
	ReadStartedAt      time.Time
	ReadCompletedAt    time.Time
	ProcessStartedAt   time.Time
	ProcessCompletedAt time.Time
}

// duration of how long it waited to be picked up for refresh
func (p *Progress) waitDuration(startTime time.Time) time.Duration {
	// If resource is explicitly skipped (e.g. tainting data source), don't show wait time
	if p.Status == progressStatusSkipped {
		return 0
	}
	// For taint / untaint it doesn't refresh state so wait time is until process start
	if !p.ProcessStartedAt.IsZero() {
		return p.ProcessStartedAt.Sub(startTime)
	}
	if p.ReadStartedAt.IsZero() {
		return time.Since(startTime)
	}
	return p.ReadStartedAt.Sub(startTime)
}

// duration of how long the refresh took place
func (p *Progress) readDuration() time.Duration {
	// For taint, there is no refreshing state
	if p.ReadStartedAt.IsZero() {
		return 0
	}

	if p.ReadCompletedAt.IsZero() {
		return time.Since(p.ReadStartedAt)
	}
	return p.ReadCompletedAt.Sub(p.ReadStartedAt)
}

// duration of how long the action took place
func (p *Progress) processDuration() time.Duration {
	if p.ProcessStartedAt.IsZero() {
		return 0
	}

	if p.ProcessCompletedAt.IsZero() {
		return time.Since(p.ProcessStartedAt)
	}
	return p.ProcessCompletedAt.Sub(p.ProcessStartedAt)
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
	m.resources = make(map[string]*terraform.Resource)
	m.rows = m.rows[:0]
	m.collapsed = make(map[string]bool)
	m.selected = make(map[string]bool)
	m.selectAll = false
	m.progresses = nil
	m.progressRows = nil
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

	m.outputLines = nil
	m.workState = workAction
	m.viewState = viewProgress
	m.cursor = 0
	m.offset = 0
	m.actionStartTime = time.Now()

	resources := m.selectedResources()
	m.progresses = make(map[string]*Progress, len(resources))
	m.progressRows = make([]*Progress, 0, len(resources))
	for _, resource := range resources {
		addr := resource.Address
		p := &Progress{
			Address: addr,
			Status:  progressStatusPending,
		}
		m.progresses[addr] = p
		m.progressRows = append(m.progressRows, p)
	}

	action := actionChoices[m.actionCursor]
	actionFuncs := map[string]func(context.Context, []string) <-chan terraform.StreamEvent{
		"plan":    m.runner.Plan,
		"apply":   m.runner.Apply,
		"destroy": m.runner.Destroy,
		"taint":   m.runner.Taint,
		"untaint": m.runner.Untaint,
	}

	addrs := m.targetedAddresses()
	if action == "taint" || action == "untaint" {
		// 'taint' and 'untaint' does not support -target module so we -target resources under it
		addrs = addrs[:0]
		for _, r := range resources {
			if r.IsDataSource() {
				m.progresses[r.Address].Status = progressStatusSkipped
				continue
			}
			addrs = append(addrs, r.Address)
		}
	}
	ch := actionFuncs[action](ctx, addrs)
	m.eventCh = ch

	return m, tea.Batch(waitForEvent(ch), tickEverySecond())
}

func (m *Model) openDetail() {
	r := m.rows[m.cursor].Item.Resource

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
