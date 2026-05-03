package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

type statePulledMsg struct {
	resources []terraform.Resource
	err       error
}

func (m *Model) waitForState() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel.fn = cancel

	return func() tea.Msg {
		resources, err := m.runner.StatePull(ctx)
		return statePulledMsg{resources: resources, err: err}
	}
}

func (m Model) handleStatePulled(msg statePulledMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		// If the error is due to trying to quit in the middle, just quit
		if m.quitState == quittingState || m.quitState == forceQuitReadyState {
			return m, tea.Quit
		}
		m.err = msg.err
		m.workState = workIdle
		m.viewState = viewError
		return m, nil
	}

	for _, r := range msg.resources {
		m.resourceIndexMap[r.Address] = len(m.resources)
		m.resources = append(m.resources, r)
	}
	m.rebuildRows()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel.fn = cancel
	m.workState = workPlan

	ch := m.runner.StreamPlan(ctx)
	m.eventCh = ch
	return m, waitForEvent(ch)
}

type (
	streamEventMsg    terraform.StreamEvent
	streamCompleteMsg struct{}
)

func waitForEvent(ch <-chan terraform.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return streamCompleteMsg{}
		}
		return streamEventMsg(event)
	}
}

func (m Model) handleStreamEvent(event terraform.StreamEvent) (tea.Model, tea.Cmd) {
	if m.workState == workAction {
		return m.handleActionEvent(event)
	}

	if event.Error != nil {
		m.err = event.Error
		return m, waitForEvent(m.eventCh)
	}

	if event.Diagnostic != nil {
		m.diagnostics = append(m.diagnostics, *event.Diagnostic)
		return m, waitForEvent(m.eventCh)
	}

	if event.Resource != nil {
		addr := event.Resource.Address
		if idx, exists := m.resourceIndexMap[addr]; exists {
			existing := m.resources[idx]
			updated := *event.Resource
			updated.Attributes = existing.Attributes
			m.resources[idx] = updated
		} else {
			newIdx := len(m.resources)
			m.resourceIndexMap[addr] = newIdx
			m.resources = append(m.resources, *event.Resource)
		}
		m.rebuildRows()
	}

	m.adjustOffset()
	return m, waitForEvent(m.eventCh)
}

func (m Model) handleActionEvent(event terraform.StreamEvent) (tea.Model, tea.Cmd) {
	if event.Message != "" {
		m.outputLines = append(m.outputLines, event.Message)
	}

	if event.Resource == nil {
		return m, waitForEvent(m.eventCh)
	}

	ar, ok := m.actionResources[event.Resource.Address]
	if !ok {
		// There are some apply_start and apply_complete from data sources that are not selected
		return m, waitForEvent(m.eventCh)
	}

	currentAction := actionChoices[m.actionCursor]
	switch event.Type {
	case "refresh_start":
		ar.Status = actionResourceReadingState
		ar.ReadStartedAt = time.Now()
	case "refresh_complete":
		if currentAction == "plan" {
			ar.Status = actionResourceSuccessful
		} else {
			ar.Status = actionResourceWaitingForAction
		}
		ar.ReadCompletedAt = time.Now()
	case "apply_start":
		ar.Status = actionResourceInProgress
		ar.ProcessStartedAt = time.Now()
	case "apply_complete":
		ar.Status = actionResourceSuccessful
		ar.ProcessCompletedAt = time.Now()
	case "apply_errored":
		ar.Status = actionResourceFailed
		ar.ProcessCompletedAt = time.Now()
	}

	return m, waitForEvent(m.eventCh)
}

type (
	outputLineMsg     string
	outputCompleteMsg struct{}
)

func waitForOutput(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return outputCompleteMsg{}
		}
		return outputLineMsg(line)
	}
}

type forceQuitReadyMsg struct{}

func waitForForceQuit() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return forceQuitReadyMsg{}
	})
}

type actionTickMsg time.Time

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return actionTickMsg(t)
	})
}
