package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/common"
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
		m.Form = NewTaskFormWithDate(m.SelectedDay)
		if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			m.Form.TaskTypeIdx = 0 // Feature
			m.Form.StatusIdx = m.SprintSwimlaneIdx
		} else {
			m.Form.TaskTypeIdx = 3 // Task
		}
		m.PopulateFormAvailableFeaturesAndBlockers()
		m.Form.TitleInput.Focus()
		return true, nil
	case "p":
		task, exists := m.GetActiveTask()
		if exists {
			isFeatureLevel := task.WorkItemType == model.WorkItemFeature ||
				task.WorkItemType == model.WorkItemDefect ||
				task.WorkItemType == model.WorkItemImprovement ||
				(m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus) ||
				task.SprintUUID != ""
			if isFeatureLevel {
				m.ConfirmTask = task
				m.ConfirmOpen = true
				m.ConfirmActionType = "create_task_from_feature"
				m.ConfirmSelectedIndex = 0
				m.ConfirmFocusArea = 0
				return true, nil
			} else {
				m.StatusMsg = "Select a feature or defect to create a corresponding task."
				return true, nil
			}
		} else {
			m.StatusMsg = "No feature or defect selected."
			return true, nil
		}
	case "I":
		if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			m.CurrentMode = ModeSprintForm
			defaultName := fmt.Sprintf("Sprint %d", len(m.Sprints)+1)
			m.SprintForm = NewSprintForm(defaultName)
			m.SprintForm.NameInput.Focus()
			return true, nil
		}
		return false, nil
	case "E":
		if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			activeSprint, ok := m.GetActiveSprint()
			if ok {
				m.CurrentMode = ModeSprintForm
				m.SprintForm = NewSprintFormFromSprint(activeSprint)
				m.SprintForm.NameInput.Focus()
				return true, nil
			}
		}
		return false, nil
	case "D":
		if m.TodoShelfFocus {
			sec, ok := m.GetActiveShelfSection()
			if ok && len(sec.Tasks) > 0 {
				m.InitiateClearShelfSection(sec)
				return true, nil
			} else if ok {
				m.StatusMsg = fmt.Sprintf("No tasks to clear in %s.", sec.Type)
				return true, nil
			} else {
				m.StatusMsg = "No active shelf section to clear."
				return true, nil
			}
		} else if m.CurrentView == SprintView && !m.SidebarFocus {
			activeSprint, ok := m.GetActiveSprint()
			if ok {
				m.InitiateDeleteSprint(activeSprint)
				return true, nil
			}
		}
		return false, nil
	case "g":
		if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			activeSprint, ok := m.GetActiveSprint()
			if ok {
				m.CurrentMode = ModeSprintForm
				m.SprintForm = NewSprintFormFromSprint(activeSprint)
				m.SprintForm.ActiveField = 3
				m.SprintForm.RecurringCountInput.SetValue("4")
				m.SprintForm.RecurringCountInput.Focus()
				return true, nil
			}
		}
		return false, nil
	case "a":
		task, exists := m.GetActiveTask()
		if exists {
			if m.CurrentView == SprintView {
				activeSprint, ok := m.GetActiveSprint()
				if !ok {
					m.StatusMsg = "No active sprint found."
					return true, nil
				}
				if m.TodoShelfFocus {
					if task.SchedulingType == model.Habit || task.SchedulingType == model.Event || task.SchedulingType == model.Reminder {
						m.StatusMsg = "Habits and events are not tasks and cannot be anchored to a sprint."
						return true, nil
					}
					// Anchor task to active sprint
					task.SprintUUID = activeSprint.UUID
					if task.LifecycleState == "" || task.LifecycleState == model.StateCompleted {
						task.LifecycleState = model.StateBacklog
					}
					task.UpdatedAt = time.Now()
					m.DB.UpdateTask(task)
					m.refreshTasks()
					m.StatusMsg = fmt.Sprintf("Task '%s' anchored to sprint '%s'.", task.Title, activeSprint.Name)
					return true, nil
				} else {
					// Deanchor task from active sprint
					task.SprintUUID = ""
					task.UpdatedAt = time.Now()
					m.DB.UpdateTask(task)
					m.refreshTasks()
					m.StatusMsg = fmt.Sprintf("Task '%s' deanchored back to global backlog.", task.Title)
					return true, nil
				}
			}

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
					defaultDur = task.StoryPoints * 45
					if defaultDur <= 0 {
						defaultDur = 60
					}
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
		} else if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			activeSprint, ok := m.GetActiveSprint()
			if ok {
				m.CurrentMode = ModeSprintForm
				m.SprintForm = NewSprintFormFromSprint(activeSprint)
				m.SprintForm.NameInput.Focus()
			}
		}
		return true, nil
	case "enter":
		if m.CurrentView == MonthView {
			return false, nil
		}
		if m.CurrentView == DashboardView || m.CurrentView == AnalyticsView {
			return true, nil
		}
		task, exists := m.GetActiveTask()
		if exists {
			m.DetailTask = task
			m.DetailOpen = true
		}
		return true, nil
	case "y":
		if m.CurrentView == SprintView && !m.SidebarFocus && !m.TodoShelfFocus {
			m.EnterSprintTaskMoveMode()
			return true, nil
		}
		m.EnterTaskMoveMode()
		return true, nil
	case "Y":
		m.EnterTaskCloneMoveMode()
		return true, nil
	case "v":
		m.EnterTaskDurationAdjustMode(true)
		return true, nil
	case "V":
		m.EnterTaskDurationAdjustMode(false)
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
			}
			common.ToggleTaskCompletion(m, task, m.SelectedDay)
		}
		return true, nil
	case "d":
		// Delete Task
		task, exists := m.GetActiveTask()
		if exists {
			common.InitiateDeleteTask(m, task)
		}
		return true, nil
	case "t":
		if m.CurrentView == SprintView {
			task, exists := m.GetActiveTask()
			if exists {
				task.AddedToToday = !task.AddedToToday
				task.UpdatedAt = time.Now()
				m.DB.UpdateTask(task)
				m.refreshTasks()
				if task.AddedToToday {
					m.StatusMsg = fmt.Sprintf("Added '%s' to today's task list (will appear on Day timeline backlog).", task.Title)
				} else {
					m.StatusMsg = fmt.Sprintf("Removed '%s' from today's task list.", task.Title)
				}
				return true, nil
			}
		}
		m.SelectedDay = time.Now()
		m.selectDefaultTaskForSelectedDay()
		m.TimelineHour = time.Now().Hour()
		m.ScrollOffset = 0
		m.StatusMsg = "Jumped to today."
		return true, nil
	case " ":
		task, exists := m.GetActiveTask()
		if exists {
			task.AddedToToday = !task.AddedToToday
			task.UpdatedAt = time.Now()
			m.DB.UpdateTask(task)
			m.refreshTasks()
			if task.AddedToToday {
				m.StatusMsg = fmt.Sprintf("Added '%s' to today's task list (will appear on Day timeline backlog).", task.Title)
			} else {
				m.StatusMsg = fmt.Sprintf("Removed '%s' from today's task list.", task.Title)
			}
			return true, nil
		}
		return false, nil
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
			m.CheckAndStartZenMode(task)
		} else {
			m.StatusMsg = "No active task selected to start Zen Mode."
		}
		return true, nil
	}
	return false, nil
}
