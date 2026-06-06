package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

var PriorityOptions = []string{"0 (Critical)", "1 (High)", "2 (Medium)", "3 (Low)"}
var SPOptions = []int{1, 2, 3, 5, 8, 13}

func (m *Model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab", "down":
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
		m.focusFormFields()
		return m, nil
	case "shift+tab", "up":
		m.Form.ActiveField = (m.Form.ActiveField - 1 + 8) % 8
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
		case 4: // Anchored
			m.Form.IsAnchored = !m.Form.IsAnchored
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
		case 4: // Anchored
			m.Form.IsAnchored = !m.Form.IsAnchored
			return m, nil
		}
	case "enter":
		if m.Form.ActiveField == 7 { // Submit
			m.submitForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
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
	}

	return m, cmd
}

func (m *Model) focusFormFields() {
	m.Form.TitleInput.Blur()
	m.Form.DescInput.Blur()
	m.Form.StartTimeInput.Blur()
	m.Form.DurationInput.Blur()

	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput.Focus()
	case 1:
		m.Form.DescInput.Focus()
	case 5:
		m.Form.StartTimeInput.Focus()
	case 6:
		m.Form.DurationInput.Focus()
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
	anchored := m.Form.IsAnchored

	var startTime time.Time
	duration := 60

	if anchored {
		timeStr := m.Form.StartTimeInput.Value()
		durStr := m.Form.DurationInput.Value()

		hour, min := 9, 0
		if parts := strings.Split(timeStr, ":"); len(parts) == 2 {
			h, _ := strconv.Atoi(parts[0])
			mVal, _ := strconv.Atoi(parts[1])
			if h >= 0 && h < 24 {
				hour = h
			}
			if mVal >= 0 && mVal < 60 {
				min = mVal
			}
		}

		if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
			duration = d
		}

		now := time.Now()
		startTime = time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())
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

	newTask := model.Task{
		UUID:          uuid.New().String(),
		WorkspaceUUID: m.ActiveWorkspaceUUID,
		Title:         title,
		Description:   m.Form.DescInput.Value(),
		Priority:      priorityVal,
		StoryPoints:   spVal,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if isEdit {
		newTask.UUID = existingTask.UUID
		newTask.WorkspaceUUID = existingTask.WorkspaceUUID
		newTask.CreatedAt = existingTask.CreatedAt
		newTask.UpdatedAt = time.Now()
		newTask.ExecutionMetrics = existingTask.ExecutionMetrics
	}

	if anchored {
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
