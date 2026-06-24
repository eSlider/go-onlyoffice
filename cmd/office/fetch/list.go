package fetch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// Loader fetches list items for a menu subject using the OnlyOffice client.
type Loader struct {
	Client *onlyoffice.Client
}

// List returns items for the given list spec (nav leaf).
func (l *Loader) List(ctx context.Context, spec model.ListSpec) ([]model.Item, error) {
	if l == nil || l.Client == nil {
		return nil, fmt.Errorf("fetch: client is nil")
	}
	switch spec.Subject {
	case model.SubjectProjects:
		return l.listProjects(ctx)
	case model.SubjectTasks:
		if spec.ProjectID != "" {
			return l.listTasksForProject(ctx, spec.ProjectID)
		}
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
		pid := spec.ProjectID
		if pid == "" {
			pid = onlyoffice.GetEnvironmentDefaults().ProjectID
		}
		if pid == "" {
			return nil, fmt.Errorf("set ONLYOFFICE_PROJECT_ID or pick a project in the tree")
		}
		return l.listProjectFiles(ctx, pid)
	case model.SubjectTaskFiles:
		if spec.TaskID == "" {
			return nil, fmt.Errorf("pick a task under Projects in the tree")
		}
		return l.listTaskFiles(ctx, spec.TaskID)
	default:
		return nil, fmt.Errorf("unsupported subject %q", spec.Subject)
	}
}

// LoadProjectsForNav returns projects to inject as dynamic tree nodes.
func (l *Loader) LoadProjectsForNav(ctx context.Context) ([]model.Item, error) {
	return l.listProjects(ctx)
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
	case model.KindTask, model.KindCRMTask:
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

// Execute runs a user-selected action on an item.
func (l *Loader) Execute(ctx context.Context, action model.ActionID, item model.Item, destPath string) (string, error) {
	switch action {
	case model.ActionView:
		return "view", nil
	case model.ActionDelete:
		return l.executeDelete(ctx, item)
	case model.ActionDownload:
		return l.executeDownload(ctx, item, destPath)
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}
}

func (l *Loader) executeDelete(ctx context.Context, item model.Item) (string, error) {
	switch item.Kind {
	case model.KindProject:
		id, err := strconv.Atoi(item.ID)
		if err != nil {
			return "", err
		}
		if _, err := l.Client.DeleteProject(id); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted project %s", item.Title), nil
	case model.KindTask:
		if _, err := l.Client.DeleteTask(ctx, item.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted task %s", item.Title), nil
	case model.KindContact:
		if _, err := l.Client.DeleteContact(ctx, item.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted contact %s", item.Title), nil
	case model.KindOpportunity:
		if _, err := l.Client.DeleteOpportunity(ctx, item.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted deal %s", item.Title), nil
	case model.KindCase:
		if _, err := l.Client.DeleteCase(ctx, item.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted case %s", item.Title), nil
	case model.KindCRMTask:
		if _, err := l.Client.DeleteCRMTask(ctx, item.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted CRM task %s", item.Title), nil
	case model.KindMail:
		id, err := strconv.Atoi(item.ID)
		if err != nil {
			return "", err
		}
		if _, err := l.Client.RemoveMailMessages(ctx, id); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted message %s", item.Title), nil
	case model.KindFile:
		id, err := strconv.Atoi(item.ID)
		if err != nil {
			return "", err
		}
		if err := l.Client.DeleteFiles(ctx, []int{id}); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted file %s", item.Title), nil
	default:
		return "", fmt.Errorf("delete not supported for %s", item.Kind)
	}
}

func (l *Loader) executeDownload(ctx context.Context, item model.Item, destPath string) (string, error) {
	if item.Kind != model.KindFile {
		return "", fmt.Errorf("download only for files")
	}
	if destPath == "" {
		destPath = filepath.Join(os.TempDir(), "office", item.Title)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := l.Client.DownloadFile(ctx, item.ID, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("Downloaded to %s", destPath), nil
}

func (l *Loader) listTasksForProject(ctx context.Context, projectID string) ([]model.Item, error) {
	rows, err := l.Client.ListTasks(ctx, projectID, "")
	if err != nil {
		return nil, err
	}
	return ItemsFromMaps(rows, model.KindTask, TaskItemFields), nil
}

func (l *Loader) listTaskFiles(ctx context.Context, taskID string) ([]model.Item, error) {
	files, err := l.Client.GetTaskFiles(ctx, taskID)
	if err != nil {
		return nil, err
	}
	var items []model.Item
	for _, f := range files {
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
		raw := map[string]any{
			"id":    id,
			"title": title,
		}
		if p.TaskCount != nil {
			raw["taskCount"] = *p.TaskCount
		}
		if p.TaskCountTotal != nil {
			raw["taskCountTotal"] = *p.TaskCountTotal
		}
		if p.DocumentsCount != nil {
			raw["documentsCount"] = *p.DocumentsCount
		}
		if p.ParticipantCount != nil {
			raw["participantCount"] = *p.ParticipantCount
		}
		items[i] = model.Item{
			ID: id, Title: title, Kind: model.KindProject, Raw: raw,
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

func (l *Loader) listProjectFiles(ctx context.Context, projectID string) ([]model.Item, error) {
	resp, err := l.Client.GetProjectFiles(ctx, projectID)
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
