package ui

import (
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
)

type columnLayout struct {
	indices []int
	widths  map[int]int
}

func (t *DataTable) computeLayout() columnLayout {
	contentW := t.lineContentWidth()
	if len(t.columns) == 0 || contentW <= 0 {
		return columnLayout{widths: map[int]int{}}
	}
	if flex, ok := model.TableFlexLayoutFor(t.spec.Subject); ok {
		return layoutFlexTable(t.columns, contentW, flex.FlexColumnKey, flex.MinFlexWidth)
	}
	return layoutScrollingTable(t.columns, t.colScroll, contentW)
}

const minFixedColumnWidth = 3

// layoutFlexTable shows every column; flexKey absorbs leftover width.
func layoutFlexTable(cols []model.TableColumn, totalW int, flexKey string, minFlexWidth int) columnLayout {
	indices := make([]int, len(cols))
	for i := range cols {
		indices[i] = i
	}
	widths := make(map[int]int, len(cols))

	flexIdx := -1
	fixed := 0
	for i, col := range cols {
		if col.Key == flexKey {
			flexIdx = i
			continue
		}
		widths[i] = col.Width
		fixed += widths[i]
	}
	if flexIdx < 0 {
		return layoutScrollingTable(cols, 0, totalW)
	}

	flexW := totalW - fixed
	if flexW < minFlexWidth {
		shrinkFixedColumns(widths, indices, flexIdx, fixed+minFlexWidth-totalW)
		fixed = 0
		for _, i := range indices {
			if i != flexIdx {
				fixed += widths[i]
			}
		}
		flexW = totalW - fixed
		if flexW < minFlexWidth {
			flexW = minFlexWidth
		}
	}
	widths[flexIdx] = flexW
	normalizeWidthSum(widths, indices, totalW, flexIdx)
	return columnLayout{indices: indices, widths: widths}
}

func shrinkFixedColumns(widths map[int]int, indices []int, flexIdx, need int) {
	for need > 0 {
		changed := false
		for _, i := range indices {
			if i == flexIdx || widths[i] <= minFixedColumnWidth {
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

func layoutScrollingTable(cols []model.TableColumn, colScroll, contentW int) columnLayout {
	indices := pickVisibleColumnIndices(cols, colScroll, contentW)
	widths := make(map[int]int, len(indices))
	if len(indices) == 0 {
		return columnLayout{indices: indices, widths: widths}
	}
	minSum := 0
	for _, i := range indices {
		minSum += cols[i].Width
	}
	widths = distributeColumnWidths(minSum, contentW, indices, cols)
	return columnLayout{indices: indices, widths: widths}
}

func pickVisibleColumnIndices(cols []model.TableColumn, colScroll, contentW int) []int {
	var out []int
	used := 0
	for colIdx := colScroll; colIdx < len(cols); colIdx++ {
		w := cols[colIdx].Width
		if len(out) > 0 && used+w > contentW {
			break
		}
		out = append(out, colIdx)
		used += w
	}
	if len(out) == 0 {
		colIdx := colScroll
		if colIdx < 0 || colIdx >= len(cols) {
			colIdx = 0
		}
		out = []int{colIdx}
	}
	return out
}

// distributeColumnWidths expands or shrinks visible columns to exactly fill total width.
func distributeColumnWidths(minSum, total int, indices []int, cols []model.TableColumn) map[int]int {
	out := make(map[int]int, len(indices))
	if len(indices) == 0 {
		return out
	}
	if total < len(indices) {
		total = len(indices)
	}
	if minSum <= 0 {
		each := total / len(indices)
		if each < 1 {
			each = 1
		}
		for _, i := range indices {
			out[i] = each
		}
		normalizeWidthSum(out, indices, total, indices[len(indices)-1])
		return out
	}
	for _, i := range indices {
		out[i] = cols[i].Width
	}
	if minSum >= total {
		for _, i := range indices {
			out[i] = cols[i].Width * total / minSum
			if out[i] < 1 {
				out[i] = 1
			}
		}
		normalizeWidthSum(out, indices, total, flexColumnIndex(indices, cols))
		return out
	}
	extra := total - minSum
	flex := flexColumnIndices(indices, cols)
	flexSum := 0
	for _, i := range flex {
		flexSum += cols[i].Width
	}
	if flexSum <= 0 {
		flexSum = len(flex)
	}
	for _, i := range flex {
		out[i] += extra * cols[i].Width / flexSum
	}
	normalizeWidthSum(out, indices, total, flexColumnIndex(indices, cols))
	return out
}

func flexColumnIndices(indices []int, cols []model.TableColumn) []int {
	flex := make([]int, 0, len(indices))
	for _, i := range indices {
		switch cols[i].Key {
		case "title", "subtitle", "description", "displayName", "primaryEmail", "from", "to", "type":
			flex = append(flex, i)
		}
	}
	if len(flex) == 0 {
		flex = append(flex, indices...)
	}
	return flex
}

func flexColumnIndex(indices []int, cols []model.TableColumn) int {
	flex := flexColumnIndices(indices, cols)
	if len(flex) == 0 {
		return indices[len(indices)-1]
	}
	for _, i := range indices {
		if cols[i].Key == "title" {
			return i
		}
	}
	return flex[0]
}

func normalizeWidthSum(widths map[int]int, indices []int, total, adjustIdx int) {
	if len(indices) == 0 {
		return
	}
	sum := 0
	for _, i := range indices {
		sum += widths[i]
	}
	if adjustIdx < 0 {
		adjustIdx = indices[len(indices)-1]
	}
	widths[adjustIdx] += total - sum
	if widths[adjustIdx] < 1 {
		widths[adjustIdx] = 1
	}
}

const cellHPadding = 0 // padding lives in outer lipgloss styles; column width is the full cell width

func truncateCellText(text string, colWidth int) string {
	if colWidth < 1 {
		colWidth = 1
	}
	if runewidth.StringWidth(text) <= colWidth {
		return text
	}
	return runewidth.Truncate(text, colWidth, "…")
}
