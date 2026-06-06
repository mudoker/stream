package tui

import (
	"fmt"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.SidebarFocus {
		switch key {
		case "j":
			m.moveSidebarView(1)
			return m, nil
		case "k":
			m.moveSidebarView(-1)
			return m, nil
		case "tab":
			m.cycleFocus()
			return m, nil
		}
	}

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
	case "tab":
		m.cycleFocus()
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
		} else if m.CurrentView == MonthView {
			// Scroll forward by colsFit months
			workspaceWidth := m.Layout.WorkspaceW - 4
			colWidth := workspaceWidth / 2
			innerW := colWidth - 6
			colsFit := innerW / 29
			if colsFit < 1 {
				colsFit = 1
			}
			m.ScrollOffset += colsFit
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
		} else if m.CurrentView == MonthView {
			// Scroll backward by colsFit months (indefinitely back)
			workspaceWidth := m.Layout.WorkspaceW - 4
			colWidth := workspaceWidth / 2
			innerW := colWidth - 6
			colsFit := innerW / 29
			if colsFit < 1 {
				colsFit = 1
			}
			m.ScrollOffset -= colsFit
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
	case "w":
		if len(m.Workspaces) > 1 {
			idx := -1
			for i, ws := range m.Workspaces {
				if ws.UUID == m.ActiveWorkspaceUUID {
					idx = i
					break
				}
			}
			if idx != -1 {
				nextIdx := (idx + 1) % len(m.Workspaces)
				m.ActiveWorkspaceUUID = m.Workspaces[nextIdx].UUID
				m.refreshTasks()
				m.selectDefaultTaskForSelectedDay()
				m.StatusMsg = fmt.Sprintf("Switched to workspace '%s'.", m.Workspaces[nextIdx].Name)
			}
		}
		return m, nil
	case "W":
		if len(m.Workspaces) > 1 {
			idx := -1
			for i, ws := range m.Workspaces {
				if ws.UUID == m.ActiveWorkspaceUUID {
					idx = i
					break
				}
			}
			if idx != -1 {
				prevIdx := (idx - 1 + len(m.Workspaces)) % len(m.Workspaces)
				m.ActiveWorkspaceUUID = m.Workspaces[prevIdx].UUID
				m.refreshTasks()
				m.selectDefaultTaskForSelectedDay()
				m.StatusMsg = fmt.Sprintf("Switched to workspace '%s'.", m.Workspaces[prevIdx].Name)
			}
		}
		return m, nil
	case "i":
		m.CurrentMode = ModeForm
		m.Form = NewTaskForm()
		m.Form.TitleInput.Focus()
		return m, nil
	case "enter":
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
			m.ConfirmTask = task
			m.ConfirmOpen = true
		}
		return m, nil
	case "t":
		m.SelectedDay = time.Now()
		m.ScrollOffset = 0
		m.StatusMsg = "Jumped to today."
		return m, nil
	case "z":
		if m.ZenTimer != nil && m.ZenTimer.Running {
			m.CurrentMode = ModeZen
			m.StatusMsg = "Returned to active Zen focus session."
			return m, nil
		}
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
	case "h":
		if !m.TodoShelfFocus {
			m.navigateHorizontal(-1)
		}
	case "l":
		if !m.TodoShelfFocus {
			m.navigateHorizontal(1)
		}
	case "j":
		if m.TodoShelfFocus {
			m.moveTaskSelection(1)
		} else {
			m.navigateVertical(1)
			m.autoScrollToSelectedTask()
		}
	case "k":
		if m.TodoShelfFocus {
			m.moveTaskSelection(-1)
		} else {
			m.navigateVertical(-1)
			m.autoScrollToSelectedTask()
		}
	case "J":
		if !m.TodoShelfFocus {
			m.TimelineHour = (m.TimelineHour + 1) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "K":
		if !m.TodoShelfFocus {
			m.TimelineHour = (m.TimelineHour - 1 + 24) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
		m.selectDefaultTaskForSelectedDay()
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
		m.selectDefaultTaskForSelectedDay()
	}
}

func (m *Model) getActiveTask() (model.Task, bool) {
	if m.SelectedTaskUUID != "" {
		for _, t := range m.Tasks {
			if t.UUID == m.SelectedTaskUUID {
				return t, true
			}
		}
	}
	if m.CurrentView == DayView {
		if m.TodoShelfFocus {
			shelf := m.getTodoShelfTasks()
			if len(shelf) > 0 {
				return shelf[0], true
			}
		} else {
			dayTasks := m.getDayTasks()
			if len(dayTasks) > 0 {
				return dayTasks[0], true
			}
		}
	} else {
		dayTasks := m.getDayTasks()
		if len(dayTasks) > 0 {
			return dayTasks[0], true
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
	// Sort by start time
	importSort(anchored)
	return anchored
}

func (m *Model) selectDefaultTaskForSelectedDay() {
	if m.TodoShelfFocus {
		return
	}
	dayTasks := m.getDayTasks()
	if len(dayTasks) > 0 {
		m.SelectedTaskUUID = dayTasks[0].UUID
		m.TimelineHour = dayTasks[0].TimeWindow.Start.Hour()
	} else {
		m.SelectedTaskUUID = ""
	}
}

func (m *Model) selectFirstTaskInCurrentHour() {
	dayTasks := m.getDayTasks()
	for _, t := range dayTasks {
		if t.TimeWindow.Start.Hour() == m.TimelineHour {
			m.SelectedTaskUUID = t.UUID
			return
		}
	}
	for _, t := range dayTasks {
		if m.TimelineHour >= t.TimeWindow.Start.Hour() && m.TimelineHour < t.TimeWindow.End.Hour() {
			m.SelectedTaskUUID = t.UUID
			return
		}
	}
}

func (m *Model) autoScrollToSelectedTask() {
	if m.SelectedTaskUUID == "" {
		return
	}
	for _, t := range m.Tasks {
		if t.UUID == m.SelectedTaskUUID && t.SchedulingType == model.Anchored {
			m.TimelineHour = t.TimeWindow.Start.Hour()
			break
		}
	}
}

func (m *Model) navigateVertical(dir int) {
	dayTasks := m.getDayTasks()
	if len(dayTasks) == 0 {
		return
	}

	idx := -1
	for i, t := range dayTasks {
		if t.UUID == m.SelectedTaskUUID {
			idx = i
			break
		}
	}

	if idx == -1 {
		if dir > 0 {
			m.SelectedTaskUUID = dayTasks[0].UUID
		} else {
			m.SelectedTaskUUID = dayTasks[len(dayTasks)-1].UUID
		}
		return
	}

	idx += dir
	if idx < 0 {
		idx = len(dayTasks) - 1
	} else if idx >= len(dayTasks) {
		idx = 0
	}
	m.SelectedTaskUUID = dayTasks[idx].UUID
}

func (m *Model) navigateHorizontal(dir int) {
	dayTasks := m.getDayTasks()
	if len(dayTasks) <= 1 {
		return
	}

	var currentTask model.Task
	found := false
	for _, t := range dayTasks {
		if t.UUID == m.SelectedTaskUUID {
			currentTask = t
			found = true
			break
		}
	}
	if !found {
		m.SelectedTaskUUID = dayTasks[0].UUID
		return
	}

	resolved := ResolveOverlaps(dayTasks)
	var currentCol int
	for _, rc := range resolved {
		if rc.Task.UUID == currentTask.UUID {
			currentCol = rc.ColIndex
			break
		}
	}

	var candidates []ScheduledColumn
	for _, rc := range resolved {
		if rc.Task.UUID == currentTask.UUID {
			continue
		}
		overlap := currentTask.TimeWindow.Start.Before(rc.Task.TimeWindow.End) &&
			rc.Task.TimeWindow.Start.Before(currentTask.TimeWindow.End)
		if overlap {
			candidates = append(candidates, rc)
		}
	}

	if len(candidates) == 0 {
		return
	}

	var targetUUID string
	if dir > 0 {
		bestCol := 999
		for _, c := range candidates {
			if c.ColIndex > currentCol && c.ColIndex < bestCol {
				bestCol = c.ColIndex
				targetUUID = c.Task.UUID
			}
		}
		if targetUUID == "" {
			bestCol = 999
			for _, c := range candidates {
				if c.ColIndex < bestCol {
					bestCol = c.ColIndex
					targetUUID = c.Task.UUID
				}
			}
		}
	} else {
		bestCol := -1
		for _, c := range candidates {
			if c.ColIndex < currentCol && c.ColIndex > bestCol {
				bestCol = c.ColIndex
				targetUUID = c.Task.UUID
			}
		}
		if targetUUID == "" {
			bestCol = -1
			for _, c := range candidates {
				if c.ColIndex > bestCol {
					bestCol = c.ColIndex
					targetUUID = c.Task.UUID
				}
			}
		}
	}

	if targetUUID != "" {
		m.SelectedTaskUUID = targetUUID
	}
}
