package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterSearch is the right-pane query input for nav/list filtering.
type FilterSearch struct {
	input   textinput.Model
	width   int
	height  int
	focused bool
	styles  filterSearchStyles
}

type filterSearchStyles struct {
	title lipgloss.Style
	hint  lipgloss.Style
}

func newFilterSearch() FilterSearch {
	in := textinput.New()
	in.Prompt = "/ "
	in.Placeholder = "Filter navigation and list…"
	in.CharLimit = 256
	return FilterSearch{
		input: in,
		styles: filterSearchStyles{
			title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
			hint:  lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		},
	}
}

func (f *FilterSearch) Query() string { return f.input.Value() }

func (f *FilterSearch) SetFocused(on bool) {
	f.focused = on
	if on {
		f.input.Focus()
	} else {
		f.input.Blur()
	}
}

func (f *FilterSearch) SetSize(w, h int) {
	if w < 12 {
		w = 12
	}
	if h < 6 {
		h = 6
	}
	f.width = w
	f.height = h
	inner := w - 2
	if inner < 8 {
		inner = 8
	}
	f.input.Width = inner
}

func (f *FilterSearch) Clear() {
	f.input.SetValue("")
}

func (f *FilterSearch) Update(msg tea.Msg) tea.Cmd {
	if !f.focused {
		return nil
	}
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return cmd
}

func (f FilterSearch) View() string {
	title := f.styles.title.Render("Filter")
	body := lipgloss.NewStyle().Width(f.width).Render(f.input.View())
	hint := f.styles.hint.Render("Filters left nav and center table · Esc clear · f focus")
	return strings.Join([]string{title, "", body, "", hint}, "\n")
}

func filterSearchBlinkCmd() tea.Cmd {
	return textinput.Blink
}
