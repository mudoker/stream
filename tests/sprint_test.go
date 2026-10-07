package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/view"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/tasks"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func setupTestSprintDB(t *testing.T) (*db.JSONDB, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "stream_sprint_test_*")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	database, err := db.NewJSONDB()
	if err != nil {
		os.RemoveAll(tempDir)
		os.Setenv("HOME", origHome)
		t.Fatalf("could not init db: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
		os.Setenv("HOME", origHome)
	}

	return database, cleanup
}

func TestSprintDBCRUDAndRecurring(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatalf("expected default sprint, got 0")
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 14)

	customSprint := model.Sprint{
		UUID:      uuid.New().String(),
		Name:      "Beta Release Sprint",
		StartDate: start,
		EndDate:   end,
	}

	err := database.AddSprint(customSprint)
	if err != nil {
		t.Fatalf("AddSprint failed: %v", err)
	}

	fetched, exists := database.GetSprint(customSprint.UUID)
	if !exists || fetched.Name != "Beta Release Sprint" {
		t.Fatalf("GetSprint mismatch: %+v", fetched)
	}

	// Update Sprint
	fetched.Name = "Beta Release Sprint v2"
	err = database.UpdateSprint(fetched)
	if err != nil {
		t.Fatalf("UpdateSprint failed: %v", err)
	}
	updated, _ := database.GetSprint(customSprint.UUID)
	if updated.Name != "Beta Release Sprint v2" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}

	// Generate 3 recurring sprints (14 days each)
	recurring, err := database.GenerateRecurringSprints(updated, 3)
	if err != nil {
		t.Fatalf("GenerateRecurringSprints failed: %v", err)
	}
	if len(recurring) != 3 {
		t.Fatalf("expected 3 recurring sprints, got %d", len(recurring))
	}
	if !recurring[0].StartDate.Equal(updated.EndDate) {
		t.Fatalf("first recurring sprint start (%v) should match base sprint end (%v)", recurring[0].StartDate, updated.EndDate)
	}
	gap := recurring[0].EndDate.Sub(recurring[0].StartDate)
	if int(gap.Hours()/24) != 14 {
		t.Fatalf("expected 14-day gap, got %v", gap)
	}

	// Delete Sprint
	err = database.DeleteSprint(customSprint.UUID)
	if err != nil {
		t.Fatalf("DeleteSprint failed: %v", err)
	}
	_, exists = database.GetSprint(customSprint.UUID)
	if exists {
		t.Fatalf("expected deleted sprint to not exist")
	}
}

func TestSprintViewNavigationAndSwimlanes(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	task1 := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Defined",
		Priority:       model.P1,
		StoryPoints:    3,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	task2 := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Progress",
		Priority:       model.P0,
		StoryPoints:    5,
		SchedulingType: model.Floating,
		LifecycleState: model.StateActive,
	}
	task3 := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Review",
		Priority:       model.P2,
		StoryPoints:    2,
		SchedulingType: model.Floating,
		LifecycleState: model.StateReview,
	}
	task4 := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Testing",
		Priority:       model.P2,
		StoryPoints:    2,
		SchedulingType: model.Floating,
		LifecycleState: model.StateTesting,
	}
	task5 := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Task Completed",
		Priority:       model.P3,
		StoryPoints:    1,
		SchedulingType: model.Floating,
		LifecycleState: model.StateCompleted,
	}
	// Global backlog task (no SprintUUID)
	globalTask := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     "",
		Title:          "Global Backlog Task",
		Priority:       model.P2,
		StoryPoints:    2,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}

	database.AddTask(task1)
	database.AddTask(task2)
	database.AddTask(task3)
	database.AddTask(task4)
	database.AddTask(task5)
	database.AddTask(globalTask)

	m := viewmodel.NewModel(database, nil)
	v := view.NewView(&m)

	// Switch to Sprint View via key '3' (Month=2, Sprint=3, Week=4, Day=5)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if m.CurrentView != viewmodel.SprintView {
		t.Fatalf("expected SprintView, got %v", m.CurrentView)
	}

	// Verify swimlane filtering
	defined, inProgress, review, testingL, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	if len(defined) != 1 || defined[0].UUID != task1.UUID {
		t.Fatalf("expected task1 in defined, got %+v", defined)
	}
	if len(inProgress) != 1 || inProgress[0].UUID != task2.UUID {
		t.Fatalf("expected task2 in inProgress, got %+v", inProgress)
	}
	if len(review) != 1 || review[0].UUID != task3.UUID {
		t.Fatalf("expected task3 in review, got %+v", review)
	}
	if len(testingL) != 1 || testingL[0].UUID != task4.UUID {
		t.Fatalf("expected task4 in testing, got %+v", testingL)
	}
	if len(completed) != 1 || completed[0].UUID != task5.UUID {
		t.Fatalf("expected task5 in completed, got %+v", completed)
	}

	// Verify Global Backlog tasks
	globalBacklog := m.GetGlobalBacklogTasks()
	foundGlobal := false
	for _, tsk := range globalBacklog {
		if tsk.UUID == globalTask.UUID {
			foundGlobal = true
			break
		}
	}
	if !foundGlobal {
		t.Fatalf("expected globalTask in global backlog, got %+v", globalBacklog)
	}

	// Render view check
	rendered := v.Render()
	if (!strings.Contains(rendered, "DEFINED") && !strings.Contains(rendered, "DEF")) ||
		(!strings.Contains(rendered, "IN PROGRESS") && !strings.Contains(rendered, "IN PROG")) ||
		(!strings.Contains(rendered, "REVIEW") && !strings.Contains(rendered, "REV")) ||
		(!strings.Contains(rendered, "TESTING") && !strings.Contains(rendered, "TEST")) ||
		(!strings.Contains(rendered, "COMPLETED") && !strings.Contains(rendered, "DONE")) {
		t.Fatalf("rendered sprint view missing swimlane headers: %s", rendered)
	}
	if !strings.Contains(rendered, "Task") || !strings.Contains(rendered, "GLOBAL BACKLOG") {
		t.Fatalf("rendered sprint view missing task content or global backlog: %s", rendered)
	}
}

func TestSprintTaskMoveAcrossSwimlanes(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	task := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Moveable Task",
		Priority:       model.P1,
		StoryPoints:    3,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog, // Defined
	}
	database.AddTask(task)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = task.UUID
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// Move right to 'In Progress' using 'L' (1)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	updatedTask, _ := database.GetTask(task.UUID)
	if updatedTask.LifecycleState != model.StateActive {
		t.Fatalf("expected StateActive, got %s", updatedTask.LifecycleState)
	}
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane 1 (In Progress), got %d", m.SprintSwimlaneIdx)
	}

	// Move right to 'Review' using 'L' (2)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	updatedTask, _ = database.GetTask(task.UUID)
	if updatedTask.LifecycleState != model.StateReview {
		t.Fatalf("expected StateReview, got %s", updatedTask.LifecycleState)
	}
	if m.SprintSwimlaneIdx != 2 {
		t.Fatalf("expected swimlane 2 (Review), got %d", m.SprintSwimlaneIdx)
	}

	// Move right to 'Testing' using 'L' (3)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	updatedTask, _ = database.GetTask(task.UUID)
	if updatedTask.LifecycleState != model.StateTesting {
		t.Fatalf("expected StateTesting, got %s", updatedTask.LifecycleState)
	}
	if m.SprintSwimlaneIdx != 3 {
		t.Fatalf("expected swimlane 3 (Testing), got %d", m.SprintSwimlaneIdx)
	}

	// Move right to 'Completed' using 'L' (4)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	updatedTask, _ = database.GetTask(task.UUID)
	if updatedTask.LifecycleState != model.StateCompleted {
		t.Fatalf("expected StateCompleted, got %s", updatedTask.LifecycleState)
	}
	if m.SprintSwimlaneIdx != 4 {
		t.Fatalf("expected swimlane 4 (Completed), got %d", m.SprintSwimlaneIdx)
	}

	// Move left back to 'Testing' using 'H' (3)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	updatedTask, _ = database.GetTask(task.UUID)
	if updatedTask.LifecycleState != model.StateTesting {
		t.Fatalf("expected StateTesting, got %s", updatedTask.LifecycleState)
	}
	if m.SprintSwimlaneIdx != 3 {
		t.Fatalf("expected swimlane 3 (Testing), got %d", m.SprintSwimlaneIdx)
	}
}

func TestSprintAnchorAndDeanchor(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	globalTask := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     "",
		Title:          "Backlog Floating Task",
		Priority:       model.P1,
		StoryPoints:    3,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	database.AddTask(globalTask)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.SprintView

	// Focus on right tab (Global Backlog shelf)
	m.TodoShelfFocus = true
	m.SidebarFocus = false
	m.SelectedTaskUUID = globalTask.UUID

	// Press 'a' to anchor to active sprint
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	anchoredTask, _ := database.GetTask(globalTask.UUID)
	if anchoredTask.SprintUUID != activeSprint.UUID {
		t.Fatalf("expected SprintUUID %s, got %s", activeSprint.UUID, anchoredTask.SprintUUID)
	}

	// Now focus on swimlanes and deanchor using 'a'
	m.TodoShelfFocus = false
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = globalTask.UUID

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	deanchoredTask, _ := database.GetTask(globalTask.UUID)
	if deanchoredTask.SprintUUID != "" {
		t.Fatalf("expected empty SprintUUID after deanchoring, got %s", deanchoredTask.SprintUUID)
	}
}

func TestSprintToggleAddToTodayAndDayTimelineBacklog(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	sprintTask := model.Task{
		UUID:           uuid.New().String(),
		SprintUUID:     activeSprint.UUID,
		Title:          "Sprint Task for Today",
		Priority:       model.P0,
		StoryPoints:    5,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
		AddedToToday:   false,
	}
	database.AddTask(sprintTask)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = sprintTask.UUID
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// Press 't' to toggle Add to Today
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	toggledTask, _ := database.GetTask(sprintTask.UUID)
	if !toggledTask.AddedToToday {
		t.Fatalf("expected AddedToToday to be true")
	}

	// Switch to Day View (5)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	if m.CurrentView != viewmodel.DayView {
		t.Fatalf("expected DayView, got %v", m.CurrentView)
	}

	// Verify task appears on Day Timeline's Todo Shelf / Backlog tab waiting to be anchored
	shelfTasks := m.GetTodoShelfTasks()
	foundInShelf := false
	for _, tsk := range shelfTasks {
		if tsk.UUID == sprintTask.UUID {
			foundInShelf = true
			break
		}
	}
	if !foundInShelf {
		t.Fatalf("expected task with AddedToToday=true to appear in Day Timeline Todo Shelf, but was not found")
	}
}

func TestInitiateDateDayTimelineVsGlobalBacklog(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	taskToday := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Task Initiated Today",
		InitiateDate:   today,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	taskYesterday := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Task Initiated Yesterday",
		InitiateDate:   yesterday,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	taskTomorrow := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Task Initiated Tomorrow",
		InitiateDate:   tomorrow,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}

	database.AddTask(taskToday)
	database.AddTask(taskYesterday)
	database.AddTask(taskTomorrow)

	m := viewmodel.NewModel(database, nil)
	m.SelectedDay = today

	// Day timeline TodoShelf should ONLY show taskToday
	dayShelf := m.GetTodoShelfTasks()
	hasToday := false
	hasYesterday := false
	hasTomorrow := false
	for _, task := range dayShelf {
		if task.UUID == taskToday.UUID {
			hasToday = true
		}
		if task.UUID == taskYesterday.UUID {
			hasYesterday = true
		}
		if task.UUID == taskTomorrow.UUID {
			hasTomorrow = true
		}
	}
	if !hasToday || hasYesterday || hasTomorrow {
		t.Fatalf("Day timeline shelf should only contain tasks for today. Got today=%v, yesterday=%v, tomorrow=%v", hasToday, hasYesterday, hasTomorrow)
	}

	// Global Backlog should show ALL floating unassigned tasks
	globalShelf := m.GetGlobalBacklogTasks()
	hasToday = false
	hasYesterday = false
	hasTomorrow = false
	for _, task := range globalShelf {
		if task.UUID == taskToday.UUID {
			hasToday = true
		}
		if task.UUID == taskYesterday.UUID {
			hasYesterday = true
		}
		if task.UUID == taskTomorrow.UUID {
			hasTomorrow = true
		}
	}
	if !hasToday || !hasYesterday || !hasTomorrow {
		t.Fatalf("Global backlog shelf must contain all floating tasks. Got today=%v, yesterday=%v, tomorrow=%v", hasToday, hasYesterday, hasTomorrow)
	}
}

func TestHabitsAndEventsExcludedFromSprint(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	now := time.Now()
	habit := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Morning Meditation",
		SchedulingType: model.Habit,
		LifecycleState: model.StateReady,
	}
	event := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Team Sync Meeting",
		SchedulingType: model.Event,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: now.Add(1 * time.Hour),
			End:   now.Add(2 * time.Hour),
		},
	}
	reminder := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Pay Internet Bill",
		SchedulingType: model.Reminder,
		LifecycleState: model.StateReady,
		TimeWindow: model.TimeWindow{
			Start: now.Add(3 * time.Hour),
		},
	}
	floatingTask := model.Task{
		UUID:           uuid.New().String(),
		Title:          "Write Design Doc",
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}

	database.AddTask(habit)
	database.AddTask(event)
	database.AddTask(reminder)
	database.AddTask(floatingTask)

	m := viewmodel.NewModel(database, nil)
	m.SelectedDay = now

	// 1. Day Timeline TodoShelf should contain habit and reminder (appear on day)
	dayShelf := m.GetTodoShelfTasks()
	hasHabitInDay := false
	hasReminderInDay := false
	for _, tsk := range dayShelf {
		if tsk.UUID == habit.UUID {
			hasHabitInDay = true
		}
		if tsk.UUID == reminder.UUID {
			hasReminderInDay = true
		}
	}
	if !hasHabitInDay || !hasReminderInDay {
		t.Fatalf("Day Timeline TodoShelf must include habits and reminders. Got habit=%v, reminder=%v", hasHabitInDay, hasReminderInDay)
	}

	// 2. Global Backlog should ONLY contain floatingTask, NOT habits, events, or reminders
	globalBacklog := m.GetGlobalBacklogTasks()
	for _, tsk := range globalBacklog {
		if tsk.SchedulingType == model.Habit || tsk.SchedulingType == model.Event || tsk.SchedulingType == model.Reminder {
			t.Fatalf("Global Backlog must not contain habits, events, or reminders. Found: %+v", tsk)
		}
	}
	if len(globalBacklog) != 1 || globalBacklog[0].UUID != floatingTask.UUID {
		t.Fatalf("Global Backlog should only have 1 floating task, got %d", len(globalBacklog))
	}

	// 3. Anchoring a habit to sprint should be rejected
	sprints := database.GetSprints()
	activeSprint := sprints[0]
	m.CurrentView = viewmodel.SprintView
	m.TodoShelfFocus = true
	m.SelectedTaskUUID = habit.UUID

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	updatedHabit, _ := database.GetTask(habit.UUID)
	if updatedHabit.SprintUUID != "" {
		t.Fatalf("Habit should not be anchorable to sprint, but got SprintUUID: %s", updatedHabit.SprintUUID)
	}

	// 4. Sprint Swimlanes should never include habits or events
	defined, inProgress, review, testingL, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	allSprintTasks := append(append(append(append(defined, inProgress...), review...), testingL...), completed...)
	for _, tsk := range allSprintTasks {
		if tsk.SchedulingType == model.Habit || tsk.SchedulingType == model.Event || tsk.SchedulingType == model.Reminder {
			t.Fatalf("Sprint swimlanes must not contain habits or events. Found: %+v", tsk)
		}
	}
}


