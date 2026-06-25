package model

// ProjectLifecycle is the OnlyOffice project status string accepted by the API.
type ProjectLifecycle string

const (
	ProjectLifecycleOpen   ProjectLifecycle = "open"
	ProjectLifecyclePaused ProjectLifecycle = "paused"
	ProjectLifecycleClosed ProjectLifecycle = "closed"
)

// ProjectStatusFromAny maps API status (int or string) to a lifecycle value.
func ProjectStatusFromAny(v any) ProjectLifecycle {
	switch x := v.(type) {
	case string:
		switch ProjectLifecycle(x) {
		case ProjectLifecycleOpen, ProjectLifecyclePaused, ProjectLifecycleClosed:
			return ProjectLifecycle(x)
		}
	case float64:
		return projectStatusFromInt(int(x))
	case int:
		return projectStatusFromInt(x)
	case int64:
		return projectStatusFromInt(int(x))
	}
	return ProjectLifecycleOpen
}

func projectStatusFromInt(n int) ProjectLifecycle {
	switch n {
	case 1:
		return ProjectLifecyclePaused
	case 2:
		return ProjectLifecycleClosed
	default:
		return ProjectLifecycleOpen
	}
}

// ProjectStatusLabel returns a short UI label for list cells.
func ProjectStatusLabel(raw map[string]any) string {
	if raw == nil {
		return "Open"
	}
	return ProjectStatusFromAny(raw["status"]).Label()
}

// ProjectStatusIcon returns the list-cell emoji for a project lifecycle state.
func (s ProjectLifecycle) Icon() string {
	switch s {
	case ProjectLifecycleClosed:
		return "🔴"
	case ProjectLifecyclePaused:
		return "🟡"
	default:
		return "🟢"
	}
}

// ProjectStatusCell returns emoji and label on one line for project table cells.
func ProjectStatusCell(raw map[string]any) string {
	if raw == nil {
		return ProjectLifecycleOpen.Icon() + " Open"
	}
	s := ProjectStatusFromAny(raw["status"])
	return s.Icon() + " " + s.Label()
}

// ProjectIsOpen is true when the project is not closed.
func ProjectIsOpen(raw map[string]any) bool {
	if raw == nil {
		return true
	}
	return ProjectStatusFromAny(raw["status"]) != ProjectLifecycleClosed
}

// Label returns a human-readable status name.
func (s ProjectLifecycle) Label() string {
	switch s {
	case ProjectLifecycleClosed:
		return "Closed"
	case ProjectLifecyclePaused:
		return "Paused"
	default:
		return "Open"
	}
}

// ToggleOpenClosed flips between open and closed (paused maps to closed on toggle).
func (s ProjectLifecycle) ToggleOpenClosed() ProjectLifecycle {
	if s == ProjectLifecycleClosed {
		return ProjectLifecycleOpen
	}
	return ProjectLifecycleClosed
}

// Next cycles open → closed → open for keyboard toggling.
func (s ProjectLifecycle) Next() ProjectLifecycle {
	return s.ToggleOpenClosed()
}

// Prev cycles closed → open → closed.
func (s ProjectLifecycle) Prev() ProjectLifecycle {
	return s.ToggleOpenClosed()
}
