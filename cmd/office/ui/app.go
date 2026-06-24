package ui

import (
	"context"
	"fmt"
	"os/exec"
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
	items   []model.Item
	subject model.Subject
	err     error
}

type previewLoadedMsg struct {
	text string
	err  error
}

// Model is the root Bubble Tea model for the office TUI.
type Model struct {
	client   *onlyoffice.Client
	loader   *fetch.Loader
	menu     *model.MenuTree
	items    []model.Item
	selection *model.Selection
	subject  model.Subject
	listIdx  int
	focus    model.FocusPane
	width    int
	height   int
	status   string
	err      string
	loading  bool
	preview  viewport.Model
	previewMD string
	ready    bool
}

// NewModel constructs the TUI with an authenticated client.
func NewModel(client *onlyoffice.Client) Model {
	vp := viewport.New(40, 20)
	return Model{
		client:    client,
		loader:    &fetch.Loader{Client: client},
		menu:      model.DefaultMenuTree(),
		selection: model.NewSelection(),
		focus:     model.FocusMenu,
		preview:   vp,
		status:    "office — Tab: pane  Space: select  Enter: preview  q: quit",
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutViewports()
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if key == "?" {
			m.status = helpText()
			return m, nil
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
		case ActionMoveUp:
			m.moveUp()
			return m, nil
		case ActionMoveDown:
			m.moveDown()
			return m, nil
		case ActionToggleSelect:
			m.selection.Toggle(&m.items, m.listIdx)
			return m, nil
		case ActionOpenPreview:
			m.selection.Toggle(&m.items, m.listIdx)
			return m, m.loadPreviewCmd()
		case ActionRefresh:
			m.loading = true
			return m, m.loadListCmd(m.subject)
		case ActionOpenVex:
			return m, m.openVexCmd()
		}
		if m.focus == model.FocusMenu && (key == "enter" || key == " ") {
			if subj, ok := m.menu.CurrentSubject(); ok {
				m.subject = subj
				m.selection.Clear()
				m.listIdx = 0
				m.focus = model.FocusList
				m.loading = true
				return m, m.loadListCmd(subj)
			}
			if m.menu.IsExpandable(m.menu.Cursor()) {
				m.menu.ToggleExpand(m.menu.Cursor())
			}
			return m, nil
		}
	case listLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.items = nil
			return m, nil
		}
		m.err = ""
		m.items = msg.items
		m.subject = msg.subject
		return m, nil
	case previewLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.previewMD = fmt.Sprintf("# Error\n\n%s\n", msg.err.Error())
		} else {
			m.previewMD = msg.text
		}
		m.renderPreview()
		return m, nil
	}

	var cmd tea.Cmd
	if m.focus == model.FocusPreview {
		m.preview, cmd = m.preview.Update(msg)
	}
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading…\n"
	}
	menuW, listW, prevW := LayoutWidths(m.width)
	menuStyle := paneStyle(m.focus == model.FocusMenu).Width(menuW).Height(m.height - 2)
	listStyle := paneStyle(m.focus == model.FocusList).Width(listW).Height(m.height - 2)
	prevStyle := paneStyle(m.focus == model.FocusPreview).Width(prevW).Height(m.height - 2)

	status := m.status
	if m.loading {
		status = "Loading…"
	}
	if m.err != "" {
		status = "Error: " + m.err
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(status)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		menuStyle.Render(m.renderMenu(menuW)),
		listStyle.Render(m.renderList(listW)),
		prevStyle.Render(m.preview.View()),
	) + "\n" + bar
}

func (m *Model) layoutViewports() {
	_, _, prevW := LayoutWidths(m.width)
	h := m.height - 4
	if h < 4 {
		h = 4
	}
	m.preview.Width = prevW - 2
	m.preview.Height = h
	m.renderPreview()
}

func (m *Model) renderPreview() {
	_, _, prevW := LayoutWidths(m.width)
	text, err := preview.RenderMarkdown(m.previewMD, prevW-4)
	if err != nil {
		text = m.previewMD
	}
	m.preview.SetContent(text)
}

func (m *Model) renderMenu(w int) string {
	var b strings.Builder
	b.WriteString("Modules\n\n")
	for i := 0; i < m.menu.VisibleCount(); i++ {
		label, depth := m.menu.LabelAt(i)
		prefix := strings.Repeat("  ", depth)
		line := prefix + label
		if m.menu.IsExpandable(i) {
			if m.menu.IsExpanded(i) {
				line = prefix + "▾ " + label
			} else {
				line = prefix + "▸ " + label
			}
		}
		if i == m.menu.Cursor() && m.focus == model.FocusMenu {
			line = "> " + line
		} else {
			line = "  " + line
		}
		if len(line) > w-1 {
			line = line[:w-2] + "…"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m *Model) renderList(w int) string {
	title := string(m.subject)
	if title == "" {
		title = "Items"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d)\n\n", title, len(m.items))
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
		if len(line) > w-1 {
			line = line[:w-2] + "…"
		}
		b.WriteString(line + "\n")
	}
	if len(m.items) == 0 && !m.loading {
		b.WriteString("(empty)\n")
	}
	return b.String()
}

func (m *Model) moveUp() {
	switch m.focus {
	case model.FocusMenu:
		m.menu.MoveUp()
	case model.FocusList:
		if m.listIdx > 0 {
			m.listIdx--
		}
	case model.FocusPreview:
		m.preview.LineUp(1)
	}
}

func (m *Model) moveDown() {
	switch m.focus {
	case model.FocusMenu:
		m.menu.MoveDown()
	case model.FocusList:
		if m.listIdx < len(m.items)-1 {
			m.listIdx++
		}
	case model.FocusPreview:
		m.preview.LineDown(1)
	}
}

func (m *Model) loadListCmd(subject model.Subject) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		items, err := m.loader.List(ctx, subject)
		return listLoadedMsg{items: items, subject: subject, err: err}
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

func (m *Model) openVexCmd() tea.Cmd {
	if m.listIdx < 0 || m.listIdx >= len(m.items) {
		return nil
	}
	item := m.items[m.listIdx]
	if item.Kind != model.KindFile {
		return func() tea.Msg {
			return previewLoadedMsg{text: "_Press v on a spreadsheet file row._\n"}
		}
	}
	return nil
}

func paneStyle(focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().Padding(0, 1)
	if focused {
		return s.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	}
	return s.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
}

func helpText() string {
	return "Tab: pane | j/k: move | Space: select | Enter: preview | r: refresh | v: vex | q: quit"
}

// OpenVex launches the vex-tui binary for a file path when available.
func OpenVex(path string) error {
	if _, err := exec.LookPath("vex"); err != nil {
		return fmt.Errorf("vex not found on PATH")
	}
	cmd := exec.Command("vex", path)
	return cmd.Run()
}
