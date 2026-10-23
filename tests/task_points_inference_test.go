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

func TestStoryPointsInference(t *testing.T) {
	spCases := []struct {
		durationMins int
		expectedSP   int
	}{
		{0, 0},
		{-5, 0},
		{1, 1},
		{15, 1},
		{45, 1},
		{46, 2},
		{60, 2},
		{90, 2},
		{91, 3},
		{120, 3},
		{135, 3},
		{136, 5},
		{180, 5},
		{225, 5},
		{226, 8},
		{300, 8},
		{360, 8},
		{361, 13},
		{500, 13},
	}

	for _, c := range spCases {
		got := model.InferStoryPoints(c.durationMins)
		if got != c.expectedSP {
			t.Errorf("InferStoryPoints(%d) = %d, expected %d", c.durationMins, got, c.expectedSP)
		}
	}

	durCases := []struct {
		points      int
		expectedDur int
	}{
		{0, 0},
		{1, 45},
		{2, 90},
		{3, 135},
		{5, 225},
		{8, 360},
		{13, 585},
	}

	for _, c := range durCases {
		got := model.InferDurationFromPoints(c.points)
		if got != c.expectedDur {
			t.Errorf("InferDurationFromPoints(%d) = %d, expected %d", c.points, got, c.expectedDur)
		}
	}
}

func TestTaskCreationInference(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, _ := sync.NewSyncEngine(database, nil, nil)
	m := viewmodel.NewModel(database, syncEngine)

	// 1. Default form values: 1 SP and 45 mins
	form := viewmodel.NewTaskForm()
	if form.SPIdx != 1 {
		t.Errorf("expected default SPIdx to be 1 (1 SP), got %d", form.SPIdx)
	}
	if form.DurationInput.Value() != "45" {
		t.Errorf("expected default DurationInput to be '45', got %q", form.DurationInput.Value())
	}

	// 2. Cycling SP updates DurationInput
	m.CurrentMode = viewmodel.ModeForm
	m.Form = viewmodel.NewTaskForm()
	m.Form.ActiveField = 3 // SPIdx
	// Press right to move from 1 SP (idx 1) to 2 SP (idx 2)
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Form.DurationInput.Value() != "90" {
		t.Errorf("expected DurationInput '90' for 2 SP, got %q", m.Form.DurationInput.Value())
	}

	// 3. Typing DurationInput updates SPIdx
	m.Form.ActiveField = 6 // DurationInput
	m.Form.DurationInput.SetValue("135")
	// Send any rune to trigger DurationInput update
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})
	// 135 mins -> 3 SP (idx 3 in SPOptions {0, 1, 2, 3, 5, 8, 13})
	if m.Form.SPIdx != 3 {
		t.Errorf("expected SPIdx 3 (3 SP) for 135 mins, got %d", m.Form.SPIdx)
	}

	// 4. Submit form creates floating task with inferred SP and Duration
	m.Form.TitleInput.SetValue("Auto Inferred Task")
	m.Form.IsAnchoredIdx = 0 // Floating
	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task created, got %d", len(tasks))
	}
	tk := tasks[0]
	if tk.EstimatedDurationMins != 135 {
		t.Errorf("expected EstimatedDurationMins 135, got %d", tk.EstimatedDurationMins)
	}
	if tk.StoryPoints != 3 {
		t.Errorf("expected StoryPoints 3, got %d", tk.StoryPoints)
	}
}

func TestSeparateTaskSemanticsOnDeanchorAndResize(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, _ := sync.NewSyncEngine(database, nil, nil)
	m := viewmodel.NewModel(database, syncEngine)

	// Setup: Anchored Task A (45m, 1 SP) and Floating Task A pt.2 (90m, 2 SP)
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	taskA := model.Task{
		UUID:                  "task-a",
		WorkspaceUUID:         m.ActiveWorkspaceUUID,
		Title:                 "Task A",
		SchedulingType:        model.Anchored,
		TimeWindow:            model.TimeWindow{Start: startTime, End: startTime.Add(45 * time.Minute)},
		StoryPoints:           1,
		EstimatedDurationMins: 45,
		LifecycleState:        model.StateScheduled,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	taskApt2 := model.Task{
		UUID:                  "task-a-pt2",
		WorkspaceUUID:         m.ActiveWorkspaceUUID,
		Title:                 "Task A pt.2",
		SchedulingType:        model.Floating,
		StoryPoints:           2,
		EstimatedDurationMins: 90,
		LifecycleState:        model.StateReady,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	database.AddTask(taskA)
	database.AddTask(taskApt2)
	m.RefreshTasks()

	// 1. Deanchor Task A
	m.SelectedTaskUUID = "task-a"
	m.CurrentView = viewmodel.DayView
	m.TodoShelfFocus = false

	// Press 'a' to open deanchor confirmation
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.ConfirmOpen || m.ConfirmActionType != "deanchor" {
		t.Fatalf("expected deanchor confirmation modal, got %t %s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// Press Enter to confirm deanchor
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify Task A is floating with 45m and 1 SP
	uA, _ := database.GetTask("task-a")
	if uA.SchedulingType != model.Floating {
		t.Errorf("expected task-a to be floating after deanchor, got %s", uA.SchedulingType)
	}
	if uA.EstimatedDurationMins != 45 || uA.StoryPoints != 1 {
		t.Errorf("expected task-a 45m and 1 SP, got %dm and %d SP", uA.EstimatedDurationMins, uA.StoryPoints)
	}

	// Verify Task A pt.2 was completely UNTOUCHED
	uApt2, _ := database.GetTask("task-a-pt2")
	if uApt2.Title != "Task A pt.2" || uApt2.SchedulingType != model.Floating || uApt2.EstimatedDurationMins != 90 || uApt2.StoryPoints != 2 {
		t.Errorf("Task A pt.2 was unexpectedly altered by deanchoring Task A: %+v", uApt2)
	}

	// 2. Re-anchor Task A and resize / increase its duration
	uA.SchedulingType = model.Anchored
	uA.TimeWindow = model.TimeWindow{Start: startTime, End: startTime.Add(60 * time.Minute)}
	uA.EstimatedDurationMins = 60
	uA.StoryPoints = 2
	database.UpdateTask(uA)
	m.RefreshTasks()

	// Verify Task A pt.2 is still completely unaffected
	uApt2After, _ := database.GetTask("task-a-pt2")
	if uApt2After.EstimatedDurationMins != 90 || uApt2After.StoryPoints != 2 {
		t.Errorf("Task A pt.2 was altered when Task A duration changed: %+v", uApt2After)
	}
}
