package ui

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
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
	actionResources  map[string]*ActionResource
	actionStartTime  time.Time

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

// We wrap cancel function by struct so that we can move around the pointer to this wrapper around copies
// This is needed because we don't want to have context as a field but Bubble tea uses methods with value receiver
type cancelWrapper struct{ fn func() }

var actionChoices []string = []string{"plan", "apply", "destroy", "taint", "untaint"}

type viewState int

const (
	viewList viewState = iota
	viewFilter
	viewActionPicker
	viewConfirm
	viewActionResources
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

type actionResourceStatus int

const (
	actionResourcePending actionResourceStatus = iota
	actionResourceInProgress
	actionResourceSuccessful
	actionResourceFailed
	actionResourceSkipped // This happens when you want to apply change to a resource with no change
)

type ActionResource struct {
	Address       string
	Status        actionResourceStatus
	TimeToStart   time.Duration
	TimeToProcess time.Duration
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
			return m.quitConfirmKeys(msg)
		}
		// ignore input if it's quitting
		if m.quitState == quittingState {
			return m, nil
		}

		switch m.viewState {
		case viewFilter:
			return m.filterKeys(msg)
		case viewActionPicker:
			return m.actionPickerKeys(msg)
		case viewConfirm:
			return m.confirmKeys(msg)
		case viewActionResources:
			return m.actionResourcesKeys(msg)
		case viewOutput:
			return m.outputKeys(msg)
		case viewError:
			return m.errorKeys(msg)
		case viewDetail:
			return m.detailKeys(msg)
		default:
			return m.listKeys(msg)
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

	case streamCompleteMsg:
		m.workState = workIdle

		// If some resources are still pending that means they have no change
		for _, ar := range m.actionResources {
			if ar.Status == actionResourcePending {
				ar.Status = actionResourceSkipped
			}
		}

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

	case actionTickMsg:
		return m, tickEverySecond()
	}

	return m, nil
}

func (m Model) View() tea.View {
	var viewString string
	switch m.viewState {
	case viewActionPicker:
		viewString = m.renderActionPickerView()
	case viewConfirm:
		viewString = m.renderConfirmView()
	case viewActionResources, viewOutput:
		viewString = m.renderActionResourcesView()
		if m.viewState == viewOutput {
			viewString = m.renderOutputLayer(viewString)
		}
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
