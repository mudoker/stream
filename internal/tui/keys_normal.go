package tui

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
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
	case "6":
		m.CurrentView = SettingsView
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
		m.HelpScrollOffset = 0
		m.StatusMsg = "Help opened. Press Esc/? to exit."
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
	case "a":
		task, exists := m.getActiveTask()
		if exists {
			if task.SchedulingType == model.Anchored {
				// De-anchor
				task.SchedulingType = model.Floating
				task.LifecycleState = model.StateReady
				if m.DB != nil {
					m.DB.UpdateTask(task)
					m.refreshTasks()
				} else {
					m.updateTaskInMemory(task)
				}
				if m.Sync != nil {
					m.Sync.TriggerSync()
				}
				m.StatusMsg = fmt.Sprintf("Task '%s' de-anchored to backlog.", task.Title)
			} else {
				// Anchor: open start time prompt
				m.AnchorPromptTask = task
				m.AnchorTimeInput = textinput.New()
				now := time.Now()
				m.AnchorTimeInput.SetValue(now.Format("15:04"))
				m.AnchorTimeInput.Focus()

				m.AnchorDurationInput = textinput.New()
				defaultDur := task.StoryPoints * 45
				if defaultDur <= 0 {
					defaultDur = 60
				}
				m.AnchorDurationInput.SetValue(strconv.Itoa(defaultDur))

				m.AnchorActiveField = 0
				m.AnchorPromptOpen = true
				m.StatusMsg = "Enter start time and duration to anchor task."
			}
		}
		return m, nil
	case "e":
		task, exists := m.getActiveTask()
		if exists {
			m.startEditMode(task)
		}
		return m, nil
	case "enter":
		task, exists := m.getActiveTask()
		if exists {
			m.DetailTask = task
			m.DetailOpen = true
		}
		return m, nil
	case "v":
		m.enterTaskMoveMode()
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
		m.selectDefaultTaskForSelectedDay()
		m.TimelineHour = time.Now().Hour()
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
	case DashboardView, AnalyticsView:
		m.handleDashboardOrAnalyticsNav(key)
	}

	return m, nil
}

func (m *Model) handleDashboardOrAnalyticsNav(key string) {
	if m.CurrentView == DashboardView {
		switch key {
		case "h", "left":
			m.DashboardFocusCol = 0
		case "l", "right":
			m.DashboardFocusCol = 1
		case "k", "up":
			m.DashboardFocusRow--
			if m.DashboardFocusRow < 0 {
				m.DashboardFocusRow = 2
			}
		case "j", "down":
			m.DashboardFocusRow = (m.DashboardFocusRow + 1) % 3
		}

		availH := m.Height - 8
		if availH < 10 {
			availH = 10
		}

		rowHeights := []int{15, 15, 15}
		if availH > 45 {
			rowHeights = partitionHeights(availH, 3)
		}

		var yStart, yEnd int
		for i := 0; i < m.DashboardFocusRow; i++ {
			yStart += rowHeights[i]
		}
		yEnd = yStart + rowHeights[m.DashboardFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+availH {
			m.ScrollOffset = yEnd - availH
		}
	} else if m.CurrentView == AnalyticsView {
		totalLayers := 6
		switch key {
		case "h", "left":
			m.AnalyticsFocusCol = 0
		case "l", "right":
			m.AnalyticsFocusCol = 1
		case "k", "up":
			m.AnalyticsFocusRow--
			if m.AnalyticsFocusRow < 0 {
				m.AnalyticsFocusRow = totalLayers - 1
			}
		case "j", "down":
			m.AnalyticsFocusRow = (m.AnalyticsFocusRow + 1) % totalLayers
		}

		yStart := m.AnalyticsFocusRow * 13
		yEnd := (m.AnalyticsFocusRow + 1) * 13

		gridHeight := m.Height - 8
		if gridHeight < 10 {
			gridHeight = 10
		}

		rowHeights := make([]int, totalLayers)
		if gridHeight > totalLayers*13 {
			rowHeights = partitionHeights(gridHeight, totalLayers)
		} else {
			for i := range rowHeights {
				rowHeights[i] = 13
			}
		}

		var yStartAcc int
		for i := 0; i < m.AnalyticsFocusRow; i++ {
			yStartAcc += rowHeights[i]
		}
		yStart = yStartAcc
		yEnd = yStartAcc + rowHeights[m.AnalyticsFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+gridHeight {
			m.ScrollOffset = yEnd - gridHeight
		}
	}
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
		shelf := m.getTodoShelfTasks()
		if len(shelf) > 0 {
			m.SelectedTaskUUID = shelf[0].UUID
		} else {
			m.SelectedTaskUUID = ""
		}
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

	rects := m.BuildDayTaskRects(dayTasks)
	if len(rects) == 0 {
		return
	}

	var current TaskRect
	found := false
	for _, r := range rects {
		if r.Task.UUID == m.SelectedTaskUUID {
			current = r
			found = true
			break
		}
	}
	if !found {
		m.SelectedTaskUUID = dayTasks[0].UUID
		return
	}

	bestScore := 1_000_000
	bestUUID := ""
	var bestRect TaskRect
	for _, r := range rects {
		if r.Task.UUID == current.Task.UUID {
			continue
		}
		if dir > 0 {
			if r.Top <= current.CenterY {
				continue
			}
			dy := absInt(r.Top - current.CenterY)
			dx := absInt(r.CenterX - current.CenterX)
			score := dy + dx*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Left < bestRect.Left || (r.Left == bestRect.Left && r.Top < bestRect.Top))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		} else {
			if r.Bottom >= current.CenterY {
				continue
			}
			dy := absInt(current.CenterY - r.Bottom)
			dx := absInt(r.CenterX - current.CenterX)
			score := dy + dx*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Left < bestRect.Left || (r.Left == bestRect.Left && r.Top < bestRect.Top))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		}
	}

	if bestUUID != "" {
		m.SelectedTaskUUID = bestUUID
	}
}

func (m *Model) navigateHorizontal(dir int) {
	dayTasks := m.getDayTasks()
	if len(dayTasks) <= 1 {
		return
	}

	rects := m.BuildDayTaskRects(dayTasks)
	if len(rects) == 0 {
		return
	}

	var current TaskRect
	found := false
	for _, r := range rects {
		if r.Task.UUID == m.SelectedTaskUUID {
			current = r
			found = true
			break
		}
	}
	if !found {
		m.SelectedTaskUUID = dayTasks[0].UUID
		return
	}

	bestScore := 1_000_000
	bestUUID := ""
	var bestRect TaskRect
	for _, r := range rects {
		if r.Task.UUID == current.Task.UUID {
			continue
		}
		if dir > 0 {
			if r.Left <= current.CenterX {
				continue
			}
			dx := absInt(r.Left - current.CenterX)
			dy := absInt(r.CenterY - current.CenterY)
			score := dx + dy*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Top < bestRect.Top || (r.Top == bestRect.Top && r.Left < bestRect.Left))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		} else {
			if r.Right >= current.CenterX {
				continue
			}
			dx := absInt(current.CenterX - r.Right)
			dy := absInt(r.CenterY - current.CenterY)
			score := dx + dy*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Top < bestRect.Top || (r.Top == bestRect.Top && r.Left < bestRect.Left))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		}
	}

	if bestUUID != "" {
		m.SelectedTaskUUID = bestUUID
	}
}

func (m *Model) enterTaskMoveMode() {
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if task.SchedulingType != model.Anchored {
		m.StatusMsg = "Only anchored tasks can be moved with v."
		return
	}

	m.CurrentMode = ModeTaskMove
	m.TaskMovePrefix = ""
	m.TaskMoveOriginalTimeWindow = task.TimeWindow
	m.StatusMsg = fmt.Sprintf("Locked '%s'. Use j/k or count+j/k to move in 15m steps. Enter to confirm, Esc to cancel.", task.Title)
}

func (m *Model) handleTaskMoveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		m.TaskMovePrefix += key
		m.StatusMsg = fmt.Sprintf("Move count set to %s. Press j/k to apply.", m.TaskMovePrefix)
		return m, nil
	}

	switch key {
	case "j", "down":
		m.applyTaskMove(1)
	case "k", "up":
		m.applyTaskMove(-1)
	case "enter":
		m.confirmTaskMove()
	case "esc":
		m.cancelTaskMove()
	}

	return m, nil
}

func (m *Model) parseTaskMoveCount() int {
	count := 1
	if m.TaskMovePrefix != "" {
		if parsed, err := strconv.Atoi(m.TaskMovePrefix); err == nil && parsed > 0 {
			count = parsed
		}
	}
	return count
}

func (m *Model) applyTaskMove(direction int) {
	count := m.parseTaskMoveCount()
	steps := count * direction
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if task.SchedulingType != model.Anchored {
		m.StatusMsg = "Only anchored tasks can be moved with v."
		return
	}

	delta := time.Duration(steps*15) * time.Minute
	task.TimeWindow.Start = task.TimeWindow.Start.Add(delta)
	task.TimeWindow.End = task.TimeWindow.End.Add(delta)
	m.updateTaskInMemory(task)
	m.TaskMovePrefix = ""
	moveDir := "down"
	if direction < 0 {
		moveDir = "up"
	}
	m.StatusMsg = fmt.Sprintf("Moved '%s' %d minutes %s. Enter to confirm, Esc to cancel.", task.Title, absInt(steps*15), moveDir)
}

func (m *Model) confirmTaskMove() {
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		m.CurrentMode = ModeNormal
		return
	}
	if m.DB != nil {
		m.DB.UpdateTask(task)
		m.refreshTasks()
	}
	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = fmt.Sprintf("Task '%s' moved to %s.", task.Title, task.TimeWindow.Start.Format("15:04"))
}

func (m *Model) cancelTaskMove() {
	if m.SelectedTaskUUID != "" {
		for i, t := range m.Tasks {
			if t.UUID == m.SelectedTaskUUID {
				t.TimeWindow = m.TaskMoveOriginalTimeWindow
				m.Tasks[i] = t
				break
			}
		}
	}
	if m.DB != nil {
		m.refreshTasks()
	}
	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = "Task move canceled."
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

