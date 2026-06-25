package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

type entityField int

const (
	entityFieldPrimary entityField = iota
	entityFieldSecondary
	entityFieldStatus
	entityFieldResponsible
)

// EntityForm is the top section of the detail pane for editable entities.
type EntityForm struct {
	active          bool
	kind            model.Kind
	itemID          string
	primaryLabel    string
	secondaryLabel  string
	readOnly        bool
	hasStatus       bool
	status          model.ProjectLifecycle
	hasTaskStatus   bool
	taskStatus      model.TaskLifecycle
	hasResponsible  bool
	responsibleID   string
	userChoices     []model.UserOption
	responsibleIdx  int
	projectTitle    string
	timingSummary   string
	hasUserEdit     bool
	userEnabled     bool
	userACL         model.UserACLState
	groupsText      string
	userFieldIdx    int
	primary         textinput.Model
	secondary       textarea.Model
	password        textinput.Model
	field           entityField
	focused         bool
	dirty           bool
	width           int
	height          int
	styles          entityFormStyles
}

type entityFormStyles struct {
	header   lipgloss.Style
	label    lipgloss.Style
	labelAct lipgloss.Style
	meta     lipgloss.Style
}

func newEntityFormStyles() entityFormStyles {
	return entityFormStyles{
		header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		label:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		labelAct: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")),
		meta:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

func newEntityForm() EntityForm {
	primary := textinput.New()
	primary.CharLimit = 512
	primary.Prompt = "> "
	secondary := textarea.New()
	secondary.ShowLineNumbers = false
	secondary.CharLimit = 8000
	secondary.Prompt = "> "
	password := textinput.New()
	password.CharLimit = 128
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.Prompt = "> "
	return EntityForm{
		primary:   primary,
		secondary: secondary,
		password:  password,
		field:     entityFieldPrimary,
		styles:    newEntityFormStyles(),
	}
}

func (f *EntityForm) FieldCount() int {
	if f.hasUserEdit {
		return f.userFieldCount()
	}
	n := 2
	if f.hasStatus || f.hasTaskStatus {
		n++
	}
	if f.hasResponsible {
		n++
	}
	return n
}

func (f *EntityForm) FormFields() model.FormFields {
	return model.FormFields{
		PrimaryLabel:   f.primaryLabel,
		SecondaryLabel: f.secondaryLabel,
		Primary:        f.primary.Value(),
		Secondary:      f.secondary.Value(),
		ReadOnly:       f.readOnly,
		HasStatus:      f.hasStatus,
		Status:         f.status,
		HasTaskStatus:  f.hasTaskStatus,
		TaskStatus:     f.taskStatus,
		HasResponsible: f.hasResponsible,
		ResponsibleID:  f.responsibleID,
		UserChoices:    f.userChoices,
		ProjectTitle:   f.projectTitle,
		TimingSummary:  f.timingSummary,
		HasUserEdit:    f.hasUserEdit,
		UserEnabled:    f.userEnabled,
		UserACL:        f.userACL,
		GroupsText:     f.groupsText,
		UserPassword:   f.password.Value(),
	}
}

func (f *EntityForm) Active() bool { return f.active }

func (f *EntityForm) ItemID() string { return f.itemID }

func (f *EntityForm) Primary() string { return f.primary.Value() }

func (f *EntityForm) Secondary() string { return f.secondary.Value() }

func (f *EntityForm) MarkClean() { f.dirty = false }

func (f *EntityForm) Load(kind model.Kind, itemID string, fields model.FormFields) {
	f.active = true
	f.kind = kind
	f.itemID = itemID
	f.primaryLabel = fields.PrimaryLabel
	f.secondaryLabel = fields.SecondaryLabel
	f.readOnly = fields.ReadOnly
	f.hasStatus = fields.HasStatus
	f.status = fields.Status
	f.hasTaskStatus = fields.HasTaskStatus
	f.taskStatus = fields.TaskStatus
	f.hasResponsible = fields.HasResponsible
	f.userChoices = fields.UserChoices
	f.projectTitle = fields.ProjectTitle
	f.timingSummary = fields.TimingSummary
	f.responsibleID = fields.ResponsibleID
	f.responsibleIdx = indexUserChoice(fields.UserChoices, fields.ResponsibleID)
	f.hasUserEdit = fields.HasUserEdit
	f.userEnabled = fields.UserEnabled
	f.userACL = copyUserACLState(fields.UserACL)
	f.groupsText = fields.GroupsText
	f.userFieldIdx = 0
	f.primary.SetValue(fields.Primary)
	f.secondary.SetValue(fields.Secondary)
	f.password.SetValue("")
	f.dirty = false
	if f.hasUserEdit {
		f.userFieldIdx = 0
	} else {
		f.field = entityFieldPrimary
	}
	f.applyFocus()
	f.layoutFields()
}

func indexUserChoice(choices []model.UserOption, id string) int {
	for i, c := range choices {
		if c.ID == id {
			return i
		}
	}
	return 0
}

func (f *EntityForm) Clear() {
	f.active = false
	f.itemID = ""
	f.hasStatus = false
	f.hasTaskStatus = false
	f.hasResponsible = false
	f.status = model.ProjectLifecycleOpen
	f.taskStatus = model.TaskLifecycleOpen
	f.responsibleID = ""
	f.userChoices = nil
	f.responsibleIdx = 0
	f.projectTitle = ""
	f.timingSummary = ""
	f.hasUserEdit = false
	f.userEnabled = false
	f.userACL = model.UserACLState{}
	f.groupsText = ""
	f.userFieldIdx = 0
	f.primary.SetValue("")
	f.secondary.SetValue("")
	f.password.SetValue("")
	f.dirty = false
	f.primary.Blur()
	f.secondary.Blur()
	f.password.Blur()
}

func (f *EntityForm) SetFocused(on bool) {
	f.focused = on && f.active
	if f.focused {
		f.applyFocus()
	} else {
		f.primary.Blur()
		f.secondary.Blur()
		f.password.Blur()
	}
}

func (f *EntityForm) SetSize(w, h int) {
	if w < 12 {
		w = 12
	}
	if h < 6 {
		h = 6
	}
	f.width = w
	f.height = h
	f.layoutFields()
}

func (f *EntityForm) layoutFields() {
	if f.width == 0 {
		return
	}
	inner := f.width - 2
	if inner < 8 {
		inner = 8
	}
	f.primary.Width = inner
	f.password.Width = inner
	if f.hasUserEdit {
		return
	}
	metaLines := 0
	if f.projectTitle != "" {
		metaLines++
	}
	if f.timingSummary != "" {
		metaLines++
	}
	if metaLines > 0 {
		metaLines++ // blank line before meta
	}
	extra := 6 + metaLines
	if f.hasStatus || f.hasTaskStatus {
		extra += 2
	}
	if f.hasResponsible {
		extra += 2
	}
	secH := f.height - extra
	if secH < 3 {
		secH = 3
	}
	f.secondary.SetWidth(inner)
	f.secondary.SetHeight(secH)
}

func (f *EntityForm) lastField() entityField {
	if f.hasResponsible {
		return entityFieldResponsible
	}
	if f.hasStatus || f.hasTaskStatus {
		return entityFieldStatus
	}
	return entityFieldSecondary
}

func (f *EntityForm) FocusNext() {
	if f.field < f.lastField() {
		f.field++
	}
	f.applyFocus()
}

func (f *EntityForm) FocusPrev() {
	if f.field > entityFieldPrimary {
		f.field--
	}
	f.applyFocus()
}

func (f *EntityForm) SetFieldIndex(i int) {
	if f.hasUserEdit {
		f.setUserFieldIndex(i)
		return
	}
	switch i {
	case 0:
		f.field = entityFieldPrimary
	case 1:
		f.field = entityFieldSecondary
	case 2:
		if f.hasStatus || f.hasTaskStatus {
			f.field = entityFieldStatus
		} else if f.hasResponsible {
			f.field = entityFieldResponsible
		}
	case 3:
		if f.hasResponsible && (f.hasStatus || f.hasTaskStatus) {
			f.field = entityFieldResponsible
		}
	default:
		return
	}
	f.applyFocus()
}

func (f *EntityForm) cycleStatus(delta int) {
	if f.readOnly {
		return
	}
	if f.hasTaskStatus {
		if delta > 0 {
			f.taskStatus = f.taskStatus.Next()
		} else {
			f.taskStatus = f.taskStatus.Prev()
		}
		f.dirty = true
		return
	}
	if !f.hasStatus {
		return
	}
	if delta > 0 {
		f.status = f.status.Next()
	} else {
		f.status = f.status.Prev()
	}
	f.dirty = true
}

func (f *EntityForm) cycleResponsible(delta int) {
	if !f.hasResponsible || f.readOnly || len(f.userChoices) == 0 {
		return
	}
	n := len(f.userChoices)
	f.responsibleIdx = (f.responsibleIdx + delta%n + n) % n
	f.responsibleID = f.userChoices[f.responsibleIdx].ID
	f.dirty = true
}

func (f *EntityForm) responsibleLabel() string {
	if f.responsibleID == "" {
		return "—"
	}
	for _, c := range f.userChoices {
		if c.ID == f.responsibleID {
			if c.Name != "" {
				return c.Name
			}
			return c.ID
		}
	}
	return f.responsibleID
}

func (f *EntityForm) applyFocus() {
	if f.hasUserEdit {
		f.applyUserFocus()
		return
	}
	if f.readOnly {
		f.primary.Blur()
		f.secondary.Blur()
		return
	}
	switch f.field {
	case entityFieldPrimary:
		f.primary.Focus()
		f.secondary.Blur()
	case entityFieldSecondary:
		f.primary.Blur()
		f.secondary.Focus()
	default:
		f.primary.Blur()
		f.secondary.Blur()
	}
}

func (f *EntityForm) Update(msg tea.Msg) tea.Cmd {
	if !f.active || !f.focused || f.readOnly {
		return nil
	}
	if f.hasUserEdit {
		return f.updateUserForm(msg)
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch f.field {
		case entityFieldStatus:
			switch key.String() {
			case "left", "h":
				f.cycleStatus(-1)
				return nil
			case "right", "l":
				f.cycleStatus(1)
				return nil
			case " ":
				f.cycleStatus(1)
				return nil
			}
		case entityFieldResponsible:
			switch key.String() {
			case "left", "h":
				f.cycleResponsible(-1)
				return nil
			case "right", "l":
				f.cycleResponsible(1)
				return nil
			case " ":
				f.cycleResponsible(1)
				return nil
			}
		}
	}
	var cmd tea.Cmd
	switch f.field {
	case entityFieldPrimary:
		f.primary, cmd = f.primary.Update(msg)
	case entityFieldSecondary:
		f.secondary, cmd = f.secondary.Update(msg)
	default:
		return nil
	}
	if _, ok := msg.(tea.KeyMsg); ok {
		f.dirty = true
	}
	return cmd
}

func (f EntityForm) View() string {
	if !f.active {
		return ""
	}
	if f.hasUserEdit {
		return f.userView()
	}
	header := f.styles.header.Render(model.KindHeading(f.kind, f.itemID))
	pLabel := f.styles.label.Render(f.primaryLabel)
	if f.focused && f.field == entityFieldPrimary && !f.readOnly {
		pLabel = f.styles.labelAct.Render(f.primaryLabel)
	}
	sLabel := f.styles.label.Render(f.secondaryLabel)
	if f.focused && f.field == entityFieldSecondary && !f.readOnly {
		sLabel = f.styles.labelAct.Render(f.secondaryLabel)
	}
	statusBlock := f.renderStatusBlock()
	responsibleBlock := f.renderResponsibleBlock()
	metaBlock := f.renderMetaBlock()
	ro := ""
	if f.readOnly {
		ro = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" (read-only)") + "\n"
	}
	parts := []string{
		header + ro,
		pLabel,
		f.primary.View(),
		"",
		sLabel,
		f.secondary.View(),
	}
	if statusBlock != "" {
		parts = append(parts, statusBlock)
	}
	if responsibleBlock != "" {
		parts = append(parts, responsibleBlock)
	}
	if metaBlock != "" {
		parts = append(parts, metaBlock)
	}
	return strings.Join(parts, "\n")
}

func (f EntityForm) renderStatusBlock() string {
	if !f.hasStatus && !f.hasTaskStatus {
		return ""
	}
	stLabel := f.styles.label.Render("Status")
	if f.focused && f.field == entityFieldStatus && !f.readOnly {
		stLabel = f.styles.labelAct.Render("Status")
	}
	label := f.status.Label()
	valStyle := f.styles.label
	if f.hasTaskStatus {
		label = f.taskStatus.Label()
		if f.taskStatus == model.TaskLifecycleClosed {
			valStyle = valStyle.Foreground(lipgloss.Color("245"))
		} else if f.taskStatus == model.TaskLifecycleOpen {
			valStyle = valStyle.Foreground(lipgloss.Color("42"))
		}
	} else if f.status == model.ProjectLifecycleClosed {
		valStyle = valStyle.Foreground(lipgloss.Color("245"))
	} else {
		valStyle = valStyle.Foreground(lipgloss.Color("42"))
	}
	hint := ""
	if f.focused && f.field == entityFieldStatus && !f.readOnly {
		hint = f.styles.label.Render("  ←/→ cycle")
	}
	return strings.Join([]string{"", stLabel, valStyle.Render(label) + hint}, "\n")
}

func (f EntityForm) renderResponsibleBlock() string {
	if !f.hasResponsible {
		return ""
	}
	rLabel := f.styles.label.Render("Responsible")
	if f.focused && f.field == entityFieldResponsible && !f.readOnly {
		rLabel = f.styles.labelAct.Render("Responsible")
	}
	hint := ""
	if f.focused && f.field == entityFieldResponsible && !f.readOnly && len(f.userChoices) > 0 {
		hint = f.styles.label.Render("  ←/→ choose")
	}
	return strings.Join([]string{
		"",
		rLabel,
		f.styles.meta.Render(f.responsibleLabel()) + hint,
	}, "\n")
}

func (f EntityForm) renderMetaBlock() string {
	if f.projectTitle == "" && f.timingSummary == "" {
		return ""
	}
	lines := []string{""}
	if f.projectTitle != "" {
		lines = append(lines, f.styles.label.Render("Project"), f.styles.meta.Render(f.projectTitle))
	}
	if f.timingSummary != "" {
		lines = append(lines, f.styles.label.Render("Timing"), f.styles.meta.Render(f.timingSummary))
	}
	return strings.Join(lines, "\n")
}

func entityFormBlinkCmd() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

// NewEntityFormForTest exposes a form for unit tests.
func NewEntityFormForTest() EntityForm {
	return newEntityForm()
}
