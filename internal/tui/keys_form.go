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
var TaskTypeOptions = []string{"Anchored", "Floating", "Reminder"}
var SPOptions = []int{0, 1, 2, 3, 5, 8, 13}

func (m *Model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	visible := m.Form.VisibleFields()

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
		case 2:
			m.Form.PriorityIdx = (m.Form.PriorityIdx - 1 + 4) % 4
			return m, nil
		case 3:
			m.Form.SPIdx = (m.Form.SPIdx - 1 + len(SPOptions)) % len(SPOptions)
			return m, nil
		case 4:
			m.Form.TaskTypeIdx = (m.Form.TaskTypeIdx - 1 + len(TaskTypeOptions)) % len(TaskTypeOptions)
			return m, nil
		}
	case "right", " ":
		switch m.Form.ActiveField {
		case 2:
			m.Form.PriorityIdx = (m.Form.PriorityIdx + 1) % 4
			return m, nil
		case 3:
			m.Form.SPIdx = (m.Form.SPIdx + 1) % len(SPOptions)
			return m, nil
		case 4:
			m.Form.TaskTypeIdx = (m.Form.TaskTypeIdx + 1) % len(TaskTypeOptions)
			return m, nil
		}
	case "enter":
		if m.Form.ActiveField == 8 {
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
		newTask.StoryPoints = 0
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


