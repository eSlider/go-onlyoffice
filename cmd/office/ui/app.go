package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/fetch"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

type listLoadedMsg struct {
	items []model.Item
	spec  model.ListSpec
	err   error
}

type previewLoadedMsg struct {
	text string
	err  error
}

type actionDoneMsg struct {
	message string
	err     error
}

type navProjectsMsg struct {
	projects []model.Item
	err      error
}

// Model is the root Bubble Tea model for the office TUI.
type Model struct {
	client    *onlyoffice.Client
	loader    *fetch.Loader
	nav       *model.NavTree
	listSpec  model.ListSpec
	hasList   bool
	items     []model.Item
	selection *model.Selection
	listIdx   int
	focus     model.FocusPane
	width     int
	height    int
	status    string
	err       string
	loading   bool
	menuVP    viewport.Model
	listVP    viewport.Model
	previewVP viewport.Model
	previewMD string
	// Action menu overlay on list pane
	actionMode  bool
	actionIdx   int
	itemActions []model.ItemAction
}

// NewModel constructs the TUI with an authenticated client.
func NewModel(client *onlyoffice.Client) Model {
	return Model{
		client:    client,
		loader:    &fetch.Loader{Client: client},
		nav:       model.DefaultNavTree(),
		selection: model.NewSelection(),
		focus:     model.FocusMenu,
		status:    "Tab/Shift+Tab: pane · Enter: open/activate · a: actions · q: quit",
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadNavProjectsCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutViewports()
		m.syncAllContent()
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if key == "?" {
			m.status = helpText()
			return m, nil
		}
		if m.actionMode && m.focus == model.FocusList {
			return m.handleActionKey(key)
		}
		action := KeyAction(key, m.focus)
		if action == ActionNone && (key == "up" || key == "k") {
			action = ResolveMoveUp(m.focus)
		}
		switch action {
		case ActionQuit:
			return m, tea.Quit
		case ActionNextPane:
			m.focus = model.NextFocusPane(m.focus)
			return m, nil
		case ActionPrevPane:
			m.focus = model.PrevFocusPane(m.focus)
			return m, nil
		case ActionMoveUp:
			m.moveUp()
			m.syncAllContent()
			return m, nil
		case ActionMoveDown:
			m.moveDown()
			m.syncAllContent()
			return m, nil
		case ActionToggleSelect:
			if m.hasList {
				m.selection.Toggle(&m.items, m.listIdx)
				m.syncListContent()
			}
			return m, nil
		case ActionOpenActions:
			if m.hasList && len(m.items) > 0 {
				m.openActionMenu()
				m.syncListContent()
			}
			return m, nil
		case ActionOpenPreview:
			if m.hasList {
				m.selection.Toggle(&m.items, m.listIdx)
				return m, m.loadPreviewCmd()
			}
			return m, nil
		case ActionRefresh:
			if m.hasList {
				m.loading = true
				return m, m.loadListCmd(m.listSpec)
			}
			return m, nil
		}
		if m.focus == model.FocusMenu {
			switch key {
			case " ":
				m.nav.ToggleExpand(m.nav.Cursor())
				m.syncMenuContent()
				return m, nil
			case "enter":
				if spec, ok := m.nav.Activate(); ok {
					return m.withList(*spec)
				}
				m.syncMenuContent()
				return m, nil
			case "right", "l":
				if spec, ok := m.nav.CurrentListSpec(); ok {
					return m.withList(*spec)
				}
				m.nav.Activate()
				m.syncMenuContent()
				return m, nil
			}
		}
		if m.focus == model.FocusList && key == "enter" && m.hasList {
			m.openActionMenu()
			m.syncListContent()
			return m, nil
		}

	case listLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.items = nil
			m.hasList = false
		} else {
			m.err = ""
			m.items = msg.items
			m.listSpec = msg.spec
			m.hasList = true
			m.listIdx = 0
		}
		m.syncListContent()
		return m, nil

	case previewLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.previewMD = fmt.Sprintf("# Error\n\n%s\n", msg.err.Error())
		} else {
			m.previewMD = msg.text
		}
		m.syncPreviewContent()
		return m, nil

	case actionDoneMsg:
		m.loading = false
		m.actionMode = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.status = msg.message
			m.err = ""
			if m.hasList {
				m.loading = true
				return m, m.loadListCmd(m.listSpec)
			}
		}
		return m, nil

	case navProjectsMsg:
		if msg.err == nil {
			m.nav.InjectProjectNodes(msg.projects)
			m.syncMenuContent()
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case model.FocusMenu:
		m.menuVP, cmd = m.menuVP.Update(msg)
	case model.FocusList:
		m.listVP, cmd = m.listVP.Update(msg)
	case model.FocusPreview:
		m.previewVP, cmd = m.previewVP.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…\n"
	}
	menuW, listW, prevW := LayoutWidths(m.width)
	h := m.height - 2
	menuStyle := paneStyle(m.focus == model.FocusMenu).Width(menuW).Height(h)
	listStyle := paneStyle(m.focus == model.FocusList).Width(listW).Height(h)
	prevStyle := paneStyle(m.focus == model.FocusPreview).Width(prevW).Height(h)

	status := m.status
	if m.loading {
		status = "Loading…"
	}
	if m.err != "" {
		status = "Error: " + m.err
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(status)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		menuStyle.Render(m.menuVP.View()),
		listStyle.Render(m.listVP.View()),
		prevStyle.Render(m.previewVP.View()),
	) + "\n" + bar
}

func (m Model) withList(spec model.ListSpec) (Model, tea.Cmd) {
	m.listSpec = spec
	m.selection.Clear()
	m.listIdx = 0
	m.hasList = false
	m.actionMode = false
	m.focus = model.FocusList
	m.loading = true
	return m, m.loadListCmd(spec)
}

func (m *Model) openActionMenu() {
	if m.listIdx < 0 || m.listIdx >= len(m.items) {
		return
	}
	m.itemActions = model.ActionsFor(m.items[m.listIdx].Kind)
	m.actionIdx = 0
	m.actionMode = true
}

func (m *Model) handleActionKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.actionMode = false
		m.syncListContent()
		return m, nil
	case "up", "k":
		if m.actionIdx > 0 {
			m.actionIdx--
		}
		m.syncListContent()
		return m, nil
	case "down", "j":
		if m.actionIdx < len(m.itemActions)-1 {
			m.actionIdx++
		}
		m.syncListContent()
		return m, nil
	case "enter":
		if m.listIdx < 0 || m.listIdx >= len(m.items) || m.actionIdx >= len(m.itemActions) {
			return m, nil
		}
		act := m.itemActions[m.actionIdx]
		item := m.items[m.listIdx]
		m.actionMode = false
		if act.ID == model.ActionView {
			return m, m.loadPreviewCmd()
		}
		m.loading = true
		return m, m.executeActionCmd(act.ID, item)
	}
	return m, nil
}

func (m *Model) layoutViewports() {
	menuW, listW, prevW := LayoutWidths(m.width)
	h := m.height - 4
	if h < 4 {
		h = 4
	}
	inner := func(w int) int {
		if w > 4 {
			return w - 2
		}
		return w
	}
	m.menuVP = viewport.New(inner(menuW), h)
	m.listVP = viewport.New(inner(listW), h)
	m.previewVP = viewport.New(inner(prevW), h)
}

func (m *Model) syncAllContent() {
	m.syncMenuContent()
	m.syncListContent()
	m.syncPreviewContent()
}

func (m *Model) syncMenuContent() {
	var b strings.Builder
	b.WriteString("Navigation\n\n")
	for i := 0; i < m.nav.VisibleCount(); i++ {
		n, _ := m.nav.NodeAtVisible(i)
		depth := m.nav.DepthAtVisible(i)
		prefix := strings.Repeat("  ", depth)
		line := prefix + n.Label
		if m.nav.IsExpandable(i) {
			if m.nav.IsExpanded(i) {
				line = prefix + "▾ " + n.Label
			} else {
				line = prefix + "▸ " + n.Label
			}
		} else if n.List != nil {
			line = prefix + "• " + n.Label
		}
		if i == m.nav.Cursor() && m.focus == model.FocusMenu {
			line = "> " + line
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	m.menuVP.SetContent(b.String())
	syncVPToLine(&m.menuVP, m.nav.Cursor()+2)
}

func (m *Model) syncListContent() {
	var b strings.Builder
	if !m.hasList {
		b.WriteString("List\n\n")
		b.WriteString("Select a leaf node in the tree\n")
		b.WriteString("(marked with •) and press Enter.\n")
	} else {
		fmt.Fprintf(&b, "%s (%d)\n\n", m.listSpec.Subject, len(m.items))
		if m.actionMode {
			item := m.items[m.listIdx]
			fmt.Fprintf(&b, "Actions for: %s\n\n", item.Title)
			for i, act := range m.itemActions {
				cursor := "  "
				if i == m.actionIdx {
					cursor = "> "
				}
				label := act.Label
				if act.Danger {
					label = "⚠ " + label
				}
				fmt.Fprintf(&b, "%s%s\n", cursor, label)
			}
			b.WriteString("\nEnter: run · Esc: cancel\n")
		} else {
			for i, it := range m.items {
				mark := "[ ]"
				if it.Selected {
					mark = "[x]"
				}
				cursor := "  "
				if i == m.listIdx && m.focus == model.FocusList {
					cursor = "> "
				}
				line := fmt.Sprintf("%s%s %s", cursor, mark, it.Title)
				if it.Subtitle != "" {
					line += " — " + it.Subtitle
				}
				b.WriteString(line + "\n")
			}
			if len(m.items) == 0 && !m.loading {
				b.WriteString("(empty)\n")
			}
			b.WriteString("\nEnter/a: actions · Space: select\n")
		}
	}
	m.listVP.SetContent(b.String())
	line := m.listIdx + 3
	if m.actionMode {
		line = m.actionIdx + 4
	}
	syncVPToLine(&m.listVP, line)
}

func (m *Model) syncPreviewContent() {
	menuW, listW, prevW := LayoutWidths(m.width)
	_ = menuW
	_ = listW
	w := prevW - 4
	if w < 20 {
		w = 20
	}
	text, err := preview.RenderMarkdown(m.previewMD, w)
	if err != nil {
		text = m.previewMD
	}
	m.previewVP.SetContent(text)
}

func syncVPToLine(vp *viewport.Model, line int) {
	if line < 0 {
		line = 0
	}
	if line < vp.YOffset {
		vp.YOffset = line
	} else if line >= vp.YOffset+vp.Height {
		vp.YOffset = line - vp.Height + 1
	}
	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
}

func (m *Model) moveUp() {
	switch m.focus {
	case model.FocusMenu:
		m.nav.MoveUp()
	case model.FocusList:
		if m.hasList && !m.actionMode && m.listIdx > 0 {
			m.listIdx--
		}
	case model.FocusPreview:
		m.previewVP.LineUp(1)
	}
}

func (m *Model) moveDown() {
	switch m.focus {
	case model.FocusMenu:
		m.nav.MoveDown()
	case model.FocusList:
		if m.hasList && !m.actionMode && m.listIdx < len(m.items)-1 {
			m.listIdx++
		}
	case model.FocusPreview:
		m.previewVP.LineDown(1)
	}
}

func (m *Model) loadListCmd(spec model.ListSpec) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		items, err := m.loader.List(ctx, spec)
		return listLoadedMsg{items: items, spec: spec, err: err}
	}
}

func (m *Model) loadPreviewCmd() tea.Cmd {
	if m.listIdx < 0 || m.listIdx >= len(m.items) {
		return nil
	}
	item := m.items[m.listIdx]
	return func() tea.Msg {
		ctx := context.Background()
		raw, err := m.loader.Detail(ctx, item)
		if err != nil {
			return previewLoadedMsg{err: err}
		}
		md := preview.EntityMarkdown(string(item.Kind), raw)
		return previewLoadedMsg{text: md}
	}
}

func (m *Model) executeActionCmd(action model.ActionID, item model.Item) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		msg, err := m.loader.Execute(ctx, action, item, "")
		return actionDoneMsg{message: msg, err: err}
	}
}

func (m *Model) loadNavProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		projects, err := m.loader.LoadProjectsForNav(ctx)
		return navProjectsMsg{projects: projects, err: err}
	}
}

func paneStyle(focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().Padding(0, 1)
	if focused {
		return s.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	}
	return s.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
}

func helpText() string {
	return "Tab/Shift+Tab: pane · ↑↓/jk: scroll · Enter: open leaf/actions · Space: select · a: actions · r: refresh · q: quit"
}
