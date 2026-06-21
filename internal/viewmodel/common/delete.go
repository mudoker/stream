package common

import (
	"fmt"

	"stream/internal/model"
)

func DeleteTaskOccurrence(ctx ModelContext, task model.Task) {
	ctx.AdjustSelectionBeforeDeletion(task.UUID)
	ctx.DeleteTask(task.UUID)
	ctx.RefreshTasks()
	ctx.TriggerGCalPushIfAnchored(task)
	if ctx.IsDetailOpen() && ctx.GetDetailTask().UUID == task.UUID {
		ctx.SetDetailOpen(false)
	}
	if zt := ctx.GetZenTimer(); zt != nil && zt.Task.UUID == task.UUID {
		ctx.SetZenTimer(nil)
	}
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
	ctx.SetStatusMsg(fmt.Sprintf("Task '%s' deleted.", task.Title))
}

func DeleteAllOccurrences(ctx ModelContext, task model.Task, allTasks []model.Task) {
	var tasksToDelete []string
	for _, t := range allTasks {
		if t.RecurringParentUUID == task.RecurringParentUUID {
			if t.UUID == task.UUID || !t.TimeWindow.Start.Before(task.TimeWindow.Start) {
				tasksToDelete = append(tasksToDelete, t.UUID)
			}
		}
	}
	ctx.AdjustSelectionBeforeDeletion(task.UUID)
	for _, uid := range tasksToDelete {
		ctx.DeleteTask(uid)
	}
	ctx.RefreshTasks()
	if ctx.IsDetailOpen() && ctx.GetDetailTask().UUID == task.UUID {
		ctx.SetDetailOpen(false)
	}
	if zt := ctx.GetZenTimer(); zt != nil && zt.Task.UUID == task.UUID {
		ctx.SetZenTimer(nil)
	}
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
	ctx.SetStatusMsg("This and all future occurrences deleted.")
}

func InitiateDeleteTask(ctx ModelContext, task model.Task) {
	ctx.SetConfirmTask(task)
	ctx.SetConfirmOpen(true)
	ctx.SetConfirmSelectedIndex(0)
	if task.RecurringParentUUID != "" {
		ctx.SetConfirmActionType("delete_recurring")
	} else {
		ctx.SetConfirmActionType("delete")
	}
}
