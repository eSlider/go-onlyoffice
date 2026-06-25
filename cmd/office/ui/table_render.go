package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// renderTableCell renders one column using the bubbles/table pattern:
// truncate plain text, constrain with an inline inner box, then apply the outer style.
func renderTableCell(outer lipgloss.Style, text string, width int) string {
	if width < 1 {
		width = 1
	}
	frame := outer.GetHorizontalFrameSize()
	innerW := width - frame
	if innerW < 1 {
		innerW = 1
	}
	truncated := runewidth.Truncate(text, innerW, "…")
	inner := lipgloss.NewStyle().Width(innerW).MaxWidth(innerW).Inline(true)
	return outer.Render(inner.Render(truncated))
}

// padANSIWidth pads or truncates a styled line without breaking ANSI sequences.
func padANSIWidth(line string, width int) string {
	if width < 1 {
		return ""
	}
	w := ansi.StringWidth(line)
	if w > width {
		return ansi.Truncate(line, width, "")
	}
	if w < width {
		return line + repeatSpace(width-w)
	}
	return line
}

func repeatSpace(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
