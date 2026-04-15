package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
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

	resources  terraform.Resources
	selected   map[string]bool
	indexMap   map[string]int
	cursor     int // indicates which resource idx we are pointing at
	offset     int // indicates which resource is shown at the top
	viewHeight int

	filteredIdx   []int
	filterInput   textinput.Model
	filterFocused bool

	hideNoops  bool
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
		selected:     make(map[string]bool),
		indexMap:     make(map[string]int),
		filterInput:  newFilterInput(),
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
		m.filterInput.SetWidth(msg.Width - 7)
		return m, nil

	case tea.KeyPressMsg:
		if m.filterFocused {
			return m.filterModeKeys(msg)
		} else {
			return m.normalModeKeys(msg)
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
			newIdx := len(m.resources)
			m.indexMap[addr] = newIdx
			m.resources = append(m.resources, *event.Resource)
		}
		if m.matchesFilter(*event.Resource) {
			m.filteredIdx = append(m.filteredIdx, m.indexMap[addr])
		}
	}

	m.adjustOffset()
	return m, waitForEvent(m.eventChannel)
}

func (m Model) View() tea.View {
	var s strings.Builder

	filterIcon := "⌕ "
	filterContent := filterIcon + m.filterInput.View()
	if m.filterFocused {
		fmt.Fprintln(&s, focusedBorderStyle.Render(filterContent))
	} else {
		fmt.Fprintln(&s, borderStyle.Render(filterContent))
	}
	fmt.Fprintln(&s)

	end := min(m.offset+m.visibleRows(), len(m.filteredIdx))
	for i := m.offset; i < end; i++ {
		r := m.resources[m.filteredIdx[i]]
		symbol := r.Action.Symbol()
		reason := ""
		if r.Reason != "" {
			reason = fmt.Sprintf(" (%s)", r.Reason)
		}
		line := fmt.Sprintf("%s %s%s", symbol, r.Address, reason)

		switch {
		case i == m.cursor:
			line = cursorStyle.Render(line)
		case m.selected[r.Address]:
			line = selectedStyle.Render(line)
		}
		if style, ok := actionStyles[r.Action]; ok {
			line = style.Render(line)
		}

		fmt.Fprintln(&s, line)
	}

	var infoLine string
	if m.isScanning {
		infoLine = fmt.Sprintf("\n %s Scanning... (%d resources found)", m.spinner.View(), len(m.resources))
	} else {
		infoLine = fmt.Sprintf("\n Scan Complete (%d resources found)", len(m.resources))
	}
	if m.filterInput.Value() != "" {
		infoLine += fmt.Sprintf(" | showing %d", len(m.filteredIdx))
	}
	if len(m.selected) > 0 {
		infoLine += fmt.Sprintf(" | %d selected", len(m.selected))
	}
	fmt.Fprintln(&s, infoLine)

	if m.err != nil {
		fmt.Fprintf(&s, "\n error occurred: %v\n", m.err)
	}

	var hKeyInfo string
	if m.hideNoops {
		hKeyInfo = "h to show unchanged"
	} else {
		hKeyInfo = "h to hide unchanged"
	}
	keyInfoLine := fmt.Sprintf("\n / to filter | <Space> to select | %s | q or ctrl+C to quit.\n", hKeyInfo)
	s.WriteString(keyInfoLine)

	return tea.NewView(s.String())
}

// 4(search bar) + 3(info) + 2(Key info) + 1(extra)
const defaultReservedRows = 10

func (m Model) visibleRows() int {
	reserved := defaultReservedRows
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
