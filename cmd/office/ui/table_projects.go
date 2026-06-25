package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

const (
	projectMinTitleWidth = 12
	projectMinFixedWidth = 3
)

// layoutProjectTable shows every project column; the title column absorbs leftover width.
func layoutProjectTable(cols []model.TableColumn, totalW int) columnLayout {
	indices := make([]int, len(cols))
	for i := range cols {
		indices[i] = i
	}
	widths := make(map[int]int, len(cols))

	titleIdx := -1
	fixed := 0
	for i, col := range cols {
		if col.Key == "title" {
			titleIdx = i
			continue
		}
		widths[i] = col.Width
		fixed += widths[i]
	}
	if titleIdx < 0 {
		return layoutScrollingTable(cols, 0, totalW)
	}

	titleW := totalW - fixed
	if titleW < projectMinTitleWidth {
		shrinkProjectFixedColumns(widths, indices, titleIdx, fixed+projectMinTitleWidth-totalW)
		fixed = 0
		for _, i := range indices {
			if i != titleIdx {
				fixed += widths[i]
			}
		}
		titleW = totalW - fixed
		if titleW < projectMinTitleWidth {
			titleW = projectMinTitleWidth
		}
	}
	widths[titleIdx] = titleW
	normalizeWidthSum(widths, indices, totalW, titleIdx)
	return columnLayout{indices: indices, widths: widths}
}

func shrinkProjectFixedColumns(widths map[int]int, indices []int, titleIdx, need int) {
	for need > 0 {
		changed := false
		for _, i := range indices {
			if i == titleIdx || widths[i] <= projectMinFixedWidth {
				continue
			}
			widths[i]--
			need--
			changed = true
			if need == 0 {
				return
			}
		}
		if !changed {
			return
		}
	}
}

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
