package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type CommandEntry struct {
	Name string
	Desc string
}

var DefaultCommands = []CommandEntry{
	{"create", "Anchor a new task for today at 9:00 AM"},
	{"todo", "Add a floating task to the backlog shelf"},
	{"complete", "Complete the selected task"},
	{"delete", "Delete the selected task"},
	{"sync", "Force Google Calendar sync"},
	{"auth", "Authenticate with Google Calendar"},
	{"stop", "Stop/Abort active Zen focus session"},
	{"review", "Open daily shutdown review"},
	{"dashboard", "Switch to dashboard view"},
	{"month", "Switch to month grid view"},
	{"week", "Switch to week lanes view"},
	{"day", "Switch to day timeline view"},
	{"analytics", "Switch to analytics view"},
	{"ws-create", "Create a new workspace"},
	{"ws-edit", "Edit active workspace"},
	{"ws-delete", "Delete active workspace or specify name"},
	{"ws-switch", "Switch to a workspace by name (type: ws-switch <name>)"},
	{"help", "Open command reference / help"},
	{"quit", "Exit stream"},
}

func (m *Model) getCommandList() []CommandEntry {
	var list []CommandEntry
	list = append(list, DefaultCommands...)

	for _, ws := range m.Workspaces {
		list = append(list, CommandEntry{
			Name: fmt.Sprintf("ws-switch %s", ws.Name),
			Desc: fmt.Sprintf("Switch to workspace: %s %s", ws.Icon, ws.Name),
		})
	}
	return list
}

func (m *Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	val := strings.ToLower(m.CommandInput.Value())
	var filtered []CommandEntry
	for _, c := range m.getCommandList() {
		if strings.Contains(strings.ToLower(c.Name), val) || strings.Contains(strings.ToLower(c.Desc), val) {
			filtered = append(filtered, c)
		}
	}

	switch msg.String() {
	case "enter":
		commandToRun := m.CommandInput.Value()
		if len(filtered) > 0 && m.CommandSelectedIndex >= 0 && m.CommandSelectedIndex < len(filtered) {
			commandToRun = filtered[m.CommandSelectedIndex].Name
		}
		m.CurrentMode = ModeNormal
		m.CommandSelectedIndex = 0
		m.CommandInput.SetValue("")
		return m.runCommand(commandToRun)
	case "esc":
		m.CurrentMode = ModeNormal
		m.CommandSelectedIndex = 0
		m.CommandInput.SetValue("")
		return m, nil
	case "up", "ctrl+k":
		if len(filtered) > 0 {
			m.CommandSelectedIndex--
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = len(filtered) - 1
			}
		}
		return m, nil
	case "down", "ctrl+j":
		if len(filtered) > 0 {
			m.CommandSelectedIndex++
			if m.CommandSelectedIndex >= len(filtered) {
				m.CommandSelectedIndex = 0
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.CommandInput, cmd = m.CommandInput.Update(msg)
	m.CommandSelectedIndex = 0
	return m, cmd
}

func (m *Model) runCommand(val string) (tea.Model, tea.Cmd) {
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
	case "help", "h", "?":
		m.HelpOpen = true
		m.HelpScrollOffset = 0
		m.StatusMsg = "Help opened. Press Esc/? to exit."

	case "create", "todo":
		if len(parts) < 2 {
			m.StatusMsg = "Syntax: create <task title>"
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
			// Anchor it for today 9:00 - 10:00
			now := time.Now()
			start := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), 9, 0, 0, 0, now.Location())
			end := start.Add(1 * time.Hour)
			newTask.SchedulingType = model.Anchored
			newTask.TimeWindow = model.TimeWindow{Start: start, End: end}
			newTask.LifecycleState = model.StateScheduled
		}
		m.DB.AddTask(newTask)
		m.refreshTasks()
		m.Sync.TriggerSync()
		m.StatusMsg = fmt.Sprintf("Task '%s' created.", title)

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
		task, exists := m.getActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			m.Sync.TriggerSync()
			m.StatusMsg = fmt.Sprintf("Task '%s' completed.", task.Title)
		}

	case "delete":
		task, exists := m.getActiveTask()
		if exists {
			m.ConfirmTask = task
			m.ConfirmOpen = true
		}

	case "sync":
		m.Sync.TriggerSync()
		m.StatusMsg = "Triggering Google Calendar sync..."

	case "auth":
		url, err := m.Sync.StartAuthServer(8080)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Auth server error: %v", err)
		} else {
			m.StatusMsg = "Go to: " + url
		}

	case "stop":
		if m.ZenTimer != nil {
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
