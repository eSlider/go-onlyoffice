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
	tabStop   int
	zone      detailZone
	focused   bool
	width     int
	height    int
	form      EntityForm
	docVP     viewport.Model
	formVP    viewport.Model
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
		formVP: viewport.New(40, 10),
		styles: newDetailStyles(),
	}
	d.docVP.MouseWheelEnabled = true
	d.formVP.MouseWheelEnabled = true
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
	d.tabStop = 0
	d.docText = ""
	d.form.Clear()
	d.docVP.SetContent("")
	d.formVP.SetContent("")
}

func (d *DetailPane) SetFocused(on bool) {
	d.focused = on
	d.applyTabStop(d.tabStop)
}

func (d *DetailPane) FocusFirstStop() {
	d.tabStop = 0
	d.applyTabStop(0)
}

func (d *DetailPane) maxTabStop() int {
	switch d.mode {
	case detailForm:
		if d.form.readOnly {
			if len(d.actions) == 0 {
				return 0
			}
			return len(d.actions) - 1
		}
		fields := d.form.FieldCount()
		if len(d.actions) == 0 {
			if fields == 0 {
				return 0
			}
			return fields - 1
		}
		return fields + len(d.actions) - 1
	case detailDocument:
		if len(d.actions) == 0 {
			return 0
		}
		return len(d.actions)
	default:
		return 0
	}
}

func (d *DetailPane) applyTabStop(stop int) {
	if stop < 0 {
		stop = 0
	}
	max := d.maxTabStop()
	if stop > max {
		stop = max
	}
	d.tabStop = stop

	switch d.mode {
	case detailForm:
		if d.form.readOnly {
			d.zone = detailZoneActions
			d.actionIdx = stop
			d.form.SetFocused(false)
			return
		}
		fields := d.form.FieldCount()
		if stop < fields {
			d.zone = detailZoneContent
			d.form.SetFieldIndex(stop)
			d.form.SetFocused(d.focused)
			return
		}
		d.zone = detailZoneActions
		d.actionIdx = stop - fields
		d.form.SetFocused(false)
	case detailDocument:
		if stop == 0 {
			d.zone = detailZoneContent
			d.form.SetFocused(false)
			return
		}
		d.zone = detailZoneActions
		d.actionIdx = stop - 1
		d.form.SetFocused(false)
	default:
		d.zone = detailZoneContent
		d.form.SetFocused(false)
	}
}

// TabForward moves to the next field/button. It returns true when focus should leave the pane.
func (d *DetailPane) TabForward() bool {
	max := d.maxTabStop()
	if d.tabStop < max {
		d.applyTabStop(d.tabStop + 1)
		return false
	}
	d.FocusFirstStop()
	return true
}

// TabBackward moves to the previous field/button. It returns true when focus should leave the pane.
func (d *DetailPane) TabBackward() bool {
	if d.tabStop > 0 {
		d.applyTabStop(d.tabStop - 1)
		return false
	}
	d.applyTabStop(d.maxTabStop())
	return true
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
	d.formVP.Width = w
	d.formVP.Height = contentH
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
	d.tabStop = 0
	d.form.Load(item.Kind, item.ID, fields)
	d.refreshFormViewport()
	d.applyTabStop(0)
	d.layoutContent()
}

func (d *DetailPane) LoadDocument(item model.Item, markdown string, renderWidth int) {
	d.mode = detailDocument
	d.item = item
	d.loadedID = item.ID
	d.actions = model.ActionsFor(item.Kind)
	d.actionIdx = 0
	d.tabStop = 0
	d.form.Clear()
	text, err := preview.RenderMarkdown(markdown, renderWidth)
	if err != nil {
		text = markdown
	}
	d.docText = text
	d.docVP.SetContent(text)
	d.docVP.GotoTop()
	d.applyTabStop(0)
	d.layoutContent()
}

func (d *DetailPane) layoutContent() {
	contentH, _ := d.splitHeights()
	d.form.SetSize(d.width, contentH)
	d.docVP.Width = d.width
	d.docVP.Height = contentH
	d.formVP.Width = d.width
	d.formVP.Height = contentH
	d.refreshFormViewport()
}

func (d *DetailPane) refreshFormViewport() {
	if d.mode != detailForm || !d.form.readOnly {
		d.formVP.SetContent("")
		return
	}
	d.formVP.SetContent(d.form.View())
}

func (d *DetailPane) LoadedID() string { return d.loadedID }

func (d *DetailPane) Item() model.Item { return d.item }

func (d *DetailPane) Zone() detailZone { return d.zone }

func (d *DetailPane) FocusContent() {
	d.applyTabStop(d.tabStop)
	if d.zone == detailZoneContent && d.mode == detailForm {
		d.form.SetFocused(d.focused)
	}
}

func (d *DetailPane) FocusActions() {
	max := d.maxTabStop()
	switch d.mode {
	case detailForm:
		if d.form.readOnly {
			d.applyTabStop(max)
			return
		}
		if len(d.actions) > 0 {
			d.applyTabStop(d.form.FieldCount())
		}
	case detailDocument:
		if len(d.actions) > 0 {
			d.applyTabStop(1)
		}
	}
}

// MoveTabStop moves between title, description, and action buttons without leaving the pane.
func (d *DetailPane) MoveTabStop(delta int) {
	if delta == 0 {
		return
	}
	next := d.tabStop + delta
	if next < 0 {
		next = 0
	}
	max := d.maxTabStop()
	if next > max {
		next = max
	}
	d.applyTabStop(next)
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

func (d DetailPane) IsDocumentContent() bool {
	return d.mode == detailDocument
}

func (d DetailPane) isReadOnlyFormContent() bool {
	return d.mode == detailForm && d.form.readOnly
}

func (d DetailPane) documentContentHeight() int {
	contentH, _ := d.splitHeights()
	return contentH
}

// ScrollDocument scrolls the read-only document viewport (mail, file preview).
func (d *DetailPane) ScrollDocument(key string) bool {
	if d.IsDocumentContent() {
		return scrollViewport(&d.docVP, key)
	}
	if d.isReadOnlyFormContent() && d.zone == detailZoneContent {
		return scrollViewport(&d.formVP, key)
	}
	return false
}

// ScrollDocumentMouse applies wheel events to scrollable preview content.
func (d *DetailPane) ScrollDocumentMouse(msg tea.MouseMsg) tea.Cmd {
	var vp *viewport.Model
	switch {
	case d.IsDocumentContent():
		vp = &d.docVP
	case d.isReadOnlyFormContent():
		vp = &d.formVP
	default:
		return nil
	}
	if msg.Y >= d.documentContentHeight() {
		return nil
	}
	if !vp.MouseWheelEnabled || msg.Action != tea.MouseActionPress {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
	default:
		return nil
	}
	var cmd tea.Cmd
	*vp, cmd = vp.Update(msg)
	return cmd
}

// DocumentYOffset exposes scroll position for tests.
func (d DetailPane) DocumentYOffset() int {
	return d.docVP.YOffset
}

func (d *DetailPane) Update(msg tea.Msg) tea.Cmd {
	if !d.focused {
		return nil
	}
	if d.zone == detailZoneContent {
		switch d.mode {
		case detailForm:
			if d.form.readOnly {
				if key, ok := msg.(tea.KeyMsg); ok {
					if scrollViewport(&d.formVP, key.String()) {
						return nil
					}
				}
				var cmd tea.Cmd
				d.formVP, cmd = d.formVP.Update(msg)
				return cmd
			}
			return d.form.Update(msg)
		case detailDocument:
			if key, ok := msg.(tea.KeyMsg); ok {
				if scrollViewport(&d.docVP, key.String()) {
					return nil
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
		if d.form.readOnly {
			top = ApplyVerticalScrollbar(
				d.formVP.View(),
				d.formVP.Width,
				d.formVP.Height,
				d.formVP.TotalLineCount(),
				d.formVP.YOffset,
			)
		} else {
			top = lipgloss.NewStyle().Width(d.width).Height(contentH).Render(d.form.View())
		}
	case detailDocument:
		top = ApplyVerticalScrollbar(
			d.docVP.View(),
			d.docVP.Width,
			d.docVP.Height,
			d.docVP.TotalLineCount(),
			d.docVP.YOffset,
		)
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
		line += d.styles.empty.Render("  Tab · Enter run")
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
