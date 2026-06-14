package tasks

import (
	"fmt"
	"sort"
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

func GetDayTasks(allTasks []model.Task, day time.Time) []model.Task {
	var list []model.Task
	for _, t := range allTasks {
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
	var reminders []model.Task
	var habits []model.Task
	var backlog []model.Task
	var completed []model.Task

	recurringCounts := make(map[string]int)
	recurringInstances := make(map[string]model.Task)

	for _, t := range allTasks {
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
			completed = append(completed, t)
			continue
		}

		if t.SchedulingType == model.Reminder {
			reminders = append(reminders, t)
		} else if t.SchedulingType == model.Habit {
			isAnchoredOnSelectedDay := !t.TimeWindow.Start.IsZero() && sameDay(t.TimeWindow.Start, selectedDay)
			if !isAnchoredOnSelectedDay {
				if t.RecurringParentUUID != "" {
					recurringCounts[t.RecurringParentUUID]++
					if _, exists := recurringInstances[t.RecurringParentUUID]; !exists {
						recurringInstances[t.RecurringParentUUID] = t
					}
				} else {
					habits = append(habits, t)
				}
			}
		} else if t.SchedulingType == model.Floating {
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

func sameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}
