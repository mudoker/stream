package viewmodel

// AdjustSelectionBeforeDeletion updates SelectedTaskUUID and LastTodoShelfTaskUUID
// to point to the next logical sibling task (or previous, or empty if none left)
// before the target task is deleted.
func (m *Model) AdjustSelectionBeforeDeletion(uuid string) {
	if uuid == "" || m.SelectedTaskUUID != uuid {
		return
	}

	// 1. Check if the task is on the todo shelf
	shelf := m.GetTodoShelfTasks()
	idx := -1
	for i, t := range shelf {
		if t.UUID == uuid {
			idx = i
			break
		}
	}
	if idx != -1 {
		nextUUID := ""
		if len(shelf) > 1 {
			if idx < len(shelf)-1 {
				nextUUID = shelf[idx+1].UUID
			} else {
				nextUUID = shelf[idx-1].UUID
			}
		}
		m.SelectedTaskUUID = nextUUID
		m.LastTodoShelfTaskUUID = nextUUID
		return
	}

	// 2. Check if the task is in the day tasks (timeline)
	dayTasks := m.GetDayTasks()
	idx = -1
	for i, t := range dayTasks {
		if t.UUID == uuid {
			idx = i
			break
		}
	}
	if idx != -1 {
		nextUUID := ""
		if len(dayTasks) > 1 {
			if idx < len(dayTasks)-1 {
				nextUUID = dayTasks[idx+1].UUID
			} else {
				nextUUID = dayTasks[idx-1].UUID
			}
		}
		m.SelectedTaskUUID = nextUUID
		if nextUUID != "" {
			for _, t := range dayTasks {
				if t.UUID == nextUUID {
					m.TimelineHour = t.TimeWindow.Start.Hour()
					break
				}
			}
		}
		return
	}

	// 3. Fallback: general tasks list
	idx = -1
	for i, t := range m.Tasks {
		if t.UUID == uuid {
			idx = i
			break
		}
	}
	if idx != -1 {
		nextUUID := ""
		if len(m.Tasks) > 1 {
			if idx < len(m.Tasks)-1 {
				nextUUID = m.Tasks[idx+1].UUID
			} else {
				nextUUID = m.Tasks[idx-1].UUID
			}
		}
		m.SelectedTaskUUID = nextUUID
	} else {
		m.SelectedTaskUUID = ""
	}
}
