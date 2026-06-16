package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/jazzlounge"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleGlobalActions(key string) (bool, tea.Cmd) {
	switch key {
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
		return true, nil
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
		return true, nil
	case "i":
		m.CurrentMode = ModeForm
		m.Form = NewTaskForm()
		m.Form.TitleInput.Focus()
		return true, nil
	case "a":
		task, exists := m.GetActiveTask()
		if exists {
			if model.IsTaskAnchored(task) {
				m.ConfirmTask = task
				m.ConfirmOpen = true
				m.ConfirmActionType = "deanchor"
				return true, nil
			} else {
				// Anchor: open start time prompt
				m.AnchorPromptTask = task
				m.AnchorTimeInput = textinput.New()
				now := time.Now()
				m.AnchorTimeInput.SetValue(now.Format("15:04"))
				m.AnchorTimeInput.Focus()

				m.AnchorDurationInput = textinput.New()
				defaultDur := task.EstimatedDurationMins
				if defaultDur <= 0 {
					defaultDur = 60
				}
				m.AnchorDurationInput.SetValue(strconv.Itoa(defaultDur))

				m.AnchorActiveField = 0
				m.AnchorPromptOpen = true
				m.StatusMsg = "Enter start time and duration to anchor task."
			}
		}
		return true, nil
	case "e":
		task, exists := m.GetActiveTask()
		if exists {
			m.startEditMode(task)
		}
		return true, nil
	case "enter":
		if m.CurrentView == DashboardView || m.CurrentView == AnalyticsView {
			return true, nil
		}
		task, exists := m.GetActiveTask()
		if exists {
			m.DetailTask = task
			m.DetailOpen = true
		}
		return true, nil
	case "v":
		m.EnterTaskMoveMode()
		return true, nil
	case "V":
		m.EnterTaskDurationAdjustMode()
		return true, nil
	case "x":
		// Complete Task
		task, exists := m.GetActiveTask()
		if exists {
			if task.SchedulingType == model.Habit {
				if isFutureDay(m.SelectedDay) {
					m.WarningMsg = "You cannot mark a habit as completed for future days!"
					m.WarningOpen = true
					return true, nil
				}

				dateStr := m.SelectedDay.Format("2006-01-02")
				foundIdx := -1
				for idx, d := range task.CompletedDates {
					if d == dateStr {
						foundIdx = idx
						break
					}
				}

				if foundIdx >= 0 {
					// Toggle off: remove from CompletedDates
					task.CompletedDates = append(task.CompletedDates[:foundIdx], task.CompletedDates[foundIdx+1:]...)
					m.StatusMsg = fmt.Sprintf("Habit '%s' marked incomplete for %s.", task.Title, dateStr)
					if len(task.CompletedDates) == 0 {
						task.LifecycleState = model.StateBacklog
					}
				} else {
					// Toggle on: add to CompletedDates
					task.CompletedDates = append(task.CompletedDates, dateStr)
					task.LifecycleState = model.StateCompleted
					m.StatusMsg = fmt.Sprintf("Habit '%s' completed for %s!", task.Title, dateStr)
				}
			} else {
				if task.LifecycleState == model.StateCompleted {
					task.LifecycleState = model.StateBacklog
					m.StatusMsg = fmt.Sprintf("Task '%s' marked incomplete.", task.Title)
				} else {
					if task.SchedulingType == model.Reminder {
						m.ConfirmTask = task
						m.ConfirmOpen = true
						m.ConfirmActionType = "complete_reminder"
						return true, nil
					}
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
		return true, nil
	case "d":
		// Delete Task
		task, exists := m.GetActiveTask()
		if exists {
			m.ConfirmTask = task
			m.ConfirmOpen = true
			if task.RecurringParentUUID != "" {
				m.ConfirmActionType = "delete_recurring"
				m.ConfirmSelectedIndex = 0
			} else {
				m.ConfirmActionType = "delete"
			}
		}
		return true, nil
	case "t":
		m.SelectedDay = time.Now()
		m.selectDefaultTaskForSelectedDay()
		m.TimelineHour = time.Now().Hour()
		m.ScrollOffset = 0
		m.StatusMsg = "Jumped to today."
		return true, nil
	case "M":
		jazzlounge.GetJazzLoungeEngine().SetPlaying(true)
		m.StatusMsg = "🔊 Jazz Lounge Engine started/resumed"
		return true, nil
	case "m":
		jazzlounge.GetJazzLoungeEngine().SetPlaying(false)
		m.StatusMsg = "🔇 Jazz Lounge Engine stopped"
		return true, nil
	case "z":
		task, exists := m.GetActiveTask()
		if exists {
			if task.SchedulingType == model.Event {
				m.StatusMsg = "Focus sessions are disabled for events."
				return true, nil
			}
			if m.ZenTimer != nil && m.ZenTimer.Task.UUID == task.UUID {
				m.CurrentMode = ModeZen
				m.StatusMsg = "Returned to active Zen focus session."
				return true, nil
			}
			if m.ZenTimer != nil {
				m.ZenTimer.RecordElapsedTimes()
				t := m.ZenTimer.Task
				t.LifecycleState = model.StateReady
				if m.DB != nil {
					m.DB.UpdateTask(t)
				}
			}
			m.StartZenMode(task)
		} else {
			m.StatusMsg = "No active task selected to start Zen Mode."
		}
		return true, nil
	}
	return false, nil
}
