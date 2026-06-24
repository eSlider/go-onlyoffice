package model

import "strings"

// InjectProjectNodes adds dynamic project branches under projects.dynamic.
func (t *NavTree) InjectProjectNodes(projects []Item) {
	// Drop prior dynamic project.* nodes.
	for id := range t.nodes {
		if strings.HasPrefix(id, "project.") {
			delete(t.nodes, id)
		}
	}
	t.children["projects.dynamic"] = nil
	for _, p := range projects {
		pid := "project." + p.ID
		t.nodes[pid] = NavNode{ID: pid, Label: p.Title, Branch: true, ParentID: "projects.dynamic"}
		t.children["projects.dynamic"] = append(t.children["projects.dynamic"], pid)
		taskID := pid + ".tasks"
		fileID := pid + ".files"
		t.nodes[taskID] = NavNode{
			ID: taskID, Label: "Tasks", Branch: false, ParentID: pid,
			List: &ListSpec{Subject: SubjectTasks, ProjectID: p.ID},
		}
		t.nodes[fileID] = NavNode{
			ID: fileID, Label: "Files", Branch: false, ParentID: pid,
			List: &ListSpec{Subject: SubjectProjectFiles, ProjectID: p.ID},
		}
		t.children[pid] = []string{taskID, fileID}
	}
	t.rebuildVisible()
}
