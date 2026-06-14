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

func TestReminderCreationFormSubmit(t *testing.T) {
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

	// 1. Submit Reminder with empty due time (should have Second() == 1)
	m.Form = viewmodel.NewTaskForm()
	m.Form.TitleInput.SetValue("Call Mom")
	m.Form.DescInput.SetValue("Weekly call")
	m.Form.PriorityIdx = 1 // High
	m.Form.SPIdx = 3       // 3 SP
	m.Form.TaskTypeIdx = 2 // Reminder
	m.Form.StartTimeInput.SetValue("") // empty due time
	m.Form.DueDateInput.SetValue("2026-06-12")

	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in DB, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Call Mom" {
		t.Errorf("expected title 'Call Mom', got %q", task.Title)
	}
	if task.SchedulingType != model.Reminder {
		t.Errorf("expected SchedulingType to be Reminder, got %s", task.SchedulingType)
	}
	if task.StoryPoints != 0 {
		t.Errorf("expected StoryPoints for reminder to be forced to 0, got %d", task.StoryPoints)
	}
	if task.TimeWindow.Start.Second() != 1 {
		t.Errorf("expected empty due time to have second sentinel == 1, got %d", task.TimeWindow.Start.Second())
	}

	// 2. Submit Reminder with specific due time (should have Second() == 0)
	m.Form = viewmodel.NewTaskForm()
	m.Form.TitleInput.SetValue("Doctor Appointment")
	m.Form.DescInput.SetValue("Checkup")
	m.Form.PriorityIdx = 0 // Critical
	m.Form.SPIdx = 4       // 5 SP
	m.Form.TaskTypeIdx = 2 // Reminder
	m.Form.StartTimeInput.SetValue("14:30")
	m.Form.DueDateInput.SetValue("2026-06-15")

	m.SubmitForm()

	tasks = database.GetTasks()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks in DB, got %d", len(tasks))
	}

	var task2 model.Task
	found := false
	for _, t := range tasks {
		if t.Title == "Doctor Appointment" {
			task2 = t
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find Doctor Appointment task")
	}
	if task2.TimeWindow.Start.Second() != 0 {
		t.Errorf("expected due time 14:30 to have second == 0, got %d", task2.TimeWindow.Start.Second())
	}
	if task2.TimeWindow.Start.Hour() != 14 || task2.TimeWindow.Start.Minute() != 30 {
		t.Errorf("expected due time 14:30, got %02d:%02d", task2.TimeWindow.Start.Hour(), task2.TimeWindow.Start.Minute())
	}
	if task2.StoryPoints != 0 {
		t.Errorf("expected StoryPoints to be forced to 0, got %d", task2.StoryPoints)
	}
}

func TestRecurringTaskLifecycle(t *testing.T) {
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
	m.SelectedDay = time.Date(2026, 6, 14, 0, 0, 0, 0, time.Local)

	// 1. Create a recurring task (Gym) on Mon, Wed, Fri
	// Expected end date: 2026-06-20 (7 days later)
	// SelectedDay 2026-06-14 is a Sunday.
	// Mon: 15, Tue: 16, Wed: 17, Thu: 18, Fri: 19, Sat: 20
	// Recurring days: Mon, Wed, Fri (15, 17, 19) -> 3 occurrences.
	m.Form = viewmodel.NewTaskForm()
	m.Form.TitleInput.SetValue("Gym Workout")
	m.Form.DescInput.SetValue("Push day")
	m.Form.PriorityIdx = 1 // High
	m.Form.SPIdx = 2       // 2 SP
	m.Form.TaskTypeIdx = 0 // Anchored
	m.Form.IsRecurringIdx = 1 // Yes
	m.Form.RecurringEndDateInput.SetValue("2026-06-20")
	m.Form.RecurringDaysInput.SetValue("Mon, Wed, Fri")
	m.Form.StartTimeInput.SetValue("08:00")
	m.Form.DurationInput.SetValue("60")

	m.SubmitForm()

	tasks := database.GetTasks()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 recurring tasks created, got %d", len(tasks))
	}

	// Verify all instances are anchored and have the same recurring parent UUID
	parentUUID := tasks[0].RecurringParentUUID
	if parentUUID == "" {
		t.Fatalf("expected RecurringParentUUID to be set")
	}

	for _, task := range tasks {
		if task.RecurringParentUUID != parentUUID {
			t.Errorf("expected parent UUID to be %s, got %s", parentUUID, task.RecurringParentUUID)
		}
		if !model.IsTaskAnchored(task) {
			t.Errorf("expected recurring task to be anchored by default")
		}
	}

	// Find the Mon 15th instance
	var firstTask model.Task
	for _, tk := range tasks {
		if tk.TimeWindow.Start.Day() == 15 {
			firstTask = tk
		}
	}

	// 2. Test edit "Only this occurrence"
	// Select the first task and press "e" to start edit
	m.SelectedDay = firstTask.TimeWindow.Start
	m.SelectedTaskUUID = firstTask.UUID
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	if m.CurrentMode != viewmodel.ModeForm || m.Form.TitleInput.Value() != "Gym Workout" {
		t.Fatalf("expected to be in edit form mode for Gym Workout")
	}

	// Change title and submit form
	m.Form.TitleInput.SetValue("Gym Workout - Mon")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Since it's a recurring task edit, it should open edit confirmation prompt
	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit recurring confirmation modal to be open")
	}

	// Submit confirmation: [1] Only this occurrence
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})

	tasks = database.GetTasks()
	// Find the edited task
	var editedTask model.Task
	for _, tk := range tasks {
		if tk.UUID == firstTask.UUID {
			editedTask = tk
		}
	}
	if editedTask.Title != "Gym Workout - Mon" {
		t.Errorf("expected only first task's title to change, got %q", editedTask.Title)
	}
	// Verify other tasks didn't change title
	for _, tk := range tasks {
		if tk.UUID != firstTask.UUID {
			if tk.Title != "Gym Workout" {
				t.Errorf("expected other tasks to remain 'Gym Workout', got %q", tk.Title)
			}
		}
	}

	// 3. Test edit "This and all remaining occurrences"
	// Let's edit the second instance (Wed 17th) to "Heavy Gym Workout"
	var secondTask model.Task
	for _, tk := range tasks {
		if tk.TimeWindow.Start.Day() == 17 {
			secondTask = tk
		}
	}
	m.SelectedDay = secondTask.TimeWindow.Start
	m.SelectedTaskUUID = secondTask.UUID
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m.Form.TitleInput.SetValue("Heavy Gym Workout")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.ConfirmOpen || m.ConfirmActionType != "edit_recurring" {
		t.Fatalf("expected edit recurring confirmation modal to be open again")
	}

	// Submit confirmation: [2] This and all remaining occurrences
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})

	tasks = database.GetTasks()
	for _, tk := range tasks {
		day := tk.TimeWindow.Start.Day()
		if day == 15 {
			if tk.Title != "Gym Workout - Mon" {
				t.Errorf("expected first task title to remain 'Gym Workout - Mon', got %q", tk.Title)
			}
		} else if day == 17 || day == 19 {
			if tk.Title != "Heavy Gym Workout" {
				t.Errorf("expected remaining task title to be 'Heavy Gym Workout', got %q", tk.Title)
			}
		}
	}

	// 4. Test delete "Only this occurrence"
	// Let's delete the third instance (Fri 19th)
	var thirdTask model.Task
	for _, tk := range tasks {
		if tk.TimeWindow.Start.Day() == 19 {
			thirdTask = tk
		}
	}
	m.SelectedDay = thirdTask.TimeWindow.Start
	m.SelectedTaskUUID = thirdTask.UUID
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if !m.ConfirmOpen || m.ConfirmActionType != "delete_recurring" {
		t.Fatalf("expected delete recurring confirmation modal to be open")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})

	tasks = database.GetTasks()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks remaining after deleting one, got %d", len(tasks))
	}
	for _, tk := range tasks {
		if tk.TimeWindow.Start.Day() == 19 {
			t.Errorf("expected Fri 19th instance to be deleted")
		}
	}

	// 5. Test delete "This and all remaining occurrences"
	// Let's delete secondTask (Wed 17th) with cascade delete.
	m.SelectedDay = secondTask.TimeWindow.Start
	m.SelectedTaskUUID = secondTask.UUID
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if !m.ConfirmOpen || m.ConfirmActionType != "delete_recurring" {
		t.Fatalf("expected delete recurring confirmation modal to be open")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})

	tasks = database.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected only 1 task (Mon 15th) to remain, got %d", len(tasks))
	}
	if tasks[0].TimeWindow.Start.Day() != 15 {
		t.Errorf("expected remaining task to be Mon 15th, got day %d", tasks[0].TimeWindow.Start.Day())
	}
}

