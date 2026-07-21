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

func TestTaskCloneMoveModeWorkflow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	modelVal := viewmodel.NewModel(database, syncEngine)
	m := &modelVal

	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	m.SelectedDay = start

	origTask := model.Task{
		UUID:           "task-clone-1",
		Title:          "Deep Work Session",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}
	database.AddTask(origTask)
	m.Tasks = []model.Task{origTask}
	m.SelectedTaskUUID = "task-clone-1"

	// 1. Enter clone move mode via Shift+Y ('Y') key press
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeTaskMove {
		t.Fatalf("expected mode ModeTaskMove, got %v (status: %s)", m.CurrentMode, m.StatusMsg)
	}
	if !m.TaskMoveIsClone {
		t.Fatalf("expected TaskMoveIsClone to be true")
	}

	// 2. Move down 2 steps (30 minutes)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = res.(*viewmodel.Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(*viewmodel.Model)

	// Verify that DURING move mode, GetDayTasks() includes BOTH the original task AND the moving clone task
	dayTasksDuringMove := m.GetDayTasks()
	if len(dayTasksDuringMove) != 2 {
		t.Fatalf("expected 2 tasks visible on timeline during clone move, got %d", len(dayTasksDuringMove))
	}
	var origVisible, cloneVisible bool
	for _, tk := range dayTasksDuringMove {
		if tk.UUID == "task-clone-1" {
			origVisible = true
		} else if tk.UUID == "task-clone-1_moving" {
			cloneVisible = true
		}
	}
	if !origVisible || !cloneVisible {
		t.Fatalf("expected both original task and clone placeholder to be visible on timeline during clone move (origVisible=%v, cloneVisible=%v)", origVisible, cloneVisible)
	}

	// 3. Confirm by pressing Enter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Fatalf("expected mode ModeNormal after confirm, got %v", m.CurrentMode)
	}
	if m.TaskMoveIsClone {
		t.Fatalf("expected TaskMoveIsClone to reset to false")
	}

	// Verify that DB and memory now contain 2 tasks: original task untouched + new cloned task at 10:30
	dbTasks := database.GetTasks()
	if len(dbTasks) != 2 {
		t.Fatalf("expected 2 tasks in DB after clone, got %d", len(dbTasks))
	}

	var foundOriginal, foundCloned bool
	var clonedUUID string
	for _, item := range dbTasks {
		if item.UUID == "task-clone-1" {
			foundOriginal = true
			if !item.TimeWindow.Start.Equal(start) {
				t.Fatalf("expected original task start to stay 10:00, got %s", item.TimeWindow.Start)
			}
		} else {
			foundCloned = true
			clonedUUID = item.UUID
			if item.Title != "Deep Work Session" {
				t.Fatalf("expected cloned task title 'Deep Work Session', got %s", item.Title)
			}
			expectedStart := start.Add(30 * time.Minute)
			if !item.TimeWindow.Start.Equal(expectedStart) {
				t.Fatalf("expected cloned task start 10:30, got %s", item.TimeWindow.Start)
			}
		}
	}

	if !foundOriginal || !foundCloned {
		t.Fatalf("expected both original and cloned tasks in DB (foundOriginal=%v, foundCloned=%v)", foundOriginal, foundCloned)
	}

	if m.SelectedTaskUUID != clonedUUID {
		t.Fatalf("expected selected task UUID to be cloned task UUID %s, got %s", clonedUUID, m.SelectedTaskUUID)
	}
}

func TestTaskCloneMoveCancelWorkflow(t *testing.T) {
	start := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	origTask := model.Task{
		UUID:           "task-clone-cancel",
		Title:          "Meeting",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: start,
			End:   start.Add(time.Hour),
		},
	}

	m := viewmodel.Model{
		Tasks:            []model.Task{origTask},
		SelectedTaskUUID: "task-clone-cancel",
		SelectedDay:      start,
	}

	m.EnterTaskCloneMoveMode()
	if m.CurrentMode != viewmodel.ModeTaskMove || !m.TaskMoveIsClone {
		t.Fatalf("expected clone move mode to be active")
	}

	// Move down
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Cancel with Esc
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Fatalf("expected mode to revert to ModeNormal")
	}
	if m.TaskMoveIsClone {
		t.Fatalf("expected TaskMoveIsClone to be false")
	}

	// Only 1 original task should remain untouched
	if len(m.Tasks) != 1 {
		t.Fatalf("expected 1 task in memory after cancel, got %d", len(m.Tasks))
	}
	if m.Tasks[0].UUID != "task-clone-cancel" || !m.Tasks[0].TimeWindow.Start.Equal(start) {
		t.Fatalf("original task should be untouched")
	}
}
