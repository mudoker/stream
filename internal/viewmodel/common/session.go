package common

import (
	"fmt"
	"time"

	"stream/internal/model"

	"github.com/google/uuid"
)

func InitiateLogSession(ctx ModelContext, task model.Task) {
	ctx.SetLogSessionPromptTask(task)

	var focusMins int
	if task.ExecutionMetrics.ElapsedFocusSeconds > 0 {
		focusMins = (task.ExecutionMetrics.ElapsedFocusSeconds + 30) / 60
	} else {
		if task.SchedulingType == model.Anchored || task.SchedulingType == model.Event {
			focusMins = int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
		} else {
			focusMins = task.StoryPoints * 45
		}
		if focusMins <= 0 {
			focusMins = 60
		}
	}

	var breakMins int
	if task.ExecutionMetrics.ElapsedBreakSeconds > 0 {
		breakMins = (task.ExecutionMetrics.ElapsedBreakSeconds + 30) / 60
	} else {
		breakMins = 0
	}

	ctx.InitLogSessionInputs(focusMins, breakMins)

	ctx.SetLogSessionPromptOpen(true)
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
	ctx.SetStatusMsg("Enter focus and break minutes to log session.")
}

func CancelLogSession(ctx ModelContext, task model.Task) {
	task.LifecycleState = model.StateCompleted
	task.UpdatedAt = time.Now()
	ctx.UpdateTask(task)
	ctx.RefreshTasks()
	if ctx.IsDetailOpen() && ctx.GetDetailTask().UUID == task.UUID {
		ctx.SetDetailOpen(false)
	}
	ctx.SetStatusMsg(fmt.Sprintf("Task '%s' completed without logging focus time.", task.Title))
	ctx.SetConfirmOpen(false)
	ctx.SetConfirmActionType("")
}

func HandleExitFocusOption(ctx ModelContext, index int) {
	zt := ctx.GetZenTimer()
	if zt == nil {
		ctx.SetConfirmOpen(false)
		ctx.SetConfirmActionType("")
		return
	}

	zt.RecordElapsedTimes()
	t := zt.Task

	switch index {
	case 0: // Mark as complete
		t.LifecycleState = model.StateCompleted
		t.UpdatedAt = time.Now()
		ctx.UpdateTask(t)
		ctx.RefreshTasks()
		ctx.SetZenTimer(nil)
		ctx.SetCurrentMode("NORMAL") // ModeNormal
		ctx.SetConfirmOpen(false)
		ctx.SetConfirmActionType("")
		ctx.SetStatusMsg(fmt.Sprintf("Task '%s' marked as completed.", t.Title))

	case 1: // Complete and resume
		originalDur := time.Duration(t.StoryPoints) * 45 * time.Minute
		if model.IsTaskAnchored(t) {
			originalDur = t.TimeWindow.End.Sub(t.TimeWindow.Start)
		}
		elapsedFocus := time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds) * time.Second
		remainingDur := originalDur - elapsedFocus
		if remainingDur < 15*time.Minute {
			remainingDur = 15 * time.Minute
		}

		if model.IsTaskAnchored(t) {
			t.TimeWindow.End = time.Now()
			if t.TimeWindow.End.Before(t.TimeWindow.Start) {
				t.TimeWindow.End = t.TimeWindow.Start.Add(1 * time.Minute)
			}
		}
		t.LifecycleState = model.StateCompleted
		t.UpdatedAt = time.Now()
		ctx.UpdateTask(t)

		sp := int((remainingDur + 44*time.Minute) / (45 * time.Minute))
		if sp < 1 {
			sp = 1
		}

		newTask := model.Task{
			UUID:           uuid.New().String(),
			WorkspaceUUID:  t.WorkspaceUUID,
			Title:          t.Title + " (Resume)",
			Description:    t.Description,
			Priority:       t.Priority,
			StoryPoints:    sp,
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		ctx.AddTask(newTask)
		ctx.RefreshTasks()

		ctx.SetZenTimer(nil)
		ctx.SetCurrentMode("NORMAL") // ModeNormal
		ctx.SetConfirmOpen(false)
		ctx.SetConfirmActionType("")
		ctx.SetStatusMsg(fmt.Sprintf("Completed '%s' and created resuming task '%s'.", t.Title, newTask.Title))

	case 2: // Discard session changes
		t.LifecycleState = model.StateReady
		t.UpdatedAt = time.Now()
		ctx.UpdateTask(t)
		ctx.RefreshTasks()
		ctx.SetZenTimer(nil)
		ctx.SetCurrentMode("NORMAL") // ModeNormal
		ctx.SetConfirmOpen(false)
		ctx.SetConfirmActionType("")
		ctx.SetStatusMsg("Focus session discarded.")
	}
}
