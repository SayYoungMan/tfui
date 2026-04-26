package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/chroma/v2/quick"
)

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
