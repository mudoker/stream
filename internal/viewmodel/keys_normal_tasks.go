package viewmodel

import (
	"sort"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/tasks"
)

func (m *Model) GetActiveTask() (model.Task, bool) {
	if m.CurrentMode == ModeTaskMove {
		for _, t := range m.Tasks {
			if t.UUID == m.SelectedTaskUUID {
				return t, true
			}
		}
	}
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.GetTodoShelfTasks()
			if len(shelf) > 0 {
				for _, t := range shelf {
					if t.UUID == m.SelectedTaskUUID {
						return t, true
					}
				}
				m.SelectedTaskUUID = shelf[0].UUID
				return shelf[0], true
			}
		} else {
			dayTasks := m.GetDayTasks()
			if len(dayTasks) > 0 {
				for _, t := range dayTasks {
					if t.UUID == m.SelectedTaskUUID {
						return t, true
					}
				}
				m.SelectedTaskUUID = dayTasks[0].UUID
				return dayTasks[0], true
			}
		}
	} else {
		if m.SelectedTaskUUID != "" {
			for _, t := range m.Tasks {
				if t.UUID == m.SelectedTaskUUID {
					return t, true
				}
			}
		}
		dayTasks := m.GetDayTasks()
		if len(dayTasks) > 0 {
			m.SelectedTaskUUID = dayTasks[0].UUID
			return dayTasks[0], true
		}
	}
	return model.Task{}, false
}

func (m *Model) GetTodoShelfTasks() []model.Task {
	var wsTasks []model.Task
	for _, t := range m.Tasks {
		if m.ActiveWorkspaceUUID == "ALL_WORKSPACES" || t.WorkspaceUUID == m.ActiveWorkspaceUUID {
			wsTasks = append(wsTasks, t)
		}
	}
	return tasks.GetTodoShelfTasks(wsTasks, m.SelectedDay)
}

func (m *Model) GetAllActiveTasks() []model.Task {
	var matching []model.Task
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.GetTodoShelfTasks()
			if len(shelf) > 0 {
				for _, t := range m.Tasks {
					if t.UUID == m.SelectedTaskUUID {
						return []model.Task{t}
					}
				}
				return []model.Task{shelf[0]}
			}
		} else {
			return m.GetDayTasks()
		}
	} else {
		return m.GetDayTasks()
	}
	return matching
}

func (m *Model) GetDayTasks() []model.Task {
	return tasks.GetDayTasks(m.Tasks, m.SelectedDay)
}

func (m *Model) updateTaskInMemory(updated model.Task) {
	for i, t := range m.Tasks {
		if t.UUID == updated.UUID {
			m.Tasks[i] = updated
			return
		}
	}
}

func (m *Model) focusAnchorPromptFields() {
	m.AnchorTimeInput.Blur()
	m.AnchorDurationInput.Blur()

	if m.AnchorActiveField == 0 {
		m.AnchorTimeInput.Focus()
	} else {
		m.AnchorDurationInput.Focus()
	}
}

func (m *Model) GetAgendaTasks() []model.Task {
	today := time.Now()
	var agendaTasks []model.Task
	for _, t := range m.Tasks {
		if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
			continue
		}
		isTodayOrUpcoming := false
		if model.IsTaskAnchored(t) {
			isTodayOrUpcoming = t.TimeWindow.Start.Year() == today.Year() &&
				t.TimeWindow.Start.Month() == today.Month() &&
				t.TimeWindow.Start.Day() == today.Day() || t.TimeWindow.Start.After(today)
		} else {
			isTodayOrUpcoming = t.CreatedAt.Year() == today.Year() &&
				t.CreatedAt.Month() == today.Month() &&
				t.CreatedAt.Day() == today.Day() && t.LifecycleState != model.StateCompleted
		}
		if isTodayOrUpcoming {
			agendaTasks = append(agendaTasks, t)
		}
	}
	sort.Slice(agendaTasks, func(i, j int) bool {
		isScheduledI := model.IsTaskAnchored(agendaTasks[i])
		isScheduledJ := model.IsTaskAnchored(agendaTasks[j])
		if isScheduledI && isScheduledJ {
			return agendaTasks[i].TimeWindow.Start.Before(agendaTasks[j].TimeWindow.Start)
		}
		if isScheduledI {
			return true
		}
		if isScheduledJ {
			return false
		}
		return agendaTasks[i].CreatedAt.Before(agendaTasks[j].CreatedAt)
	})
	return agendaTasks
}

func (m *Model) GetRecommendedCapacity() int {
	return 15
}

func (m *Model) GetUpcomingTask() (model.Task, bool) {
	now := time.Now()
	var candidates []model.Task
	for _, t := range m.Tasks {
		if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
			continue
		}
		if model.IsTaskAnchored(t) && t.LifecycleState != model.StateCompleted {
			if t.TimeWindow.Start.After(now) {
				candidates = append(candidates, t)
			}
		}
	}

	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].TimeWindow.Start.Before(candidates[j].TimeWindow.Start)
		})
		return candidates[0], true
	}

	shelf := m.GetTodoShelfTasks()
	if len(shelf) > 0 {
		activeTask, activeExists := m.GetActiveTask()
		for _, t := range shelf {
			if activeExists && t.UUID == activeTask.UUID {
				continue
			}
			return t, true
		}
	}

	return model.Task{}, false
}

