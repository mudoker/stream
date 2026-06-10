package viewmodel

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func (m *Model) RunCommand(val string) (tea.Model, tea.Cmd) {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, ":") {
		val = strings.TrimPrefix(val, ":")
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return m, nil
	}

	parts := strings.Fields(val)
	cmdName := parts[0]

	switch cmdName {
	case "q", "quit", "exit":
		return m, tea.Quit

	case "dashboard":
		m.CurrentView = DashboardView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Dashboard view."
	case "month":
		m.CurrentView = MonthView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Month view."
	case "week":
		m.CurrentView = WeekView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Week view."
	case "day":
		m.CurrentView = DayView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Day view."
	case "analytics":
		m.CurrentView = AnalyticsView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Analytics view."
	case "profile":
		settings := m.DB.GetUserSettings()
		m.ProfileForm = NewProfileForm(settings.Username, settings.LockTimeoutMinutes)
		m.ProfileForm.ActiveField = 0
		m.focusProfileFormFields()
		m.CurrentMode = ModeProfileForm
		m.StatusMsg = "Edit profile & security settings."
	case "help", "h", "?":
		m.HelpOpen = true
		m.HelpScrollOffset = 0
		m.StatusMsg = "Help opened. Press Esc/? to exit."

	case "create", "todo", "habit":
		if len(parts) < 2 {
			m.StatusMsg = fmt.Sprintf("Syntax: %s <task title>", cmdName)
			return m, nil
		}
		title := strings.Join(parts[1:], " ")
		newTask := model.Task{
			UUID:           uuid.New().String(),
			WorkspaceUUID:  m.ActiveWorkspaceUUID,
			Title:          title,
			Priority:       model.P2,
			StoryPoints:    3,
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		}
		if cmdName == "create" {
			now := time.Now()
			start := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), 9, 0, 0, 0, now.Location())
			end := start.Add(1 * time.Hour)
			newTask.SchedulingType = model.Anchored
			newTask.TimeWindow = model.TimeWindow{Start: start, End: end}
			newTask.LifecycleState = model.StateScheduled
		} else if cmdName == "habit" {
			newTask.SchedulingType = model.Habit
			newTask.StoryPoints = 0
		}
		m.DB.AddTask(newTask)
		m.refreshTasks()
		if m.Sync != nil {
			m.Sync.TriggerSync()
		}
		if cmdName == "habit" {
			m.StatusMsg = fmt.Sprintf("Habit '%s' created.", title)
		} else {
			m.StatusMsg = fmt.Sprintf("Task '%s' created.", title)
		}

	case "ws-switch":
		if len(parts) < 2 {
			m.WorkspacePickerIdx = 0
			for i, ws := range m.Workspaces {
				if ws.UUID == m.ActiveWorkspaceUUID {
					m.WorkspacePickerIdx = i
					break
				}
			}
			m.CurrentMode = ModeWorkspacePicker
			m.StatusMsg = "Select a workspace to switch to."
			return m, nil
		}
		targetName := strings.Join(parts[1:], " ")
		var targetWS *model.Workspace
		for _, ws := range m.Workspaces {
			if strings.EqualFold(ws.Name, targetName) {
				targetWS = &ws
				break
			}
		}
		if targetWS == nil {
			m.StatusMsg = fmt.Sprintf("Workspace '%s' not found.", targetName)
		} else {
			m.ActiveWorkspaceUUID = targetWS.UUID
			m.refreshTasks()
			m.selectDefaultTaskForSelectedDay()
			m.StatusMsg = fmt.Sprintf("Switched to workspace '%s'.", targetWS.Name)
		}

	case "ws-create":
		m.IsEditingWorkspace = false
		m.WorkspaceForm = NewWorkspaceForm()
		m.CurrentMode = ModeWorkspaceForm
		m.StatusMsg = "Entering Workspace Creation Form. Press Esc to cancel."

	case "ws-edit":
		var activeWS model.Workspace
		found := false
		for _, ws := range m.Workspaces {
			if ws.UUID == m.ActiveWorkspaceUUID {
				activeWS = ws
				found = true
				break
			}
		}
		if !found {
			m.StatusMsg = "No active workspace to edit."
			return m, nil
		}
		m.IsEditingWorkspace = true
		m.EditingWorkspaceUUID = activeWS.UUID
		m.WorkspaceForm = NewWorkspaceForm()
		m.WorkspaceForm.NameInput.SetValue(activeWS.Name)
		m.WorkspaceForm.IconInput.SetValue(activeWS.Icon)
		m.WorkspaceForm.BadgeInput.SetValue(activeWS.Badge)
		m.CurrentMode = ModeWorkspaceForm
		m.StatusMsg = fmt.Sprintf("Editing workspace '%s'. Press Esc to cancel.", activeWS.Name)

	case "ws-delete":
		var wsToDelete model.Workspace
		if len(parts) > 1 {
			targetName := strings.Join(parts[1:], " ")
			found := false
			for _, ws := range m.Workspaces {
				if strings.EqualFold(ws.Name, targetName) {
					wsToDelete = ws
					found = true
					break
				}
			}
			if !found {
				m.StatusMsg = fmt.Sprintf("Workspace '%s' not found.", targetName)
				return m, nil
			}
		} else {
			for _, ws := range m.Workspaces {
				if ws.UUID == m.ActiveWorkspaceUUID {
					wsToDelete = ws
					break
				}
			}
		}

		if len(m.Workspaces) <= 1 {
			m.StatusMsg = "Cannot delete the last workspace."
			return m, nil
		}

		err := m.DB.DeleteWorkspace(wsToDelete.UUID)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Error deleting workspace: %v", err)
		} else {
			if m.ActiveWorkspaceUUID == wsToDelete.UUID {
				m.ActiveWorkspaceUUID = ""
			}
			m.refreshWorkspaces()
			m.refreshTasks()
			m.selectDefaultTaskForSelectedDay()
			m.StatusMsg = fmt.Sprintf("Workspace '%s' and its tasks deleted.", wsToDelete.Name)
		}

	case "review":
		today := time.Now()
		completed := 0
		deferred := 0
		secs := 0
		for _, t := range m.Tasks {
			isToday := false
			if t.SchedulingType == model.Anchored {
				isToday = t.TimeWindow.Start.Year() == today.Year() && t.TimeWindow.Start.Month() == today.Month() && t.TimeWindow.Start.Day() == today.Day()
			} else {
				isToday = t.CreatedAt.Year() == today.Year() && t.CreatedAt.Month() == today.Month() && t.CreatedAt.Day() == today.Day()
			}

			if isToday {
				if t.LifecycleState == model.StateCompleted {
					completed++
				} else {
					deferred++
				}
				secs += t.ExecutionMetrics.ElapsedFocusSeconds
			}
		}
		m.ReviewTasksCompleted = completed
		m.ReviewTasksDeferred = deferred
		m.ReviewFocusSeconds = secs
		m.ReviewOpen = true
		m.StatusMsg = "Daily Shutdown Review opened."

	case "complete":
		task, exists := m.GetActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			if m.Sync != nil {
				m.Sync.TriggerSync()
			}
			if m.ZenTimer != nil && m.ZenTimer.Task.UUID == task.UUID {
				m.ZenTimer = nil
			}
			m.StatusMsg = fmt.Sprintf("Task '%s' completed.", task.Title)
		}

	case "delete":
		task, exists := m.GetActiveTask()
		if exists {
			m.ConfirmTask = task
			m.ConfirmOpen = true
		}

	case "sync":
		if m.Sync != nil {
			m.Sync.TriggerSync()
			m.StatusMsg = "Triggering Google Calendar sync..."
		} else {
			m.StatusMsg = "Sync engine is not initialized."
		}

	case "auth":
		if m.Sync != nil {
			url, err := m.Sync.StartAuthServer(8080)
			if err != nil {
				m.StatusMsg = fmt.Sprintf("Auth server error: %v", err)
			} else {
				m.StatusMsg = "Go to: " + url
			}
		} else {
			m.StatusMsg = "Sync engine is not initialized."
		}

	case "stop":
		if m.ZenTimer != nil {
			m.ZenTimer.RecordElapsedTimes()
			t := m.ZenTimer.Task
			t.LifecycleState = model.StateReady
			m.DB.UpdateTask(t)
			m.refreshTasks()
			m.ZenTimer = nil
			m.StatusMsg = "Zen focus session stopped."
		} else {
			m.StatusMsg = "No active focus session running."
		}

	default:
		m.StatusMsg = fmt.Sprintf("Unknown command: %s", cmdName)
	}

	return m, nil
}
