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
