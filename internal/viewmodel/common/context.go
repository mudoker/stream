package common

import (
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/timer"
)

type ModelContext interface {
	AddTask(task model.Task)
	UpdateTask(task model.Task)
	DeleteTask(uuid string)
	RefreshTasks()
	SetStatusMsg(msg string)
	SetConfirmOpen(open bool)
	SetConfirmActionType(actionType string)
	SetConfirmTask(task model.Task)
	SetConfirmSelectedIndex(idx int)
	GetConfirmTask() model.Task
	GetConfirmSelectedIndex() int
	GetConfirmFocusArea() int
	SetConfirmFocusArea(area int)
	GetZenTimer() *timer.ZenTimer
	SetZenTimer(zt *timer.ZenTimer)
	GetTasks() []model.Task
	SetTasks(tasks []model.Task)
	IsDetailOpen() bool
	SetDetailOpen(open bool)
	GetDetailTask() model.Task
	SetDetailTask(task model.Task)
	AdjustSelectionBeforeDeletion(uuid string)
	TriggerGCalPushIfAnchored(task model.Task)
	AutoScrollToSelectedTask()
	SetSelectedDay(day time.Time)
	GetSelectedDay() time.Time
	SetCurrentMode(mode string)
	SetLogSessionPromptOpen(open bool)
	SetLogSessionPromptTask(task model.Task)
	InitLogSessionInputs(focusMins, breakMins int)
}
