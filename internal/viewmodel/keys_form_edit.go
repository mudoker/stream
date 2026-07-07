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

	startDateVal := task.TimeWindow.Start.Format("2006-01-02")
	if task.TimeWindow.Start.IsZero() {
		startDateVal = m.SelectedDay.Format("2006-01-02")
	}
	m.Form.StartDateInput.SetValue(startDateVal)

	endDateVal := task.TimeWindow.End.Format("2006-01-02")
	if task.TimeWindow.End.IsZero() {
		if !task.TimeWindow.Start.IsZero() {
			endDateVal = task.TimeWindow.Start.Format("2006-01-02")
		} else {
			endDateVal = m.SelectedDay.Format("2006-01-02")
		}
	}
	m.Form.EndDateInput.SetValue(endDateVal)

	if task.SchedulingType == model.Anchored {
		m.Form.TaskTypeIdx = 0
		m.Form.IsAnchoredIdx = 1
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
	} else if task.SchedulingType == model.Floating {
		m.Form.TaskTypeIdx = 0
		m.Form.IsAnchoredIdx = 0
		m.Form.StartTimeInput.SetValue(time.Now().Format("15:04"))
		if task.EstimatedDurationMins > 0 {
			m.Form.DurationInput.SetValue(fmt.Sprintf("%d", task.EstimatedDurationMins))
		} else {
			m.Form.DurationInput.SetValue("60")
		}
	} else if task.SchedulingType == model.Reminder {
		m.Form.TaskTypeIdx = 1
		if task.TimeWindow.Start.Second() == 1 {
			m.Form.StartTimeInput.SetValue("")
		} else {
			m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		}
		m.Form.DueDateInput.SetValue(task.TimeWindow.Start.Format("2006-01-02"))
		m.Form.DurationInput.SetValue("60")
	} else if task.SchedulingType == model.Habit {
		m.Form.TaskTypeIdx = 2
		if !task.TimeWindow.Start.IsZero() {
			m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
			durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
			m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
		} else {
			m.Form.StartTimeInput.SetValue("")
			m.Form.DurationInput.SetValue("60")
		}
		m.Form.LocationInput.SetValue(task.Location)
		m.Form.CommuteInput.SetValue(fmt.Sprintf("%d", task.CommuteBuffer))
	} else if task.SchedulingType == model.Event {
		m.Form.TaskTypeIdx = 3
		m.Form.StartTimeInput.SetValue(task.TimeWindow.Start.Format("15:04"))
		durMins := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		m.Form.DurationInput.SetValue(fmt.Sprintf("%d", durMins))
		m.Form.LocationInput.SetValue(task.Location)
		m.Form.CommuteInput.SetValue(fmt.Sprintf("%d", task.CommuteBuffer))
	}

	m.Form.TagsInput.SetValue(strings.Join(task.Tags, ", "))

	// Reset recurring form fields first
	m.Form.IsRecurringIdx = 0
	m.Form.RecurringEndDateInput.SetValue("")
	m.Form.RecurringDaysInput.SetValue("")
	for i := range m.Form.RecurringDaysSelected {
		m.Form.RecurringDaysSelected[i] = false
	}

	// Populate recurring fields if editing a recurring task/habit
	if task.RecurringParentUUID != "" || task.SchedulingType == model.Habit {
		m.Form.IsRecurringIdx = 1
		var maxEnd time.Time
		weekdays := make(map[time.Weekday]bool)
		for _, tVal := range m.Tasks {
			if tVal.RecurringParentUUID == task.RecurringParentUUID && task.RecurringParentUUID != "" {
				if tVal.TimeWindow.Start.After(maxEnd) {
					maxEnd = tVal.TimeWindow.Start
				}
				weekdays[tVal.TimeWindow.Start.Weekday()] = true
			}
		}
		// If no other parent instances, use default values from task or active day
		if maxEnd.IsZero() {
			if !task.TimeWindow.Start.IsZero() {
				maxEnd = task.TimeWindow.Start.AddDate(0, 0, 7)
				weekdays[task.TimeWindow.Start.Weekday()] = true
			} else {
				maxEnd = m.SelectedDay.AddDate(0, 0, 7)
				weekdays[m.SelectedDay.Weekday()] = true
			}
		}
		m.Form.RecurringEndDateInput.SetValue(maxEnd.Format("2006-01-02"))

		var dayNames []string
		orderedDays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
		shortNames := map[time.Weekday]string{
			time.Monday:    "mon",
			time.Tuesday:   "tue",
			time.Wednesday: "wed",
			time.Thursday:  "thu",
			time.Friday:    "fri",
			time.Saturday:  "sat",
			time.Sunday:    "sun",
		}
		for i, d := range orderedDays {
			m.Form.RecurringDaysSelected[i] = weekdays[d]
			if weekdays[d] {
				dayNames = append(dayNames, shortNames[d])
			}
		}
		m.Form.RecurringDaysInput.SetValue(strings.Join(dayNames, ", "))
	}

	m.Form.ActiveField = 0
	m.focusFormFields()
	m.CurrentMode = ModeForm
}
