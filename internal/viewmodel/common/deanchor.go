package common

import (
	"fmt"
	"time"

	"stream/internal/model"
)

func ConfirmDeanchor(ctx ModelContext, task model.Task) {
	if task.SchedulingType == model.Habit {
		task.TimeWindow = model.TimeWindow{} // clear time window to deanchor
	} else {
		task.SchedulingType = model.Floating
	}
	task.LifecycleState = model.StateReady
	task.UpdatedAt = time.Now()
	ctx.UpdateTask(task)
	ctx.RefreshTasks()
	ctx.TriggerGCalPushIfAnchored(task)
	ctx.SetStatusMsg(fmt.Sprintf("Task '%s' de-anchored to backlog.", task.Title))
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
}
