package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
)

const listToolbarHeight = 2

type listZone int

const (
	listZoneTable listZone = iota
	listZoneFilter
	listZoneActions
)

// ListToolbarMeta is the navigation/sort summary shown above the filter row.
type ListToolbarMeta struct {
	Subject     string
	Count       int
	SortLabel   string
	LoadingMore bool
	SaveEnabled bool
	DeleteEnabled bool
}

// ListToolbar is the filter & navigation bar above the center table.
type ListToolbar struct {
	filter      FilterSearch
	width       int
	zone        listZone
	actionIdx   int
	actionRects []toolbarBtnRect
	btnX0       int
	styles      listToolbarStyles
}

type toolbarBtnRect struct {
	action model.ActionID
	x0, x1 int
}

type listToolbarStyles struct {
	title   lipgloss.Style
	btn     lipgloss.Style
	btnOn   lipgloss.Style
	btnOff  lipgloss.Style
	btnAct  lipgloss.Style
	btnDn   lipgloss.Style
}

func newListToolbar() ListToolbar {
	return ListToolbar{
		filter: newFilterSearch(),
		styles: listToolbarStyles{
			title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
			btn:    lipgloss.NewStyle().Padding(0, 1),
			btnOn:  lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("255")),
			btnOff: lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("238")),
			btnAct: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")),
			btnDn:  lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("52")),
		},
	}
}

func (b *ListToolbar) SetWidth(w int) {
	if w < 12 {
		w = 12
	}
	b.width = w
	inner := w - 8
	if inner < 12 {
		inner = 12
	}
	b.filter.SetInputWidth(inner)
}

func (b *ListToolbar) Zone() listZone { return b.zone }

func (b *ListToolbar) SetZone(z listZone) { b.zone = z }

func (b *ListToolbar) Query() string { return b.filter.Query() }

func (b *ListToolbar) ClearFilter() { b.filter.Clear() }

func (b *ListToolbar) syncInputFocus() {
	b.filter.SetFocused(b.zone == listZoneFilter)
}

func (b *ListToolbar) FocusFilter() {
	b.zone = listZoneFilter
	b.filter.SetFocused(true)
}

func (b *ListToolbar) FocusTable() {
	b.zone = listZoneTable
	b.filter.SetFocused(false)
}

func (b *ListToolbar) TabForward(meta ListToolbarMeta) {
	switch b.zone {
	case listZoneTable:
		b.FocusFilter()
	case listZoneFilter:
		b.zone = listZoneActions
		b.actionIdx = 0
		b.filter.SetFocused(false)
		b.clampActionIdx(meta)
	case listZoneActions:
		if b.nextEnabledAction(meta, 1) {
			return
		}
		b.FocusTable()
	}
}

func (b *ListToolbar) TabBackward(meta ListToolbarMeta) {
	switch b.zone {
	case listZoneTable:
		b.zone = listZoneActions
		b.actionIdx = b.lastEnabledAction(meta)
		b.filter.SetFocused(false)
	case listZoneActions:
		if b.nextEnabledAction(meta, -1) {
			return
		}
		b.FocusFilter()
	case listZoneFilter:
		b.FocusTable()
	}
}

func (b *ListToolbar) toolbarActions(meta ListToolbarMeta) []model.ActionID {
	var out []model.ActionID
	if meta.SaveEnabled {
		out = append(out, model.ActionSave)
	}
	if meta.DeleteEnabled {
		out = append(out, model.ActionDelete)
	}
	return out
}

func (b *ListToolbar) clampActionIdx(meta ListToolbarMeta) {
	acts := b.toolbarActions(meta)
	if len(acts) == 0 {
		b.actionIdx = 0
		return
	}
	if b.actionIdx >= len(acts) {
		b.actionIdx = len(acts) - 1
	}
	if b.actionIdx < 0 {
		b.actionIdx = 0
	}
}

func (b *ListToolbar) lastEnabledAction(meta ListToolbarMeta) int {
	acts := b.toolbarActions(meta)
	if len(acts) == 0 {
		return 0
	}
	return len(acts) - 1
}

func (b *ListToolbar) nextEnabledAction(meta ListToolbarMeta, delta int) bool {
	acts := b.toolbarActions(meta)
	if len(acts) == 0 {
		return false
	}
	next := b.actionIdx + delta
	if next >= 0 && next < len(acts) {
		b.actionIdx = next
		return true
	}
	return false
}

func (b *ListToolbar) SelectedAction(meta ListToolbarMeta) (model.ActionID, bool) {
	acts := b.toolbarActions(meta)
	if b.zone != listZoneActions || len(acts) == 0 {
		return "", false
	}
	b.clampActionIdx(meta)
	return acts[b.actionIdx], true
}

func (b *ListToolbar) MoveAction(delta int, meta ListToolbarMeta) {
	if b.zone != listZoneActions {
		return
	}
	b.nextEnabledAction(meta, delta)
}

func (b *ListToolbar) Update(msg tea.Msg, meta ListToolbarMeta) tea.Cmd {
	if b.zone == listZoneFilter {
		if key, ok := msg.(tea.KeyMsg); ok {
			cmd := b.filter.Update(msg)
			switch key.String() {
			case "tab":
				b.TabForward(meta)
				return cmd
			case "shift+tab", "backtab":
				b.TabBackward(meta)
				return cmd
			case "enter":
				b.zone = listZoneActions
				b.actionIdx = 0
				b.filter.SetFocused(false)
				b.clampActionIdx(meta)
				return cmd
			}
			return cmd
		}
		return b.filter.Update(msg)
	}
	if b.zone == listZoneActions {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "left", "h":
				b.MoveAction(-1, meta)
			case "right", "l":
				b.MoveAction(1, meta)
			case "tab":
				b.TabForward(meta)
			case "shift+tab", "backtab":
				b.TabBackward(meta)
			}
		}
	}
	return nil
}

func (b *ListToolbar) View(meta ListToolbarMeta) string {
	title := b.styles.title.Render(meta.Subject)
	if meta.Count > 0 || meta.Subject != "" {
		title = b.styles.title.Render(formatListToolbarTitle(meta))
	}
	if meta.LoadingMore {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  …")
	}

	filter := b.filter.CompactView()
	buttons, rects := b.renderButtons(meta)

	avail := b.width - runewidth.StringWidth(buttons) - 1
	if avail < 8 {
		avail = 8
	}
	b.btnX0 = avail + 1
	for i := range rects {
		rects[i].x0 += b.btnX0
		rects[i].x1 += b.btnX0
	}
	b.actionRects = rects

	filterLine := padDisplayWidth(filter, avail) + " " + buttons
	if runewidth.StringWidth(filterLine) > b.width {
		filterLine = runewidth.Truncate(filterLine, b.width, "")
	}

	lines := []string{
		padANSIWidth(title, b.width),
		padANSIWidth(filterLine, b.width),
	}
	return lipgloss.NewStyle().Width(b.width).MaxWidth(b.width).Render(strings.Join(lines, "\n"))
}

func formatListToolbarTitle(meta ListToolbarMeta) string {
	s := meta.Subject
	if meta.Count > 0 {
		s = fmt.Sprintf("%s (%d)", meta.Subject, meta.Count)
	}
	if meta.SortLabel != "" {
		s += "  " + meta.SortLabel
	}
	return s
}

func (b *ListToolbar) renderButtons(meta ListToolbarMeta) (string, []toolbarBtnRect) {
	type btn struct {
		action  model.ActionID
		icon    string
		enabled bool
		danger  bool
	}
	defs := []btn{
		{model.ActionSave, "💾", meta.SaveEnabled, false},
		{model.ActionDelete, "🗑", meta.DeleteEnabled, true},
	}
	acts := b.toolbarActions(meta)
	rects := make([]toolbarBtnRect, 0, len(defs))
	parts := make([]string, 0, len(defs))
	x := 0
	for i, d := range defs {
		active := b.zone == listZoneActions && len(acts) > 0 && b.actionIdx < len(acts) && acts[b.actionIdx] == d.action
		var style lipgloss.Style
		switch {
		case !d.enabled:
			style = b.styles.btnOff
		case active && d.danger:
			style = b.styles.btnDn
		case active:
			style = b.styles.btnAct
		case d.danger:
			style = b.styles.btn.Foreground(lipgloss.Color("203"))
		default:
			style = b.styles.btnOn
		}
		rendered := style.Render(d.icon)
		w := runewidth.StringWidth(rendered)
		rects = append(rects, toolbarBtnRect{action: d.action, x0: x, x1: x + w})
		x += w
		parts = append(parts, rendered)
		_ = i
	}
	return strings.Join(parts, " "), rects
}

func (b *ListToolbar) ActionAt(x int, meta ListToolbarMeta) (model.ActionID, bool) {
	_ = meta
	if x < 0 {
		return "", false
	}
	for _, r := range b.actionRects {
		if x >= r.x0 && x < r.x1 {
			action := r.action
			if b.IsActionEnabled(action, meta) {
				return action, true
			}
			return "", false
		}
	}
	return "", false
}

func (b *ListToolbar) IsActionEnabled(action model.ActionID, meta ListToolbarMeta) bool {
	switch action {
	case model.ActionSave:
		return meta.SaveEnabled
	case model.ActionDelete:
		return meta.DeleteEnabled
	default:
		return false
	}
}
