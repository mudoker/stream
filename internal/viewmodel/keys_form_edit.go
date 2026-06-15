package viewmodel

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
)

func (m *Model) startEditMode(task model.Task) {
	m.IsEditing = true
	m.Form.IsEditing = true
	m.EditingTaskUUID = task.UUID

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

	m.Form.LocationInput.SetValue("")
	m.Form.CommuteInput.SetValue("0")

	if task.SchedulingType == model.Anchored {
		m.Form.TaskTypeIdx = 0
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
	} else if task.SchedulingType == model.Event {
		m.Form.TaskTypeIdx = 4
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
		m.Form.LocationInput.SetValue(task.Location)
		m.Form.CommuteInput.SetValue(fmt.Sprintf("%d", task.CommuteBuffer))
	} else if task.SchedulingType == model.Reminder {
		m.Form.TaskTypeIdx = 2
		if task.TimeWindow.Start.Second() == 1 {
			m.Form.StartTimeInput.SetValue("")
		} else {
			m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		}
		m.Form.DueDateInput.SetValue(task.TimeWindow.Start.Format("2006-01-02"))
		m.Form.DurationInput.SetValue("60")
	} else if task.SchedulingType == model.Habit {
		m.Form.TaskTypeIdx = 3
		if !task.TimeWindow.Start.IsZero() {
			m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
			durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
			m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
		} else {
			m.Form.StartTimeInput.SetValue("")
			m.Form.DurationInput.SetValue("60")
		}
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
