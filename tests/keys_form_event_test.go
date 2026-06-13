package tests

import (
	"testing"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"
)

func TestEventTaskFormCreation(t *testing.T) {
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

	// Fill the form fields for an Event
	m.Form.TitleInput.SetValue("Launch Event")
	m.Form.DescInput.SetValue("Google I/O keynote")
	m.Form.PriorityIdx = 1 // High priority
	m.Form.TaskTypeIdx = 4 // Event task type
	m.Form.StartTimeInput.SetValue("10:00")
	m.Form.DurationInput.SetValue("120")
	m.Form.LocationInput.SetValue("Mountain View")
	m.Form.CommuteInput.SetValue("30")
	m.Form.TagsInput.SetValue("tech, google")

	// Submit the form
	m.SubmitForm()

	// Retrieve the created task from database
	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in database, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Launch Event" {
		t.Errorf("expected Title 'Launch Event', got '%s'", task.Title)
	}
	if task.SchedulingType != model.Event {
		t.Errorf("expected SchedulingType 'EVENT', got '%s'", task.SchedulingType)
	}
	if task.Location != "Mountain View" {
		t.Errorf("expected Location 'Mountain View', got '%s'", task.Location)
	}
	if task.CommuteBuffer != 30 {
		t.Errorf("expected CommuteBuffer 30, got %d", task.CommuteBuffer)
	}

	// Verify that CalculateTaskRestTime returns 0 for Event
	rest := viewmodel.CalculateTaskRestTime(task)
	if rest != 0 {
		t.Errorf("expected CalculateTaskRestTime to return 0 for Event, got %v", rest)
	}
}
