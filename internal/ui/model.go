package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

type (
	streamEventMsg  terraform.StreamEvent
	scanCompleteMsg struct{}
)

type Model struct {
	eventChannel <-chan terraform.StreamEvent
	cancel       func()

	resources  []terraform.Resource
	indexMap   map[string]int
	cursor     int // indicates which resource idx we are pointing at
	offset     int // indicates which resource is shown at the top
	viewHeight int

	isScanning bool
	spinner    spinner.Model
	err        error
}

func NewModel(ch <-chan terraform.StreamEvent, cancel func()) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		eventChannel: ch,
		cancel:       cancel,
		resources:    []terraform.Resource{},
		indexMap:     make(map[string]int),
		isScanning:   true,
		spinner:      s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForEvent(m.eventChannel),
	)
}

func waitForEvent(ch <-chan terraform.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return scanCompleteMsg{}
		}
		return streamEventMsg(event)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewHeight = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.resources)-1 {
				m.cursor++
				m.adjustOffset()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.adjustOffset()
			}
		}

	case streamEventMsg:
		return m.handleStreamEvent(terraform.StreamEvent(msg))

	case scanCompleteMsg:
		m.isScanning = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleStreamEvent(event terraform.StreamEvent) (tea.Model, tea.Cmd) {
	if event.Error != nil {
		m.err = event.Error
		return m, waitForEvent(m.eventChannel)
	}

	if event.Resource != nil {
		addr := event.Resource.Address
		if idx, exists := m.indexMap[addr]; exists {
			m.resources[idx] = *event.Resource
		} else {
			m.indexMap[addr] = len(m.resources)
			m.resources = append(m.resources, *event.Resource)
		}
	}

	m.adjustOffset()
	return m, waitForEvent(m.eventChannel)
}

func (m Model) View() tea.View {
	var s strings.Builder

	end := min(m.offset+m.visibleRows(), len(m.resources))

	for i := m.offset; i < end; i++ {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		r := m.resources[i]
		symbol := r.Action.Symbol()
		reason := ""
		if r.Reason != "" {
			reason = fmt.Sprintf(" (%s)", r.Reason)
		}
		fmt.Fprintf(&s, "%s %s %s%s\n", cursor, symbol, r.Address, reason)
	}

	if m.isScanning {
		fmt.Fprintf(&s, "\n %s Scanning... (%d resources found)\n", m.spinner.View(), len(m.resources))
	} else {
		fmt.Fprintf(&s, "\n Scan Complete (%d resources found)\n", len(m.resources))
	}

	if m.err != nil {
		fmt.Fprintf(&s, "\n error occurred: %v\n", m.err)
	}

	s.WriteString("\n q or ctrl+C to quit.\n")

	return tea.NewView(s.String())
}

func (m Model) visibleRows() int {
	reserved := 5
	if m.err != nil {
		reserved++
	}

	rows := m.viewHeight - reserved

	return max(1, rows)
}

func (m *Model) adjustOffset() {
	visible := m.visibleRows()

	// Cursor went below visible area — scroll down
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}

	// Cursor went above visible area — scroll up
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}
