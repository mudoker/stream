package tui

import (
	"time"

	"stream/internal/model"
)

type ZenTimer struct {
	Task              model.Task
	Sessions          []Session
	CurrentSessionIdx int
	TimeRemaining     time.Duration
	TotalDuration     time.Duration // Total duration of the current session
	IsPaused          bool
	Running           bool
}

func NewZenTimer(t model.Task) *ZenTimer {
	dur := time.Duration(t.StoryPoints) * 45 * time.Minute
	if t.SchedulingType == model.Anchored {
		dur = t.TimeWindow.End.Sub(t.TimeWindow.Start)
	}

	elapsed := time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds) * time.Second
	dur -= elapsed
	if dur < 0 {
		dur = 0
	}

	sessions := PartitionTask(dur)
	var timeRemaining time.Duration
	if len(sessions) > 0 {
		timeRemaining = sessions[0].Duration
	}

	return &ZenTimer{
		Task:              t,
		Sessions:          sessions,
		CurrentSessionIdx: 0,
		TimeRemaining:     timeRemaining,
		TotalDuration:     timeRemaining,
		IsPaused:          false,
		Running:           true,
	}
}

func (zt *ZenTimer) RecordElapsedTimes() int {
	if zt.CurrentSessionIdx >= 0 && zt.CurrentSessionIdx < len(zt.Sessions) {
		sess := zt.Sessions[zt.CurrentSessionIdx]
		elapsed := zt.TotalDuration - zt.TimeRemaining
		if elapsed > 0 {
			elapsedSecs := int(elapsed.Seconds())
			if sess.Type == FocusSession {
				zt.Task.ExecutionMetrics.ElapsedFocusSeconds += elapsedSecs
			} else if sess.Type == BreakSession {
				zt.Task.ExecutionMetrics.ElapsedBreakSeconds += elapsedSecs
			}
			zt.TotalDuration = zt.TimeRemaining
			return elapsedSecs
		}
	}
	return 0
}

func (zt *ZenTimer) Tick() bool {
	if zt.IsPaused || !zt.Running {
		return false
	}

	zt.TimeRemaining -= time.Second
	if zt.TimeRemaining <= 0 {
		zt.RecordElapsedTimes()

		if zt.CurrentSessionIdx >= 0 && zt.CurrentSessionIdx < len(zt.Sessions) {
			if zt.Sessions[zt.CurrentSessionIdx].Type == FocusSession {
				zt.Task.ExecutionMetrics.TotalCompletedPomodoros++
			}
		}

		return zt.NextSession()
	}
	return false
}

func (zt *ZenTimer) NextSession() bool {
	zt.CurrentSessionIdx++
	if zt.CurrentSessionIdx >= len(zt.Sessions) {
		zt.Running = false
		zt.TimeRemaining = 0
		return true
	}
	zt.TimeRemaining = zt.Sessions[zt.CurrentSessionIdx].Duration
	zt.TotalDuration = zt.TimeRemaining
	return false
}

func (zt *ZenTimer) AddTime(d time.Duration) {
	zt.TimeRemaining += d
	if zt.TimeRemaining < 0 {
		zt.TimeRemaining = 0
	}
	zt.TotalDuration += d
	if zt.TotalDuration < 0 {
		zt.TotalDuration = 0
	}
}

func (zt *ZenTimer) UpdateTaskDuration(newTask model.Task) {
	zt.Task = newTask
	var newDur time.Duration
	if newTask.SchedulingType == model.Anchored {
		newDur = newTask.TimeWindow.End.Sub(newTask.TimeWindow.Start)
	} else {
		newDur = time.Duration(newTask.StoryPoints) * 45 * time.Minute
	}

	elapsedTotal := time.Duration(0)
	for i := 0; i < zt.CurrentSessionIdx; i++ {
		elapsedTotal += zt.Sessions[i].Duration
	}
	elapsedCurrent := zt.TotalDuration - zt.TimeRemaining
	elapsedTotal += elapsedCurrent

	if newDur <= elapsedTotal {
		zt.Sessions[zt.CurrentSessionIdx].Duration = elapsedCurrent
		zt.TotalDuration = elapsedCurrent
		zt.TimeRemaining = 0
		zt.Sessions = zt.Sessions[:zt.CurrentSessionIdx+1]
		zt.Running = false
		return
	}

	remainingToSchedule := newDur - elapsedTotal
	if remainingToSchedule <= zt.TimeRemaining {
		zt.TimeRemaining = remainingToSchedule
		zt.TotalDuration = elapsedCurrent + remainingToSchedule
		zt.Sessions[zt.CurrentSessionIdx].Duration = zt.TotalDuration
		zt.Sessions = zt.Sessions[:zt.CurrentSessionIdx+1]
	} else {
		subRemaining := remainingToSchedule - zt.TimeRemaining
		keepCount := zt.CurrentSessionIdx + 1

		for j := zt.CurrentSessionIdx + 1; j < len(zt.Sessions); j++ {
			if subRemaining <= 0 {
				break
			}
			if zt.Sessions[j].Duration <= subRemaining {
				subRemaining -= zt.Sessions[j].Duration
				keepCount++
			} else {
				zt.Sessions[j].Duration = subRemaining
				subRemaining = 0
				keepCount++
				break
			}
		}
		zt.Sessions = zt.Sessions[:keepCount]

		if subRemaining > 0 {
			extraSessions := PartitionTask(subRemaining)
			zt.Sessions = append(zt.Sessions, extraSessions...)
		}
	}
}
