package common

import (
	"fmt"
	"time"

	"stream/internal/model"
)

func CompleteReminder(ctx ModelContext, task model.Task) {
	task.LifecycleState = model.StateCompleted
	task.UpdatedAt = time.Now()
	ctx.UpdateTask(task)
	ctx.RefreshTasks()
	if ctx.IsDetailOpen() && ctx.GetDetailTask().UUID == task.UUID {
		ctx.SetDetailOpen(false)
	}
	ctx.SetStatusMsg(fmt.Sprintf("Reminder '%s' completed!", task.Title))
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
}

func ToggleTaskCompletion(ctx ModelContext, task model.Task, selectedDay time.Time) {
	if task.SchedulingType == model.Habit {
		dateStr := selectedDay.Format("2006-01-02")
		foundIdx := -1
		for idx, d := range task.CompletedDates {
			if d == dateStr {
				foundIdx = idx
				break
			}
		}
		if foundIdx != -1 {
			// Toggle off: remove from CompletedDates
			task.CompletedDates = append(task.CompletedDates[:foundIdx], task.CompletedDates[foundIdx+1:]...)
			if len(task.CompletedDates) == 0 {
				task.LifecycleState = model.StateReady
			}
			ctx.SetStatusMsg(fmt.Sprintf("Habit '%s' marked incomplete for %s.", task.Title, dateStr))
		} else {
			// Toggle on: add to CompletedDates
			task.CompletedDates = append(task.CompletedDates, dateStr)
			task.LifecycleState = model.StateCompleted
			ctx.SetStatusMsg(fmt.Sprintf("Habit '%s' completed for %s!", task.Title, dateStr))
		}
		task.UpdatedAt = time.Now()
		ctx.UpdateTask(task)
		ctx.RefreshTasks()
	} else {
		if task.LifecycleState == model.StateCompleted {
			task.LifecycleState = model.StateBacklog
			ctx.SetStatusMsg(fmt.Sprintf("Task '%s' marked incomplete.", task.Title))
			task.UpdatedAt = time.Now()
			ctx.UpdateTask(task)
			ctx.RefreshTasks()
		} else {
			if task.SchedulingType == model.Reminder {
				ctx.SetConfirmTask(task)
				ctx.SetConfirmOpen(true)
				ctx.SetConfirmActionType("complete_reminder")
				ctx.SetConfirmSelectedIndex(0)
			} else {
				if ctx.GetZenTimer() == nil || ctx.GetZenTimer().Task.UUID != task.UUID {
					ctx.SetConfirmTask(task)
					ctx.SetConfirmOpen(true)
					ctx.SetConfirmActionType("log_session_confirm")
					ctx.SetConfirmSelectedIndex(0)
				} else {
					task.LifecycleState = model.StateCompleted
					ctx.SetStatusMsg(fmt.Sprintf("Task '%s' completed!", task.Title))
					task.UpdatedAt = time.Now()
					ctx.UpdateTask(task)
					ctx.RefreshTasks()
					ctx.SetZenTimer(nil)
				}
			}
		}
	}
}
