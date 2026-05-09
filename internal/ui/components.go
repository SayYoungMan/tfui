package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type modalOpts struct {
	width        int
	height       int
	contentStyle *lipgloss.Style
}

func (m Model) renderModal(content string, opts *modalOpts) *lipgloss.Layer {
	modal := focusedBorderStyle.Render(content)
	if opts != nil && opts.contentStyle != nil {
		modal = opts.contentStyle.Render(content)
	}

	modalWidth := lipgloss.Width(modal)
	if opts != nil && opts.width != 0 {
		modalWidth = opts.width
	}
	modalHeight := lipgloss.Height(modal)
	if opts != nil && opts.height != 0 {
		modalHeight = opts.height
	}

	x := max(0, (m.viewWidth-modalWidth)/2)
	y := max(0, (m.viewHeight-modalHeight)/2)

	return lipgloss.NewLayer(modal).X(x).Y(y).Z(1)
}

func (m Model) renderModalWithBackground(fg, bg string, opts *modalOpts) string {
	modalLayer := m.renderModal(fg, opts)
	background := lipgloss.NewLayer(bg)

	return lipgloss.NewCompositor(background, modalLayer).Render()
}

type keyInfo struct {
	key  string
	info string
}

func (m Model) renderKeyInfo(keyInfos []keyInfo) string {
	var styledKeyInfos []string
	for _, k := range keyInfos {
		key := helpKeyStyle.Render("'" + k.key + "'")
		info := helpInfoStyle.Render(" " + k.info)
		styledKeyInfos = append(styledKeyInfos, key+info)
	}

	infoLines := strings.Join(styledKeyInfos, "  ")
	if m.viewWidth <= lipgloss.Width(infoLines) {
		mid := (len(styledKeyInfos) + 1) / 2
		line1 := strings.Join(styledKeyInfos[:mid], "  ")
		line2 := strings.Join(styledKeyInfos[mid:], "  ")
		infoLines = line1 + "\n" + line2
	}

	return infoLines
}

func (m Model) renderConfirmCancelButtons() string {
	cancelButton := buttonStyle.Render("Cancel")
	confirmButton := buttonStyle.Render("Confirm")
	if m.confirmCursor == 0 {
		cancelButton = focusedButtonStyle.Render("Cancel")
	} else {
		confirmButton = focusedButtonStyle.Render("Confirm")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, "  ", confirmButton)
}
