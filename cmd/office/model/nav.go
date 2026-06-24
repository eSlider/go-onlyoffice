package model

import "strings"

// ListSpec tells the fetch layer what to load when a nav leaf is active.
type ListSpec struct {
	Subject   Subject
	ProjectID string
	TaskID    string
}

// NavNode is one row in the navigation tree.
type NavNode struct {
	ID       string
	Label    string
	Branch   bool
	List     *ListSpec // set on leaves that open the center list
	ParentID string
}

// NavTree is a hierarchical navigator; the center list loads only on leaves.
type NavTree struct {
	nodes    map[string]NavNode
	roots    []string
	children map[string][]string
	expanded map[string]bool
	cursor   int
	visible  []string
	filter   string
}

// DefaultNavTree returns the OnlyOffice module tree (static skeleton).
func DefaultNavTree() *NavTree {
	t := &NavTree{
		nodes:    make(map[string]NavNode),
		children: make(map[string][]string),
		expanded: make(map[string]bool),
	}
	add := func(id, label, parent string, branch bool, list *ListSpec) {
		t.nodes[id] = NavNode{ID: id, Label: label, Branch: branch, List: list, ParentID: parent}
		if parent == "" {
			t.roots = append(t.roots, id)
		} else {
			t.children[parent] = append(t.children[parent], id)
		}
	}

	add("projects", "Projects", "", false, &ListSpec{Subject: SubjectProjects})
	add("tasks", "Tasks", "", false, &ListSpec{Subject: SubjectTasks})
	add("projects.dynamic", "By project", "", true, nil)

	add("calendar", "Calendar", "", false, &ListSpec{Subject: SubjectCalendar})

	add("crm", "CRM", "", true, nil)
	add("crm.contacts", "Contacts", "crm", false, &ListSpec{Subject: SubjectContacts})
	add("crm.persons", "Persons", "crm", false, &ListSpec{Subject: SubjectPersons})
	add("crm.companies", "Companies", "crm", false, &ListSpec{Subject: SubjectCompanies})
	add("crm.opportunities", "Opportunities", "crm", false, &ListSpec{Subject: SubjectOpportunities})
	add("crm.cases", "Cases", "crm", false, &ListSpec{Subject: SubjectCases})
	add("crm.tasks", "CRM tasks", "crm", false, &ListSpec{Subject: SubjectCRMTasks})

	add("mail", "Mail", "", true, nil)
	add("mail.inbox", "Inbox", "mail", false, &ListSpec{Subject: SubjectMailInbox})
	add("mail.sent", "Sent", "mail", false, &ListSpec{Subject: SubjectMailSent})
	add("mail.drafts", "Drafts", "mail", false, &ListSpec{Subject: SubjectMailDrafts})
	add("mail.trash", "Trash", "mail", false, &ListSpec{Subject: SubjectMailTrash})
	add("mail.spam", "Spam", "mail", false, &ListSpec{Subject: SubjectMailSpam})

	add("users", "Users", "", true, nil)
	add("users.dir", "Directory", "users", false, &ListSpec{Subject: SubjectUsers})

	t.rebuildVisible()
	return t
}

// RootLabels returns top-level labels (for tests).
func (t *NavTree) RootLabels() []string {
	out := make([]string, len(t.roots))
	for i, id := range t.roots {
		out[i] = t.nodes[id].Label
	}
	return out
}

func (t *NavTree) Cursor() int { return t.cursor }

func (t *NavTree) VisibleCount() int { return len(t.visible) }

func (t *NavTree) NodeAtVisible(i int) (NavNode, bool) {
	if i < 0 || i >= len(t.visible) {
		return NavNode{}, false
	}
	n, ok := t.nodes[t.visible[i]]
	return n, ok
}

func (t *NavTree) DepthAtVisible(i int) int {
	if i < 0 || i >= len(t.visible) {
		return 0
	}
	depth := 0
	id := t.visible[i]
	for {
		n, ok := t.nodes[id]
		if !ok || n.ParentID == "" {
			break
		}
		depth++
		id = n.ParentID
	}
	return depth
}

func (t *NavTree) IsExpandable(i int) bool {
	n, ok := t.NodeAtVisible(i)
	if !ok {
		return false
	}
	return n.Branch && len(t.children[n.ID]) > 0
}

func (t *NavTree) IsExpanded(i int) bool {
	n, ok := t.NodeAtVisible(i)
	if !ok {
		return false
	}
	return t.expanded[n.ID]
}

func (t *NavTree) ToggleExpand(i int) {
	n, ok := t.NodeAtVisible(i)
	if !ok || !n.Branch {
		return
	}
	if t.expanded[n.ID] {
		delete(t.expanded, n.ID)
	} else {
		t.expanded[n.ID] = true
	}
	t.rebuildVisible()
}

func (t *NavTree) MoveUp() { t.SetCursor(t.cursor - 1) }
func (t *NavTree) MoveDown() { t.SetCursor(t.cursor + 1) }

func (t *NavTree) SetCursor(c int) {
	if c < 0 {
		c = 0
	}
	if c >= len(t.visible) {
		c = len(t.visible) - 1
	}
	if c < 0 {
		c = 0
	}
	t.cursor = c
}

// CurrentListSpec returns the list spec when the cursor is on a leaf.
func (t *NavTree) CurrentListSpec() (*ListSpec, bool) {
	n, ok := t.NodeAtVisible(t.cursor)
	if !ok || n.List == nil {
		return nil, false
	}
	spec := *n.List
	return &spec, true
}

// Activate expands a branch or returns leaf list spec on Enter.
func (t *NavTree) Activate() (*ListSpec, bool) {
	n, ok := t.NodeAtVisible(t.cursor)
	if !ok {
		return nil, false
	}
	if n.Branch {
		t.expanded[n.ID] = true
		t.rebuildVisible()
		return nil, false
	}
	if n.List != nil {
		spec := *n.List
		return &spec, true
	}
	return nil, false
}

func (t *NavTree) SetFilter(query string) {
	t.filter = strings.ToLower(strings.TrimSpace(query))
	t.rebuildVisible()
}

func (t *NavTree) ClearFilter() {
	t.filter = ""
	t.rebuildVisible()
}

func (t *NavTree) FilterQuery() string { return t.filter }

func (t *NavTree) nodeLabelMatches(id string) bool {
	if t.filter == "" {
		return true
	}
	n, ok := t.nodes[id]
	if !ok {
		return false
	}
	return strings.Contains(strings.ToLower(n.Label), t.filter)
}

func (t *NavTree) hasMatchingDescendant(id string) bool {
	for _, child := range t.children[id] {
		if t.nodeLabelMatches(child) || t.hasMatchingDescendant(child) {
			return true
		}
	}
	return false
}

func (t *NavTree) visibleUnderFilter(id string) bool {
	if t.filter == "" {
		return true
	}
	return t.nodeLabelMatches(id) || t.hasMatchingDescendant(id)
}

func (t *NavTree) rebuildVisible() {
	t.visible = t.visible[:0]
	var walk func(id string)
	walk = func(id string) {
		if !t.visibleUnderFilter(id) {
			return
		}
		t.visible = append(t.visible, id)
		expanded := t.expanded[id]
		if t.filter != "" && t.hasMatchingDescendant(id) {
			expanded = true
		}
		if !expanded {
			return
		}
		for _, child := range t.children[id] {
			walk(child)
		}
	}
	for _, root := range t.roots {
		walk(root)
	}
	if t.cursor >= len(t.visible) {
		t.cursor = len(t.visible) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}
