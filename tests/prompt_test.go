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

func TestPromptModalInteractiveSelection(t *testing.T) {
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

	startTime := time.Date(2026, 6, 6, 9, 0, 0, 0, time.Local)
	task := model.Task{
		UUID:           "task-prompt-1",
		Title:          "Morning Standup",
		Priority:       model.P1,
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(30 * time.Minute),
		},
		LifecycleState: model.StateScheduled,
	}
	database.AddTask(task)
	m.Tasks = []model.Task{task}

	// Trigger the prompt
	m.PromptTask = task
	m.PromptOpen = true
	m.PromptSelectedIdx = 0

	// 1. Check default selection
	if m.PromptSelectedIdx != 0 {
		t.Errorf("expected default PromptSelectedIdx to be 0, got %d", m.PromptSelectedIdx)
	}

	// 2. Pressing "s" changes focus to index 1 (Snooze), but does not execute yet
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = res.(*viewmodel.Model)
	if m.PromptSelectedIdx != 1 {
		t.Errorf("expected pressing 's' to select Snooze (1), got %d", m.PromptSelectedIdx)
	}
	if !m.PromptOpen {
		t.Errorf("expected prompt to remain open after pressing 's'")
	}
	if !m.Tasks[0].TimeWindow.Start.Equal(startTime) {
		t.Errorf("expected task time to remain unchanged before Enter confirmation")
	}

	// 3. Pressing "d" changes focus to index 2 (Dismiss), but does not execute yet
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = res.(*viewmodel.Model)
	if m.PromptSelectedIdx != 2 {
		t.Errorf("expected pressing 'd' to select Dismiss (2), got %d", m.PromptSelectedIdx)
	}
	if !m.PromptOpen {
		t.Errorf("expected prompt to remain open after pressing 'd'")
	}

	// 4. Pressing "tab" cycles selection from index 2 to 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	m = res.(*viewmodel.Model)
	if m.PromptSelectedIdx != 0 {
		t.Errorf("expected pressing 'tab' at index 2 to cycle to 0, got %d", m.PromptSelectedIdx)
	}

	// 5. Pressing "left" arrow key cycles selection to index 2
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = res.(*viewmodel.Model)
	if m.PromptSelectedIdx != 2 {
		t.Errorf("expected pressing 'left' at index 0 to cycle to 2, got %d", m.PromptSelectedIdx)
	}

	// 6. Focus back to snooze and press Enter to confirm
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}) // Select snooze
	m = res.(*viewmodel.Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm snooze
	m = res.(*viewmodel.Model)

	if m.PromptOpen {
		t.Errorf("expected prompt to close after confirming snooze via Enter")
	}

	updatedTasks := database.GetTasks()
	if len(updatedTasks) != 1 {
		t.Fatalf("expected 1 task in database, got %d", len(updatedTasks))
	}
	expectedStart := startTime.Add(5 * time.Minute)
	if !updatedTasks[0].TimeWindow.Start.Equal(expectedStart) {
		t.Errorf("expected task start time to be snoozed to %s, got %s", expectedStart, updatedTasks[0].TimeWindow.Start)
	}
}
