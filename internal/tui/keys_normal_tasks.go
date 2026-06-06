package tui

import (
	"stream/internal/model"
)

func (m *Model) getActiveTask() (model.Task, bool) {
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.getTodoShelfTasks()
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
			dayTasks := m.getDayTasks()
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
		dayTasks := m.getDayTasks()
		if len(dayTasks) > 0 {
			m.SelectedTaskUUID = dayTasks[0].UUID
			return dayTasks[0], true
		}
	}
	return model.Task{}, false
}

func (m *Model) getTodoShelfTasks() []model.Task {
	var reminders []model.Task
	var backlog []model.Task
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			continue
		}
		if t.SchedulingType == model.Reminder {
			reminders = append(reminders, t)
		} else if t.SchedulingType == model.Floating {
			backlog = append(backlog, t)
		}
	}
	sortReminders(reminders)
	importSort(backlog)
	return append(reminders, backlog...)
}

func sortReminders(tasks []model.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].TimeWindow.Start.Before(tasks[i].TimeWindow.Start) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if tasks[j].TimeWindow.Start.Equal(tasks[i].TimeWindow.Start) {
				if tasks[j].SortingWeight() > tasks[i].SortingWeight() {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

func importSort(tasks []model.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].SortingWeight() > tasks[i].SortingWeight() {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func (m *Model) getAllActiveTasks() []model.Task {
	var matching []model.Task
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.getTodoShelfTasks()
			if len(shelf) > 0 {
				for _, t := range m.Tasks {
					if t.UUID == m.SelectedTaskUUID {
						return []model.Task{t}
					}
				}
				return []model.Task{shelf[0]}
			}
		} else {
			return m.getDayTasks()
		}
	} else {
		return m.getDayTasks()
	}
	return matching
}

func (m *Model) getDayTasks() []model.Task {
	var anchored []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, m.SelectedDay) {
			anchored = append(anchored, t)
		}
	}
	importSort(anchored)
	return anchored
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
