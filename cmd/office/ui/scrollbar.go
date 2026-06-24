package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ScrollbarMetrics computes thumb position for a vertical scrollbar.
func ScrollbarMetrics(totalLines, viewportHeight, yOffset int) (show bool, thumbStart, thumbEnd int) {
	if totalLines <= viewportHeight || viewportHeight < 1 {
		return false, 0, 0
	}
	maxOff := totalLines - viewportHeight
	if maxOff < 0 {
		maxOff = 0
	}
	thumbSize := viewportHeight * viewportHeight / totalLines
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > viewportHeight {
		thumbSize = viewportHeight
	}
	travel := viewportHeight - thumbSize
	thumbStart = 0
	if maxOff > 0 {
		thumbStart = yOffset * travel / maxOff
	}
	return true, thumbStart, thumbStart + thumbSize
}

// ApplyVerticalScrollbar draws a 1-column scrollbar on the right when content overflows.
func ApplyVerticalScrollbar(view string, width, height, totalLines, yOffset int) string {
	show, thumbStart, thumbEnd := ScrollbarMetrics(totalLines, height, yOffset)
	if !show || width < 2 {
		return view
	}
	contentWidth := width - 1
	lines := strings.Split(view, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	track := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	thumb := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	out := make([]string, height)
	for row := 0; row < height; row++ {
		line := padDisplayWidth(lines[row], contentWidth)
		ch := "│"
		style := track
		if row >= thumbStart && row < thumbEnd {
			ch = "█"
			style = thumb
		}
		out[row] = line + style.Render(ch)
	}
	return strings.Join(out, "\n")
}

func padDisplayWidth(line string, width int) string {
	if width < 1 {
		return ""
	}
	w := runewidth.StringWidth(line)
	if w > width {
		return runewidth.Truncate(line, width, "")
	}
	if w < width {
		line += strings.Repeat(" ", width-w)
	}
	return line
}
