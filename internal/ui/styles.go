package ui

import "charm.land/lipgloss/v2"

var (
	colorCreamWhite = lipgloss.Color("230")
	colorLightGrey  = lipgloss.Color("240")
)

var (
	cursorStyle   = lipgloss.NewStyle().Background(colorCreamWhite).Foreground(lipgloss.Color("234"))
	selectedStyle = lipgloss.NewStyle().Background(colorLightGrey)
)
