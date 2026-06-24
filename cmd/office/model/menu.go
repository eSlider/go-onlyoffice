package model

// menuNode is one row in the left navigation tree (flat slice with depth).
type menuNode struct {
	label    string
	depth    int
	subject  Subject // zero when branch node
	expandable bool
	parent   int // index of parent branch, -1 for roots
}

// MenuTree is the left-pane OnlyOffice module navigator.
type MenuTree struct {
	nodes      []menuNode
	cursor     int
	expanded   map[int]bool
	visible    []int // indices into nodes, recomputed on expand/collapse
}

// DefaultMenuTree returns the standard OnlyOffice Workspace module tree.
func DefaultMenuTree() *MenuTree {
	nodes := []menuNode{
		{label: "Projects", depth: 0, expandable: true, parent: -1},
		{label: "All projects", depth: 1, subject: SubjectProjects, parent: 0},
		{label: "Tasks", depth: 1, subject: SubjectTasks, parent: 0},
		{label: "Calendar", depth: 0, expandable: true, parent: -1},
		{label: "Calendars", depth: 1, subject: SubjectCalendars, parent: 3},
		{label: "Events", depth: 1, subject: SubjectEvents, parent: 3},
		{label: "CRM", depth: 0, expandable: true, parent: -1},
		{label: "Contacts", depth: 1, subject: SubjectContacts, parent: 6},
		{label: "Persons", depth: 1, subject: SubjectPersons, parent: 6},
		{label: "Companies", depth: 1, subject: SubjectCompanies, parent: 6},
		{label: "Opportunities", depth: 1, subject: SubjectOpportunities, parent: 6},
		{label: "Cases", depth: 1, subject: SubjectCases, parent: 6},
		{label: "CRM Tasks", depth: 1, subject: SubjectCRMTasks, parent: 6},
		{label: "Mail", depth: 0, expandable: true, parent: -1},
		{label: "Inbox", depth: 1, subject: SubjectMailInbox, parent: 13},
		{label: "Sent", depth: 1, subject: SubjectMailSent, parent: 13},
		{label: "Drafts", depth: 1, subject: SubjectMailDrafts, parent: 13},
		{label: "Trash", depth: 1, subject: SubjectMailTrash, parent: 13},
		{label: "Spam", depth: 1, subject: SubjectMailSpam, parent: 13},
		{label: "Documents", depth: 0, expandable: true, parent: -1},
		{label: "Project files", depth: 1, subject: SubjectProjectFiles, parent: 19},
		{label: "Task files", depth: 1, subject: SubjectTaskFiles, parent: 19},
		{label: "Users", depth: 0, expandable: true, parent: -1},
		{label: "Directory", depth: 1, subject: SubjectUsers, parent: 22},
	}
	m := &MenuTree{
		nodes:    nodes,
		expanded: map[int]bool{},
	}
	m.rebuildVisible()
	return m
}

// RootLabels returns top-level module names in order.
func (m *MenuTree) RootLabels() []string {
	var out []string
	for i, n := range m.nodes {
		if n.depth == 0 {
			out = append(out, n.label)
		}
		_ = i
	}
	return out
}

// Cursor returns the visible-row cursor index.
func (m *MenuTree) Cursor() int { return m.cursor }

// SetCursor sets the visible-row cursor.
func (m *MenuTree) SetCursor(c int) {
	if c < 0 {
		c = 0
	}
	if c >= len(m.visible) {
		c = len(m.visible) - 1
	}
	if c < 0 {
		c = 0
	}
	m.cursor = c
}

// MoveDown advances the menu cursor.
func (m *MenuTree) MoveDown() {
	m.SetCursor(m.cursor + 1)
}

// MoveUp moves the menu cursor up.
func (m *MenuTree) MoveUp() {
	m.SetCursor(m.cursor - 1)
}

// VisibleCount returns the number of visible menu rows.
func (m *MenuTree) VisibleCount() int { return len(m.visible) }

// LabelAt returns the label and depth for visible row i.
func (m *MenuTree) LabelAt(i int) (label string, depth int) {
	if i < 0 || i >= len(m.visible) {
		return "", 0
	}
	n := m.nodes[m.visible[i]]
	return n.label, n.depth
}

// IsExpandable reports whether visible row i is a branch node.
func (m *MenuTree) IsExpandable(visibleIdx int) bool {
	if visibleIdx < 0 || visibleIdx >= len(m.visible) {
		return false
	}
	return m.nodes[m.visible[visibleIdx]].expandable
}

// IsExpanded reports whether the branch at visible row i is expanded.
func (m *MenuTree) IsExpanded(visibleIdx int) bool {
	if visibleIdx < 0 || visibleIdx >= len(m.visible) {
		return false
	}
	return m.expanded[m.visible[visibleIdx]]
}

// ToggleExpand expands or collapses the branch at visible row i.
func (m *MenuTree) ToggleExpand(visibleIdx int) {
	if !m.IsExpandable(visibleIdx) {
		return
	}
	idx := m.visible[visibleIdx]
	if m.expanded[idx] {
		delete(m.expanded, idx)
	} else {
		m.expanded[idx] = true
	}
	m.rebuildVisible()
}

// SelectIndex returns the subject for visible row i when it is a leaf.
func (m *MenuTree) SelectIndex(visibleIdx int) (Subject, bool) {
	if visibleIdx < 0 || visibleIdx >= len(m.visible) {
		return "", false
	}
	n := m.nodes[m.visible[visibleIdx]]
	if n.subject == "" {
		return "", false
	}
	return n.subject, true
}

// CurrentSubject returns the subject at the current cursor if it is a leaf.
func (m *MenuTree) CurrentSubject() (Subject, bool) {
	return m.SelectIndex(m.cursor)
}

func (m *MenuTree) rebuildVisible() {
	m.visible = m.visible[:0]
	for i, n := range m.nodes {
		if n.depth == 0 {
			m.visible = append(m.visible, i)
			if m.expanded[i] {
				m.appendChildren(i)
			}
			continue
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *MenuTree) appendChildren(branchIdx int) {
	for i, n := range m.nodes {
		if n.parent == branchIdx {
			m.visible = append(m.visible, i)
		}
	}
}
