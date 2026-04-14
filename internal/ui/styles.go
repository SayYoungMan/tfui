package ui

import "charm.land/lipgloss/v2"

var (
	colorCreamWhite = lipgloss.Color("230")
	colorCharcoal   = lipgloss.Color("234")
	colorLightGrey  = lipgloss.Color("240")
	colorDimGrey    = lipgloss.Color("245")
)

var (
	cursorStyle        = lipgloss.NewStyle().Background(colorCreamWhite).Foreground(colorCharcoal)
	selectedStyle      = lipgloss.NewStyle().Background(colorLightGrey)
	borderStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDimGrey).Padding(0, 1)
	focusedBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorCreamWhite).Padding(0, 1)
)
