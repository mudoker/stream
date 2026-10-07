package tasks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"stream/internal/model"
)

func SortReminders(tasks []model.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].TimeWindow.Start.Before(tasks[i].TimeWindow.Start) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if tasks[j].TimeWindow.Start.Equal(tasks[i].TimeWindow.Start) {
				if tasks[j].SortingWeight() > tasks[i].SortingWeight() {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

func getPriorityVal(p model.Priority) int {
	switch p {
	case model.P0:
		return 4
	case model.P1:
		return 3
	case model.P2:
		return 2
	case model.P3:
		return 1
	default:
		return 0
	}
}

func ImportSort(tasks []model.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		pI := getPriorityVal(tasks[i].Priority)
		pJ := getPriorityVal(tasks[j].Priority)
		if pI != pJ {
			return pI > pJ
		}
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].UUID < tasks[j].UUID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
}

func GetDayTasks(allTasks []model.Task, day time.Time, isCloneMove ...bool) []model.Task {
	clones := make(map[string]bool)
	isClone := len(isCloneMove) > 0 && isCloneMove[0]
	if !isClone {
		for _, t := range allTasks {
			if strings.HasSuffix(t.UUID, "_moving") {
				clones[strings.TrimSuffix(t.UUID, "_moving")] = true
			} else if strings.HasSuffix(t.UUID, "_adjusting") {
				clones[strings.TrimSuffix(t.UUID, "_adjusting")] = true
			}
		}
	}

	var list []model.Task
	for _, t := range allTasks {
		if clones[t.UUID] {
			continue
		}
		if model.IsTaskAnchored(t) && sameDay(t.TimeWindow.Start, day) {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TimeWindow.Start.Before(list[j].TimeWindow.Start)
	})
	return list
}

func GetTodoShelfTasks(allTasks []model.Task, selectedDay time.Time) []model.Task {
	if selectedDay.IsZero() {
		selectedDay = time.Now()
	}

	var reminders []model.Task
	var habits []model.Task
	var backlog []model.Task
	var completed []model.Task

	recurringCounts := make(map[string]int)
	recurringInstances := make(map[string]model.Task)

	for _, t := range allTasks {
		// Anchored habits (has start time and end time) should not show on the todo shelf anymore
		if t.SchedulingType == model.Habit && !t.TimeWindow.Start.IsZero() && !t.TimeWindow.End.IsZero() {
			continue
		}

		// Filter un-anchored habits by the selectedDay if they are bound to a date
		if t.SchedulingType == model.Habit && !t.TimeWindow.Start.IsZero() {
			if !sameDay(t.TimeWindow.Start, selectedDay) {
				continue
			}
		}

		isDone := false
		if t.SchedulingType == model.Habit {
			dateStr := selectedDay.Format("2006-01-02")
			for _, d := range t.CompletedDates {
				if d == dateStr {
					isDone = true
					break
				}
			}
		} else {
			isDone = t.LifecycleState == model.StateCompleted
		}

		if isDone {
			if t.SchedulingType == model.Reminder {
				continue
			}
			// Only show completed tasks completed or initiated on this selected day
			isCompletedToday := sameDay(t.UpdatedAt, selectedDay) || sameDay(t.GetInitiateDate(), selectedDay)
			if t.SchedulingType == model.Habit {
				isCompletedToday = false
				dateStr := selectedDay.Format("2006-01-02")
				for _, d := range t.CompletedDates {
					if d == dateStr {
						isCompletedToday = true
						break
					}
				}
			}
			if isCompletedToday {
				completed = append(completed, t)
			}
			continue
		}

		if t.SchedulingType == model.Reminder {
			reminders = append(reminders, t)
		} else if t.SchedulingType == model.Habit {
			if t.RecurringParentUUID != "" {
				recurringCounts[t.RecurringParentUUID]++
				if _, exists := recurringInstances[t.RecurringParentUUID]; !exists {
					recurringInstances[t.RecurringParentUUID] = t
				}
			} else {
				habits = append(habits, t)
			}
		} else if t.SchedulingType == model.Floating || t.AddedToToday {
			// Day view shelf: only show tasks linked to that day
			// 1. Explicitly added to today
			// 2. InitiateDate matches selectedDay
			// 3. Fallback: CreatedAt matches selectedDay if InitiateDate is zero
			// 4. Fallback for tests/legacy without dates: both InitiateDate and CreatedAt are zero
			var isForDay bool
			if t.AddedToToday && sameDay(time.Now(), selectedDay) {
				isForDay = true
			} else if !t.InitiateDate.IsZero() {
				isForDay = sameDay(t.InitiateDate, selectedDay)
			} else if !t.CreatedAt.IsZero() {
				isForDay = sameDay(t.CreatedAt, selectedDay)
			} else {
				isForDay = true
			}

			if isForDay {
				if t.RecurringParentUUID != "" {
					recurringCounts[t.RecurringParentUUID]++
					if _, exists := recurringInstances[t.RecurringParentUUID]; !exists {
						recurringInstances[t.RecurringParentUUID] = t
					}
				} else {
					backlog = append(backlog, t)
				}
			}
		}
	}

	for parentUUID, count := range recurringCounts {
		task := recurringInstances[parentUUID]
		if count > 1 {
			task.Title = fmt.Sprintf("%s (%d)", task.Title, count)
		}
		if task.SchedulingType == model.Habit {
			habits = append(habits, task)
		} else {
			backlog = append(backlog, task)
		}
	}

	SortReminders(reminders)
	ImportSort(habits)
	ImportSort(backlog)
	ImportSort(completed)

	res := append(reminders, habits...)
	res = append(res, backlog...)
	return append(res, completed...)
}

func GetGlobalBacklogShelfTasks(allTasks []model.Task, selectedDay time.Time) []model.Task {
	// Global backlog contains ALL floating tasks not assigned to a sprint across all dates.
	// Habits, events, and reminders are not tasks and appear directly on the day timeline.
	var backlog []model.Task
	var completed []model.Task

	for _, t := range allTasks {
		if t.SchedulingType != model.Floating {
			continue
		}
		if t.SprintUUID != "" {
			continue
		}
		if model.IsTaskAnchored(t) {
			continue
		}

		isDone := t.LifecycleState == model.StateCompleted
		if isDone {
			completed = append(completed, t)
		} else {
			backlog = append(backlog, t)
		}
	}

	ImportSort(backlog)
	ImportSort(completed)

	return append(backlog, completed...)
}

func GetSprintSwimlaneTasks(allTasks []model.Task, sprintUUID string) (defined, inProgress, review, testing, completed []model.Task) {
	for _, t := range allTasks {
		// Habits, events, and reminders are not tasks and cannot belong to a sprint
		if t.SchedulingType != model.Floating && !t.AddedToToday {
			continue
		}
		if t.SprintUUID != sprintUUID {
			continue
		}
		switch t.LifecycleState {
		case model.StateCompleted:
			completed = append(completed, t)
		case model.StateTesting:
			testing = append(testing, t)
		case model.StateReview:
			review = append(review, t)
		case model.StateActive, model.StateScheduled, model.StatePaused:
			inProgress = append(inProgress, t)
		default: // StateBacklog, StateReady, StateOverdue, etc.
			defined = append(defined, t)
		}
	}
	ImportSort(defined)
	ImportSort(inProgress)
	ImportSort(review)
	ImportSort(testing)
	ImportSort(completed)
	return defined, inProgress, review, testing, completed
}

func SameDay(a, b time.Time) bool {
	return sameDay(a, b)
}

func sameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}

