package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

const detailContentRatio = 0.72

type detailMode int

const (
	detailEmpty detailMode = iota
	detailDocument
	detailForm
)

type detailZone int

const (
	detailZoneContent detailZone = iota
	detailZoneActions
)

// DetailPane is the right column: content top, CRUD actions bottom.
type DetailPane struct {
	mode      detailMode
	item      model.Item
	loadedID  string
	actions   []model.ItemAction
	actionIdx int
	zone      detailZone
	focused   bool
	width     int
	height    int
	form      EntityForm
	docVP     viewport.Model
	docText   string
	styles    detailStyles
}

type detailStyles struct {
	sep      lipgloss.Style
	action   lipgloss.Style
	actionOn lipgloss.Style
	actionDn lipgloss.Style
	empty    lipgloss.Style
}

func newDetailPane() DetailPane {
	d := DetailPane{
		form:   newEntityForm(),
		docVP:  viewport.New(40, 10),
		styles: newDetailStyles(),
	}
	d.docVP.MouseWheelEnabled = true
	return d
}

func newDetailStyles() detailStyles {
	btn := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)
	return detailStyles{
		sep:      lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		action:   btn.Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236")),
		actionOn: btn.Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")),
		actionDn: btn.Foreground(lipgloss.Color("255")).Background(lipgloss.Color("52")),
		empty:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
}

func (d *DetailPane) Clear() {
	d.mode = detailEmpty
	d.item = model.Item{}
	d.loadedID = ""
	d.actions = nil
	d.actionIdx = 0
	d.docText = ""
	d.form.Clear()
	d.docVP.SetContent("")
}

func (d *DetailPane) SetFocused(on bool) {
	d.focused = on
	d.form.SetFocused(on && d.zone == detailZoneContent && d.mode == detailForm)
}

func (d *DetailPane) SetSize(w, h int) {
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
	}
	d.width = w
	d.height = h
	contentH, _ := d.splitHeights()
	d.form.SetSize(w, contentH)
	d.docVP.Width = w
	d.docVP.Height = contentH
}

func (d *DetailPane) splitHeights() (contentH, actionH int) {
	actionH = int(float64(d.height) * (1 - detailContentRatio))
	if actionH < 3 {
		actionH = 3
	}
	if actionH > 6 {
		actionH = 6
	}
	contentH = d.height - actionH - 1
	if contentH < 4 {
		contentH = 4
	}
	return contentH, actionH
}

func (d *DetailPane) LoadForm(item model.Item, fields model.FormFields) {
	d.mode = detailForm
	d.item = item
	d.loadedID = item.ID
	d.actions = model.ActionsFor(item.Kind)
	d.actionIdx = 0
	d.form.Load(item.Kind, item.ID, fields)
	d.layoutContent()
}

func (d *DetailPane) LoadDocument(item model.Item, markdown string, renderWidth int) {
	d.mode = detailDocument
	d.item = item
	d.loadedID = item.ID
	d.actions = model.ActionsFor(item.Kind)
	d.actionIdx = 0
	d.form.Clear()
	text, err := preview.RenderMarkdown(markdown, renderWidth)
	if err != nil {
		text = markdown
	}
	d.docText = text
	d.docVP.SetContent(text)
	d.layoutContent()
}

func (d *DetailPane) layoutContent() {
	contentH, _ := d.splitHeights()
	d.form.SetSize(d.width, contentH)
	d.docVP.Width = d.width
	d.docVP.Height = contentH
}

func (d *DetailPane) LoadedID() string { return d.loadedID }

func (d *DetailPane) Item() model.Item { return d.item }

func (d *DetailPane) Zone() detailZone { return d.zone }

func (d *DetailPane) FocusActions() {
	d.zone = detailZoneActions
	d.form.SetFocused(false)
}

func (d *DetailPane) FocusContent() {
	d.zone = detailZoneContent
	d.form.SetFocused(d.focused && d.mode == detailForm)
}

func (d *DetailPane) ToggleZone() {
	if d.zone == detailZoneContent {
		d.FocusActions()
	} else {
		d.FocusContent()
	}
}

func (d *DetailPane) MoveAction(delta int) {
	if len(d.actions) == 0 {
		return
	}
	d.actionIdx = clampInt(d.actionIdx+delta, 0, len(d.actions)-1)
}

func (d *DetailPane) SelectedAction() (model.ItemAction, bool) {
	if d.actionIdx < 0 || d.actionIdx >= len(d.actions) {
		return model.ItemAction{}, false
	}
	return d.actions[d.actionIdx], true
}

func (d *DetailPane) Update(msg tea.Msg) tea.Cmd {
	if !d.focused {
		return nil
	}
	if d.zone == detailZoneContent {
		switch d.mode {
		case detailForm:
			return d.form.Update(msg)
		case detailDocument:
			if _, ok := msg.(tea.KeyMsg); ok {
				switch msg.(tea.KeyMsg).String() {
				case "pgdown", "pgdn", "f", "ctrl+d":
					d.docVP.ViewDown()
				case "pgup", "b", "ctrl+u":
					d.docVP.ViewUp()
				}
			}
			var cmd tea.Cmd
			d.docVP, cmd = d.docVP.Update(msg)
			return cmd
		}
	}
	return nil
}

func (d DetailPane) View() string {
	if d.mode == detailEmpty {
		return d.styles.empty.Render("Select a row to preview or edit.")
	}
	contentH, actionH := d.splitHeights()
	var top string
	switch d.mode {
	case detailForm:
		top = lipgloss.NewStyle().Width(d.width).Height(contentH).Render(d.form.View())
	case detailDocument:
		top = d.docVP.View()
	default:
		top = ""
	}
	sep := d.styles.sep.Width(d.width).Render(strings.Repeat("─", max(1, d.width)))
	bottom := lipgloss.NewStyle().Width(d.width).Height(actionH).Render(d.renderActions())
	return lipgloss.JoinVertical(lipgloss.Left, top, sep, bottom)
}

func (d DetailPane) renderActions() string {
	if len(d.actions) == 0 {
		hint := "No actions"
		if d.focused && d.zone == detailZoneActions {
			hint = lipgloss.NewStyle().Background(lipgloss.Color("238")).Render(" No actions ")
		}
		return d.styles.empty.Render(hint)
	}
	parts := make([]string, 0, len(d.actions))
	for i, act := range d.actions {
		style := d.styles.action
		if d.focused && d.zone == detailZoneActions && i == d.actionIdx {
			if act.Danger {
				style = d.styles.actionDn
			} else {
				style = d.styles.actionOn
			}
		} else if act.Danger {
			style = d.styles.actionDn
		}
		parts = append(parts, style.Render(act.Label))
	}
	line := strings.Join(parts, "")
	if d.focused && d.zone == detailZoneActions {
		line += d.styles.empty.Render("  ←/→ select · Enter run")
	}
	return line
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (d DetailPane) BlinkCmd() tea.Cmd {
	return entityFormBlinkCmd()
}

// NewDetailPaneForTest exposes detail pane for tests.
func NewDetailPaneForTest() DetailPane {
	return newDetailPane()
}
