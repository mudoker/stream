package tests

import (
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"
)

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
		defaultH int
		defaultM int
	}{
		{"single digit hour", "14", 14, 0, 9, 0},
		{"single digit hour with default", "9", 9, 0, 9, 0},
		{"hour with colon and minutes", "14:30", 14, 30, 9, 0},
		{"leading zero", "09:15", 9, 15, 9, 0},
		{"invalid hour out of range", "25", 9, 0, 9, 0},
		{"invalid hour with colon", "25:30", 9, 0, 9, 0},
		{"invalid minutes", "14:75", 9, 0, 9, 0},
		{"zero hour", "0", 0, 0, 9, 0},
		{"zero hour with minutes", "0:15", 0, 15, 9, 0},
		{"empty string", "", 9, 0, 9, 0},
		{"whitespace", "  ", 9, 0, 9, 0},
		{"hour with whitespace", "  14  ", 14, 0, 9, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := viewmodel.ParseFlexibleTime(tt.input, tt.defaultH, tt.defaultM)
			if h != tt.wantHour || m != tt.wantMin {
				t.Errorf("ParseFlexibleTime(%q) = (%d, %d), want (%d, %d)",
					tt.input, h, m, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestFlexibleTimeInContext(t *testing.T) {
	tests := []struct {
		name         string
		timeInput    string
		expectedHour int
		expectedMin  int
	}{
		{"14 becomes 14:00", "14", 14, 0},
		{"14:30 stays 14:30", "14:30", 14, 30},
		{"9 becomes 9:00", "9", 9, 0},
		{"09:15 stays 09:15", "09:15", 9, 15},
	}

	selectedDay := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, min := viewmodel.ParseFlexibleTime(tt.timeInput, 9, 0)

			result := time.Date(selectedDay.Year(), selectedDay.Month(), selectedDay.Day(), hour, min, 0, 0, time.UTC)

			if result.Hour() != tt.expectedHour || result.Minute() != tt.expectedMin {
				t.Errorf("Expected %02d:%02d, got %02d:%02d",
					tt.expectedHour, tt.expectedMin, result.Hour(), result.Minute())
			}
		})
	}
}

func TestGetTodoShelfTasksIncludesReminders(t *testing.T) {
	now := time.Now()
	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "2",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow: model.TimeWindow{
					Start: now.Add(10 * time.Minute),
				},
			},
			{
				UUID:           "3",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateScheduled,
				TimeWindow: model.TimeWindow{
					Start: now.Add(30 * time.Minute),
					End:   now.Add(90 * time.Minute),
				},
			},
		},
	}

	shelf := m.GetTodoShelfTasks()
	if len(shelf) != 2 {
		t.Fatalf("expected 2 tasks in todo shelf, got %d", len(shelf))
	}
	if shelf[0].UUID != "1" && shelf[0].UUID != "2" {
		t.Fatalf("unexpected TODO shelf task UUID %s", shelf[0].UUID)
	}
}

func TestHabitCreationFormSubmit(t *testing.T) {
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
	m.Form = viewmodel.NewTaskForm()
	m.Form.TitleInput.SetValue("Drink Water")
	m.Form.DescInput.SetValue("8 glasses a day")
	m.Form.PriorityIdx = 2 // Medium
	m.Form.SPIdx = 2       // 2 SP
	m.Form.TaskTypeIdx = 3 // Habit
	m.Form.TagsInput.SetValue("health, daily")

	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in DB, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Drink Water" {
		t.Errorf("expected title 'Drink Water', got %q", task.Title)
	}
	if task.Description != "8 glasses a day" {
		t.Errorf("expected description '8 glasses a day', got %q", task.Description)
	}
	if task.SchedulingType != model.Habit {
		t.Errorf("expected SchedulingType to be Habit, got %s", task.SchedulingType)
	}
	if len(task.Tags) != 2 || task.Tags[0] != "health" || task.Tags[1] != "daily" {
		t.Errorf("unexpected tags: %v", task.Tags)
	}
}

func TestHabitCommandPalette(t *testing.T) {
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
	m.RunCommand("habit Read Books")

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in DB, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Read Books" {
		t.Errorf("expected title 'Read Books', got %q", task.Title)
	}
	if task.SchedulingType != model.Habit {
		t.Errorf("expected SchedulingType to be Habit, got %s", task.SchedulingType)
	}
	if task.StoryPoints != 0 {
		t.Errorf("expected StoryPoints to be 0, got %d", task.StoryPoints)
	}
}
