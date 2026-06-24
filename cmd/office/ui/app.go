package ui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/fetch"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

type listLoadedMsg struct {
	items []model.Item
	spec  model.ListSpec
	err   error
}

type listMoreLoadedMsg struct {
	items []model.Item
	err   error
}

type detailLoadedMsg struct {
	item     model.Item
	document bool
	fields   model.FormFields
	markdown string
	err      error
}

type detailSavedMsg struct {
	item   model.Item
	fields model.FormFields
	err    error
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
	listHasMore    bool
	listLoadingMore bool
	focus     model.FocusPane
	width     int
	height    int
	status    string
	err       string
	loading   bool
	menuVP    viewport.Model
	listTable DataTable
	detail    DetailPane
	showMenu  bool
	showList  bool
	showDetail bool
	filterActive bool
	filterSearch FilterSearch
	customPaneLayout bool
	paneSizes        PaneWidths
	resize           paneResizeState
}

// NewModel constructs the TUI with an authenticated client.
func NewModel(client *onlyoffice.Client) Model {
	h, w := 24, 80
	m := Model{
		client:    client,
		loader:    &fetch.Loader{Client: client},
		nav:       model.DefaultNavTree(),
		selection: model.NewSelection(),
		focus:     model.FocusMenu,
		status:    "Tab: pane · f: filter · row select loads detail · Ctrl+S save · q quit",
		height:    h,
		width:     w,
	}
	m.menuVP = viewport.New(m.paneInnerWidth(22), m.paneHeight())
	m.listTable = newDataTable()
	m.detail = newDetailPane()
	m.filterSearch = newFilterSearch()
	m.showMenu, m.showList, m.showDetail = true, true, true
	m.menuVP.MouseWheelEnabled = true
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadNavProjectsCmd(), m.detail.BlinkCmd(), filterSearchBlinkCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.customPaneLayout {
			m.paneSizes = FitPaneWidths(m.width, m.paneVis(), m.paneSizes)
		}
		m.layoutViewports()
		m.syncAllContent()
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if key == "esc" && m.filterActive {
			m.exitFilter()
			return m, nil
		}
		if m.filterActive && m.focus == model.FocusPreview && m.showDetail {
			switch key {
			case "tab", "shift+tab", "backtab":
				// allow pane switching while filter stays active
			default:
				cmd := m.filterSearch.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}
		if key == "?" {
			m.status = helpText()
			return m, nil
		}
		if m.scrollFocusedPane(key) {
			return m, nil
		}
		if m.filterActive && m.focus == model.FocusPreview {
			// filter input owns keys except pane switching handled above
		} else if m.focus == model.FocusPreview {
			if cmd, handled := m.handleDetailKey(key, msg); handled {
				return m, cmd
			}
		}
		action := KeyAction(key, m.focus)
		if action == ActionNone && (key == "up" || key == "k") {
			action = ResolveMoveUp(m.focus)
		}
		switch action {
		case ActionQuit:
			return m, tea.Quit
		case ActionNextPane:
			prev := m.focus
			m.focus = NextVisibleFocus(m.focus, m.paneVis())
			if m.focus == model.FocusPreview && prev != model.FocusPreview && !m.filterActive {
				m.detail.FocusFirstStop()
			}
			m.syncPaneFocus()
			return m, nil
		case ActionPrevPane:
			prev := m.focus
			m.focus = PrevVisibleFocus(m.focus, m.paneVis())
			if m.focus == model.FocusPreview && prev != model.FocusPreview && !m.filterActive {
				m.detail.FocusFirstStop()
			}
			m.syncPaneFocus()
			return m, nil
		case ActionMoveUp:
			if m.focus == model.FocusPreview {
				return m.handleDetailMove(-1)
			}
			m.moveUp()
			m.syncFocusedPane()
			return m, m.onListRowChanged()
		case ActionMoveDown:
			if m.focus == model.FocusPreview {
				return m.handleDetailMove(1)
			}
			m.moveDown()
			m.syncFocusedPane()
			return m, m.onListRowChanged()
		case ActionMoveLeft:
			if m.focus == model.FocusPreview && m.detail.Zone() == detailZoneActions {
				m.detail.MoveTabStop(-1)
				return m, nil
			}
			if m.focus == model.FocusList && m.hasList {
				m.listTable.MoveCol(-1)
			}
			return m, nil
		case ActionMoveRight:
			if m.focus == model.FocusPreview && m.detail.Zone() == detailZoneActions {
				m.detail.MoveTabStop(1)
				return m, nil
			}
			if m.focus == model.FocusList && m.hasList {
				m.listTable.MoveCol(1)
			}
			return m, nil
		case ActionSort:
			if m.focus == model.FocusList && m.hasList {
				m.listTable.ToggleSort()
			}
			return m, nil
		case ActionToggleSelect:
			if m.hasList {
				if it, ok := m.listTableItem(); ok {
					for i := range m.items {
						if m.items[i].ID == it.ID {
							m.selection.Toggle(&m.items, i)
							break
						}
					}
					m.syncListTable()
				}
			}
			return m, nil
		case ActionFocusDetail:
			if m.hasList && m.showDetail {
				m.focus = model.FocusPreview
				m.syncPaneFocus()
				m.detail.FocusFirstStop()
				return m, m.onListRowChanged()
			}
			return m, nil
		case ActionToggleMenuPane:
			m.togglePane(1)
			return m, nil
		case ActionToggleListPane:
			m.togglePane(2)
			return m, nil
		case ActionToggleDetailPane:
			m.togglePane(3)
			return m, nil
		case ActionFilter:
			m.enterFilter()
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

	case listLoadedMsg:
		m.loading = false
		m.listLoadingMore = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.items = nil
			m.hasList = false
			m.listHasMore = false
			m.listTable.Clear()
			m.detail.Clear()
		} else {
			m.err = ""
			m.items = msg.items
			m.listSpec = msg.spec
			m.hasList = true
			m.listHasMore = model.SubjectIsMail(msg.spec.Subject) && len(msg.items) >= fetch.MailListPageSize
			m.listTable.SetData(m.listSpec, m.items)
			if m.filterActive {
				m.applyFilter()
			}
		}
		return m, tea.Batch(m.onListRowChanged(), m.maybeLoadMoreList())

	case listMoreLoadedMsg:
		m.listLoadingMore = false
		m.listTable.SetLoadingMore(false)
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if len(msg.items) == 0 {
			m.listHasMore = false
			return m, nil
		}
		added := m.listTable.AppendItems(msg.items)
		if added > 0 {
			m.items = m.listTable.Items()
		}
		m.listHasMore = len(msg.items) >= fetch.MailListPageSize
		m.err = ""
		return m, nil

	case detailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.detail.Clear()
		} else {
			pw := m.paneLayout()
			w := pw.Detail - 4
			if w < 20 {
				w = 20
			}
			if msg.document {
				m.detail.LoadDocument(msg.item, msg.markdown, w)
			} else {
				m.detail.LoadForm(msg.item, msg.fields)
			}
			m.detail.SetFocused(m.focus == model.FocusPreview)
			m.err = ""
		}
		return m, nil

	case detailSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.updateItemAfterSave(msg.item, msg.fields)
			m.detail.LoadForm(msg.item, msg.fields)
			m.detail.SetFocused(m.focus == model.FocusPreview)
			m.status = "Saved"
			m.err = ""
		}
		return m, nil

	case actionDoneMsg:
		m.loading = false
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

	case tea.MouseMsg:
		if m.handlePaneResizeMouse(msg) {
			return m, nil
		}
		if cmd, handled := m.routeDetailDocumentMouse(msg); handled {
			return m, cmd
		}
		var cmd tea.Cmd
		switch m.focus {
		case model.FocusMenu:
			m.menuVP, cmd = m.menuVP.Update(msg)
		case model.FocusList:
			cmd = m.listTable.Update(msg)
		case model.FocusPreview:
			if !m.filterActive {
				cmd = m.detail.Update(msg)
			}
		}
		return m, cmd
	}

	var cmd tea.Cmd
	switch m.focus {
	case model.FocusMenu:
		m.menuVP, cmd = m.menuVP.Update(msg)
	case model.FocusList:
		cmd = m.listTable.Update(msg)
	case model.FocusPreview:
		if !m.filterActive {
			cmd = m.detail.Update(msg)
		}
	}
	return m, cmd
}

func (m *Model) handleDetailKey(key string, msg tea.KeyMsg) (tea.Cmd, bool) {
	switch key {
	case "tab", "shift+tab", "backtab":
		reverse := key != "tab"
		if reverse {
			if m.detail.TabBackward() {
				m.focus = PrevVisibleFocus(m.focus, m.paneVis())
			}
		} else if m.detail.TabForward() {
			m.focus = NextVisibleFocus(m.focus, m.paneVis())
		}
		m.syncPaneFocus()
		return nil, true
	case "ctrl+s":
		m.loading = true
		return m.saveDetailCmd(), true
	case "enter":
		if m.detail.Zone() == detailZoneActions {
			act, ok := m.detail.SelectedAction()
			if !ok {
				return nil, true
			}
			m.loading = true
			if act.ID == model.ActionSave {
				return m.saveDetailCmd(), true
			}
			return m.executeActionCmd(act.ID, m.detail.Item()), true
		}
	}
	if m.detail.Zone() == detailZoneContent {
		return m.detail.Update(msg), true
	}
	return nil, false
}

func (m *Model) routeDetailDocumentMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m.filterActive || !m.showDetail {
		return nil, false
	}
	if !m.detail.IsDocumentContent() && !m.detail.isReadOnlyFormContent() {
		return nil, false
	}
	if msg.Y > m.paneHeight()+1 {
		return nil, false
	}
	x0, x1 := DetailPaneXRange(m.paneLayout())
	if x0 >= x1 || msg.X < x0 || msg.X >= x1 {
		return nil, false
	}
	cmd := m.detail.ScrollDocumentMouse(msg)
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return cmd, true
	}
	return cmd, false
}

func (m Model) handleDetailMove(delta int) (Model, tea.Cmd) {
	if m.detail.Zone() == detailZoneContent && (m.detail.IsDocumentContent() || m.detail.isReadOnlyFormContent()) {
		key := "down"
		if delta < 0 {
			key = "up"
		}
		m.detail.ScrollDocument(key)
		return m, nil
	}
	m.detail.MoveTabStop(delta)
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…\n"
	}
	pw := m.paneLayout()
	h := m.paneHeight() + 2
	var parts []string
	if pw.Visibility.Menu {
		menuStyle := paneStyle(m.focus == model.FocusMenu).Width(pw.Menu).Height(h)
		menuBody := ApplyVerticalScrollbar(
			m.menuVP.View(),
			m.menuVP.Width,
			m.menuVP.Height,
			m.menuVP.TotalLineCount(),
			m.menuVP.YOffset,
		)
		parts = append(parts, menuStyle.Render(menuBody))
	}
	if pw.Visibility.List {
		listStyle := paneStyle(m.focus == model.FocusList).Width(pw.List).Height(h)
		parts = append(parts, listStyle.Render(m.listTable.View()))
	}
	if pw.Visibility.Detail {
		prevStyle := paneStyle(m.focus == model.FocusPreview).Width(pw.Detail).Height(h)
		var detailBody string
		if m.filterActive {
			detailBody = m.filterSearch.View()
		} else {
			detailBody = m.detail.View()
		}
		parts = append(parts, prevStyle.Render(detailBody))
	}

	status := m.status
	if m.loading {
		status = "Loading…"
	}
	if m.err != "" {
		status = "Error: " + m.err
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(status)

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...) + "\n" + bar
}

func (m Model) paneVis() PaneVisibility {
	return PaneVisibility{Menu: m.showMenu, List: m.showList, Detail: m.showDetail}
}

func (m Model) paneLayout() PaneWidths {
	vis := m.paneVis()
	if m.customPaneLayout {
		return FitPaneWidths(m.width, vis, m.paneSizes)
	}
	return LayoutWidths(m.width, vis)
}

func (m *Model) syncPaneFocus() {
	m.listTable.SetFocused(m.focus == model.FocusList && m.showList)
	if m.filterActive {
		m.filterSearch.SetFocused(m.focus == model.FocusPreview && m.showDetail)
		m.detail.SetFocused(false)
		return
	}
	m.filterSearch.SetFocused(false)
	m.detail.SetFocused(m.focus == model.FocusPreview && m.showDetail)
}

func (m *Model) togglePane(which int) {
	switch which {
	case 1:
		m.showMenu = !m.showMenu
	case 2:
		m.showList = !m.showList
	case 3:
		m.showDetail = !m.showDetail
	}
	if !m.showMenu && !m.showList && !m.showDetail {
		switch which {
		case 1:
			m.showMenu = true
		case 2:
			m.showList = true
		case 3:
			m.showDetail = true
		}
	}
	if !paneVisible(m.focus, m.paneVis()) {
		m.focus = firstVisibleFocus(m.paneVis())
	}
	if m.customPaneLayout {
		m.paneSizes = FitPaneWidths(m.width, m.paneVis(), m.paneSizes)
	}
	m.layoutViewports()
	m.syncAllContent()
	m.syncPaneFocus()
}

func (m Model) withList(spec model.ListSpec) (Model, tea.Cmd) {
	m.listSpec = spec
	m.selection.Clear()
	m.hasList = false
	m.listHasMore = false
	m.listLoadingMore = false
	m.focus = model.FocusList
	m.listTable.SetFocused(true)
	m.detail.Clear()
	m.loading = true
	return m, m.loadListCmd(spec)
}

func (m *Model) layoutViewports() {
	pw := m.paneLayout()
	h := m.paneHeight()
	if pw.Visibility.Menu {
		m.menuVP.Width = m.paneInnerWidth(pw.Menu)
		m.menuVP.Height = h
	}
	if pw.Visibility.List {
		m.listTable.SetSize(m.paneInnerWidth(pw.List), h)
	}
	if pw.Visibility.Detail {
		m.detail.SetSize(m.paneInnerWidth(pw.Detail), h)
		m.filterSearch.SetSize(m.paneInnerWidth(pw.Detail), h)
	}
}

func (m *Model) syncAllContent() {
	m.syncMenuContent()
	m.syncListTable()
}

func (m *Model) syncListTable() {
	if !m.hasList {
		m.listTable.Clear()
		return
	}
	items := m.items
	if m.filterActive {
		items = model.FilterItems(m.items, m.filterSearch.Query())
	}
	m.listTable.UpdateItems(items)
}

func (m *Model) enterFilter() {
	if !m.showDetail {
		m.showDetail = true
		m.layoutViewports()
	}
	m.filterActive = true
	m.focus = model.FocusPreview
	m.syncPaneFocus()
	m.applyFilter()
}

func (m *Model) exitFilter() {
	m.filterActive = false
	m.filterSearch.Clear()
	m.nav.ClearFilter()
	m.syncMenuContent()
	m.syncListTable()
	m.syncPaneFocus()
}

func (m *Model) applyFilter() {
	q := m.filterSearch.Query()
	m.nav.SetFilter(q)
	m.syncMenuContent()
	m.syncListTable()
}

func (m Model) listTableItem() (model.Item, bool) {
	idx := m.listTable.ItemIndex()
	if idx < 0 {
		return model.Item{}, false
	}
	items := m.items
	if m.filterActive {
		items = model.FilterItems(m.items, m.filterSearch.Query())
	}
	if idx >= len(items) {
		return model.Item{}, false
	}
	return items[idx], true
}

func (m *Model) syncMenuContent() {
	var b strings.Builder
	title := "Navigation"
	if m.filterActive && m.filterSearch.Query() != "" {
		title += " (filtered)"
	}
	b.WriteString(title + "\n\n")
	menuW := m.menuVP.Width
	if menuW < 10 {
		menuW = 20
	}
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("255"))
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
			line = selectedStyle.Width(menuW).Render(truncateRunes(line, menuW))
		}
		b.WriteString(line + "\n")
	}
	m.menuVP.SetContent(b.String())
	syncVPToLine(&m.menuVP, m.nav.Cursor()+2)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func (m *Model) syncFocusedPane() {
	switch m.focus {
	case model.FocusMenu:
		m.syncMenuContent()
	case model.FocusList:
		m.syncListTable()
	}
}

func (m *Model) moveUp() {
	switch m.focus {
	case model.FocusMenu:
		m.nav.MoveUp()
	case model.FocusList:
		if m.hasList {
			m.listTable.MoveRow(-1)
		}
	}
}

func (m *Model) moveDown() {
	switch m.focus {
	case model.FocusMenu:
		m.nav.MoveDown()
	case model.FocusList:
		if m.hasList {
			m.listTable.MoveRow(1)
		}
	}
}

func syncVPToLine(vp *viewport.Model, line int) {
	if line < 0 {
		line = 0
	}
	max := vp.TotalLineCount()
	if max == 0 {
		vp.YOffset = 0
		return
	}
	if line >= max {
		line = max - 1
	}
	if line < vp.YOffset {
		vp.YOffset = line
	} else if line >= vp.YOffset+vp.Height {
		vp.YOffset = line - vp.Height + 1
	}
	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
	maxOff := max - vp.Height
	if maxOff < 0 {
		maxOff = 0
	}
	if vp.YOffset > maxOff {
		vp.YOffset = maxOff
	}
}

func (m *Model) loadListCmd(spec model.ListSpec) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		items, err := m.loader.List(ctx, spec)
		return listLoadedMsg{items: items, spec: spec, err: err}
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

func (m *Model) onListRowChanged() tea.Cmd {
	if m.filterActive {
		return nil
	}
	if !m.hasList {
		m.detail.Clear()
		return m.maybeLoadMoreList()
	}
	it, ok := m.listTableItem()
	if !ok {
		m.detail.Clear()
		return m.maybeLoadMoreList()
	}
	var cmds []tea.Cmd
	cmds = append(cmds, m.maybeLoadMoreList())
	if m.detail.LoadedID() != it.ID {
		m.loading = true
		cmds = append(cmds, m.loadDetailCmd(it))
	}
	return tea.Batch(cmds...)
}

const mailListLoadThreshold = 3

func (m *Model) maybeLoadMoreList() tea.Cmd {
	if !m.hasList || m.listLoadingMore || !m.listHasMore || m.filterActive {
		return nil
	}
	if !model.SubjectIsMail(m.listSpec.Subject) {
		return nil
	}
	if !m.listTable.NearEnd(mailListLoadThreshold) {
		return nil
	}
	m.listLoadingMore = true
	m.listTable.SetLoadingMore(true)
	return m.loadListMoreCmd()
}

func (m *Model) loadListMoreCmd() tea.Cmd {
	start := len(m.items)
	spec := m.listSpec
	return func() tea.Msg {
		ctx := context.Background()
		items, err := m.loader.ListMailMore(ctx, spec, start)
		return listMoreLoadedMsg{items: items, err: err}
	}
}

func (m *Model) loadDetailCmd(item model.Item) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if model.IsDocumentKind(item.Kind) {
			md, err := m.loader.PreviewMarkdown(ctx, item)
			return detailLoadedMsg{item: item, document: true, markdown: md, err: err}
		}
		fields, err := m.loader.DetailForm(ctx, item)
		return detailLoadedMsg{item: item, fields: fields, err: err}
	}
}

func (m *Model) saveDetailCmd() tea.Cmd {
	item := m.detail.Item()
	fields := m.detail.form.FormFields()
	return func() tea.Msg {
		ctx := context.Background()
		err := m.loader.SaveItem(ctx, item, fields)
		return detailSavedMsg{item: item, fields: fields, err: err}
	}
}

func (m *Model) updateItemAfterSave(item model.Item, fields model.FormFields) {
	for i := range m.items {
		if m.items[i].ID != item.ID {
			continue
		}
		m.items[i].Title = fields.Primary
		if m.items[i].Raw == nil {
			m.items[i].Raw = map[string]any{}
		}
		m.items[i].Raw["title"] = fields.Primary
		m.items[i].Raw["description"] = fields.Secondary
		if fields.HasStatus {
			m.items[i].Raw["status"] = statusInt(fields.Status)
		}
		m.syncListTable()
		return
	}
}

func statusInt(s model.ProjectLifecycle) int {
	switch s {
	case model.ProjectLifecycleClosed:
		return 2
	case model.ProjectLifecyclePaused:
		return 1
	default:
		return 0
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
	return "Alt+1/2/3: toggle panes · drag borders to resize · f: filter · Tab: fields/actions/pane · j/k: scroll mail · v: detail · Ctrl+S: save · q: quit"
}
