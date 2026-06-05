package tui

import (
	"fmt"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global View Selectors
	switch key {
	case "1":
		m.CurrentView = DashboardView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return m, nil
	case "2":
		m.CurrentView = MonthView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return m, nil
	case "3":
		m.CurrentView = WeekView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return m, nil
	case "4":
		m.CurrentView = DayView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return m, nil
	case "5":
		m.CurrentView = AnalyticsView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		return m, nil
	case "ctrl+d":
		if m.CurrentView == DayView {
			if m.TodoShelfFocus {
				m.ShelfScrollOffset += 2
				shelfTasks := m.getTodoShelfTasks()
				if m.ShelfScrollOffset > len(shelfTasks)-3 {
					m.ShelfScrollOffset = len(shelfTasks) - 3
				}
				if m.ShelfScrollOffset < 0 {
					m.ShelfScrollOffset = 0
				}
			} else {
				m.TimelineHour = (m.TimelineHour + 2) % 24
			}
		} else {
			m.ScrollOffset += 2
		}
		return m, nil
	case "ctrl+u":
		if m.CurrentView == DayView {
			if m.TodoShelfFocus {
				m.ShelfScrollOffset -= 2
				if m.ShelfScrollOffset < 0 {
					m.ShelfScrollOffset = 0
				}
			} else {
				m.TimelineHour = (m.TimelineHour - 2 + 24) % 24
			}
		} else {
			m.ScrollOffset -= 2
			if m.ScrollOffset < 0 {
				m.ScrollOffset = 0
			}
		}
		return m, nil
	case "?":
		m.HelpOpen = true
		m.StatusMsg = "Help opened. Press Esc/Enter to exit."
		return m, nil
	case ":":
		m.CurrentMode = ModeCommand
		m.CommandInput.SetValue("")
		m.CommandInput.Focus()
		return m, nil
	case "i":
		m.CurrentMode = ModeForm
		m.Form = NewTaskForm()
		m.Form.TitleInput.Focus()
		return m, nil
	case "enter":
		// Open Detail panel of active task
		task, exists := m.getActiveTask()
		if exists {
			m.DetailTask = task
			m.DetailOpen = true
		}
		return m, nil
	case "x":
		// Complete Task
		task, exists := m.getActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			m.StatusMsg = fmt.Sprintf("Task '%s' completed!", task.Title)
		}
		return m, nil
	case "d":
		// Delete Task
		task, exists := m.getActiveTask()
		if exists {
			m.DB.DeleteTask(task.UUID)
			m.refreshTasks()
			m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", task.Title)
		}
		return m, nil
	case "z":
		task, exists := m.getActiveTask()
		if exists {
			m.startZenMode(task)
		} else {
			m.StatusMsg = "No active task selected to start Zen Mode."
		}
		return m, nil
	}

	// Navigation keys depending on active view
	switch m.CurrentView {
	case MonthView:
		m.handleMonthNav(key)
	case WeekView:
		m.handleWeekNav(key)
	case DayView:
		m.handleDayNav(key)
	}

	return m, nil
}

func (m *Model) handleMonthNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "j":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	case "k":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, -1, 0)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 1, 0)
	case "enter":
		m.CurrentView = DayView
	}
}

func (m *Model) handleWeekNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	}
}

func (m *Model) handleDayNav(key string) {
	switch key {
	case "h", "l", "tab":
		m.TodoShelfFocus = !m.TodoShelfFocus
	case "j":
		if m.TodoShelfFocus {
			m.moveTaskSelection(1)
		} else {
			m.TimelineHour = (m.TimelineHour + 1) % 24
		}
	case "k":
		if m.TodoShelfFocus {
			m.moveTaskSelection(-1)
		} else {
			m.TimelineHour = (m.TimelineHour - 1 + 24) % 24
		}
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	}
}

func (m *Model) getActiveTask() (model.Task, bool) {
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.getTodoShelfTasks()
			if len(shelf) > 0 {
				for _, t := range m.Tasks {
					if t.UUID == m.SelectedTaskUUID {
						return t, true
					}
				}
				return shelf[0], true
			}
		} else {
			for _, t := range m.Tasks {
				if t.SchedulingType == model.Anchored {
					startH := t.TimeWindow.Start.Hour()
					endH := t.TimeWindow.End.Hour()
					if t.TimeWindow.Start.Day() == m.SelectedDay.Day() &&
						t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
						t.TimeWindow.Start.Year() == m.SelectedDay.Year() &&
						m.TimelineHour >= startH && m.TimelineHour < endH {
						return t, true
					}
				}
			}
		}
	} else {
		var todayTasks []model.Task
		for _, t := range m.Tasks {
			if t.SchedulingType == model.Anchored &&
				t.TimeWindow.Start.Day() == m.SelectedDay.Day() &&
				t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
				t.TimeWindow.Start.Year() == m.SelectedDay.Year() {
				todayTasks = append(todayTasks, t)
			}
		}
		if len(todayTasks) > 0 {
			return todayTasks[0], true
		}
	}
	return model.Task{}, false
}

func (m *Model) getTodoShelfTasks() []model.Task {
	var shelf []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Floating && t.LifecycleState != model.StateCompleted {
			shelf = append(shelf, t)
		}
	}
	importSort(shelf)
	return shelf
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

func (m *Model) moveTaskSelection(dir int) {
	shelf := m.getTodoShelfTasks()
	if len(shelf) == 0 {
		return
	}

	idx := -1
	for i, t := range shelf {
		if t.UUID == m.SelectedTaskUUID {
			idx = i
			break
		}
	}

	if idx == -1 {
		m.SelectedTaskUUID = shelf[0].UUID
		return
	}

	idx += dir
	if idx < 0 {
		idx = len(shelf) - 1
	} else if idx >= len(shelf) {
		idx = 0
	}
	m.SelectedTaskUUID = shelf[idx].UUID
}
