package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func (t *DataTable) projectBaseStyle(item model.Item) lipgloss.Style {
	if model.ProjectIsOpen(item.Raw) {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("108"))
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("252"))
}

func (t *DataTable) projectCellStyle(row, col int, colKey string, selected bool, item model.Item) lipgloss.Style {
	isRow := row == t.cursorRow
	isCol := col == t.cursorCol
	isCell := t.focused && isRow && isCol

	switch {
	case selected && isCell:
		return t.styles.cellSelect
	case selected && isRow:
		return t.styles.rowSelect
	case selected:
		return t.styles.rowSelect
	case isCell:
		return t.styles.cellActive
	case t.focused && (isRow || isCol):
		return t.projectStatusStyle(colKey, item, t.projectBaseStyle(item))
	default:
		return t.projectStatusStyle(colKey, item, t.projectBaseStyle(item))
	}
}

func (t *DataTable) projectStatusStyle(colKey string, item model.Item, base lipgloss.Style) lipgloss.Style {
	if colKey != "status" {
		return base
	}
	if model.ProjectIsOpen(item.Raw) {
		return base.Foreground(lipgloss.Color("42")).Bold(true)
	}
	return base.Foreground(lipgloss.Color("245"))
}
