package db

import (
	"testing"
	"time"

	"stream/internal/model"
)

func TestEventPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	db1, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db1: %v", err)
	}

	eventTask := model.Task{
		UUID:           "event-persistence-1",
		Title:          "Conference Keynote",
		SchedulingType: model.Event,
		Location:       "Main Stage",
		TimeWindow: model.TimeWindow{
			Start: time.Now(),
			End:   time.Now().Add(2 * time.Hour),
		},
	}

	err = db1.AddTask(eventTask)
	if err != nil {
		t.Fatalf("failed to add event task: %v", err)
	}

	// Re-initialize JSONDB to load from the same path (which uses HOME directory path config)
	db2, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db2: %v", err)
	}

	tasks := db2.GetTasks()
	found := false
	for _, tk := range tasks {
		if tk.UUID == "event-persistence-1" {
			found = true
			if tk.Title != "Conference Keynote" {
				t.Errorf("expected Title 'Conference Keynote', got '%s'", tk.Title)
			}
			if tk.SchedulingType != model.Event {
				t.Errorf("expected SchedulingType EVENT, got '%s'", tk.SchedulingType)
			}
			if tk.Location != "Main Stage" {
				t.Errorf("expected Location 'Main Stage', got '%s'", tk.Location)
			}
			break
		}
	}

	if !found {
		t.Errorf("event task was not loaded back from the database file")
	}
}

func TestEventPersistenceNoLocation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	db1, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db1: %v", err)
	}

	eventTask := model.Task{
		UUID:           "event-persistence-2",
		Title:          "Empty Loc Event",
		SchedulingType: model.Event,
		Location:       "",
		TimeWindow: model.TimeWindow{
			Start: time.Now(),
			End:   time.Now().Add(1 * time.Hour),
		},
	}

	err = db1.AddTask(eventTask)
	if err != nil {
		t.Fatalf("failed to add event task: %v", err)
	}

	db2, err := NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create db2: %v", err)
	}

	tasks := db2.GetTasks()
	found := false
	for _, tk := range tasks {
		if tk.UUID == "event-persistence-2" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("event task with empty location was not loaded back from the database file")
	}
}
