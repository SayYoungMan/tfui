package ui

import "charm.land/lipgloss/v2"

var (
	colorCreamWhite = lipgloss.Color("230")
	colorCharcoal   = lipgloss.Color("234")
	colorLightGrey  = lipgloss.Color("240")
)

var (
	cursorStyle   = lipgloss.NewStyle().Background(colorCreamWhite).Foreground(colorCharcoal)
	selectedStyle = lipgloss.NewStyle().Background(colorLightGrey)
)
