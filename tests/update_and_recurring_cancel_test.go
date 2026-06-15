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

func TestUpdatePromptModal(t *testing.T) {
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

	// Simulate getting update commits msg
	res, _ := m.Update(viewmodel.UpdateCheckMsg{
		Commits: []string{"feat: awesome update", "fix: minor bug"},
	})
	m = res.(*viewmodel.Model)

	if !m.UpdatePromptOpen {
		t.Fatalf("expected update prompt to be open")
	}
	if len(m.UpdateCommits) != 2 || m.UpdateCommits[0] != "feat: awesome update" {
		t.Errorf("unexpected update commits list: %v", m.UpdateCommits)
	}
	if m.UpdatePromptSelectedIdx != 0 {
		t.Errorf("expected default selected option to be 0 (Update), got %d", m.UpdatePromptSelectedIdx)
	}

	// Press right/l to select Snooze
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = res.(*viewmodel.Model)
	if m.UpdatePromptSelectedIdx != 1 {
		t.Errorf("expected selected option to be 1 (Snooze), got %d", m.UpdatePromptSelectedIdx)
	}

	// Confirm Snooze
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.UpdatePromptOpen {
		t.Errorf("expected update prompt to close after enter on snooze")
	}

	settings := database.GetUserSettings()
	if settings.UpdateSnoozedUntil.Before(time.Now()) {
		t.Errorf("expected update snoozed until to be set in the future, got %v", settings.UpdateSnoozedUntil)
	}
}

func TestRecurringTaskMoveCancel(t *testing.T) {
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

	startTime := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	task := model.Task{
		UUID:                "recurring-task-1",
		WorkspaceUUID:       m.ActiveWorkspaceUUID,
		RecurringParentUUID: "parent-uuid",
		Title:               "Weekly Gym",
		Priority:            model.P2,
		SchedulingType:      model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(1 * time.Hour),
		},
		LifecycleState: model.StateScheduled,
	}
	database.AddTask(task)
	m.RefreshTasks()

	// Select the task
	m.SelectedTaskUUID = task.UUID
	m.SelectedDay = startTime

	// Enter move mode
	m.EnterTaskMoveMode()
	if m.CurrentMode != viewmodel.ModeTaskMove {
		t.Fatalf("expected mode to be TASK_MOVE, got %s", m.CurrentMode)
	}

	// Move task down by 1 step (adds constant.TaskMoveStepMinutes, usually 15m)
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(*viewmodel.Model)

	// Confirm move -> this triggers edit_recurring confirm modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit_recurring confirm modal to open, got ConfirmOpen=%v ConfirmActionType=%q", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Cancel the edit_recurring confirmation (press Esc)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(*viewmodel.Model)

	if m.ConfirmOpen {
		t.Errorf("expected confirm modal to close on Esc")
	}

	// Verify the task's time window in memory is reverted
	var revertedTask model.Task
	for _, t := range m.Tasks {
		if t.UUID == task.UUID {
			revertedTask = t
			break
		}
	}
	if !revertedTask.TimeWindow.Start.Equal(startTime) {
		t.Errorf("expected task start time to revert to %v, got %v", startTime, revertedTask.TimeWindow.Start)
	}
}
