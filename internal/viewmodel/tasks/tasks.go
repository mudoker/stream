package tasks

import (
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

func ImportSort(tasks []model.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].SortingWeight() > tasks[i].SortingWeight() {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func GetDayTasks(allTasks []model.Task, day time.Time) []model.Task {
	var list []model.Task
	for _, t := range allTasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, day) {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TimeWindow.Start.Before(list[j].TimeWindow.Start)
	})
	return list
}

func GetTodoShelfTasks(allTasks []model.Task) []model.Task {
	var reminders []model.Task
	var habits []model.Task
	var backlog []model.Task
	for _, t := range allTasks {
		if t.LifecycleState == model.StateCompleted && t.SchedulingType != model.Habit {
			continue
		}
		if t.SchedulingType == model.Reminder {
			reminders = append(reminders, t)
		} else if t.SchedulingType == model.Habit {
			habits = append(habits, t)
		} else if t.SchedulingType == model.Floating {
			backlog = append(backlog, t)
		}
	}
	SortReminders(reminders)
	ImportSort(habits)
	ImportSort(backlog)
	res := append(reminders, habits...)
	return append(res, backlog...)
}

func sameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}
