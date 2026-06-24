package fetch

import (
	"context"
	"fmt"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// Loader fetches list items for a menu subject using the OnlyOffice client.
type Loader struct {
	Client *onlyoffice.Client
}

// List returns items for the given subject.
func (l *Loader) List(ctx context.Context, subject model.Subject) ([]model.Item, error) {
	if l == nil || l.Client == nil {
		return nil, fmt.Errorf("fetch: client is nil")
	}
	switch subject {
	case model.SubjectProjects:
		return l.listProjects(ctx)
	case model.SubjectTasks:
		return l.listTasks(ctx)
	case model.SubjectCalendars:
		return l.listCalendars(ctx)
	case model.SubjectEvents:
		return l.listEvents(ctx)
	case model.SubjectContacts:
		return l.listContacts(ctx, nil)
	case model.SubjectPersons:
		falseVal := false
		return l.listContacts(ctx, &falseVal)
	case model.SubjectCompanies:
		trueVal := true
		return l.listContacts(ctx, &trueVal)
	case model.SubjectOpportunities:
		return l.listOpportunities(ctx)
	case model.SubjectCases:
		return l.listCases(ctx)
	case model.SubjectCRMTasks:
		return l.listCRMTasks(ctx)
	case model.SubjectMailInbox:
		return l.listMail(ctx, onlyoffice.MailFolderInbox)
	case model.SubjectMailSent:
		return l.listMail(ctx, onlyoffice.MailFolderSent)
	case model.SubjectMailDrafts:
		return l.listMail(ctx, onlyoffice.MailFolderDrafts)
	case model.SubjectMailTrash:
		return l.listMail(ctx, onlyoffice.MailFolderTrash)
	case model.SubjectMailSpam:
		return l.listMail(ctx, onlyoffice.MailFolderSpam)
	case model.SubjectUsers:
		return l.listUsers(ctx)
	case model.SubjectProjectFiles:
		return l.listProjectFiles(ctx)
	case model.SubjectTaskFiles:
		return nil, fmt.Errorf("select a task in Tasks first (task files need task id)")
	default:
		return nil, fmt.Errorf("unsupported subject %q", subject)
	}
}

// Detail fetches full record data for preview when list row is insufficient.
func (l *Loader) Detail(ctx context.Context, item model.Item) (map[string]any, error) {
	switch item.Kind {
	case model.KindOpportunity:
		return l.Client.GetOpportunity(ctx, item.ID)
	case model.KindContact:
		return l.Client.GetContact(ctx, item.ID)
	case model.KindMail:
		return l.Client.GetMailMessage(ctx, item.ID)
	case model.KindTask:
		return l.Client.GetTaskByID(ctx, item.ID)
	case model.KindProject:
		return l.Client.GetProjectByID(ctx, item.ID)
	default:
		if item.Raw != nil {
			return item.Raw, nil
		}
		return map[string]any{"id": item.ID, "title": item.Title}, nil
	}
}

func (l *Loader) listProjects(ctx context.Context) ([]model.Item, error) {
	projects, err := l.Client.GetProjects()
	if err != nil {
		return nil, err
	}
	items := make([]model.Item, len(projects))
	for i, p := range projects {
		id := ""
		if p.ID != nil {
			id = fmt.Sprint(*p.ID)
		}
		title := ""
		if p.Title != nil {
			title = *p.Title
		}
		status := ""
		if p.Status != nil {
			status = fmt.Sprint(*p.Status)
		}
		items[i] = model.Item{
			ID: id, Title: title, Subtitle: status, Kind: model.KindProject,
			Raw: map[string]any{"id": id, "title": title, "status": status},
		}
	}
	return items, nil
}

func (l *Loader) listTasks(ctx context.Context) ([]model.Item, error) {
	rows, err := l.Client.ListAllTasks(ctx, "")
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindTask, TaskItemFields), nil
}

func (l *Loader) listCalendars(ctx context.Context) ([]model.Item, error) {
	rows, err := l.Client.ListCalendars(ctx, "", "")
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindCalendar, FieldMap{IDKey: "objectId", TitleKey: "title"}), nil
}

func (l *Loader) listEvents(ctx context.Context) ([]model.Item, error) {
	start := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	rows, err := l.Client.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindEvent, FieldMap{IDKey: "objectId", TitleKey: "title", SubtitleKey: "start"}), nil
}

func (l *Loader) listContacts(ctx context.Context, companyOnly *bool) ([]model.Item, error) {
	rows, err := l.Client.ListAllContacts(ctx)
	if err != nil {
		return nil, err
	}
	if companyOnly != nil {
		filtered := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			isCo, _ := r["isCompany"].(bool)
			if isCo == *companyOnly {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	return ItemsFromMaps(rows, model.KindContact, ContactItemFields), nil
}

func (l *Loader) listOpportunities(ctx context.Context) ([]model.Item, error) {
	rows, err := l.Client.ListAllOpportunities(ctx)
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindOpportunity, FieldMap{IDKey: "id", TitleKey: "title", SubtitleKey: "stage"}), nil
}

func (l *Loader) listCases(ctx context.Context) ([]model.Item, error) {
	cases, _, err := l.Client.ListCases(ctx, 100, 0)
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(cases, model.KindCase, FieldMap{IDKey: "id", TitleKey: "title"}), nil
}

func (l *Loader) listCRMTasks(ctx context.Context) ([]model.Item, error) {
	rows, _, err := l.Client.ListCRMTasks(ctx, 100, 0)
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindCRMTask, TaskItemFields), nil
}

func (l *Loader) listMail(ctx context.Context, folder int) ([]model.Item, error) {
	rows, err := l.Client.ListMailMessages(ctx, onlyoffice.MailMessagesFilter{Folder: folder, Count: 50})
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindMail, MailItemFields), nil
}

func (l *Loader) listUsers(ctx context.Context) ([]model.Item, error) {
	users, err := l.Client.GetUsers()
	if err != nil {
		return nil, err
	}
	items := make([]model.Item, len(users))
	for i, u := range users {
		id := ""
		if u.ID != nil {
			id = fmt.Sprint(*u.ID)
		}
		title := ""
		if u.DisplayName != nil {
			title = *u.DisplayName
		}
		email := ""
		if u.Email != nil {
			email = *u.Email
		}
		items[i] = model.Item{
			ID: id, Title: title, Subtitle: email, Kind: model.KindUser,
			Raw: map[string]any{"id": id, "displayName": title, "email": email},
		}
	}
	return items, nil
}

func (l *Loader) listProjectFiles(ctx context.Context) ([]model.Item, error) {
	def := l.Client // need project id from defaults
	_ = def
	pid := onlyoffice.GetEnvironmentDefaults().ProjectID
	if pid == "" {
		return nil, fmt.Errorf("set ONLYOFFICE_PROJECT_ID for project files")
	}
	resp, err := l.Client.GetProjectFiles(ctx, pid)
	if err != nil {
		return nil, err
	}
	var items []model.Item
	for _, f := range resp.Files {
		id, title := "", ""
		if f.ID != nil {
			id = f.ID.String()
		}
		if f.Title != nil {
			title = *f.Title
		}
		items = append(items, model.Item{
			ID: id, Title: title, Kind: model.KindFile,
			Raw: map[string]any{"id": id, "title": title},
		})
	}
	return items, nil
}
