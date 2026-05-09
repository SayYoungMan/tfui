package ui

import "charm.land/lipgloss/v2"

type modalOpts struct {
	width        int
	height       int
	contentStyle *lipgloss.Style
}

func (m *Model) renderModal(content string, opts *modalOpts) *lipgloss.Layer {
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

func (m *Model) renderModalWithBackground(fg, bg string, opts *modalOpts) string {
	modalLayer := m.renderModal(fg, opts)
	background := lipgloss.NewLayer(bg)

	return lipgloss.NewCompositor(background, modalLayer).Render()
}
