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
	m.Form.TaskTypeIdx = 3 // Event task type
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

func TestEventTaskFormCreationNoLocation(t *testing.T) {
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

	// Fill the form fields for an Event, but keep Location empty
	m.Form.TitleInput.SetValue("Remote Sync")
	m.Form.DescInput.SetValue("Zoom meeting")
	m.Form.PriorityIdx = 1
	m.Form.TaskTypeIdx = 3 // Event task type
	m.Form.StartTimeInput.SetValue("14:00")
	m.Form.DurationInput.SetValue("60")
	m.Form.LocationInput.SetValue("") // empty location
	m.Form.CommuteInput.SetValue("30") // entered but location is empty
	m.Form.TagsInput.SetValue("meeting")

	// Verify field 8 is not visible
	visible := m.Form.VisibleFields()
	for _, fld := range visible {
		if fld == 8 {
			t.Errorf("field 8 (Commute Buffer) should not be visible when location is empty")
		}
	}

	// Submit the form
	m.SubmitForm()

	// Retrieve the created task from database
	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in database, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Remote Sync" {
		t.Errorf("expected Title 'Remote Sync', got '%s'", task.Title)
	}
	if task.Location != "" {
		t.Errorf("expected Location to be empty, got '%s'", task.Location)
	}
	if task.CommuteBuffer != 0 {
		t.Errorf("expected CommuteBuffer to be 0 since location was empty, got %d", task.CommuteBuffer)
	}
}

func TestEventTaskFormCreationStartEndDates(t *testing.T) {
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

	// Fill the form fields for an Event (scoped within a single day)
	m.Form.TitleInput.SetValue("Single-day Event")
	m.Form.DescInput.SetValue("Hackathon weekend")
	m.Form.PriorityIdx = 1
	m.Form.TaskTypeIdx = 3 // Event
	m.Form.StartDateInput.SetValue("2026-06-20")
	m.Form.StartTimeInput.SetValue("09:00")
	m.Form.DurationInput.SetValue("180") // 3 hours duration

	// Submit form
	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in database, got %d", len(tasks))
	}

	task := tasks[0]
	if task.SchedulingType != model.Event {
		t.Errorf("expected EVENT scheduling type, got %s", task.SchedulingType)
	}

	expectedStartStr := "2026-06-20 09:00:00"
	expectedEndStr := "2026-06-20 12:00:00" // 09:00 + 180 min

	actualStartStr := task.TimeWindow.Start.Local().Format("2006-01-02 15:04:05")
	actualEndStr := task.TimeWindow.End.Local().Format("2006-01-02 15:04:05")

	if actualStartStr != expectedStartStr {
		t.Errorf("expected start %s, got %s", expectedStartStr, actualStartStr)
	}
	if actualEndStr != expectedEndStr {
		t.Errorf("expected end %s, got %s", expectedEndStr, actualEndStr)
	}
}
