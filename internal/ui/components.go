package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type modalOpts struct {
	width        int
	height       int
	contentStyle *lipgloss.Style
}

func (m Model) renderModal(content string, opts *modalOpts) *lipgloss.Layer {
	var modal string
	if opts != nil && opts.contentStyle != nil {
		modal = opts.contentStyle.Render(content)
	} else {
		modal = focusedBorderStyle.Render(content)
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
		infoLines = line1 + "\n " + line2
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

func (m Model) renderScrollableBox(contents []string, width, height int) string {
	// take borders and padding into account
	innerHeight := height - 6
	innerWidth := width - 6

	var contentBuilder strings.Builder
	visualRows := 0
	for i := m.offset; i < len(contents); i++ {
		// calculate how many rows this line occupies
		lineRows := max(1, (lipgloss.Width(contents[i])+innerWidth-1)/innerWidth)
		if visualRows+lineRows > innerHeight {
			break
		}
		fmt.Fprintln(&contentBuilder, m.outputLines[i])
		visualRows += lineRows
	}

	content := strings.TrimSuffix(contentBuilder.String(), "\n")
	return borderStyle.Width(width).Height(height).Padding(1, 2).Render(content)
}
