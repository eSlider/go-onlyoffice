package model

// TableFlexLayout describes a list table where every column stays visible and
// one flex column absorbs leftover pane width.
type TableFlexLayout struct {
	FlexColumnKey string
	MinFlexWidth  int
}

// TableFlexLayoutFor returns a flex layout policy for subjects that show all columns.
func TableFlexLayoutFor(subject Subject) (TableFlexLayout, bool) {
	switch subject {
	case SubjectProjects:
		return TableFlexLayout{FlexColumnKey: "title", MinFlexWidth: 12}, true
	case SubjectUsers:
		return TableFlexLayout{FlexColumnKey: "email", MinFlexWidth: 16}, true
	default:
		return TableFlexLayout{}, false
	}
}
