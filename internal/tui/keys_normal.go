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
			if task.SchedulingType == model.Habit {
				isCompletedToday := task.LifecycleState == model.StateCompleted && sameDay(task.UpdatedAt, m.SelectedDay)
				if isCompletedToday {
					task.LifecycleState = model.StateBacklog
					m.StatusMsg = fmt.Sprintf("Habit '%s' marked incomplete.", task.Title)
				} else {
					task.LifecycleState = model.StateCompleted
					m.StatusMsg = fmt.Sprintf("Habit '%s' completed!", task.Title)
				}
			} else {
				if task.LifecycleState == model.StateCompleted {
					task.LifecycleState = model.StateBacklog
					m.StatusMsg = fmt.Sprintf("Task '%s' marked incomplete.", task.Title)
				} else {
					task.LifecycleState = model.StateCompleted
					m.StatusMsg = fmt.Sprintf("Task '%s' completed!", task.Title)
				}
			}
			task.UpdatedAt = time.Now()
			m.DB.UpdateTask(task)
			m.refreshTasks()
			if task.LifecycleState == model.StateCompleted {
				if m.ZenTimer != nil && m.ZenTimer.Task.UUID == task.UUID {
					m.ZenTimer = nil
				}
			}
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
		task, exists := m.getActiveTask()
		if exists {
			if m.ZenTimer != nil && m.ZenTimer.Task.UUID == task.UUID {
				m.CurrentMode = ModeZen
				m.StatusMsg = "Returned to active Zen focus session."
				return m, nil
			}
			if m.ZenTimer != nil {
				m.ZenTimer.RecordElapsedTimes()
				t := m.ZenTimer.Task
				t.LifecycleState = model.StateReady
				if m.DB != nil {
					m.DB.UpdateTask(t)
				}
			}
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
