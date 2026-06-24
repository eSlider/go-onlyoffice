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
)

// EntityForm is the top section of the detail pane for editable entities.
type EntityForm struct {
	active         bool
	kind           model.Kind
	itemID         string
	primaryLabel   string
	secondaryLabel string
	readOnly       bool
	hasStatus      bool
	status         model.ProjectLifecycle
	responsibleID  string
	primary        textinput.Model
	secondary      textarea.Model
	field          entityField
	focused        bool
	dirty          bool
	width          int
	height         int
	styles         entityFormStyles
}

type entityFormStyles struct {
	header   lipgloss.Style
	label    lipgloss.Style
	labelAct lipgloss.Style
}

func newEntityFormStyles() entityFormStyles {
	return entityFormStyles{
		header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		label:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		labelAct: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")),
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
	return EntityForm{
		primary:  primary,
		secondary: secondary,
		field:    entityFieldPrimary,
		styles:   newEntityFormStyles(),
	}
}

func (f *EntityForm) FieldCount() int {
	n := 2
	if f.hasStatus {
		n = 3
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
		ResponsibleID:  f.responsibleID,
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
	f.responsibleID = fields.ResponsibleID
	f.primary.SetValue(fields.Primary)
	f.secondary.SetValue(fields.Secondary)
	f.dirty = false
	f.field = entityFieldPrimary
	if f.hasStatus && f.readOnly {
		f.field = entityFieldStatus
	}
	f.applyFocus()
	f.layoutFields()
}

func (f *EntityForm) Clear() {
	f.active = false
	f.itemID = ""
	f.hasStatus = false
	f.status = model.ProjectLifecycleOpen
	f.responsibleID = ""
	f.primary.SetValue("")
	f.secondary.SetValue("")
	f.dirty = false
	f.primary.Blur()
	f.secondary.Blur()
}

func (f *EntityForm) SetFocused(on bool) {
	f.focused = on && f.active
	if f.focused {
		f.applyFocus()
	} else {
		f.primary.Blur()
		f.secondary.Blur()
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
	secH := f.height - 6
	if secH < 3 {
		secH = 3
	}
	f.secondary.SetWidth(inner)
	f.secondary.SetHeight(secH)
}

func (f *EntityForm) FocusNext() {
	last := entityFieldSecondary
	if f.hasStatus {
		last = entityFieldStatus
	}
	if f.field < last {
		f.field++
	}
	f.applyFocus()
}

func (f *EntityForm) FocusPrev() {
	first := entityFieldPrimary
	if f.field > first {
		f.field--
	}
	f.applyFocus()
}

func (f *EntityForm) SetFieldIndex(i int) {
	switch i {
	case 0:
		f.field = entityFieldPrimary
	case 1:
		f.field = entityFieldSecondary
	case 2:
		if f.hasStatus {
			f.field = entityFieldStatus
		}
	default:
		return
	}
	f.applyFocus()
}

func (f *EntityForm) cycleStatus(delta int) {
	if !f.hasStatus || f.readOnly {
		return
	}
	if delta > 0 {
		f.status = f.status.Next()
	} else {
		f.status = f.status.Prev()
	}
	f.dirty = true
}

func (f *EntityForm) applyFocus() {
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
	if key, ok := msg.(tea.KeyMsg); ok && f.field == entityFieldStatus {
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
	header := f.styles.header.Render(model.KindHeading(f.kind, f.itemID))
	pLabel := f.styles.label.Render(f.primaryLabel)
	if f.focused && f.field == entityFieldPrimary && !f.readOnly {
		pLabel = f.styles.labelAct.Render(f.primaryLabel)
	}
	sLabel := f.styles.label.Render(f.secondaryLabel)
	if f.focused && f.field == entityFieldSecondary && !f.readOnly {
		sLabel = f.styles.labelAct.Render(f.secondaryLabel)
	}
	statusBlock := ""
	if f.hasStatus {
		stLabel := f.styles.label.Render("Status")
		if f.focused && f.field == entityFieldStatus && !f.readOnly {
			stLabel = f.styles.labelAct.Render("Status")
		}
		valStyle := f.styles.label
		if f.status == model.ProjectLifecycleClosed {
			valStyle = valStyle.Foreground(lipgloss.Color("245"))
		} else {
			valStyle = valStyle.Foreground(lipgloss.Color("42"))
		}
		hint := ""
		if f.focused && f.field == entityFieldStatus && !f.readOnly {
			hint = f.styles.label.Render("  ←/→ toggle")
		}
		statusBlock = strings.Join([]string{
			"",
			stLabel,
			valStyle.Render(f.status.Label()) + hint,
		}, "\n")
	}
	ro := ""
	if f.readOnly {
		ro = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" (read-only)") + "\n"
	}
	return strings.Join([]string{
		header + ro,
		pLabel,
		f.primary.View(),
		"",
		sLabel,
		f.secondary.View(),
		statusBlock,
	}, "\n")
}

func entityFormBlinkCmd() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

// NewEntityFormForTest exposes a form for unit tests.
func NewEntityFormForTest() EntityForm {
	return newEntityForm()
}
