package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSplitTitleGeneration(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Task A", "Task A pt.2"},
		{"Task A pt.2", "Task A pt.3"},
		{"Task A pt.3", "Task A pt.4"},
		{"Task A pt. 2", "Task A pt.3"},
		{"Task A (pt. 2)", "Task A pt.3"},
		{"Task A (pt.2)", "Task A pt.3"},
		{"Task A - pt.2", "Task A pt.3"},
		{"Task A pt.9", "Task A pt.10"},
		{"Task A (remaining)", "Task A pt.2"},
		{"Refactor auth engine", "Refactor auth engine pt.2"},
		{"Refactor auth engine pt.2", "Refactor auth engine pt.3"},
	}

	for _, c := range cases {
		got := model.GenerateSplitTitle(c.input)
		if got != c.expected {
			t.Errorf("GenerateSplitTitle(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestTaskAnchoringModalAndMultipleSplit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, _ := sync.NewSyncEngine(database, nil, nil)
	m := viewmodel.NewModel(database, syncEngine)

	// Create a floating task with 135 mins (3 SP)
	taskA := model.Task{
		UUID:                  "task-split-a",
		WorkspaceUUID:         m.ActiveWorkspaceUUID,
		Title:                 "Design System",
		SchedulingType:        model.Floating,
		StoryPoints:           3,
		EstimatedDurationMins: 135,
		LifecycleState:        model.StateReady,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	database.AddTask(taskA)
	m.RefreshTasks()

	// 1. Anchor first 45 mins
	m.SelectedTaskUUID = "task-split-a"
	m.CurrentView = viewmodel.DayView
	m.TodoShelfFocus = true

	// Press 'a'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.AnchorPromptOpen {
		t.Fatalf("expected anchor prompt to be open")
	}

	m.AnchorTimeInput.SetValue("10:00")
	m.AnchorDurationInput.SetValue("45")

	// Press Enter -> should NOT silently split, should open confirm modal
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.ConfirmOpen || m.ConfirmActionType != "anchor_remaining_confirm" {
		t.Fatalf("expected anchor_remaining_confirm modal, got ConfirmOpen=%t, Action=%s", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.ShrinkRemainingMins != 90 {
		t.Errorf("expected remaining mins 90, got %d", m.ShrinkRemainingMins)
	}

	// Press Enter to confirm "Log to Shelf" (option 0)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check DB state: Task A is anchored (45m, 1 SP); "Design System pt.2" is floating (90m, 2 SP)
	tasks := database.GetTasks()
	var orig, pt2 *model.Task
	for _, tk := range tasks {
		tkCopy := tk
		if tk.UUID == "task-split-a" {
			orig = &tkCopy
		} else if strings.Contains(tk.Title, "pt.2") {
			pt2 = &tkCopy
		}
	}

	if orig == nil || pt2 == nil {
		t.Fatalf("expected orig and pt2 to exist, got orig=%v, pt2=%v", orig, pt2)
	}
	if orig.SchedulingType != model.Anchored {
		t.Errorf("expected orig to be anchored, got %s", orig.SchedulingType)
	}
	if orig.EstimatedDurationMins != 45 || orig.StoryPoints != 1 {
		t.Errorf("expected orig 45m and 1 SP, got %dm and %d SP", orig.EstimatedDurationMins, orig.StoryPoints)
	}
	if pt2.Title != "Design System pt.2" {
		t.Errorf("expected pt2 title 'Design System pt.2', got %q", pt2.Title)
	}
	if pt2.SchedulingType != model.Floating {
		t.Errorf("expected pt2 to be floating, got %s", pt2.SchedulingType)
	}
	if pt2.EstimatedDurationMins != 90 || pt2.StoryPoints != 2 {
		t.Errorf("expected pt2 90m and 2 SP, got %dm and %d SP", pt2.EstimatedDurationMins, pt2.StoryPoints)
	}

	// 2. Multiple split: Now anchor "Design System pt.2" for 45 mins -> should produce "Design System pt.3"
	m.SelectedTaskUUID = pt2.UUID
	m.TodoShelfFocus = true

	// Press 'a' on pt2
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.AnchorPromptOpen {
		t.Fatalf("expected anchor prompt to be open for pt2")
	}

	m.AnchorTimeInput.SetValue("11:00")
	m.AnchorDurationInput.SetValue("45")

	// Press Enter -> modal
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.ConfirmOpen || m.ConfirmActionType != "anchor_remaining_confirm" {
		t.Fatalf("expected anchor_remaining_confirm modal for pt2, got ConfirmOpen=%t, Action=%s", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.ShrinkRemainingMins != 45 {
		t.Errorf("expected remaining mins 45, got %d", m.ShrinkRemainingMins)
	}

	// Press Enter to confirm "Log to Shelf"
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check DB state: should now also have "Design System pt.3" with 45m
	tasks = database.GetTasks()
	var pt3 *model.Task
	for _, tk := range tasks {
		tkCopy := tk
		if strings.Contains(tk.Title, "pt.3") {
			pt3 = &tkCopy
		}
	}
	if pt3 == nil {
		t.Fatalf("expected pt3 to be created from multiple split")
	}
	if pt3.Title != "Design System pt.3" {
		t.Errorf("expected pt3 title 'Design System pt.3', got %q", pt3.Title)
	}
	if pt3.SchedulingType != model.Floating {
		t.Errorf("expected pt3 to be floating, got %s", pt3.SchedulingType)
	}
	if pt3.EstimatedDurationMins != 45 || pt3.StoryPoints != 1 {
		t.Errorf("expected pt3 45m and 1 SP, got %dm and %d SP", pt3.EstimatedDurationMins, pt3.StoryPoints)
	}
}

func TestPartialAnchoringDiscard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	syncEngine, _ := sync.NewSyncEngine(database, nil, nil)
	m := viewmodel.NewModel(database, syncEngine)

	taskB := model.Task{
		UUID:                  "task-split-b",
		WorkspaceUUID:         m.ActiveWorkspaceUUID,
		Title:                 "Review Pull Request",
		SchedulingType:        model.Floating,
		StoryPoints:           2,
		EstimatedDurationMins: 90,
		LifecycleState:        model.StateReady,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	database.AddTask(taskB)
	m.RefreshTasks()

	m.SelectedTaskUUID = "task-split-b"
	m.CurrentView = viewmodel.DayView
	m.TodoShelfFocus = true

	// Press 'a'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.AnchorTimeInput.SetValue("14:00")
	m.AnchorDurationInput.SetValue("45")

	// Submit anchor prompt
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.ConfirmOpen || m.ConfirmActionType != "anchor_remaining_confirm" {
		t.Fatalf("expected anchor_remaining_confirm modal")
	}

	// Press right to select "Discard" (index 1), then Enter
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.ConfirmSelectedIndex != 1 {
		t.Fatalf("expected ConfirmSelectedIndex=1 for Discard, got %d", m.ConfirmSelectedIndex)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check DB: taskB is anchored for 45 mins, NO remainder task exists
	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected only 1 task in DB (remaining discarded), got %d", len(tasks))
	}
	if tasks[0].SchedulingType != model.Anchored {
		t.Errorf("expected task to be anchored, got %s", tasks[0].SchedulingType)
	}
	if tasks[0].EstimatedDurationMins != 45 {
		t.Errorf("expected duration 45m, got %d", tasks[0].EstimatedDurationMins)
	}
}
