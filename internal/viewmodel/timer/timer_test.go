package timer

import (
	"testing"
	"time"

	"stream/internal/model"
)

func TestNewZenTimerAndOperations(t *testing.T) {
	task := model.Task{
		UUID:        "timer-task",
		Title:       "Timer Task",
		StoryPoints: 3, // 135 minutes -> 6 sessions (50m focus, 10m break, 50m focus, 10m break, 35m focus, 5m break)
	}

	zt := NewZenTimer(task)
	if zt.Task.UUID != "timer-task" {
		t.Errorf("expected task UUID to be timer-task")
	}
	if len(zt.Sessions) != 6 {
		t.Errorf("expected 6 sessions, got %d", len(zt.Sessions))
	}

	// Test Tick focus session
	startRemaining := zt.TimeRemaining
	zt.Tick()
	if zt.TimeRemaining != startRemaining-time.Second {
		t.Errorf("expected time remaining to decrease by 1 second, got %v", zt.TimeRemaining)
	}

	// Test Add Time
	zt.AddTime(5 * time.Minute)
	if zt.TimeRemaining != startRemaining-time.Second+5*time.Minute {
		t.Errorf("expected time remaining to increase after AddTime")
	}

	// Test Next Session
	zt.NextSession()
	if zt.CurrentSessionIdx != 1 {
		t.Errorf("expected next session index to be 1, got %d", zt.CurrentSessionIdx)
	}

	// Test Update Duration
	task.StoryPoints = 4
	zt.UpdateTaskDuration(task)
	if len(zt.Sessions) != 7 {
		t.Errorf("expected sessions to be updated to 7, got %d", len(zt.Sessions))
	}

	// Test RecordElapsedTimes
	zt.TimeRemaining -= 10 * time.Second
	recorded := zt.RecordElapsedTimes()
	if recorded != 10 {
		t.Errorf("expected 10 seconds recorded, got %d", recorded)
	}
	if zt.Task.ExecutionMetrics.ElapsedBreakSeconds != 10 {
		t.Errorf("expected 10 seconds elapsed break time, got %d", zt.Task.ExecutionMetrics.ElapsedBreakSeconds)
	}
}

func TestTimerEdgeCases(t *testing.T) {
	// 1. Partition leftover total <= 0
	sessEmpty := PartitionTask(0)
	if len(sessEmpty) != 2 || sessEmpty[0].Type != FocusSession {
		t.Errorf("expected default sessions for zero duration")
	}

	// 2. Partition small duration < 25m
	sessSmall := PartitionTask(15 * time.Minute)
	if len(sessSmall) != 1 || sessSmall[0].Duration != 15*time.Minute {
		t.Errorf("expected 1 session of 15m")
	}

	// 3. Ticking when paused
	zt := NewZenTimer(model.Task{StoryPoints: 1})
	zt.IsPaused = true
	if zt.Tick() {
		t.Errorf("expected false when ticking paused timer")
	}
	zt.IsPaused = false
	zt.Running = false
	if zt.Tick() {
		t.Errorf("expected false when ticking stopped timer")
	}

	// 4. Timer expiry and session completion
	zt2 := NewZenTimer(model.Task{StoryPoints: 1})
	zt2.TimeRemaining = time.Second
	finished := zt2.Tick()
	if finished {
		t.Errorf("expected false since there are more sessions (break session)")
	}
	if zt2.Task.ExecutionMetrics.TotalCompletedPomodoros != 1 {
		t.Errorf("expected 1 completed pomodoro")
	}

	// 5. Shrinking duration (new duration <= elapsed total)
	zt3 := NewZenTimer(model.Task{StoryPoints: 2})
	zt3.TimeRemaining = 0
	zt3.UpdateTaskDuration(model.Task{StoryPoints: 0})
	if zt3.Running {
		t.Errorf("expected timer to stop running after duration shrinks below elapsed")
	}

	// 6. AddTime with negative values
	zt4 := NewZenTimer(model.Task{StoryPoints: 1})
	zt4.AddTime(-100 * time.Hour)
	if zt4.TimeRemaining != 0 || zt4.TotalDuration != 0 {
		t.Errorf("expected AddTime to clamp at 0")
	}
}

func TestTimerMoreEdgeCases(t *testing.T) {
	// 1. NewZenTimer with Anchored task
	now := time.Now()
	taskAnchored := model.Task{
		UUID:           "anchored-task",
		Title:          "Anchored Task",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: now,
			End:   now.Add(90 * time.Minute),
		},
	}
	zt := NewZenTimer(taskAnchored)
	// 90 minutes should yield 4 sessions (50m focus, 10m break, 40m focus, 5m break)
	if len(zt.Sessions) != 4 {
		t.Errorf("expected 4 sessions for 90m Anchored task, got %d", len(zt.Sessions))
	}

	// 2. NewZenTimer where elapsed times exceed total sessions
	taskElapsedOverflow := model.Task{
		UUID:        "overflow-task",
		Title:       "Overflow Task",
		StoryPoints: 1, // 45m -> 1 focus session
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 3600, // 60m focus elapsed (exceeds 45m!)
		},
	}
	ztOverflow := NewZenTimer(taskElapsedOverflow)
	if !ztOverflow.Running || ztOverflow.CurrentSessionIdx != 1 || ztOverflow.Sessions[ztOverflow.CurrentSessionIdx].Type != BreakSession {
		t.Errorf("expected break session to be running since focus session is fully elapsed, got %+v", ztOverflow)
	}

	// 3. UpdateTaskDuration with Anchored task
	zt.UpdateTaskDuration(model.Task{
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: now,
			End:   now.Add(180 * time.Minute),
		},
	})

	// 4. UpdateTaskDuration with longer duration requiring extra sessions
	zt2 := NewZenTimer(model.Task{StoryPoints: 1}) // 45m
	zt2.UpdateTaskDuration(model.Task{StoryPoints: 3}) // 135m -> expands sessions
	if len(zt2.Sessions) < 3 {
		t.Errorf("expected sessions to expand, got %d", len(zt2.Sessions))
	}
}
