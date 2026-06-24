package model

// Kind identifies the OnlyOffice entity type for list items and preview routing.
type Kind string

const (
	KindProject     Kind = "project"
	KindTask        Kind = "task"
	KindContact     Kind = "contact"
	KindOpportunity Kind = "opportunity"
	KindCase        Kind = "case"
	KindCRMTask     Kind = "crm_task"
	KindMail        Kind = "mail"
	KindEvent       Kind = "event"
	KindCalendar    Kind = "calendar"
	KindFile        Kind = "file"
	KindUser        Kind = "user"
)

// Subject identifies a menu leaf / list source.
type Subject string

const (
	SubjectProjects      Subject = "projects"
	SubjectTasks         Subject = "tasks"
	SubjectCalendars     Subject = "calendars"
	SubjectEvents        Subject = "events"
	SubjectContacts      Subject = "contacts"
	SubjectPersons       Subject = "persons"
	SubjectCompanies     Subject = "companies"
	SubjectOpportunities Subject = "opportunities"
	SubjectCases         Subject = "cases"
	SubjectCRMTasks      Subject = "crm_tasks"
	SubjectMailInbox     Subject = "mail_inbox"
	SubjectMailSent      Subject = "mail_sent"
	SubjectMailDrafts    Subject = "mail_drafts"
	SubjectMailTrash     Subject = "mail_trash"
	SubjectMailSpam      Subject = "mail_spam"
	SubjectProjectFiles  Subject = "project_files"
	SubjectTaskFiles     Subject = "task_files"
	SubjectUsers         Subject = "users"
)

// FocusPane is which column has keyboard focus.
type FocusPane int

const (
	FocusMenu FocusPane = iota
	FocusList
	FocusPreview
)

// PrevFocusPane cycles preview → list → menu → preview.
func PrevFocusPane(p FocusPane) FocusPane {
	switch p {
	case FocusMenu:
		return FocusPreview
	case FocusList:
		return FocusMenu
	default:
		return FocusList
	}
}

// NextFocusPane cycles menu → list → preview → menu.
func NextFocusPane(p FocusPane) FocusPane {
	switch p {
	case FocusMenu:
		return FocusList
	case FocusList:
		return FocusPreview
	default:
		return FocusMenu
	}
}

// Item is one row in the center list pane.
type Item struct {
	ID, Title, Subtitle string
	Kind                Kind
	Raw                 map[string]any
	Selected            bool
}

// Selection tracks multi-selected item IDs within the current subject.
type Selection struct {
	ids map[string]struct{}
}

// NewSelection returns an empty selection set.
func NewSelection() *Selection {
	return &Selection{ids: make(map[string]struct{})}
}

// Toggle flips selection on items[idx] and updates the ID set.
func (s *Selection) Toggle(items *[]Item, idx int) {
	if idx < 0 || idx >= len(*items) {
		return
	}
	it := &(*items)[idx]
	it.Selected = !it.Selected
	if it.Selected {
		s.ids[it.ID] = struct{}{}
	} else {
		delete(s.ids, it.ID)
	}
}

// Count returns the number of selected items.
func (s *Selection) Count() int {
	return len(s.ids)
}

// Clear removes all selections.
func (s *Selection) Clear() {
	s.ids = make(map[string]struct{})
}

// IDs returns selected item IDs in stable order (map iteration order is fine for display).
func (s *Selection) IDs() []string {
	out := make([]string, 0, len(s.ids))
	for id := range s.ids {
		out = append(out, id)
	}
	return out
}
