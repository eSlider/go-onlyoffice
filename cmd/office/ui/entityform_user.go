package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func (f *EntityForm) userFieldCount() int {
	return 2 + len(model.UserACLDefs)
}

func (f *EntityForm) userView() string {
	header := f.styles.header.Render(model.KindHeading(f.kind, f.itemID))
	lines := []string{
		header,
		f.styles.label.Render("ID"),
		f.styles.meta.Render(f.itemID),
		"",
		f.renderUserEnabledRow(),
		"",
		f.renderUserPasswordRow(),
	}
	for i, def := range model.UserACLDefs {
		lines = append(lines, f.renderUserACLRow(i, def.Label, f.userACL.ACLModuleOn(def.Key)))
	}
	lines = append(lines, "", f.styles.label.Render("Groups"), f.styles.meta.Render(f.groupsText))
	return strings.Join(lines, "\n")
}

func (f *EntityForm) renderUserEnabledRow() string {
	label := f.styles.label.Render("Account")
	if f.focused && f.userFieldIdx == 0 {
		label = f.styles.labelAct.Render("Account")
	}
	val := "Disabled"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	if f.userEnabled {
		val = "Enabled"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	}
	hint := ""
	if f.focused && f.userFieldIdx == 0 {
		hint = f.styles.label.Render("  space toggle")
	}
	return strings.Join([]string{label, style.Render(val) + hint}, "\n")
}

func (f *EntityForm) renderUserPasswordRow() string {
	label := f.styles.label.Render("Password")
	if f.focused && f.userFieldIdx == 1 {
		label = f.styles.labelAct.Render("Password")
	}
	return strings.Join([]string{label, f.password.View(), f.styles.meta.Render("(leave blank to keep)")}, "\n")
}

func (f *EntityForm) renderUserACLRow(idx int, title string, on bool) string {
	fieldIdx := idx + 2
	label := f.styles.label.Render(title)
	if f.focused && f.userFieldIdx == fieldIdx {
		label = f.styles.labelAct.Render(title)
	}
	val := "Off"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	if on {
		val = "On"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	}
	hint := ""
	if f.focused && f.userFieldIdx == fieldIdx {
		hint = f.styles.label.Render("  space toggle")
	}
	return strings.Join([]string{label, style.Render(val) + hint}, "\n")
}

func (f *EntityForm) setUserFieldIndex(i int) {
	if i < 0 {
		i = 0
	}
	max := f.userFieldCount() - 1
	if i > max {
		i = max
	}
	f.userFieldIdx = i
	f.applyFocus()
}

func (f *EntityForm) toggleUserEnabled() {
	f.userEnabled = !f.userEnabled
	f.dirty = true
}

func (f *EntityForm) toggleUserACL(idx int) {
	if idx < 0 || idx >= len(model.UserACLDefs) {
		return
	}
	if idx == 0 {
		f.userACL.FullAccess = !f.userACL.FullAccess
		if f.userACL.FullAccess {
			for k := range f.userACL.Modules {
				f.userACL.Modules[k] = true
			}
		}
		f.dirty = true
		return
	}
	def := model.UserACLDefs[idx]
	if f.userACL.FullAccess {
		f.userACL.FullAccess = false
		for k := range f.userACL.Modules {
			f.userACL.Modules[k] = true
		}
	}
	f.userACL.Modules[def.Key] = !f.userACL.Modules[def.Key]
	f.dirty = true
}

func (f *EntityForm) updateUserForm(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch f.userFieldIdx {
		case 0:
			switch key.String() {
			case " ", "left", "right", "l", "h":
				f.toggleUserEnabled()
				return nil
			}
		case 1:
			var cmd tea.Cmd
			f.password, cmd = f.password.Update(msg)
			f.dirty = true
			return cmd
		default:
			switch key.String() {
			case " ", "left", "right", "l", "h":
				f.toggleUserACL(f.userFieldIdx - 2)
				return nil
			}
		}
	}
	return nil
}

func (f *EntityForm) applyUserFocus() {
	f.primary.Blur()
	f.secondary.Blur()
	if f.userFieldIdx == 1 {
		f.password.Focus()
	} else {
		f.password.Blur()
	}
}

func copyUserACLState(s model.UserACLState) model.UserACLState {
	modules := make(map[string]bool, len(s.Modules))
	for k, v := range s.Modules {
		modules[k] = v
	}
	return model.UserACLState{FullAccess: s.FullAccess, Modules: modules}
}
