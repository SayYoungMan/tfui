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
	context    context.Context
	runner     *terraform.TerraformRunner
	viewState  viewState
	viewHeight int
	viewWidth  int
	eventCh    <-chan terraform.StreamEvent

	cancel    func()
	quitState quitState

	resources        terraform.Resources
	resourceIndexMap map[string]int

	rows      []Row
	collapsed map[string]bool
	selected  map[string]bool
	cursor    int // indicates which resource idx we are pointing at
	offset    int // indicates which resource is shown at the top

	filterInput   textinput.Model
	hideUnchanged bool

	workState   workState
	spinner     spinner.Model
	err         error
	diagnostics []terraform.Diagnostic

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

type workState int

const (
	workIdle      workState = iota // Not doing any terraform work
	workStatePull                  // doing `terraform state pull` for inital population of resources
	workPlan                       // doing `terraform plan` to scan resource states
	workAction                     // doing action chosen by user
)

type quitState int

const (
	noneQuitState quitState = iota
	confirmQuitState
	quittingState
	forceQuitReadyState
)

type rowKind int

const (
	rowResource rowKind = iota
	rowModule
)

type Row struct {
	Kind       rowKind
	TreePrefix string
	Address    string
	Parent     string
}

func NewModel(runner *terraform.TerraformRunner, ctx context.Context, cancel func()) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		context:          ctx,
		runner:           runner,
		cancel:           cancel,
		resources:        []terraform.Resource{},
		collapsed:        make(map[string]bool),
		selected:         make(map[string]bool),
		resourceIndexMap: make(map[string]int),
		filterInput:      newFilterInput(),
		workState:        workStatePull,
		spinner:          s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.waitForState(),
	)
}

type statePulledMsg struct {
	resources []terraform.Resource
	err       error
}

func (m *Model) waitForState() tea.Cmd {
	return func() tea.Msg {
		resources, err := m.runner.StatePull(m.context)
		return statePulledMsg{resources: resources, err: err}
	}
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
	m.quitState = quittingState
	m.cancel()
	if !m.isRunning() {
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
			if m.quitState == forceQuitReadyState {
				return m, tea.Quit
			}
			if m.quitState == noneQuitState {
				m.quitState = confirmQuitState
			}
			return m, nil
		}
		if m.quitState == confirmQuitState {
			return m.quitViewConfirmKeys(msg)
		}
		// ignore input if it's quitting
		if m.quitState == quittingState {
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
				if m.offset < m.maxOutputOffset() {
					m.offset++
				}
			}
		case viewList:
			if msg.Button == tea.MouseWheelUp && m.cursor > 0 {
				m.cursor--
				m.adjustOffset()
			} else if msg.Button == tea.MouseWheelDown && m.cursor < len(m.rows)-1 {
				m.cursor++
				m.adjustOffset()
			}
		}
		return m, nil

	case statePulledMsg:
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
		m.workState = workPlan
		ch := m.runner.StreamPlan(m.context)
		m.eventCh = ch
		return m, waitForEvent(ch)

	case streamEventMsg:
		return m.handleStreamEvent(terraform.StreamEvent(msg))

	case scanCompleteMsg:
		m.workState = workIdle
		if m.quitState == quittingState || m.quitState == forceQuitReadyState {
			return m, tea.Quit
		}
		if m.hasError() {
			m.viewState = viewError
		}
		return m, nil

	case outputLineMsg:
		m.outputLines = append(m.outputLines, string(msg))
		maxOff := m.maxOutputOffset()
		if m.offset >= maxOff {
			m.offset = maxOff
		}
		return m, waitForOutput(m.outputCh)

	case outputCompleteMsg:
		m.workState = workIdle
		if m.quitState == quittingState || m.quitState == forceQuitReadyState {
			return m, tea.Quit
		}
		return m, nil

	case forceQuitReadyMsg:
		m.quitState = forceQuitReadyState
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) isRunning() bool {
	return m.workState != workIdle
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

	switch m.quitState {
	case confirmQuitState:
		viewString = lipgloss.NewCompositor(lipgloss.NewLayer(dimStyle.Render(viewString)), m.renderQuitConfirmLayer()).Render()
	case quittingState, forceQuitReadyState:
		viewString = lipgloss.NewCompositor(lipgloss.NewLayer(dimStyle.Render(viewString)), m.renderShutdownLayer()).Render()
	}

	v := tea.NewView(viewString)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}

// filter box (3) + resource borders (2) + info bar (1) + blank + help bar
const defaultReservedRows = 8

func (m Model) visibleRows() int {
	reserved := defaultReservedRows
	if m.viewWidth < 90 {
		reserved++
	}

	return max(1, m.viewHeight-reserved)
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

const (
	defaultReservedOutputWidth = 6
	defaultReservedOutputRows  = 6
)

func (m Model) maxOutputOffset() int {
	contentWidth := max(1, m.viewWidth-defaultReservedOutputWidth)
	boxHeight := max(1, m.viewHeight-defaultReservedOutputRows)

	total := 0
	for i := len(m.outputLines) - 1; i >= 0; i-- {
		lineWidth := lipgloss.Width(m.outputLines[i])
		rows := max(1, (lineWidth+contentWidth-1)/contentWidth)
		total += rows
		if total >= boxHeight {
			return i
		}
	}
	return 0
}

func (m Model) startRescan() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// initialize
	m.context = ctx
	m.resources = m.resources[:0]
	m.resourceIndexMap = make(map[string]int)
	m.rows = m.rows[:0]
	m.collapsed = make(map[string]bool)
	m.selected = make(map[string]bool)
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
	m.workState = workAction
	m.offset = 0
	m.viewState = viewOutput

	return m, waitForOutput(m.outputCh)
}
