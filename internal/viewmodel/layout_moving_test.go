package viewmodel

import (
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResolveOverlaps_MovingOverlay(t *testing.T) {
	// A: 10:00-11:00, B: 11:00-12:00, C: 12:00-13:00 (all consecutive normal events)
	// D: 10:30-11:30 (moving clone event, overlaps with A and B)
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	taskA := model.Task{
		UUID:           "task-A",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(10 * time.Hour),
			End:   dayStart.Add(11 * time.Hour),
		},
	}
	taskB := model.Task{
		UUID:           "task-B",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(11 * time.Hour),
			End:   dayStart.Add(12 * time.Hour),
		},
	}
	taskC := model.Task{
		UUID:           "task-C",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(12 * time.Hour),
			End:   dayStart.Add(13 * time.Hour),
		},
	}
	taskDMoving := model.Task{
		UUID:           "task-D_moving",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(10*time.Hour + 30*time.Minute),
			End:   dayStart.Add(11*time.Hour + 30*time.Minute),
		},
	}

	tasks := []model.Task{taskA, taskB, taskC, taskDMoving}

	// 1. ResolveOverlaps test
	cols := ResolveOverlaps(tasks)
	if len(cols) != 4 {
		t.Fatalf("expected 4 column resolutions, got %d", len(cols))
	}

	// Ensure normal tasks (A, B, C) are not influenced by the moving clone
	for _, col := range cols {
		if strings.HasSuffix(col.Task.UUID, "_moving") {
			// Special moving task must have ColIndex = 0, TotalCol = 1
			if col.ColIndex != 0 || col.TotalCol != 1 {
				t.Errorf("special moving task should have ColIndex = 0 and TotalCol = 1, got ColIndex=%d TotalCol=%d", col.ColIndex, col.TotalCol)
			}
		} else {
			// Normal tasks do not overlap with each other, so they must have ColIndex = 0, TotalCol = 1
			if col.ColIndex != 0 || col.TotalCol != 1 {
				t.Errorf("normal task %s should have ColIndex = 0 and TotalCol = 1, got ColIndex=%d TotalCol=%d", col.Task.UUID, col.ColIndex, col.TotalCol)
			}
		}
	}

	// 2. BuildDayTaskRects test (shifting logic bypass)
	m := &Model{}
	m.Layout.TimelineW = 50
	rects := m.BuildDayTaskRects(tasks)
	if len(rects) != 4 {
		t.Fatalf("expected 4 task rects, got %d", len(rects))
	}

	for _, r := range rects {
		expectedStartRow := TimeToRow(r.Task.TimeWindow.Start) / 5
		expectedEndRow := TimeToRow(r.Task.TimeWindow.End) / 5
		expectedHeight := expectedEndRow - expectedStartRow

		if r.Top != expectedStartRow {
			t.Errorf("task %s rect top was shifted: expected %d, got %d", r.Task.UUID, expectedStartRow, r.Top)
		}
		if r.Height != expectedHeight {
			t.Errorf("task %s rect height was altered: expected %d, got %d", r.Task.UUID, expectedHeight, r.Height)
		}
	}
}

func TestManualSessionLogging(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := NewModel(database, syncEngine)

	day := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	task := model.Task{
		UUID:           "task-log-test",
		Title:          "Log Time Task",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(10 * time.Hour),
			End:   day.Add(11 * time.Hour), // 60 mins planned
		},
	}
	database.AddTask(task)
	m.refreshTasks()

	m.SelectedTaskUUID = "task-log-test"
	m.SelectedDay = day

	// 1. Completing a task when NOT in zen mode prompts log_session_confirm
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.ConfirmOpen || m.ConfirmActionType != "log_session_confirm" {
		t.Fatalf("expected log_session_confirm dialog, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// 2. Pressing "y" on log_session_confirm opens LogSessionPromptOpen
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.ConfirmOpen || !m.LogSessionPromptOpen {
		t.Fatalf("expected LogSessionPromptOpen to be active, got LogSessionPromptOpen=%t", m.LogSessionPromptOpen)
	}

	// Default value of focus minutes should be planned duration (60)
	if m.LogSessionFocusInput.Value() != "60" {
		t.Errorf("expected default focus input value '60', got %q", m.LogSessionFocusInput.Value())
	}
	if m.LogSessionBreakInput.Value() != "0" {
		t.Errorf("expected default break input value '0', got %q", m.LogSessionBreakInput.Value())
	}

	// Fill in new focus/break minutes: 45 focus, 5 break
	m.LogSessionFocusInput.SetValue("45")
	m.LogSessionBreakInput.SetValue("5")

	// 3. Pressing Enter saves focus/break elapsed time and completes the task
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if m.LogSessionPromptOpen {
		t.Fatal("expected LogSessionPromptOpen to be closed after Enter")
	}

	updated, _ := m.DB.GetTask("task-log-test")
	if updated.LifecycleState != model.StateCompleted {
		t.Errorf("expected task to be completed, got %s", updated.LifecycleState)
	}
	if updated.ExecutionMetrics.ElapsedFocusSeconds != 45*60 {
		t.Errorf("expected 45m (2700s) focus time, got %d", updated.ExecutionMetrics.ElapsedFocusSeconds)
	}
	if updated.ExecutionMetrics.ElapsedBreakSeconds != 5*60 {
		t.Errorf("expected 5m (300s) break time, got %d", updated.ExecutionMetrics.ElapsedBreakSeconds)
	}

	// 4. Test "No" selection completing task without logging time (0 minutes)
	task2 := model.Task{
		UUID:           "task-log-test-2",
		Title:          "No Log Task",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(12 * time.Hour),
			End:   day.Add(13 * time.Hour),
		},
	}
	database.AddTask(task2)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-log-test-2"

	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.ConfirmOpen || m.ConfirmActionType != "log_session_confirm" {
		t.Fatalf("expected log_session_confirm dialog, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Press "n" to decline logging
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.ConfirmOpen || m.LogSessionPromptOpen {
		t.Fatal("expected confirmation and prompt modals to be closed")
	}

	updated2, _ := m.DB.GetTask("task-log-test-2")
	if updated2.LifecycleState != model.StateCompleted {
		t.Errorf("expected task to be completed, got %s", updated2.LifecycleState)
	}
	if updated2.ExecutionMetrics.ElapsedFocusSeconds != 0 {
		t.Errorf("expected 0 focus time logged, got %d", updated2.ExecutionMetrics.ElapsedFocusSeconds)
	}

	// 5. Test Log Session prompt modal buttons navigation
	task3 := model.Task{
		UUID:           "task-log-test-3",
		Title:          "Log Buttons Task",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(14 * time.Hour),
			End:   day.Add(15 * time.Hour),
		},
	}
	database.AddTask(task3)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-log-test-3"

	// Complete task -> prompt confirm
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	// Press "y" -> opens Log Session prompt modal
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.LogSessionPromptOpen {
		t.Fatalf("expected LogSessionPromptOpen to be active")
	}

	// Default active field should be 0 (Focus input)
	if m.LogSessionActiveField != 0 {
		t.Errorf("expected LogSessionActiveField=0, got %d", m.LogSessionActiveField)
	}

	// Press Tab -> active field 1 (Break input)
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyTab})
	if m.LogSessionActiveField != 1 {
		t.Errorf("expected LogSessionActiveField=1, got %d", m.LogSessionActiveField)
	}

	// Press Tab -> active field 2 (Save button)
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyTab})
	if m.LogSessionActiveField != 2 {
		t.Errorf("expected LogSessionActiveField=2, got %d", m.LogSessionActiveField)
	}

	// Press Tab -> active field 3 (Cancel button)
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyTab})
	if m.LogSessionActiveField != 3 {
		t.Errorf("expected LogSessionActiveField=3, got %d", m.LogSessionActiveField)
	}

	// Press Left -> active field 2 (Save button)
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyLeft})
	if m.LogSessionActiveField != 2 {
		t.Errorf("expected LogSessionActiveField=2 after Left, got %d", m.LogSessionActiveField)
	}

	// Press Right -> active field 3 (Cancel button)
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRight})
	if m.LogSessionActiveField != 3 {
		t.Errorf("expected LogSessionActiveField=3 after Right, got %d", m.LogSessionActiveField)
	}

	// Press Enter while on Cancel button -> modal should close and task should NOT be completed
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if m.LogSessionPromptOpen {
		t.Fatal("expected LogSessionPromptOpen to be closed after Enter on cancel button")
	}

	updated3, _ := m.DB.GetTask("task-log-test-3")
	if updated3.LifecycleState != model.StateOverdue {
		t.Errorf("expected task to remain overdue (canceled completion), got %s", updated3.LifecycleState)
	}

	// 6. Test inheritance of existing execution metrics
	task4 := model.Task{
		UUID:           "task-log-test-4",
		Title:          "Inherit Metrics Task",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(16 * time.Hour),
			End:   day.Add(17 * time.Hour),
		},
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 1500, // 25 minutes
			ElapsedBreakSeconds: 300,  // 5 minutes
		},
	}
	database.AddTask(task4)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-log-test-4"

	// Complete task -> prompt confirm
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	// Press "y" -> opens Log Session prompt modal
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.LogSessionPromptOpen {
		t.Fatalf("expected LogSessionPromptOpen to be active")
	}

	// Should inherit focus and break minutes
	if m.LogSessionFocusInput.Value() != "25" {
		t.Errorf("expected inherited focus input value '25', got %q", m.LogSessionFocusInput.Value())
	}
	if m.LogSessionBreakInput.Value() != "5" {
		t.Errorf("expected inherited break input value '5', got %q", m.LogSessionBreakInput.Value())
	}

	// Press Esc to cancel
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
}

func TestTaskShrinkRemaining(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := NewModel(database, syncEngine)
	day := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	m.SelectedDay = day

	// Test case 1: Log to shelf
	task1 := model.Task{
		UUID:           "task-shrink-1",
		Title:          "Shrink Task 1",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(10 * time.Hour),
			End:   day.Add(11 * time.Hour), // 60 mins
		},
	}
	database.AddTask(task1)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-shrink-1"

	// Enter duration adjust mode (adjustTop = false)
	m.EnterTaskDurationAdjustMode(false)
	if m.CurrentMode != ModeTaskDurationAdjust {
		t.Fatalf("expected mode %v, got %v", ModeTaskDurationAdjust, m.CurrentMode)
	}

	// Shrink duration by 15 minutes
	m.applyTaskDurationAdjust(-1)

	// Confirm duration adjust
	m.confirmTaskDurationAdjust()
	if !m.ConfirmOpen || m.ConfirmActionType != "shrink_remaining_confirm" {
		t.Fatalf("expected shrink_remaining_confirm dialog, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Press "y" to log remaining time to shelf
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.ConfirmOpen {
		t.Fatal("expected confirmation modal to be closed")
	}

	// Verify task 1 is shrunk to 45 mins
	u1, _ := database.GetTask("task-shrink-1")
	dur1 := u1.TimeWindow.End.Sub(u1.TimeWindow.Start)
	if dur1 != 45*time.Minute {
		t.Errorf("expected task-shrink-1 duration to be 45m, got %s", dur1)
	}

	// Verify a floating task was logged to shelf with 15 mins estimated duration
	allTasks := database.GetTasks()
	var floatingFound bool
	for _, tk := range allTasks {
		if tk.SchedulingType == model.Floating && tk.Title == "Shrink Task 1 (remaining)" && tk.EstimatedDurationMins == 15 {
			floatingFound = true
			break
		}
	}
	if !floatingFound {
		t.Error("expected a floating task with 15m remaining logged to Todo Shelf")
	}

	// Test case 2: Discard remaining
	task2 := model.Task{
		UUID:           "task-shrink-2",
		Title:          "Shrink Task 2",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(12 * time.Hour),
			End:   day.Add(13 * time.Hour), // 60 mins
		},
	}
	database.AddTask(task2)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-shrink-2"

	m.EnterTaskDurationAdjustMode(false)
	m.applyTaskDurationAdjust(-1)
	m.confirmTaskDurationAdjust()
	if !m.ConfirmOpen || m.ConfirmActionType != "shrink_remaining_confirm" {
		t.Fatalf("expected shrink_remaining_confirm dialog, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Press "n" to discard remaining
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.ConfirmOpen {
		t.Fatal("expected confirmation modal to be closed")
	}

	u2, _ := database.GetTask("task-shrink-2")
	dur2 := u2.TimeWindow.End.Sub(u2.TimeWindow.Start)
	if dur2 != 45*time.Minute {
		t.Errorf("expected task-shrink-2 duration to be 45m, got %s", dur2)
	}

	// Verify no new floating task for "Shrink Task 2"
	allTasks = database.GetTasks()
	floatingFound = false
	for _, tk := range allTasks {
		if tk.SchedulingType == model.Floating && tk.Title == "Shrink Task 2" {
			floatingFound = true
			break
		}
	}
	if floatingFound {
		t.Error("expected no floating task logged to Todo Shelf for task-shrink-2")
	}

	// Test case 3: Cancel shrink confirmation
	task3 := model.Task{
		UUID:           "task-shrink-3",
		Title:          "Shrink Task 3",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(14 * time.Hour),
			End:   day.Add(15 * time.Hour), // 60 mins
		},
	}
	database.AddTask(task3)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-shrink-3"

	m.EnterTaskDurationAdjustMode(false)
	m.applyTaskDurationAdjust(-1)
	m.confirmTaskDurationAdjust()
	if !m.ConfirmOpen || m.ConfirmActionType != "shrink_remaining_confirm" {
		t.Fatalf("expected shrink_remaining_confirm dialog, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Press Esc to cancel
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if m.ConfirmOpen {
		t.Fatal("expected confirmation modal to be closed")
	}

	// Verify duration remains 60 mins
	u3, _ := database.GetTask("task-shrink-3")
	dur3 := u3.TimeWindow.End.Sub(u3.TimeWindow.Start)
	if dur3 != 60*time.Minute {
		t.Errorf("expected task-shrink-3 duration to remain 60m, got %s", dur3)
	}
}

func TestTaskDurationAdjustScrolling(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := NewModel(database, syncEngine)
	m.Height = 32
	day := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
	m.SelectedDay = day

	task := model.Task{
		UUID:           "task-scroll",
		Title:          "Scroll Task",
		WorkspaceUUID:  m.ActiveWorkspaceUUID,
		SchedulingType: model.Anchored,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: day.Add(10 * time.Hour),
			End:   day.Add(11 * time.Hour), // 60 mins
		},
	}
	database.AddTask(task)
	m.refreshTasks()
	m.SelectedTaskUUID = "task-scroll"

	// 1. Enter duration adjust mode
	m.EnterTaskDurationAdjustMode(false) // Adjust bottom boundary
	m.TimelineHour = 10 // Center around 10:00 AM

	// Trigger AutoScroll - should keep TimelineHour = 10 since task is in view
	m.AutoScrollToSelectedTask()
	if m.TimelineHour != 10 {
		t.Errorf("expected TimelineHour to remain 10, got %d", m.TimelineHour)
	}

	// 2. Extend bottom boundary extensively to 8 hours duration (moving taskEnd down)
	m.applyTaskDurationAdjust(28) // +7 hours (total 8 hours)

	// Since active boundary taskEnd is now at 18:00 (6:00 PM), it is outside the viewport when centered at 10:00 AM.
	// AutoScroll should scroll down (TimelineHour should increase)
	m.AutoScrollToSelectedTask()
	if m.TimelineHour <= 10 {
		t.Errorf("expected TimelineHour to increase to keep extended bottom in view, got %d", m.TimelineHour)
	}
}
