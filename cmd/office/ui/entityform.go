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
)

// EntityForm is the top section of the detail pane for editable entities.
type EntityForm struct {
	active         bool
	kind           model.Kind
	itemID         string
	primaryLabel   string
	secondaryLabel string
	readOnly       bool
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
	f.primary.SetValue(fields.Primary)
	f.secondary.SetValue(fields.Secondary)
	f.dirty = false
	f.field = entityFieldPrimary
	f.applyFocus()
	f.layoutFields()
}

func (f *EntityForm) Clear() {
	f.active = false
	f.itemID = ""
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
	if f.field < entityFieldSecondary {
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
	default:
		f.primary.Blur()
		f.secondary.Focus()
	}
}

func (f *EntityForm) Update(msg tea.Msg) tea.Cmd {
	if !f.active || !f.focused || f.readOnly {
		return nil
	}
	var cmd tea.Cmd
	switch f.field {
	case entityFieldPrimary:
		f.primary, cmd = f.primary.Update(msg)
	default:
		f.secondary, cmd = f.secondary.Update(msg)
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
	}, "\n")
}

func entityFormBlinkCmd() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

// NewEntityFormForTest exposes a form for unit tests.
func NewEntityFormForTest() EntityForm {
	return newEntityForm()
}
