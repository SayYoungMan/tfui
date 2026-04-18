package ui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

type (
	streamEventMsg  terraform.StreamEvent
	scanCompleteMsg struct{}
)

type Model struct {
	runner     *terraform.TerraformRunner
	viewState  viewState
	viewHeight int
	viewWidth  int
	eventCh    <-chan terraform.StreamEvent

	cancel         func()
	isQuitting     bool
	forceQuitReady bool

	resources terraform.Resources
	selected  map[string]bool
	indexMap  map[string]int
	cursor    int // indicates which resource idx we are pointing at
	offset    int // indicates which resource is shown at the top
	isRunning bool

	filteredIdx []int
	filterInput textinput.Model

	hideUnchanged bool
	spinner       spinner.Model
	err           error
	diagnostics   []terraform.Diagnostic

	actionCursor  int
	confirmCursor int

	outputLines []string
	outputCh    <-chan string
}

var actionChoices []string = []string{"plan", "apply", "destroy", "taint", "untaint"}

type viewState int

const (
	viewList viewState = iota
	viewFilter
	viewActionPicker
	viewConfirm
	viewOutput
	viewError
)

func NewModel(runner *terraform.TerraformRunner, ch <-chan terraform.StreamEvent, cancel func()) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		runner:      runner,
		eventCh:     ch,
		cancel:      cancel,
		resources:   []terraform.Resource{},
		selected:    make(map[string]bool),
		indexMap:    make(map[string]int),
		filterInput: newFilterInput(),
		isRunning:   true,
		spinner:     s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForEvent(m.eventCh),
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

func (m Model) gracefulQuit() (tea.Model, tea.Cmd) {
	m.isQuitting = true
	m.cancel()
	if !m.isRunning {
		return m, tea.Quit
	}
	return m, waitForForceQuit()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewHeight = msg.Height
		m.viewWidth = msg.Width
		m.filterInput.SetWidth(msg.Width - 7)
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if m.isQuitting && m.forceQuitReady {
				return m, tea.Quit
			}
			if !m.isQuitting {
				return m.gracefulQuit()
			}
			return m, nil
		}
		// ignore input if it's quitting
		if m.isQuitting {
			return m, nil
		}

		switch m.viewState {
		case viewFilter:
			return m.filterModeKeys(msg)
		case viewActionPicker:
			return m.actionPickerKeys(msg)
		case viewConfirm:
			return m.confirmKeys(msg)
		case viewOutput:
			return m.outputKeys(msg)
		case viewError:
			return m.errorKeys(msg)
		default:
			return m.normalModeKeys(msg)
		}

	case tea.MouseWheelMsg:
		switch m.viewState {
		case viewOutput:
			if msg.Button == tea.MouseWheelUp && m.offset > 0 {
				m.offset--
			} else if msg.Button == tea.MouseWheelDown {
				maxOff := max(0, len(m.outputLines)-m.visibleOutputRows())
				if m.offset < maxOff {
					m.offset++
				}
			}
		case viewList:
			if msg.Button == tea.MouseWheelUp && m.cursor > 0 {
				m.cursor--
				m.adjustOffset()
			} else if msg.Button == tea.MouseWheelDown && m.cursor < len(m.filteredIdx)-1 {
				m.cursor++
				m.adjustOffset()
			}
		}
		return m, nil

	case streamEventMsg:
		return m.handleStreamEvent(terraform.StreamEvent(msg))

	case scanCompleteMsg:
		m.isRunning = false
		if m.isQuitting {
			return m, tea.Quit
		}
		if m.hasError() {
			m.viewState = viewError
		}
		return m, nil

	case outputLineMsg:
		m.outputLines = append(m.outputLines, string(msg))
		maxOff := max(0, len(m.outputLines)-m.visibleOutputRows())
		if m.offset >= maxOff {
			m.offset = maxOff
		}
		return m, waitForOutput(m.outputCh)

	case outputCompleteMsg:
		m.isRunning = false
		if m.isQuitting {
			return m, tea.Quit
		}
		return m, nil

	case forceQuitReadyMsg:
		m.forceQuitReady = true
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) hasError() bool {
	for _, d := range m.diagnostics {
		if d.Severity == "error" {
			return true
		}
	}
	return m.err != nil
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
	return m, waitForEvent(m.eventCh)
}

func (m Model) View() tea.View {
	var viewString string
	switch m.viewState {
	case viewActionPicker:
		viewString = m.renderActionPickerView()
	case viewConfirm:
		viewString = m.renderConfirmView()
	case viewOutput:
		viewString = m.renderOutputView()
	case viewError:
		viewString = m.renderErrorView()
	default:
		viewString = m.renderListView()
	}

	if m.isQuitting {
		viewString = lipgloss.NewCompositor(lipgloss.NewLayer(dimStyle.Render(viewString)), m.renderShutdownLayer()).Render()
	}

	v := tea.NewView(viewString)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}

// 3(search bar) + 3(info) + 2(Key info) + 1(extra)
const defaultReservedRows = 9

func (m Model) visibleRows() int {
	rows := m.viewHeight - defaultReservedRows

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

// 2(borders) + 1(title) + 2(blank) + 1(help)
// TODO: There is a bug where the upper border explodes to top with lots of outputs
const defaultReservedOutputRows = 10

func (m Model) visibleOutputRows() int {
	reserved := defaultReservedRows
	if m.viewWidth < 90 {
		reserved++
	}
	return max(1, m.viewHeight-reserved)
}

func (m Model) startRescan() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// initialize
	m.resources = m.resources[:0]
	m.indexMap = make(map[string]int)
	m.filteredIdx = m.filteredIdx[:0]
	m.selected = make(map[string]bool)
	m.cursor = 0
	m.offset = 0
	m.err = nil
	m.diagnostics = nil
	m.isRunning = true
	m.outputLines = nil
	m.outputCh = nil
	m.viewState = viewList

	ch := m.runner.StreamPlan(ctx)
	m.eventCh = ch

	return m, tea.Batch(
		m.spinner.Tick,
		waitForEvent(ch),
	)
}

func (m Model) startAction() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	addrs := m.selectedAddresses()
	action := actionChoices[m.actionCursor]

	actionFuncs := map[string]func(context.Context, []string) <-chan string{
		"plan":    m.runner.Plan,
		"apply":   m.runner.Apply,
		"destroy": m.runner.Destroy,
		"taint":   m.runner.Taint,
		"untaint": m.runner.Untaint,
	}
	m.outputCh = actionFuncs[action](ctx, addrs)
	m.outputLines = nil
	m.isRunning = true
	m.offset = 0
	m.viewState = viewOutput

	return m, waitForOutput(m.outputCh)
}
