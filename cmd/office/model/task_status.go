package model

// TaskLifecycle is the OnlyOffice project task status code.
type TaskLifecycle int

const (
	TaskLifecycleNotAccept      TaskLifecycle = 0
	TaskLifecycleOpen           TaskLifecycle = 1
	TaskLifecycleClosed         TaskLifecycle = 2
	TaskLifecycleDisabled       TaskLifecycle = 3
	TaskLifecycleUnclassified   TaskLifecycle = 4
	TaskLifecycleNotInMilestone TaskLifecycle = 5
)

var taskLifecycleCycle = []TaskLifecycle{
	TaskLifecycleOpen,
	TaskLifecycleClosed,
	TaskLifecycleNotAccept,
	TaskLifecycleDisabled,
	TaskLifecycleUnclassified,
	TaskLifecycleNotInMilestone,
}

// TaskStatusFromAny maps API status to a lifecycle value.
func TaskStatusFromAny(v any) TaskLifecycle {
	if v == nil {
		return TaskLifecycleOpen
	}
	if s, ok := v.(string); ok {
		switch taskStatusFromString(s) {
		case "Open":
			return TaskLifecycleOpen
		case "Closed":
			return TaskLifecycleClosed
		case "Not accepted":
			return TaskLifecycleNotAccept
		case "Disabled":
			return TaskLifecycleDisabled
		case "Unclassified":
			return TaskLifecycleUnclassified
		case "Not in milestone":
			return TaskLifecycleNotInMilestone
		}
	}
	switch n := intRawVal(map[string]any{"status": v}, "status"); n {
	case 0:
		return TaskLifecycleNotAccept
	case 2:
		return TaskLifecycleClosed
	case 3:
		return TaskLifecycleDisabled
	case 4:
		return TaskLifecycleUnclassified
	case 5:
		return TaskLifecycleNotInMilestone
	default:
		return TaskLifecycleOpen
	}
}

// Label returns a human-readable status name.
func (s TaskLifecycle) Label() string {
	return TaskStatusLabel(int(s))
}

// Next cycles task status for keyboard toggling.
func (s TaskLifecycle) Next() TaskLifecycle {
	return cycleTaskLifecycle(s, 1)
}

// Prev cycles task status backward.
func (s TaskLifecycle) Prev() TaskLifecycle {
	return cycleTaskLifecycle(s, -1)
}

func cycleTaskLifecycle(cur TaskLifecycle, delta int) TaskLifecycle {
	idx := 0
	for i, v := range taskLifecycleCycle {
		if v == cur {
			idx = i
			break
		}
	}
	n := len(taskLifecycleCycle)
	idx = (idx + delta%n + n) % n
	return taskLifecycleCycle[idx]
}
