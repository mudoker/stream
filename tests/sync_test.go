package tests

import (
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/sync"
	"stream/internal/viewmodel"
)

func TestIsGCalSyncable(t *testing.T) {
	tests := []struct {
		name     string
		taskType model.SchedulingType
		want     bool
	}{
		{"Anchored task is syncable", model.Anchored, true},
		{"Event task is syncable", model.Event, true},
		{"Floating task is not syncable", model.Floating, false},
		{"Reminder task is not syncable", model.Reminder, false},
		{"Habit task is not syncable", model.Habit, false},
		{"Recurring task is not syncable", model.Recurring, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := model.Task{SchedulingType: tt.taskType}
			got := model.IsGCalSyncable(task)
			if got != tt.want {
				t.Errorf("IsGCalSyncable() for %s = %v, want %v", tt.taskType, got, tt.want)
			}
		})
	}
}

func TestManualSyncCommandsExecution(t *testing.T) {
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

	// Test :pull command routing
	_, _ = m.RunCommand("pull")
	if m.StatusMsg != "Pulling tasks from Google Calendar..." {
		t.Errorf("expected pull command message, got %q", m.StatusMsg)
	}

	// Test :push command routing
	_, _ = m.RunCommand("push")
	if m.StatusMsg != "Pushing tasks to Google Calendar..." {
		t.Errorf("expected push command message, got %q", m.StatusMsg)
	}

	// Test :sync command routing (should now be unknown since they are separate manual commands)
	_, _ = m.RunCommand("sync")
	if m.StatusMsg != "Unknown command: sync" {
		t.Errorf("expected unknown command message for sync, got %q", m.StatusMsg)
	}
}

func TestReminderDueDateSubmit(t *testing.T) {
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
	m.Form.TitleInput.SetValue("Doctor appointment")
	m.Form.DescInput.SetValue("Annual checkup")
	m.Form.PriorityIdx = 1 // High
	m.Form.SPIdx = 0
	m.Form.TaskTypeIdx = 1 // Reminder
	m.Form.DueDateInput.SetValue("2026-07-20")
	m.Form.StartTimeInput.SetValue("14:30") // Due time

	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Doctor appointment" {
		t.Errorf("expected title 'Doctor appointment', got %q", task.Title)
	}
	if task.SchedulingType != model.Reminder {
		t.Errorf("expected SchedulingType to be Reminder, got %s", task.SchedulingType)
	}

	expectedTime := time.Date(2026, 7, 20, 14, 30, 0, 0, time.Now().Location())
	if !task.TimeWindow.Start.Equal(expectedTime) {
		t.Errorf("expected due datetime %s, got %s", expectedTime, task.TimeWindow.Start)
	}
}

func TestDBInvalidTasksCleanup(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	database.AddTaskNoLedger(model.Task{
		UUID:           "valid-task",
		Title:          "Valid Task",
		SchedulingType: model.Anchored,
		LifecycleState: model.StateScheduled,
	})
	database.AddTaskNoLedger(model.Task{
		UUID:           "empty-title-task",
		Title:          "",
		SchedulingType: model.Anchored,
		LifecycleState: model.StateScheduled,
	})
	database.AddTaskNoLedger(model.Task{
		UUID:           "invalid-type-task",
		Title:          "Invalid Type Task",
		SchedulingType: "UNKNOWN",
		LifecycleState: model.StateReady,
	})
	database.AddTaskNoLedger(model.Task{
		UUID:           "completed-invalid-type-task",
		Title:          "Completed Invalid Type Task",
		SchedulingType: "UNKNOWN",
		LifecycleState: model.StateCompleted,
	})

	database2, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to reload database: %v", err)
	}

	tasks := database2.GetTasks()
	
	hasTask := func(uuid string) bool {
		for _, tk := range tasks {
			if tk.UUID == uuid {
				return true
			}
		}
		return false
	}

	if !hasTask("valid-task") {
		t.Errorf("expected valid task to be retained")
	}
	if hasTask("empty-title-task") {
		t.Errorf("expected empty title task to be cleaned up")
	}
	if hasTask("invalid-type-task") {
		t.Errorf("expected invalid type task to be cleaned up")
	}
	if !hasTask("completed-invalid-type-task") {
		t.Errorf("expected completed invalid type task to be retained")
	}
}

func TestAutoSyncEndToEndIntegration(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	ws := model.Workspace{UUID: "ws-1", Name: "Default"}
	_ = database.AddWorkspace(ws)

	var logMessages []string
	syncEngine, err := sync.NewSyncEngine(database, func(msg string) {
		logMessages = append(logMessages, msg)
	}, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	m := viewmodel.NewModel(database, syncEngine)

	// Verify default sync settings: Two-Way mode
	settings := database.GetUserSettings().NormalizedGCalSync()
	if settings.GCalSyncMode != model.GCalSyncTwoWay {
		t.Errorf("expected default sync mode to be GCalSyncTwoWay, got %s", settings.GCalSyncMode)
	}

	// 1. Add anchored task via ViewModel -> should trigger push sync
	task := model.Task{
		UUID:           "task-anchored-auto",
		WorkspaceUUID:  "ws-1",
		Title:          "Morning Standup",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 11, 9, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 11, 9, 30, 0, 0, time.UTC),
		},
	}
	m.AddTask(task)

	ledger := database.GetLedger()
	if len(ledger) != 1 || ledger[0].Op != "CREATE" {
		t.Errorf("expected 1 CREATE entry in ledger for added anchored task, got %d", len(ledger))
	}

	// 2. Update task via ViewModel -> should record UPDATE in ledger
	task.Title = "Morning Standup & Sync"
	m.UpdateTask(task)

	ledger = database.GetLedger()
	if len(ledger) != 2 || ledger[1].Op != "UPDATE" {
		t.Errorf("expected UPDATE entry in ledger, got %v", ledger)
	}

	// 3. Delete task via ViewModel -> should record DELETE in ledger
	m.DeleteTask("task-anchored-auto")

	ledger = database.GetLedger()
	if len(ledger) != 3 || ledger[2].Op != "DELETE" {
		t.Errorf("expected DELETE entry in ledger, got %v", ledger)
	}
}







