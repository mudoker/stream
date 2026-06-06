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
		Tasks:           []model.Task{task},
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
	if !m.Tasks[0].TimeWindow.Start.Equal(start.Add(30 * time.Minute)) {
		t.Fatalf("expected start after move to be 10:30, got %s", m.Tasks[0].TimeWindow.Start)
	}
	if !m.Tasks[0].TimeWindow.End.Equal(start.Add(90 * time.Minute)) {
		t.Fatalf("expected end after move to be 11:30, got %s", m.Tasks[0].TimeWindow.End)
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

