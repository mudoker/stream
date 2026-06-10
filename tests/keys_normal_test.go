package tests

import (
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTaskMoveModeWorkflow(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
	}

	m.EnterTaskMoveMode()
	if m.CurrentMode != viewmodel.ModeTaskMove {
		t.Fatal("expected mode to be ModeTaskMove")
	}
	if m.TaskMoveOriginalTimeWindow != task.TimeWindow {
		t.Fatal("expected original time window to be recorded")
	}

	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.TaskMovePrefix != "2" {
		t.Fatalf("expected prefix to be '2', got %q", m.TaskMovePrefix)
	}

	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.TaskMovePrefix != "" {
		t.Fatalf("expected prefix to reset after move, got %q", m.TaskMovePrefix)
	}

	// The original task should still be at its original location (10:00)
	if !m.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("expected original task start to remain unchanged (10:00), got %s", m.Tasks[0].TimeWindow.Start)
	}

	// The moving clone task should be at index 1 and moved to 10:30
	if len(m.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in memory during move, got %d", len(m.Tasks))
	}
	clone := m.Tasks[1]
	if clone.UUID != "task-1_moving" {
		t.Fatalf("expected clone UUID to be task-1_moving, got %s", clone.UUID)
	}
	if !clone.TimeWindow.Start.Equal(start.Add(30 * time.Minute)) {
		t.Fatalf("expected clone start to be 10:30, got %s", clone.TimeWindow.Start)
	}

	// Press Enter to confirm move
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// After confirmation, only the original task should remain and it should be updated
	if len(m.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory after confirm, got %d", len(m.Tasks))
	}
	if !m.Tasks[0].TimeWindow.Start.Equal(start.Add(30 * time.Minute)) {
		t.Fatalf("expected original task start to be updated to 10:30, got %s", m.Tasks[0].TimeWindow.Start)
	}
}

func TestTaskMoveModeDoesNotTriggerPriorityOverlapCollision(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-1",
		Priority:       model.P0,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}
	otherTask := model.Task{
		UUID:           "task-2",
		Priority:       model.P1,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start.Add(30 * time.Minute),
			End:   start.Add(90 * time.Minute),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task, otherTask},
		SelectedTaskUUID: "task-1",
	}

	m.EnterTaskMoveMode()

	if m.HasPriorityOverlapCollision(m.Tasks[0]) {
		t.Fatal("expected original task collision warning to be disabled while moving")
	}
	if m.HasPriorityOverlapCollision(m.Tasks[2]) {
		t.Fatal("expected moving clone collision warning to be disabled while moving")
	}
}

func TestMovingCloneDoesNotSelfCollideWithOriginalTask(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-1",
		Priority:       model.P0,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}
	clone := task
	clone.UUID = "task-1_moving"

	m := &viewmodel.Model{
		CurrentMode: viewmodel.ModeNormal,
		Tasks:       []model.Task{task, clone},
	}

	if m.HasPriorityOverlapCollision(task) {
		t.Fatal("expected moving clone to be ignored as the same logical task")
	}
}

func TestPriorityOverlapCollisionTriggersAfterConfirm(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-1",
		Priority:       model.P0,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}
	otherTask := model.Task{
		UUID:           "task-2",
		Priority:       model.P2,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start.Add(65 * time.Minute),
			End:   start.Add(2 * time.Hour),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task, otherTask},
		SelectedTaskUUID: "task-1",
	}

	m.EnterTaskMoveMode()
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.HasPriorityOverlapCollision(m.Tasks[0]) {
		t.Fatal("expected priority overlap collision to trigger after confirming move")
	}
}

func TestTaskMoveModeCancel(t *testing.T) {
	start := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-2",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-2",
	}

	m.EnterTaskMoveMode()
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Fatal("expected mode to return to ModeNormal after cancel")
	}
	if !m.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("expected start to revert to original 12:00, got %s", m.Tasks[0].TimeWindow.Start)
	}
}

func TestQuickAnchorDeAnchorWorkflow(t *testing.T) {
	// 1. Test De-anchoring an Anchored task
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-1",
		Title:          "Task 1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
		LifecycleState: model.StateScheduled,
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
	}

	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if m.Tasks[0].SchedulingType != model.Floating {
		t.Fatal("expected task to be de-anchored to Floating")
	}
	if m.Tasks[0].LifecycleState != model.StateReady {
		t.Fatal("expected de-anchored task to have StateReady")
	}

	// 2. Test Anchoring a Floating task
	task2 := model.Task{
		UUID:           "task-2",
		Title:          "Task 2",
		SchedulingType: model.Floating,
		StoryPoints:    2,
		LifecycleState: model.StateReady,
	}

	m2 := &viewmodel.Model{
		Tasks:            []model.Task{task2},
		SelectedTaskUUID: "task-2",
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
	}

	m2.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if !m2.AnchorPromptOpen {
		t.Fatal("expected AnchorPromptOpen to be true")
	}
	if m2.AnchorPromptTask.UUID != "task-2" {
		t.Fatalf("expected AnchorPromptTask UUID to be task-2, got %s", m2.AnchorPromptTask.UUID)
	}

	m2.AnchorTimeInput.SetValue("14:30")

	resTabRaw, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	m2Tab := resTabRaw.(*viewmodel.Model)
	if m2Tab.AnchorActiveField != 1 {
		t.Fatalf("expected active field to be 1 (Duration) after tab, got %d", m2Tab.AnchorActiveField)
	}

	m2Tab.AnchorDurationInput.SetValue("120")

	resModelRaw, _ := m2Tab.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := resModelRaw.(*viewmodel.Model)

	if m3.AnchorPromptOpen {
		t.Fatal("expected AnchorPromptOpen to be false after submit")
	}

	updatedTask := m3.Tasks[0]
	if updatedTask.SchedulingType != model.Anchored {
		t.Fatal("expected task to be anchored after submit")
	}
	if updatedTask.TimeWindow.Start.Hour() != 14 || updatedTask.TimeWindow.Start.Minute() != 30 {
		t.Fatalf("expected start time 14:30, got %s", updatedTask.TimeWindow.Start.Format("15:04"))
	}
	if updatedTask.TimeWindow.End.Hour() != 16 || updatedTask.TimeWindow.End.Minute() != 30 {
		t.Fatalf("expected end time 16:30, got %s", updatedTask.TimeWindow.End.Format("15:04"))
	}
	if updatedTask.LifecycleState != model.StateScheduled {
		t.Fatal("expected anchored task to have StateScheduled")
	}
}

func TestTaskMoveModeAutoScroll(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.Local)
	task := model.Task{
		UUID:           "task-1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
		Height:           20,
		TimelineHour:     12,
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	m.EnterTaskMoveMode()
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("8")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	if m.TimelineHour != 9 {
		t.Fatalf("expected TimelineHour to scroll to 9, got %d", m.TimelineHour)
	}

	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	expectedDay := time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local)
	if !viewmodel.SameDay(m.SelectedDay, expectedDay) {
		t.Fatalf("expected SelectedDay to transition to June 5, got %s", m.SelectedDay.Format("2006-01-02"))
	}
}

func TestTaskMoveModeCancelViaUpdate(t *testing.T) {
	start := time.Date(2026, 6, 6, 12, 0, 0, 0, time.Local)
	task := model.Task{
		UUID:           "task-2",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-2",
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	m.EnterTaskMoveMode()
	if len(m.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in memory, got %d", len(m.Tasks))
	}

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mRes := toModelVal(res)

	if mRes.CurrentMode != viewmodel.ModeNormal {
		t.Fatal("expected mode to return to ModeNormal after cancel")
	}
	if len(mRes.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory after cancel, got %d", len(mRes.Tasks))
	}
	if !mRes.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("expected start to revert to original 12:00, got %s", mRes.Tasks[0].TimeWindow.Start)
	}
}

func TestRescheduleOverdueTaskClearsStatus(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	task := model.Task{
		UUID:           "task-overdue-1",
		SchedulingType: model.Anchored,
		LifecycleState: model.StateOverdue,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-overdue-1",
	}

	m.EnterTaskMoveMode()
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory, got %d", len(m.Tasks))
	}
	updatedTask := m.Tasks[0]
	if updatedTask.LifecycleState != model.StateScheduled {
		t.Errorf("expected LifecycleState to clear to StateScheduled on reschedule, got %s", updatedTask.LifecycleState)
	}
}

func toModelVal(tm tea.Model) viewmodel.Model {
	return *(tm.(*viewmodel.Model))
}

func TestHabitCompletionFutureDayValidation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := viewmodel.NewModel(database, syncEngine)
	habit := model.Task{
		UUID:           "habit-validation-test",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		Title:          "Read Go Code",
		SchedulingType: model.Habit,
		LifecycleState: model.StateBacklog,
	}
	database.AddTask(habit)
	m.RefreshTasks()

	m.SelectedTaskUUID = habit.UUID
	m.TodoShelfFocus = true

	m.SelectedDay = time.Now()
	resModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = toModelVal(resModel)

	updatedHabit, _ := m.DB.GetTask(habit.UUID)
	dateStrToday := time.Now().Format("2006-01-02")
	if len(updatedHabit.CompletedDates) != 1 || updatedHabit.CompletedDates[0] != dateStrToday {
		t.Errorf("expected today's date %s in CompletedDates, got %v", dateStrToday, updatedHabit.CompletedDates)
	}
	if m.WarningOpen {
		t.Errorf("expected WarningOpen to be false, got true")
	}

	m.SelectedDay = time.Now().AddDate(0, 0, 1)
	resModel2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = toModelVal(resModel2)

	updatedHabit2, _ := m.DB.GetTask(habit.UUID)
	if len(updatedHabit2.CompletedDates) != 1 {
		t.Errorf("expected CompletedDates to still contain only today's date, got %v", updatedHabit2.CompletedDates)
	}
	if !m.WarningOpen {
		t.Errorf("expected WarningOpen to be true for future day habit checkoff, got false")
	}
	if m.WarningMsg != "You cannot mark a habit as completed for future days!" {
		t.Errorf("expected warning message to be set, got %q", m.WarningMsg)
	}

	resModel3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = toModelVal(resModel3)
	if m.WarningOpen {
		t.Errorf("expected WarningOpen to be false after dismissing warning, got true")
	}
}
