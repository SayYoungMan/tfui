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
	streamEventMsg  terraform.StreamEvent
	scanCompleteMsg struct{}
)

func waitForEvent(ch <-chan terraform.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return scanCompleteMsg{}
		}
		return streamEventMsg(event)
	}
}

func (m Model) handleStreamEvent(event terraform.StreamEvent) (tea.Model, tea.Cmd) {
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
