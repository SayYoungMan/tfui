package ui

import (
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
	viewState    viewState
	viewHeight   int
	viewWidth    int
	eventChannel <-chan terraform.StreamEvent
	cancel       func()

	resources terraform.Resources
	selected  map[string]bool
	indexMap  map[string]int
	cursor    int // indicates which resource idx we are pointing at
	offset    int // indicates which resource is shown at the top

	filteredIdx []int
	filterInput textinput.Model

	hideUnchanged bool
	isScanning    bool
	spinner       spinner.Model
	err           error

	actionCursor  int
	confirmCursor int
}

var actionChoices []string = []string{"plan", "apply", "destroy", "taint", "untaint"}

type viewState int

const (
	viewList viewState = iota
	viewFilter
	viewActionPicker
	viewConfirm
	viewOutput
)

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
		m.viewWidth = msg.Width
		m.filterInput.SetWidth(msg.Width - 7)
		return m, nil

	case tea.KeyPressMsg:
		switch m.viewState {
		case viewFilter:
			return m.filterModeKeys(msg)
		case viewActionPicker:
			return m.actionPickerKeys(msg)
		case viewConfirm:
			return m.confirmKeys(msg)
		default:
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
			wasUnchanged := isUnchanged(m.resources[idx])
			m.resources[idx] = *event.Resource

			// handle the case where it was matching but hidden due to hideUnchanged but now showing because it's changed now
			if m.hideUnchanged && wasUnchanged && !isUnchanged(*event.Resource) && m.matchesFilter(*event.Resource) {
				m.filteredIdx = append(m.filteredIdx, idx)
			}
		} else {
			newIdx := len(m.resources)
			m.indexMap[addr] = newIdx
			m.resources = append(m.resources, *event.Resource)

			if m.matchesFilter(*event.Resource) {
				m.filteredIdx = append(m.filteredIdx, m.indexMap[addr])
			}
		}
	}

	m.adjustOffset()
	return m, waitForEvent(m.eventChannel)
}

func (m Model) View() tea.View {
	switch m.viewState {
	case viewActionPicker:
		return tea.NewView(m.renderActionPickerView())
	case viewConfirm:
		return tea.NewView(m.renderConfirmView())
	default:
		return tea.NewView(m.renderListView())
	}
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
