package ui

import "github.com/eslider/go-onlyoffice/cmd/office/model"

const (
	userMinEmailWidth = 16
	userMinFixedWidth = 3
)

// layoutUserTable shows every user column; email absorbs leftover width.
func layoutUserTable(cols []model.TableColumn, totalW int) columnLayout {
	indices := make([]int, len(cols))
	for i := range cols {
		indices[i] = i
	}
	widths := make(map[int]int, len(cols))

	emailIdx := -1
	fixed := 0
	for i, col := range cols {
		if col.Key == "email" {
			emailIdx = i
			continue
		}
		widths[i] = col.Width
		fixed += widths[i]
	}
	if emailIdx < 0 {
		return layoutScrollingTable(cols, 0, totalW)
	}

	emailW := totalW - fixed
	if emailW < userMinEmailWidth {
		shrinkUserFixedColumns(widths, indices, emailIdx, fixed+userMinEmailWidth-totalW)
		fixed = 0
		for _, i := range indices {
			if i != emailIdx {
				fixed += widths[i]
			}
		}
		emailW = totalW - fixed
		if emailW < userMinEmailWidth {
			emailW = userMinEmailWidth
		}
	}
	widths[emailIdx] = emailW
	normalizeWidthSum(widths, indices, totalW, emailIdx)
	return columnLayout{indices: indices, widths: widths}
}

func shrinkUserFixedColumns(widths map[int]int, indices []int, emailIdx, need int) {
	for need > 0 {
		changed := false
		for _, i := range indices {
			if i == emailIdx || widths[i] <= userMinFixedWidth {
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
