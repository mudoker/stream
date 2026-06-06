package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

var PriorityOptions = []string{"0 (Critical)", "1 (High)", "2 (Medium)", "3 (Low)"}
var TaskTypeOptions = []string{"Anchored", "Floating", "Reminder"}
var SPOptions = []int{1, 2, 3, 5, 8, 13}

func (m *Model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	visible := m.Form.VisibleFields()

	// Find index of current ActiveField in visible
	curIdx := -1
	for idx, val := range visible {
		if val == m.Form.ActiveField {
			curIdx = idx
			break
		}
	}
	if curIdx == -1 {
		curIdx = 0
		m.Form.ActiveField = visible[0]
	}

	switch key {
	case "tab", "down":
		curIdx = (curIdx + 1) % len(visible)
		m.Form.ActiveField = visible[curIdx]
		m.focusFormFields()
		return m, nil
	case "shift+tab", "up":
		curIdx = (curIdx - 1 + len(visible)) % len(visible)
		m.Form.ActiveField = visible[curIdx]
		m.focusFormFields()
		return m, nil
	case "left":
		switch m.Form.ActiveField {
		case 2: // Priority
			m.Form.PriorityIdx = (m.Form.PriorityIdx - 1 + 4) % 4
			return m, nil
		case 3: // Story Points
			m.Form.SPIdx = (m.Form.SPIdx - 1 + 6) % 6
			return m, nil
		case 4: // Task Type
			m.Form.TaskTypeIdx = (m.Form.TaskTypeIdx - 1 + len(TaskTypeOptions)) % len(TaskTypeOptions)
			return m, nil
		}
	case "right", " ":
		switch m.Form.ActiveField {
		case 2: // Priority
			m.Form.PriorityIdx = (m.Form.PriorityIdx + 1) % 4
			return m, nil
		case 3: // Story Points
			m.Form.SPIdx = (m.Form.SPIdx + 1) % 6
			return m, nil
		case 4: // Task Type
			m.Form.TaskTypeIdx = (m.Form.TaskTypeIdx + 1) % len(TaskTypeOptions)
			return m, nil
		}
	case "enter":
		if m.Form.ActiveField == 8 { // Submit
			m.submitForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		curIdx = (curIdx + 1) % len(visible)
		m.Form.ActiveField = visible[curIdx]
		m.focusFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		m.IsEditing = false
		m.EditingTaskUUID = ""
		return m, nil
	}

	var cmd tea.Cmd
	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput, cmd = m.Form.TitleInput.Update(msg)
	case 1:
		m.Form.DescInput, cmd = m.Form.DescInput.Update(msg)
	case 5:
		m.Form.StartTimeInput, cmd = m.Form.StartTimeInput.Update(msg)
	case 6:
		m.Form.DurationInput, cmd = m.Form.DurationInput.Update(msg)
	case 7:
		m.Form.TagsInput, cmd = m.Form.TagsInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusFormFields() {
	m.Form.TitleInput.Blur()
	m.Form.DescInput.Blur()
	m.Form.StartTimeInput.Blur()
	m.Form.DurationInput.Blur()
	m.Form.TagsInput.Blur()

	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput.Focus()
	case 1:
		m.Form.DescInput.Focus()
	case 5:
		if m.Form.TaskTypeIdx != 1 {
			m.Form.StartTimeInput.Focus()
		}
	case 6:
		if m.Form.TaskTypeIdx == 0 {
			m.Form.DurationInput.Focus()
		}
	case 7:
		m.Form.TagsInput.Focus()
	}
}

func (m *Model) submitForm() {
	title := m.Form.TitleInput.Value()
	if title == "" {
		m.StatusMsg = "Title cannot be empty."
		return
	}

	var priorityVal model.Priority
	switch m.Form.PriorityIdx {
	case 0:
		priorityVal = model.P0
	case 1:
		priorityVal = model.P1
	case 2:
		priorityVal = model.P2
	case 3:
		priorityVal = model.P3
	}

	spVal := SPOptions[m.Form.SPIdx]
	taskType := m.Form.TaskTypeIdx

	var startTime time.Time
	duration := 60

	if taskType != 1 {
		timeStr := m.Form.StartTimeInput.Value()

		hour, min := ParseFlexibleTime(timeStr, 9, 0)
		now := time.Now()
		startTime = time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())
		if taskType == 0 {
			durStr := m.Form.DurationInput.Value()
			if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
				duration = d
			}
		}
	}

	var isEdit = m.IsEditing
	var existingTask model.Task
	if isEdit {
		for _, t := range m.Tasks {
			if t.UUID == m.EditingTaskUUID {
				existingTask = t
				break
			}
		}
	}

	tagsStr := m.Form.TagsInput.Value()
	var tags []string
	for _, part := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}

	newTask := model.Task{
		UUID:          uuid.New().String(),
		WorkspaceUUID: m.ActiveWorkspaceUUID,
		Title:         title,
		Description:   m.Form.DescInput.Value(),
		Priority:      priorityVal,
		StoryPoints:   spVal,
		Tags:          tags,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if isEdit {
		newTask.UUID = existingTask.UUID
		newTask.WorkspaceUUID = existingTask.WorkspaceUUID
		newTask.CreatedAt = existingTask.CreatedAt
		newTask.UpdatedAt = time.Now()
		newTask.ExecutionMetrics = existingTask.ExecutionMetrics
		newTask.GCalMetadata = existingTask.GCalMetadata
		newTask.Notes = existingTask.Notes
	}

	if taskType == 0 {
		newTask.SchedulingType = model.Anchored
		newTask.TimeWindow = model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(time.Duration(duration) * time.Minute),
		}
		if isEdit && existingTask.LifecycleState == model.StateCompleted {
			newTask.LifecycleState = model.StateCompleted
		} else {
			newTask.LifecycleState = model.StateScheduled
		}
	} else if taskType == 2 {
		newTask.SchedulingType = model.Reminder
		newTask.TimeWindow = model.TimeWindow{
			Start: startTime,
		}
		if isEdit && existingTask.LifecycleState == model.StateCompleted {
			newTask.LifecycleState = model.StateCompleted
		} else {
			newTask.LifecycleState = model.StateReady
		}
	} else {
		newTask.SchedulingType = model.Floating
		if isEdit && existingTask.LifecycleState == model.StateCompleted {
			newTask.LifecycleState = model.StateCompleted
		} else {
			newTask.LifecycleState = model.StateReady
		}
	}

	if isEdit {
		m.DB.UpdateTask(newTask)
		if m.ZenTimer != nil && m.ZenTimer.Task.UUID == newTask.UUID {
			m.ZenTimer.UpdateTaskDuration(newTask)
		}
		m.IsEditing = false
		m.EditingTaskUUID = ""
		m.refreshTasks()
		m.Sync.TriggerSync()
		m.StatusMsg = fmt.Sprintf("Task '%s' updated successfully.", title)
	} else {
		m.DB.AddTask(newTask)
		m.refreshTasks()
		m.Sync.TriggerSync()
		m.StatusMsg = fmt.Sprintf("Task '%s' created successfully.", title)
	}
}

func (m *Model) handleWorkspaceFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "shift+tab":
		m.WorkspaceForm.ActiveField = (m.WorkspaceForm.ActiveField - 1 + 4) % 4
		m.focusWorkspaceFormFields()
		return m, nil
	case "down", "tab":
		m.WorkspaceForm.ActiveField = (m.WorkspaceForm.ActiveField + 1) % 4
		m.focusWorkspaceFormFields()
		return m, nil
	case "enter":
		if m.WorkspaceForm.ActiveField == 3 { // Submit
			m.submitWorkspaceForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		m.WorkspaceForm.ActiveField = (m.WorkspaceForm.ActiveField + 1) % 4
		m.focusWorkspaceFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		m.IsEditingWorkspace = false
		m.EditingWorkspaceUUID = ""
		return m, nil
	}

	var cmd tea.Cmd
	switch m.WorkspaceForm.ActiveField {
	case 0:
		m.WorkspaceForm.NameInput, cmd = m.WorkspaceForm.NameInput.Update(msg)
	case 1:
		m.WorkspaceForm.IconInput, cmd = m.WorkspaceForm.IconInput.Update(msg)
	case 2:
		m.WorkspaceForm.BadgeInput, cmd = m.WorkspaceForm.BadgeInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusWorkspaceFormFields() {
	m.WorkspaceForm.NameInput.Blur()
	m.WorkspaceForm.IconInput.Blur()
	m.WorkspaceForm.BadgeInput.Blur()

	switch m.WorkspaceForm.ActiveField {
	case 0:
		m.WorkspaceForm.NameInput.Focus()
	case 1:
		m.WorkspaceForm.IconInput.Focus()
	case 2:
		m.WorkspaceForm.BadgeInput.Focus()
	}
}

func (m *Model) submitWorkspaceForm() {
	name := m.WorkspaceForm.NameInput.Value()
	if strings.TrimSpace(name) == "" {
		m.StatusMsg = "Workspace name cannot be empty."
		return
	}

	icon := m.WorkspaceForm.IconInput.Value()
	if strings.TrimSpace(icon) == "" {
		icon = "💼"
	}

	badge := m.WorkspaceForm.BadgeInput.Value()

	var ws model.Workspace
	isEdit := m.IsEditingWorkspace
	if isEdit {
		for _, w := range m.Workspaces {
			if w.UUID == m.EditingWorkspaceUUID {
				ws = w
				break
			}
		}
	} else {
		ws = model.Workspace{
			UUID:      uuid.New().String(),
			CreatedAt: time.Now(),
		}
	}

	ws.Name = name
	ws.Icon = icon
	ws.Badge = badge
	ws.UpdatedAt = time.Now()

	if isEdit {
		err := m.DB.UpdateWorkspace(ws)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Error updating workspace: %v", err)
		} else {
			m.StatusMsg = fmt.Sprintf("Workspace '%s' updated successfully.", name)
		}
	} else {
		err := m.DB.AddWorkspace(ws)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Error creating workspace: %v", err)
		} else {
			m.ActiveWorkspaceUUID = ws.UUID // Switch to it!
			m.StatusMsg = fmt.Sprintf("Workspace '%s' created successfully.", name)
		}
	}

	m.IsEditingWorkspace = false
	m.EditingWorkspaceUUID = ""
	m.refreshWorkspaces()
	m.refreshTasks()
	m.selectDefaultTaskForSelectedDay()
}

func (m *Model) startEditMode(task model.Task) {
	m.IsEditing = true
	m.EditingTaskUUID = task.UUID

	// Pre-fill form fields
	m.Form.TitleInput.SetValue(task.Title)
	m.Form.DescInput.SetValue(task.Description)
	switch task.Priority {
	case model.P0:
		m.Form.PriorityIdx = 0
	case model.P1:
		m.Form.PriorityIdx = 1
	case model.P2:
		m.Form.PriorityIdx = 2
	case model.P3:
		m.Form.PriorityIdx = 3
	}

	m.Form.SPIdx = 2
	for idx, val := range SPOptions {
		if val == task.StoryPoints {
			m.Form.SPIdx = idx
			break
		}
	}

	if task.SchedulingType == model.Anchored {
		m.Form.TaskTypeIdx = 0
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
	} else if task.SchedulingType == model.Reminder {
		m.Form.TaskTypeIdx = 2
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		m.Form.DurationInput.SetValue("60")
	} else {
		m.Form.TaskTypeIdx = 1
		m.Form.StartTimeInput.SetValue(time.Now().Format("15:04"))
		m.Form.DurationInput.SetValue("60")
	}

	m.Form.TagsInput.SetValue(strings.Join(task.Tags, ", "))

	m.Form.ActiveField = 0
	m.focusFormFields()
	m.CurrentMode = ModeForm
}

func (m *Model) handleProfileFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "shift+tab":
		m.ProfileForm.ActiveField = (m.ProfileForm.ActiveField - 1 + 4) % 4
		m.focusProfileFormFields()
		return m, nil
	case "down", "tab":
		m.ProfileForm.ActiveField = (m.ProfileForm.ActiveField + 1) % 4
		m.focusProfileFormFields()
		return m, nil
	case "enter":
		if m.ProfileForm.ActiveField == 3 { // Submit
			m.submitProfileForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		m.ProfileForm.ActiveField = (m.ProfileForm.ActiveField + 1) % 4
		m.focusProfileFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	switch m.ProfileForm.ActiveField {
	case 0:
		m.ProfileForm.UsernameInput, cmd = m.ProfileForm.UsernameInput.Update(msg)
	case 1:
		m.ProfileForm.PasswordInput, cmd = m.ProfileForm.PasswordInput.Update(msg)
	case 2:
		m.ProfileForm.LockTimeoutInput, cmd = m.ProfileForm.LockTimeoutInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusProfileFormFields() {
	m.ProfileForm.UsernameInput.Blur()
	m.ProfileForm.PasswordInput.Blur()
	m.ProfileForm.LockTimeoutInput.Blur()

	switch m.ProfileForm.ActiveField {
	case 0:
		m.ProfileForm.UsernameInput.Focus()
	case 1:
		m.ProfileForm.PasswordInput.Focus()
	case 2:
		m.ProfileForm.LockTimeoutInput.Focus()
	}
}

func (m *Model) submitProfileForm() {
	user := strings.TrimSpace(m.ProfileForm.UsernameInput.Value())
	if user == "" {
		m.StatusMsg = "Username cannot be empty."
		return
	}

	passVal := m.ProfileForm.PasswordInput.Value()
	timeoutStr := m.ProfileForm.LockTimeoutInput.Value()

	timeoutVal := 5
	if val, err := strconv.Atoi(timeoutStr); err == nil && val > 0 {
		timeoutVal = val
	}

	settings := m.DB.GetUserSettings()
	settings.Username = user
	settings.LockTimeoutMinutes = timeoutVal

	// Update password if provided
	if passVal != "" {
		if strings.EqualFold(passVal, "none") {
			settings.PasswordHash = ""
			m.IsLocked = false
			m.StatusMsg = "Display username and settings updated. Password lock disabled."
		} else {
			// Hash it
			settings.PasswordHash = db.HashPassword(passVal)
			m.StatusMsg = "Display username, settings, and password updated successfully."
		}
	} else {
		m.StatusMsg = "Display username and lock settings updated successfully."
	}

	if err := m.DB.UpdateUserSettings(settings); err != nil {
		m.StatusMsg = fmt.Sprintf("Error saving settings: %v", err)
	} else {
		// Reset lock timer to match new timeout
		m.SessionTimeRemainingSeconds = timeoutVal * 60
	}

	// Sync local copy in model too
	m.ProfileForm.Username = user
	m.ProfileForm.LockTimeoutMins = timeoutVal
	m.ProfileForm.PasswordInput.SetValue("")
}
