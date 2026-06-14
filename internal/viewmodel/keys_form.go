package viewmodel

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
var TaskTypeOptions = []string{"Anchored", "Floating", "Reminder", "Habit", "Event"}
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
		case 11:
			m.Form.IsRecurringIdx = (m.Form.IsRecurringIdx - 1 + 2) % 2
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
		case 11:
			m.Form.IsRecurringIdx = (m.Form.IsRecurringIdx + 1) % 2
			return m, nil
		}
	case "enter":
		m.SubmitForm()
		m.CurrentMode = ModeNormal
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		m.IsEditing = false
		m.Form.IsEditing = false
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
		if m.Form.TaskTypeIdx == 0 || m.Form.TaskTypeIdx == 4 {
			m.Form.StartTimeInput, cmd = m.Form.StartTimeInput.Update(msg)
		} else if m.Form.TaskTypeIdx == 2 {
			m.Form.DueDateInput, cmd = m.Form.DueDateInput.Update(msg)
		}
	case 6:
		if m.Form.TaskTypeIdx == 0 || m.Form.TaskTypeIdx == 4 {
			m.Form.DurationInput, cmd = m.Form.DurationInput.Update(msg)
		} else if m.Form.TaskTypeIdx == 2 {
			m.Form.StartTimeInput, cmd = m.Form.StartTimeInput.Update(msg)
		}
	case 7:
		if m.Form.TaskTypeIdx == 4 {
			m.Form.LocationInput, cmd = m.Form.LocationInput.Update(msg)
		} else {
			m.Form.TagsInput, cmd = m.Form.TagsInput.Update(msg)
		}
	case 8:
		if m.Form.TaskTypeIdx == 4 {
			m.Form.CommuteInput, cmd = m.Form.CommuteInput.Update(msg)
		}
	case 9:
		m.Form.TagsInput, cmd = m.Form.TagsInput.Update(msg)
	case 12:
		m.Form.RecurringEndDateInput, cmd = m.Form.RecurringEndDateInput.Update(msg)
	case 13:
		m.Form.RecurringDaysInput, cmd = m.Form.RecurringDaysInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusFormFields() {
	m.Form.TitleInput.Blur()
	m.Form.DescInput.Blur()
	m.Form.StartTimeInput.Blur()
	m.Form.DurationInput.Blur()
	m.Form.TagsInput.Blur()
	m.Form.DueDateInput.Blur()
	m.Form.LocationInput.Blur()
	m.Form.CommuteInput.Blur()
	m.Form.RecurringEndDateInput.Blur()
	m.Form.RecurringDaysInput.Blur()

	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput.Focus()
	case 1:
		m.Form.DescInput.Focus()
	case 5:
		if m.Form.TaskTypeIdx == 0 || m.Form.TaskTypeIdx == 4 {
			m.Form.StartTimeInput.Focus()
		} else if m.Form.TaskTypeIdx == 2 {
			m.Form.DueDateInput.Focus()
		}
	case 6:
		if m.Form.TaskTypeIdx == 0 || m.Form.TaskTypeIdx == 4 {
			m.Form.DurationInput.Focus()
		} else if m.Form.TaskTypeIdx == 2 {
			m.Form.StartTimeInput.Focus()
		}
	case 7:
		if m.Form.TaskTypeIdx == 4 {
			m.Form.LocationInput.Focus()
		} else {
			m.Form.TagsInput.Focus()
		}
	case 8:
		if m.Form.TaskTypeIdx == 4 {
			m.Form.CommuteInput.Focus()
		}
	case 9:
		m.Form.TagsInput.Focus()
	case 12:
		m.Form.RecurringEndDateInput.Focus()
	case 13:
		m.Form.RecurringDaysInput.Focus()
	}
}

func (m *Model) SubmitForm() {
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

	now := time.Now()
	startTime := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), 9, 0, 0, 0, now.Location())
	duration := 60

	if taskType == 0 || taskType == 4 {
		timeStr := m.Form.StartTimeInput.Value()
		hour, min := ParseFlexibleTime(timeStr, 9, 0)
		startTime = time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())
		durStr := m.Form.DurationInput.Value()
		if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
			duration = d
		}
	} else if taskType == 2 {
		dateStr := m.Form.DueDateInput.Value()
		timeStr := strings.TrimSpace(m.Form.StartTimeInput.Value())

		dueDay, err := time.Parse("2006-01-02", strings.TrimSpace(dateStr))
		if err != nil {
			dueDay = m.SelectedDay
		}

		var hour, min, sec int
		if timeStr == "" {
			hour, min, sec = 9, 0, 1
		} else {
			hour, min = ParseFlexibleTime(timeStr, 9, 0)
			sec = 0
		}
		startTime = time.Date(dueDay.Year(), dueDay.Month(), dueDay.Day(), hour, min, sec, 0, now.Location())
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
		newTask.RecurringParentUUID = existingTask.RecurringParentUUID
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
	} else if taskType == 4 {
		newTask.SchedulingType = model.Event
		newTask.TimeWindow = model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(time.Duration(duration) * time.Minute),
		}
		newTask.Location = m.Form.LocationInput.Value()
		commuteMins := 0
		if strings.TrimSpace(newTask.Location) != "" {
			if c, err := strconv.Atoi(m.Form.CommuteInput.Value()); err == nil && c > 0 {
				commuteMins = c
			}
		}
		newTask.CommuteBuffer = commuteMins
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
	} else if taskType == 3 {
		newTask.SchedulingType = model.Habit
		newTask.StoryPoints = 0
		// By default, a non-recurring habit starts on the shelf (zero TimeWindow)
		newTask.TimeWindow = model.TimeWindow{}
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
		if existingTask.RecurringParentUUID != "" {
			m.PendingEditTask = newTask
			m.ConfirmTask = existingTask
			m.ConfirmOpen = true
			m.ConfirmActionType = "edit_recurring"
			m.IsEditing = false
			m.Form.IsEditing = false
			m.EditingTaskUUID = ""
			m.StatusMsg = "Choose recurring update option."
			return
		}

		m.DB.UpdateTask(newTask)
		if m.ZenTimer != nil && m.ZenTimer.Task.UUID == newTask.UUID {
			m.ZenTimer.UpdateTaskDuration(newTask)
		}
		m.IsEditing = false
		m.Form.IsEditing = false
		m.EditingTaskUUID = ""
		m.refreshTasks()
		m.SelectedTaskUUID = newTask.UUID
		m.AutoScrollToSelectedTask()
		m.triggerGCalPush(newTask)
		m.StatusMsg = fmt.Sprintf("Task '%s' updated successfully.", title)
	} else {
		if m.Form.IsRecurringIdx == 1 || taskType == 3 {
			endDateStr := strings.TrimSpace(m.Form.RecurringEndDateInput.Value())
			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				endDate = startTime.AddDate(0, 0, 7)
			}
			endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, startTime.Location())

			maxEndDate := startTime.AddDate(0, 1, 0)
			if endDate.After(maxEndDate) {
				endDate = time.Date(maxEndDate.Year(), maxEndDate.Month(), maxEndDate.Day(), 23, 59, 59, 0, startTime.Location())
			}

			daysStr := strings.ToLower(m.Form.RecurringDaysInput.Value())
			days := map[time.Weekday]bool{
				time.Sunday:    strings.Contains(daysStr, "sun") || strings.Contains(daysStr, "daily"),
				time.Monday:    strings.Contains(daysStr, "mon") || strings.Contains(daysStr, "daily"),
				time.Tuesday:   strings.Contains(daysStr, "tue") || strings.Contains(daysStr, "daily"),
				time.Wednesday: strings.Contains(daysStr, "wed") || strings.Contains(daysStr, "daily"),
				time.Thursday:  strings.Contains(daysStr, "thu") || strings.Contains(daysStr, "daily"),
				time.Friday:    strings.Contains(daysStr, "fri") || strings.Contains(daysStr, "daily"),
				time.Saturday:  strings.Contains(daysStr, "sat") || strings.Contains(daysStr, "daily"),
			}
			hasAny := false
			for _, v := range days {
				if v {
					hasAny = true
					break
				}
			}
			if !hasAny {
				for k := range days {
					days[k] = true
				}
			}

			parentUUID := uuid.New().String()
			current := startTime
			count := 0
			for !current.After(endDate) {
				if days[current.Weekday()] {
					instance := newTask
					instance.UUID = uuid.New().String()
					instance.RecurringParentUUID = parentUUID
					instance.TimeWindow.Start = time.Date(current.Year(), current.Month(), current.Day(), startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
					
					if instance.SchedulingType == model.Habit {
						// A habit is a recurring task with TimeWindow set, so it is anchored by default
						instance.TimeWindow.End = instance.TimeWindow.Start.Add(time.Duration(duration) * time.Minute)
						instance.LifecycleState = model.StateReady
					} else {
						instance.SchedulingType = model.Anchored
						instance.TimeWindow.End = instance.TimeWindow.Start.Add(time.Duration(duration) * time.Minute)
						instance.LifecycleState = model.StateScheduled
					}

					m.DB.AddTask(instance)
					count++
				}
				current = current.AddDate(0, 0, 1)
			}
			m.refreshTasks()
			m.StatusMsg = fmt.Sprintf("Created %d recurring occurrences.", count)
			return
		}

		m.DB.AddTask(newTask)
		m.refreshTasks()
		m.SelectedTaskUUID = newTask.UUID
		m.AutoScrollToSelectedTask()
		m.triggerGCalPush(newTask)
		m.StatusMsg = fmt.Sprintf("Task '%s' created successfully.", title)
	}
}


