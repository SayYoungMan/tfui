package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
	"github.com/alecthomas/chroma/v2/quick"
)

type (
	streamEventMsg  terraform.StreamEvent
	scanCompleteMsg struct{}
	// We wrap cancel function by struct so that we can move around the pointer to this wrapper around copies
	// This is needed because we don't want to have context as a field but Bubble tea uses methods with value receiver
	cancelWrapper struct{ fn func() }
)

type Model struct {
	runner     *terraform.TerraformRunner
	viewState  viewState
	viewHeight int
	viewWidth  int
	eventCh    <-chan terraform.StreamEvent

	cancel    *cancelWrapper
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
	viewDetail
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

func NewModel(runner *terraform.TerraformRunner) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		runner:           runner,
		resources:        []terraform.Resource{},
		collapsed:        make(map[string]bool),
		selected:         make(map[string]bool),
		resourceIndexMap: make(map[string]int),
		filterInput:      newFilterInput(),
		workState:        workStatePull,
		cancel:           &cancelWrapper{},
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
	if m.cancel.fn != nil {
		m.cancel.fn()
	}
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
		case viewDetail:
			return m.detailKeys(msg)
		default:
			return m.normalModeKeys(msg)
		}

	case tea.MouseWheelMsg:
		switch m.viewState {
		case viewOutput, viewDetail:
			if msg.Button == tea.MouseWheelUp && m.offset > 0 {
				m.offset--
			} else if msg.Button == tea.MouseWheelDown {
				if m.offset < len(m.outputLines)-1 {
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
		return m.handleStatePulled(msg)

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
		visible := m.viewHeight - defaultReservedOutputRows
		if len(m.outputLines)-m.offset > visible {
			m.offset = len(m.outputLines) - visible
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
	case viewDetail:
		viewString = m.renderDetailView()
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

func (m Model) startRescan() (tea.Model, tea.Cmd) {
	// initialize
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
	m.cancel.fn = cancel

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
