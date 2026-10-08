package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/view"
	"stream/internal/view/modals"
	"stream/internal/view/pages"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/tasks"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	end := start.AddDate(0, 0, 13)

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
	expectedStart := updated.EndDate.AddDate(0, 0, 1)
	if !recurring[0].StartDate.Equal(expectedStart) {
		t.Fatalf("first recurring sprint start (%v) should be day after base sprint end (%v)", recurring[0].StartDate, expectedStart)
	}
	gap := recurring[0].EndDate.Sub(recurring[0].StartDate)
	if int(gap.Hours()/24) != 13 {
		t.Fatalf("expected 13-day delta (14 inclusive days), got %v", gap)
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
	m.Layout = viewmodel.ComputeLayout(180, 45)
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

func TestSprintSwimlaneSwitchingWithHL(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	task1 := model.Task{
		UUID:           "task-def",
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Defined",
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog, // Defined
	}
	task2 := model.Task{
		UUID:           "task-prog",
		SprintUUID:     activeSprint.UUID,
		Title:          "Task in Progress",
		SchedulingType: model.Floating,
		LifecycleState: model.StateActive, // In Progress
	}
	database.AddTask(task1)
	database.AddTask(task2)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = task1.UUID
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// Switch right to swimlane 1 (In Progress) using 'L'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane 1 (In Progress), got %d", m.SprintSwimlaneIdx)
	}
	if m.SelectedTaskUUID != "task-prog" {
		t.Fatalf("expected selection to be task-prog, got %s", m.SelectedTaskUUID)
	}

	// Switch right to swimlane 2 (Review) using 'L'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if m.SprintSwimlaneIdx != 2 {
		t.Fatalf("expected swimlane 2 (Review), got %d", m.SprintSwimlaneIdx)
	}

	// Switch left back to swimlane 1 using 'H'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane 1 (In Progress), got %d", m.SprintSwimlaneIdx)
	}
	if m.SelectedTaskUUID != "task-prog" {
		t.Fatalf("expected selection to be task-prog, got %s", m.SelectedTaskUUID)
	}

	// Switch left back to swimlane 0 using 'H'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	if m.SprintSwimlaneIdx != 0 {
		t.Fatalf("expected swimlane 0 (Defined), got %d", m.SprintSwimlaneIdx)
	}
	if m.SelectedTaskUUID != "task-def" {
		t.Fatalf("expected selection to be task-def, got %s", m.SelectedTaskUUID)
	}
}

func TestSidebarFocusIsolation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.MonthView
	initialDay := m.SelectedDay
	m.SidebarFocus = true // Focused on left sidebar tab!

	// Press 'h' and 'l' while sidebar is focused
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})

	// Selected day on month grid must NOT have changed!
	if !m.SelectedDay.Equal(initialDay) {
		t.Fatalf("expected selected day to remain %v when sidebar is focused, got %v", initialDay, m.SelectedDay)
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

func TestSprintHorizontalScrollingAndTaskNavigation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	activeSprint := sprints[0]

	task1 := model.Task{
		UUID:           "task-def-1",
		SprintUUID:     activeSprint.UUID,
		Title:          "Defined Task 1",
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	task2 := model.Task{
		UUID:           "task-def-2",
		SprintUUID:     activeSprint.UUID,
		Title:          "Defined Task 2",
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	task3 := model.Task{
		UUID:           "task-prog-1",
		SprintUUID:     activeSprint.UUID,
		Title:          "Progress Task 1",
		SchedulingType: model.Floating,
		LifecycleState: model.StateActive,
	}

	database.AddTask(task1)
	database.AddTask(task2)
	database.AddTask(task3)

	m := viewmodel.NewModel(database, nil)
	m.Layout = viewmodel.ComputeLayout(80, 24) // Narrow terminal width triggering horizontal scroll
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = task1.UUID
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// Navigate down with 'j' in lane 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.SelectedTaskUUID != "task-def-2" {
		t.Fatalf("expected task-def-2, got %s", m.SelectedTaskUUID)
	}

	// Navigate up with 'k' in lane 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.SelectedTaskUUID != "task-def-1" {
		t.Fatalf("expected task-def-1, got %s", m.SelectedTaskUUID)
	}

	// Lowercase 'l' should NOT change swimlanes (stays in lane 0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.SprintSwimlaneIdx != 0 {
		t.Fatalf("expected swimlane to remain 0 after lowercase 'l', got %d", m.SprintSwimlaneIdx)
	}

	// Capital 'L' switches across swimlanes (from lane 0 to lane 1)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane 1 after 'L', got %d", m.SprintSwimlaneIdx)
	}
	if m.SelectedTaskUUID != "task-prog-1" {
		t.Fatalf("expected task-prog-1, got %s", m.SelectedTaskUUID)
	}

	// Jump across multiple swimlanes using 'L' to last column
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")}) // 2
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")}) // 3
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")}) // 4 (COMPLETED)
	if m.SprintSwimlaneIdx != 4 {
		t.Fatalf("expected swimlane 4 (Completed), got %d", m.SprintSwimlaneIdx)
	}

	// In narrow terminal (workspaceW ~56, contentW ~52, visibleCols = 2), SprintScrollColOffset must adjust
	m.AutoScrollSprintLane()
	if m.SprintScrollColOffset < 2 {
		t.Fatalf("expected horizontal scroll offset >= 2 for lane 4 in narrow view, got %d", m.SprintScrollColOffset)
	}
}

func TestSprintStatusSelectionAndMoveMode(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatal("expected default sprint")
	}
	sprint := sprints[0]

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 2 // REVIEW swimlane
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// 1. Creating a feature in Sprint View defaults status to the focused swimlane (2: Review)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if m.Form.TaskTypeIdx != 0 {
		t.Fatalf("expected Feature type (0), got %d", m.Form.TaskTypeIdx)
	}
	if m.Form.StatusIdx != 2 {
		t.Fatalf("expected StatusIdx 2 (Review), got %d", m.Form.StatusIdx)
	}

	m.Form.TitleInput.SetValue("Code Review Checklist")
	m.SubmitForm()
	m.CurrentMode = viewmodel.ModeNormal

	// Verify task exists with StateReview
	var createdTask model.Task
	for _, tk := range m.Tasks {
		if tk.Title == "Code Review Checklist" {
			createdTask = tk
			break
		}
	}
	if createdTask.LifecycleState != model.StateReview {
		t.Fatalf("expected created task to have StateReview, got %s", createdTask.LifecycleState)
	}

	// 2. Sprint Move Mode: 'y' to enter move mode, j/k to reorder, h/l to move swimlane
	m.SelectedTaskUUID = createdTask.UUID
	m.SprintSwimlaneIdx = 2
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.CurrentMode != viewmodel.ModeSprintTaskMove {
		t.Fatalf("expected CurrentMode to be ModeSprintTaskMove, got %s", m.CurrentMode)
	}

	// Move left to lane 1 (IN PROGRESS) using 'h'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane 1 after 'h', got %d", m.SprintSwimlaneIdx)
	}
	activeTask, _ := m.GetActiveTask()
	if activeTask.LifecycleState != model.StateActive {
		t.Fatalf("expected active state, got %s", activeTask.LifecycleState)
	}

	// Confirm move with Enter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.CurrentMode != viewmodel.ModeNormal {
		t.Fatalf("expected ModeNormal after enter, got %s", m.CurrentMode)
	}

	// Verify updated in database
	dbTask, ok := database.GetTask(createdTask.UUID)
	if !ok || dbTask.LifecycleState != model.StateActive {
		t.Fatalf("expected persisted state StateActive in DB, got %v", dbTask.LifecycleState)
	}

	// 3. Test cancel move mode with 'esc'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}) // Move to Review
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.CurrentMode != viewmodel.ModeNormal {
		t.Fatalf("expected ModeNormal after cancel, got %s", m.CurrentMode)
	}
	dbTaskAfterCancel, _ := database.GetTask(createdTask.UUID)
	if dbTaskAfterCancel.LifecycleState != model.StateActive {
		t.Fatalf("expected state to remain StateActive after cancelling move, got %s", dbTaskAfterCancel.LifecycleState)
	}
}

func TestSprintViewFullHorizontalFill(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	m := viewmodel.NewModel(database, nil)
	m.Layout = viewmodel.ComputeLayout(200, 40)
	m.CurrentView = viewmodel.SprintView

	th := theme.NewTheme()
	rendered := pages.RenderSprintView(&m, th, 40)
	lines := strings.Split(rendered, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines in sprint view, got %d", len(lines))
	}

	headerSepLine := lines[1]
	sepWidth := lipgloss.Width(headerSepLine)

	// Lane row line (e.g. line 2 contains the headers of swimlanes joined with │)
	swimlaneLine := lines[2]
	swimlaneWidth := lipgloss.Width(swimlaneLine)

	if swimlaneWidth != sepWidth {
		t.Errorf("expected swimlanes joined width (%d) to equal separator width (%d) to fill all available horizontal space", swimlaneWidth, sepWidth)
	}

	// Verify all 5 swimlanes are rendered in standard width
	for _, laneName := range []string{"DEFINED", "IN PROGRESS", "REVIEW", "TESTING", "COMPLETED"} {
		if !strings.Contains(swimlaneLine, laneName) {
			t.Errorf("expected swimlane row to contain %s, got: %s", laneName, swimlaneLine)
		}
	}
}

func TestSprintDeleteFocusIsolationAndConfirmation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatal("expected at least 1 default sprint")
	}
	initialSprintUUID := sprints[0].UUID

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.SprintView

	// 1. When on TodoShelfFocus (Global Backlog), pressing 'D' must NOT delete sprint or open sprint confirm dialog
	m.TodoShelfFocus = true
	m.SidebarFocus = false

	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if m.ConfirmOpen {
		t.Fatalf("expected ConfirmOpen to be false when pressing Shift+D on Todo Shelf, got true (action: %s)", m.ConfirmActionType)
	}
	if len(database.GetSprints()) != len(sprints) {
		t.Fatalf("sprint count changed unexpectedly when pressing Shift+D on Todo Shelf")
	}

	// 2. When focused on the Sprint board, pressing 'D' must open the confirmation modal
	m.TodoShelfFocus = false
	m.SidebarFocus = false

	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if !m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be true when pressing Shift+D on focused sprint board")
	}
	if m.ConfirmActionType != "delete_sprint" {
		t.Fatalf("expected ConfirmActionType to be 'delete_sprint', got '%s'", m.ConfirmActionType)
	}
	if m.ConfirmSprint.UUID != initialSprintUUID {
		t.Fatalf("expected ConfirmSprint UUID to be %s, got %s", initialSprintUUID, m.ConfirmSprint.UUID)
	}

	// Verify the confirmation modal renders properly
	th := theme.NewTheme()
	modalRes := modals.RenderConfirmModal(&m, th)
	if !strings.Contains(modalRes, "DELETE SPRINT") {
		t.Errorf("expected modal to contain 'DELETE SPRINT', got:\n%s", modalRes)
	}

	// 3. Canceling with 'n' must NOT delete the sprint
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be false after cancel")
	}
	if len(database.GetSprints()) == 0 {
		t.Fatal("sprint was deleted despite cancelling")
	}

	// 4. Confirming deletion with 'y' must delete the sprint
	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if !m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be true on second Shift+D")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be false after confirming")
	}
	for _, s := range database.GetSprints() {
		if s.UUID == initialSprintUUID {
			t.Fatalf("expected sprint %s to be deleted from database", initialSprintUUID)
		}
	}
}

func TestSprintViewWorkItemCardRendering(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatal("expected at least one sprint")
	}
	sprint := sprints[0]

	// Create a Feature, a Defect, and an Improvement
	feat := model.Task{
		UUID:          uuid.New().String(),
		ID:            "FEAT-1",
		WorkItemType:  model.WorkItemFeature,
		Title:         "OAuth Login Support",
		Priority:      model.P1,
		StoryPoints:   5,
		SprintUUID:    sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
	}
	def := model.Task{
		UUID:          uuid.New().String(),
		ID:            "DEF-1",
		WorkItemType:  model.WorkItemDefect,
		Title:         "Crash on large payload",
		Priority:      model.P0,
		StoryPoints:   3,
		SprintUUID:    sprint.UUID,
		LifecycleState: model.StateActive,
		SchedulingType: model.Floating,
	}

	database.AddTask(feat)
	database.AddTask(def)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView
	m.Layout.TimelineW = 140

	th := theme.NewTheme()
	rendered := pages.RenderSprintView(&m, th, 30)

	if !strings.Contains(rendered, "FEAT-1") && !strings.Contains(rendered, "FEA-1") {
		t.Fatalf("expected sprint view to contain 'FEAT-1' or 'FEA-1', got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "DEF-1") {
		t.Fatalf("expected sprint view to contain 'DEF-1', got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "OAuth Login") {
		t.Fatalf("expected sprint view to contain 'OAuth Login', got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Crash on large") {
		t.Fatalf("expected sprint view to contain 'Crash on large', got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Features") {
		t.Fatalf("expected sprint view to contain 'Features' metric, got:\n%s", rendered)
	}
}

func TestGlobalBacklogTodayShelf(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatal("expected at least one sprint")
	}
	sprint := sprints[0]

	// 1. Feature in active sprint
	feat := model.Task{
		UUID:          uuid.New().String(),
		ID:            "FEAT-1",
		WorkItemType:  model.WorkItemFeature,
		Title:         "API Auth",
		Priority:      model.P1,
		SprintUUID:    sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
	}
	database.AddTask(feat)

	// 2. Task linked to FEAT-1 with AddedToToday = true
	todayTask := model.Task{
		UUID:            uuid.New().String(),
		ID:              "TASK-1",
		Title:           "Write JWT middleware",
		Priority:        model.P1,
		LinkedFeatureID: "FEAT-1",
		AddedToToday:    true,
		LifecycleState:  model.StateReady,
		SchedulingType:  model.Floating,
	}
	database.AddTask(todayTask)

	// 3. Regular backlog task (not added to today)
	backlogTask := model.Task{
		UUID:           uuid.New().String(),
		ID:             "TASK-2",
		Title:          "Write docs",
		Priority:       model.P2,
		AddedToToday:   false,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
	}
	database.AddTask(backlogTask)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView

	shelfData := m.GetShelfData()
	if !shelfData.IsGlobalBacklog {
		t.Fatalf("expected IsGlobalBacklog to be true on SprintView")
	}

	// Should have TASKS section
	var tasksSec *viewmodel.ShelfSection
	for _, sec := range shelfData.Sections {
		if sec.Type == viewmodel.SectionTasks {
			sCopy := sec
			tasksSec = &sCopy
			break
		}
	}
	if tasksSec == nil {
		t.Fatalf("expected TASKS section in Global Backlog")
	}
	if len(tasksSec.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in TASKS section, got %d: %+v", len(tasksSec.Tasks), tasksSec.Tasks)
	}
}

func TestBlockedByLevelIsolation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	sprint := sprints[0]

	feat1 := model.Task{
		UUID:          uuid.New().String(),
		ID:            "FEAT-1",
		WorkItemType:  model.WorkItemFeature,
		Title:         "Feature 1",
		SprintUUID:    sprint.UUID,
		SchedulingType: model.Floating,
	}
	def1 := model.Task{
		UUID:          uuid.New().String(),
		ID:            "DEF-1",
		WorkItemType:  model.WorkItemDefect,
		Title:         "Defect 1",
		SprintUUID:    sprint.UUID,
		SchedulingType: model.Floating,
	}
	task1 := model.Task{
		UUID:           uuid.New().String(),
		ID:             "TASK-1",
		Title:          "Task 1",
		SchedulingType: model.Floating,
	}
	database.AddTask(feat1)
	database.AddTask(def1)
	database.AddTask(task1)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID

	// 1. Feature Form (TaskTypeIdx = 0) -> Blockers must only be Features/Defects (FEAT-1, DEF-1)
	m.Form = viewmodel.NewTaskForm()
	m.Form.TaskTypeIdx = 0 // Feature
	m.PopulateFormAvailableFeaturesAndBlockers()

	for _, b := range m.Form.AvailableBlockers {
		if b.WorkItemType != model.WorkItemFeature && b.WorkItemType != model.WorkItemDefect && b.WorkItemType != model.WorkItemImprovement {
			t.Fatalf("expected feature blockers to only be feature-level, got: %+v", b)
		}
	}
	if len(m.Form.AvailableBlockers) != 2 {
		t.Fatalf("expected 2 feature-level blockers (FEAT-1, DEF-1), got %d", len(m.Form.AvailableBlockers))
	}

	// 2. Task Form (TaskTypeIdx = 3) -> Blockers must only be Tasks (TASK-1)
	m.Form = viewmodel.NewTaskForm()
	m.Form.TaskTypeIdx = 3 // Task
	m.PopulateFormAvailableFeaturesAndBlockers()

	for _, b := range m.Form.AvailableBlockers {
		if b.WorkItemType != "" && b.WorkItemType != model.WorkItemTask {
			t.Fatalf("expected task blockers to only be task-level, got: %+v", b)
		}
	}
	if len(m.Form.AvailableBlockers) != 1 || m.Form.AvailableBlockers[0].ID != "TASK-1" {
		t.Fatalf("expected 1 task blocker (TASK-1), got %+v", m.Form.AvailableBlockers)
	}
}

func TestCreateTaskFromFeatureShortcut(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	sprint := sprints[0]

	feat := model.Task{
		UUID:          uuid.New().String(),
		ID:            "FEAT-42",
		WorkItemType:  model.WorkItemFeature,
		Title:         "Stripe Billing Integration",
		Description:   "Support webhooks and card checkout",
		Priority:      model.P0,
		StoryPoints:   8,
		SprintUUID:    sprint.UUID,
		SchedulingType: model.Floating,
		LifecycleState: model.StateBacklog,
	}
	database.AddTask(feat)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView
	m.SprintSwimlaneIdx = 0
	m.SelectedTaskUUID = feat.UUID
	m.SidebarFocus = false
	m.TodoShelfFocus = false

	// 1. Press 'p' on focused feature
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})

	if !m.ConfirmOpen {
		t.Fatalf("expected ConfirmOpen to be true after pressing 'p'")
	}
	if m.ConfirmActionType != "create_task_from_feature" {
		t.Fatalf("expected ConfirmActionType 'create_task_from_feature', got '%s'", m.ConfirmActionType)
	}
	if m.ConfirmTask.UUID != feat.UUID {
		t.Fatalf("expected ConfirmTask UUID to match feature UUID")
	}

	// Verify modal rendering
	th := theme.NewTheme()
	renderedModal := modals.RenderConfirmModal(&m, th)
	if !strings.Contains(renderedModal, "CREATE TASK FOR FEATURE") {
		t.Fatalf("expected modal to contain 'CREATE TASK FOR FEATURE', got:\n%s", renderedModal)
	}
	if !strings.Contains(renderedModal, "FEAT-42") {
		t.Fatalf("expected modal to contain 'FEAT-42', got:\n%s", renderedModal)
	}

	// 2. Confirm creation with 'y' (or Enter)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	if m.ConfirmOpen {
		t.Fatalf("expected ConfirmOpen to be false after confirming")
	}

	// 3. Verify created task in database
	allTasks := database.GetTasks()
	var createdTask *model.Task
	for _, tsk := range allTasks {
		if tsk.LinkedFeatureID == "FEAT-42" {
			tCopy := tsk
			createdTask = &tCopy
			break
		}
	}

	if createdTask == nil {
		t.Fatalf("expected task linked to FEAT-42 to be created in DB")
	}
	if createdTask.Title != "Stripe Billing Integration" {
		t.Errorf("expected Title 'Stripe Billing Integration', got '%s'", createdTask.Title)
	}
	if createdTask.Priority != model.P0 {
		t.Errorf("expected Priority P0, got %s", createdTask.Priority)
	}
	if !createdTask.AddedToToday {
		t.Errorf("expected AddedToToday to be true")
	}
	if createdTask.SchedulingType != model.Floating {
		t.Errorf("expected Floating scheduling type, got %s", createdTask.SchedulingType)
	}
	if !strings.HasPrefix(createdTask.ID, "TSK-") {
		t.Errorf("expected ID prefix 'TSK-', got '%s'", createdTask.ID)
	}

	// 4. Verify it appears in TASKS section in Global Backlog
	shelfData := m.GetShelfData()
	var tasksSec *viewmodel.ShelfSection
	for _, sec := range shelfData.Sections {
		if sec.Type == viewmodel.SectionTasks {
			sCopy := sec
			tasksSec = &sCopy
			break
		}
	}
	if tasksSec == nil {
		t.Fatalf("expected SectionTasks to be present in Global Backlog")
	}
	foundInTasks := false
	for _, tsk := range tasksSec.Tasks {
		if tsk.UUID == createdTask.UUID {
			foundInTasks = true
			break
		}
	}
	if !foundInTasks {
		t.Fatalf("expected created task to appear in Global Backlog TASKS section")
	}
}

func TestSprintViewWorkItemCommands(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	sprint := sprints[0]

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView

	// 1. Run :feature command
	_, _ = m.RunCommand("feature User Profile Redesign")
	tasks := database.GetTasks()
	var createdFeat *model.Task
	for _, tk := range tasks {
		if tk.Title == "User Profile Redesign" {
			tCopy := tk
			createdFeat = &tCopy
			break
		}
	}
	if createdFeat == nil {
		t.Fatalf("expected feature 'User Profile Redesign' to be created")
	}
	if createdFeat.WorkItemType != model.WorkItemFeature || createdFeat.SprintUUID != sprint.UUID {
		t.Errorf("expected Feature in sprint, got type=%s sprint=%s", createdFeat.WorkItemType, createdFeat.SprintUUID)
	}

	// 2. Run :defect command
	_, _ = m.RunCommand("defect SQL Injection in search")
	tasks = database.GetTasks()
	var createdDefect *model.Task
	for _, tk := range tasks {
		if tk.Title == "SQL Injection in search" {
			tCopy := tk
			createdDefect = &tCopy
			break
		}
	}
	if createdDefect == nil {
		t.Fatalf("expected defect 'SQL Injection in search' to be created")
	}
	if createdDefect.WorkItemType != model.WorkItemDefect || createdDefect.SprintUUID != sprint.UUID {
		t.Errorf("expected Defect in sprint, got type=%s sprint=%s", createdDefect.WorkItemType, createdDefect.SprintUUID)
	}

	// 3. Run :improvement command
	_, _ = m.RunCommand("improvement Cache query results")
	tasks = database.GetTasks()
	var createdImp *model.Task
	for _, tk := range tasks {
		if tk.Title == "Cache query results" {
			tCopy := tk
			createdImp = &tCopy
			break
		}
	}
	if createdImp == nil {
		t.Fatalf("expected improvement 'Cache query results' to be created")
	}
	if createdImp.WorkItemType != model.WorkItemImprovement || createdImp.SprintUUID != sprint.UUID {
		t.Errorf("expected Improvement in sprint, got type=%s sprint=%s", createdImp.WorkItemType, createdImp.SprintUUID)
	}
}

func TestSprintViewDisallowHabitReminderEvent(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	sprint := sprints[0]

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView

	// 1. :habit command on Sprint View should be rejected
	_, _ = m.RunCommand("habit Meditate")
	if !strings.Contains(m.StatusMsg, "cannot be created in Sprint View") {
		t.Errorf("expected habit rejection message on Sprint View, got: %q", m.StatusMsg)
	}

	// 2. Form type cycling on Sprint View should only cycle 0..3 (Feature, Defect, Improvement, Task)
	m.CurrentMode = viewmodel.ModeForm
	m.Form = viewmodel.NewTaskForm()
	m.Form.ActiveField = 4 // Type field
	m.Form.TaskTypeIdx = 0 // Feature

	// Cycle right 4 times: 0 -> 1 -> 2 -> 3 -> 0 (should never hit 4=Reminder, 5=Habit, 6=Event)
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Form.TaskTypeIdx != 1 {
		t.Errorf("expected Defect (1), got %d", m.Form.TaskTypeIdx)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Form.TaskTypeIdx != 2 {
		t.Errorf("expected Improvement (2), got %d", m.Form.TaskTypeIdx)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Form.TaskTypeIdx != 3 {
		t.Errorf("expected Task (3), got %d", m.Form.TaskTypeIdx)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.Form.TaskTypeIdx != 0 {
		t.Errorf("expected cycle back to Feature (0), got %d", m.Form.TaskTypeIdx)
	}

	// Cycle left: 0 -> 3 (Task)
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.Form.TaskTypeIdx != 3 {
		t.Errorf("expected cycle left to Task (3), got %d", m.Form.TaskTypeIdx)
	}

	// 3. Submitting habit/reminder/event on Sprint View must be rejected
	m.Form.TaskTypeIdx = 5 // Habit (forced)
	m.Form.TitleInput.SetValue("Daily Run")
	m.SubmitForm()
	if !strings.Contains(m.StatusMsg, "cannot be created in Sprint View") {
		t.Errorf("expected form submit rejection for habit in Sprint View, got: %q", m.StatusMsg)
	}
}

func TestSprintCardBordersAndHorizontalNavigation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	if len(sprints) == 0 {
		t.Fatal("expected default sprint")
	}
	sprint := sprints[0]

	feat := model.Task{
		UUID:           uuid.New().String(),
		ID:             "FEAT-10",
		WorkItemType:   model.WorkItemFeature,
		Title:          "Implement user authentication flow with PKCE",
		Priority:       model.P1,
		StoryPoints:    8,
		SprintUUID:     sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
		Description:    "Ensure secure OAuth2.0 authentication",
		Tags:           []string{"security", "auth"},
	}
	database.AddTask(feat)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView
	m.Layout.TimelineW = 50 // Small width forces fewer visible columns and horizontal scrolling

	th := theme.NewTheme()
	rendered := pages.RenderSprintView(&m, th, 30)

	// Verify rounded card border characters appear intact and not wrapped
	if !strings.Contains(rendered, "╭") || !strings.Contains(rendered, "╰") {
		t.Errorf("expected rendered sprint view to contain intact rounded border cards, got:\n%s", rendered)
	}

	// Verify horizontal navigation H and L with smaller timeline width
	m.SprintSwimlaneIdx = 0
	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if m.SprintSwimlaneIdx != 1 {
		t.Fatalf("expected swimlane index 1 after 'L', got %d", m.SprintSwimlaneIdx)
	}

	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	if m.SprintSwimlaneIdx != 0 {
		t.Fatalf("expected swimlane index 0 after 'H', got %d", m.SprintSwimlaneIdx)
	}
}

func TestSprintCardRedesignAndBlockedSubcard(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	sprints := database.GetSprints()
	sprint := sprints[0]

	// Create blocked feature
	feat := model.Task{
		UUID:           uuid.New().String(),
		ID:             "FEA-1",
		WorkItemType:   model.WorkItemFeature,
		Title:          "Checkout Flow",
		Priority:       model.P0,
		StoryPoints:    5,
		SprintUUID:     sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
		BlockedBy:      "DEF-2",
		Tags:           []string{"billing"},
	}
	database.AddTask(feat)

	m := viewmodel.NewModel(database, nil)
	m.ActiveSprintUUID = sprint.UUID
	m.CurrentView = viewmodel.SprintView
	m.Layout.TimelineW = 160

	th := theme.NewTheme()
	rendered := pages.RenderSprintView(&m, th, 30)

	// 1. Should have [FEA-1] Checkout Flow directly
	if !strings.Contains(rendered, "[FEA-1] Checkout") {
		t.Errorf("expected rendered card to contain '[FEA-1] Checkout', got:\n%s", rendered)
	}

	// 2. Should NOT contain checkbox '☐'
	if strings.Contains(rendered, "☐") {
		t.Errorf("expected rendered sprint card to NOT contain checkbox '☐', got:\n%s", rendered)
	}

	// 3. Should contain priority and points
	if !strings.Contains(rendered, "P0 • 5 SP") {
		t.Errorf("expected rendered card to contain 'P0 • 5 SP', got:\n%s", rendered)
	}

	// 4. Should contain continuous attached blocked subcard
	if !strings.Contains(rendered, "Blocked by DEF-2") {
		t.Errorf("expected rendered card to contain continuous blocked subcard with 'Blocked by DEF-2', got:\n%s", rendered)
	}
}






