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

func TestEventMoveAndDurationAdjust(t *testing.T) {
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
	
	now := time.Now()
	eventTask := model.Task{
		UUID:           "event-1",
		Title:          "Keynote Event",
		SchedulingType: model.Event,
		Location:       "Online",
		TimeWindow: model.TimeWindow{
			Start: now,
			End:   now.Add(60 * time.Minute),
		},
	}
	database.AddTask(eventTask)
	m.SelectedTaskUUID = "event-1"
	m.Tasks = []model.Task{eventTask}

	// 1. Test EnterTaskMoveMode
	m.EnterTaskMoveMode()
	if m.CurrentMode != viewmodel.ModeTaskMove {
		t.Errorf("expected ModeTaskMove, got %v", m.CurrentMode)
	}
	
	// 2. Apply move down (1 step = 15 mins)
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	// Confirm move
	m.HandleTaskMoveKeys(tea.KeyMsg{Type: tea.KeyEnter})
	
	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected to return to ModeNormal, got %v", m.CurrentMode)
	}

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	movedTask := tasks[0]
	expectedStart := now.Add(15 * time.Minute)
	if movedTask.TimeWindow.Start.Sub(expectedStart).Abs() > time.Second {
		t.Errorf("expected start time %v, got %v", expectedStart, movedTask.TimeWindow.Start)
	}

	m.Tasks = []model.Task{movedTask}
	m.SelectedTaskUUID = "event-1"

	// 3. Test EnterTaskDurationAdjustMode
	m.EnterTaskDurationAdjustMode()
	if m.CurrentMode != viewmodel.ModeTaskDurationAdjust {
		t.Errorf("expected ModeTaskDurationAdjust, got %v", m.CurrentMode)
	}

	// 4. Apply duration increase (1 step = 15 mins)
	m.HandleTaskDurationAdjustKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	// Confirm duration adjust
	m.HandleTaskDurationAdjustKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected to return to ModeNormal, got %v", m.CurrentMode)
	}

	tasks = database.GetTasks()
	adjustedTask := tasks[0]
	expectedDuration := 75 * time.Minute
	actualDuration := adjustedTask.TimeWindow.End.Sub(adjustedTask.TimeWindow.Start)
	if actualDuration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, actualDuration)
	}
}
