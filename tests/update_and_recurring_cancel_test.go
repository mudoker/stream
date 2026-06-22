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

func TestEditHabitRecurringFieldsExposedAndUpdated(t *testing.T) {
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

	today := time.Now().Truncate(24 * time.Hour)
	habit := model.Task{
		UUID:                "habit-instance-1",
		WorkspaceUUID:       m.ActiveWorkspaceUUID,
		RecurringParentUUID: "habit-parent-uuid",
		Title:               "Hydrate",
		Priority:            model.P2,
		SchedulingType:      model.Habit,
		TimeWindow: model.TimeWindow{
			Start: today.Add(8 * time.Hour), // 08:00
			End:   today.Add(8 * time.Hour).Add(15 * time.Minute),
		},
		LifecycleState: model.StateReady,
	}
	database.AddTask(habit)
	m.RefreshTasks()

	// Select task
	m.SelectedTaskUUID = habit.UUID
	m.SelectedDay = today

	// Trigger Edit mode
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeForm {
		t.Fatalf("expected mode to be ModeForm (editing), got %v", m.CurrentMode)
	}

	// 2. Assert recurring fields are populated
	if m.Form.IsRecurringIdx != 1 {
		t.Errorf("expected IsRecurringIdx to be 1, got %d", m.Form.IsRecurringIdx)
	}
	if m.Form.RecurringDaysInput.Value() == "" {
		t.Errorf("expected RecurringDaysInput to be pre-populated")
	}
	if m.Form.RecurringEndDateInput.Value() == "" {
		t.Errorf("expected RecurringEndDateInput to be pre-populated")
	}

	// 3. Assert recurring fields are visible
	visible := m.Form.VisibleFields()
	hasField12 := false
	hasField13 := false
	for _, f := range visible {
		if f == 12 {
			hasField12 = true
		}
		if f == 13 {
			hasField13 = true
		}
	}
	if !hasField12 || !hasField13 {
		t.Errorf("expected recurring fields (12, 13) to be visible in form edit mode, visible: %v", visible)
	}

	// 4. Change some values (e.g. title, end date, days)
	m.Form.TitleInput.SetValue("Drink Water Edited")
	m.Form.RecurringEndDateInput.SetValue(today.AddDate(0, 0, 3).Format("2006-01-02")) // 3 days recurrence
	m.Form.RecurringDaysInput.SetValue("mon, tue, wed, thu, fri, sat, sun") // daily

	// Submit form
	m.SubmitForm()

	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit_recurring confirmation dialog to be open")
	}

	// Confirm "This and all future occurrences" (ConfirmSelectedIndex = 1)
	m.ConfirmSelectedIndex = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	// 5. Verify database has regenerated tasks
	tasks := database.GetTasks()
	var matchingTasks []model.Task
	for _, tVal := range tasks {
		if tVal.RecurringParentUUID == "habit-parent-uuid" {
			matchingTasks = append(matchingTasks, tVal)
		}
	}

	// Since we set 3 days daily, we expect around 4 instances (today, today+1, today+2, today+3)
	if len(matchingTasks) < 3 {
		t.Errorf("expected at least 3 recurring instances regenerated, got %d: %v", len(matchingTasks), matchingTasks)
	}

	for _, tVal := range matchingTasks {
		if tVal.Title != "Drink Water Edited" {
			t.Errorf("expected title to be updated to 'Drink Water Edited', got %q", tVal.Title)
		}
	}
}

func TestHabitStartEditAndMove(t *testing.T) {
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

	today := time.Now().Truncate(24 * time.Hour)
	startTime := today.Add(8 * time.Hour) // 08:00 AM

	habit := model.Task{
		UUID:                "habit-occurrence-1",
		WorkspaceUUID:       m.ActiveWorkspaceUUID,
		RecurringParentUUID: "habit-parent-uuid",
		Title:               "Read Book",
		SchedulingType:      model.Habit,
		TimeWindow: model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(30 * time.Minute),
		},
		LifecycleState: model.StateReady,
	}
	database.AddTask(habit)
	m.RefreshTasks()

	// ----------------------------------------------------
	// 1. Edit Habit Start Time via Form
	// ----------------------------------------------------
	m.SelectedTaskUUID = habit.UUID
	m.SelectedDay = today

	// Trigger Edit
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(*viewmodel.Model)

	// Set Start Time to 09:30 AM
	m.Form.StartTimeInput.SetValue("09:30")
	// Submit form (Enter key)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit_recurring confirmation dialog to be open from edit form, got: %v", m.ConfirmActionType)
	}
	if !m.RecurringEditFromForm {
		t.Errorf("expected RecurringEditFromForm to be true when editing from form")
	}

	// Confirm "This and all future occurrences" (ConfirmSelectedIndex = 1)
	m.ConfirmSelectedIndex = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	// Verify regenerated tasks have updated start time (09:30)
	tasks := database.GetTasks()
	found := false
	for _, tVal := range tasks {
		if tVal.RecurringParentUUID == "habit-parent-uuid" {
			found = true
			h, min, _ := tVal.TimeWindow.Start.Clock()
			if h != 9 || min != 30 {
				t.Errorf("expected habit time to be 09:30, got %02d:%02d", h, min)
			}
		}
	}
	if !found {
		t.Fatalf("expected to find regenerated habit instances in DB")
	}

	// ----------------------------------------------------
	// 2. Move Habit Start Time via V (Quick Move)
	// ----------------------------------------------------
	// Find the current instance UUID (could have changed due to regeneration)
	var habitUUID string
	tasks = database.GetTasks()
	for _, tVal := range tasks {
		if tVal.RecurringParentUUID == "habit-parent-uuid" {
			habitUUID = tVal.UUID
			break
		}
	}
	if habitUUID == "" {
		t.Fatalf("no regenerated habit found to test move")
	}

	m.SelectedTaskUUID = habitUUID

	// Press 'v' to enter move mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = res.(*viewmodel.Model)
	if m.CurrentMode != viewmodel.ModeTaskMove {
		t.Fatalf("expected to be in ModeTaskMove, got %v", m.CurrentMode)
	}

	// Press 'j' to move it 15 mins later (direction = 1)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(*viewmodel.Model)

	// Press Enter to confirm move
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit_recurring confirmation dialog after move")
	}
	if m.RecurringEditFromForm {
		t.Errorf("expected RecurringEditFromForm to be false when moving")
	}

	// Select option 1 (This and all future occurrences)
	m.ConfirmSelectedIndex = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	// Verify all instances are shifted to 09:45 AM
	tasks = database.GetTasks()
	foundShifted := false
	for _, tVal := range tasks {
		if tVal.RecurringParentUUID == "habit-parent-uuid" {
			foundShifted = true
			h, min, _ := tVal.TimeWindow.Start.Clock()
			if h != 9 || min != 45 {
				t.Errorf("expected habit time to be moved to 09:45, got %02d:%02d", h, min)
			}
		}
	}
	if !foundShifted {
		t.Fatalf("expected to find shifted habit instances in DB")
	}
}

