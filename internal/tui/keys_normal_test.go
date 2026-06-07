package tui

import (
	"testing"
	"time"

	"stream/internal/model"

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

	m := &Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
	}

	m.enterTaskMoveMode()
	if m.CurrentMode != ModeTaskMove {
		t.Fatal("expected mode to be ModeTaskMove")
	}
	if m.TaskMoveOriginalTimeWindow != task.TimeWindow {
		t.Fatal("expected original time window to be recorded")
	}

	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.TaskMovePrefix != "2" {
		t.Fatalf("expected prefix to be '2', got %q", m.TaskMovePrefix)
	}

	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.TaskMovePrefix != "" {
		t.Fatalf("expected prefix to reset after move, got %q", m.TaskMovePrefix)
	}

	// The original task should still be at its original location (10:00)
	if !m.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("expected original task start to remain unchanged (10:00), got %s", m.Tasks[0].TimeWindow.Start)
	}

	// The moving clone task should be at index 1 and moved to 10:10
	if len(m.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in memory during move, got %d", len(m.Tasks))
	}
	clone := m.Tasks[1]
	if clone.UUID != "task-1_moving" {
		t.Fatalf("expected clone UUID to be task-1_moving, got %s", clone.UUID)
	}
	if !clone.TimeWindow.Start.Equal(start.Add(10 * time.Minute)) {
		t.Fatalf("expected clone start to be 10:10, got %s", clone.TimeWindow.Start)
	}

	// Press Enter to confirm move
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// After confirmation, only the original task should remain and it should be updated
	if len(m.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory after confirm, got %d", len(m.Tasks))
	}
	if !m.Tasks[0].TimeWindow.Start.Equal(start.Add(10 * time.Minute)) {
		t.Fatalf("expected original task start to be updated to 10:10, got %s", m.Tasks[0].TimeWindow.Start)
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

	m := &Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-2",
	}

	m.enterTaskMoveMode()
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.CurrentMode != ModeNormal {
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

	m := &Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
	}

	// Pressing "a" on an anchored task should de-anchor it immediately
	m.handleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

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

	m2 := &Model{
		Tasks:            []model.Task{task2},
		SelectedTaskUUID: "task-2",
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
	}

	// Pressing "a" on a floating task should open the AnchorPrompt Modal
	m2.handleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if !m2.AnchorPromptOpen {
		t.Fatal("expected AnchorPromptOpen to be true")
	}
	if m2.AnchorPromptTask.UUID != "task-2" {
		t.Fatalf("expected AnchorPromptTask UUID to be task-2, got %s", m2.AnchorPromptTask.UUID)
	}

	// Set Start Time to "14:30"
	m2.AnchorTimeInput.SetValue("14:30")

	// Press Tab to switch to Duration field
	resTabRaw, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	m2Tab := resTabRaw.(Model)
	if m2Tab.AnchorActiveField != 1 {
		t.Fatalf("expected active field to be 1 (Duration) after tab, got %d", m2Tab.AnchorActiveField)
	}

	// Set Duration to "120"
	m2Tab.AnchorDurationInput.SetValue("120")

	// Press Enter to confirm anchoring
	resModelRaw, _ := m2Tab.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := resModelRaw.(Model)

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
	// Duration was set to 120 minutes. 14:30 + 120 mins = 16:30
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

	m := &Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
		Height:           20, // visibleH will be 16
		TimelineHour:     12,
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	m.enterTaskMoveMode()
	// Let's move the task up (negative direction) by 24 steps (2 hours)
	// Task start time will become 08:00
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	// Since task starts at 08:00 (row 64), it fits in the viewport of height 16.
	// Initial TimelineHour was 12. Viewport was [12*8 - 8, 12*8 + 8) = [88, 104)
	// Task row range is [64, 72)
	// 64 < 88, so we auto-scrolled up to (64 + 8)/8 = 9.
	if m.TimelineHour != 9 {
		t.Fatalf("expected TimelineHour to scroll to 9, got %d", m.TimelineHour)
	}

	// Test day wrap-around/transition:
	// Move task up by 120 steps (10 hours). Start will be 08:00 - 10h = 22:00 of June 5th.
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	m.handleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	// Check that SelectedDay transitioned to June 5th.
	expectedDay := time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local)
	if !sameDay(m.SelectedDay, expectedDay) {
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

	m := Model{
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-2",
		SelectedDay:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	m.enterTaskMoveMode()
	// Let's verify that entering move mode adds the clone placeholder task
	if len(m.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in memory, got %d", len(m.Tasks))
	}

	// Send an ESC key through the Update method
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mRes := res.(Model)

	if mRes.CurrentMode != ModeNormal {
		t.Fatal("expected mode to return to ModeNormal after cancel")
	}
	// The clone should be completely removed from memory
	if len(mRes.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory after cancel, got %d", len(mRes.Tasks))
	}
	if !mRes.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("expected start to revert to original 12:00, got %s", mRes.Tasks[0].TimeWindow.Start)
	}
}

