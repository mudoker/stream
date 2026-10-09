package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/view/components"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func TestGetTodoShelfTasksSorting(t *testing.T) {
	now := time.Now()
	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "backlog-1",
				Title:          "Backlog Low Priority",
				Priority:       model.P3,
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "reminder-2",
				Title:          "Reminder Late",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow: model.TimeWindow{
					Start: now.Add(2 * time.Hour),
				},
			},
			{
				UUID:           "backlog-2",
				Title:          "Backlog High Priority",
				Priority:       model.P0,
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "reminder-1",
				Title:          "Reminder Early",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow: model.TimeWindow{
					Start: now.Add(1 * time.Hour),
				},
			},
		},
	}

	shelfTasks := m.GetTodoShelfTasks()
	if len(shelfTasks) != 4 {
		t.Fatalf("expected 4 tasks on the shelf, got %d", len(shelfTasks))
	}

	if shelfTasks[0].UUID != "reminder-1" {
		t.Errorf("expected first shelf task to be reminder-1, got %s", shelfTasks[0].UUID)
	}
	if shelfTasks[1].UUID != "reminder-2" {
		t.Errorf("expected second shelf task to be reminder-2, got %s", shelfTasks[1].UUID)
	}

	if shelfTasks[2].UUID != "backlog-2" {
		t.Errorf("expected third shelf task to be backlog-2 (P0), got %s", shelfTasks[2].UUID)
	}
	if shelfTasks[3].UUID != "backlog-1" {
		t.Errorf("expected fourth shelf task to be backlog-1 (P3), got %s", shelfTasks[3].UUID)
	}
}

func TestTodoShelfMoveTaskSelection(t *testing.T) {
	now := time.Now()
	m := &viewmodel.Model{
		TodoShelfFocus:   true,
		SelectedTaskUUID: "reminder-1",
		Tasks: []model.Task{
			{
				UUID:           "reminder-1",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow: model.TimeWindow{Start: now},
			},
			{
				UUID:           "backlog-1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
		},
	}

	m.MoveTaskSelection(1)
	if m.SelectedTaskUUID != "backlog-1" {
		t.Errorf("expected selection to move down to backlog-1, got %s", m.SelectedTaskUUID)
	}

	m.MoveTaskSelection(1)
	if m.SelectedTaskUUID != "reminder-1" {
		t.Errorf("expected selection to wrap to reminder-1, got %s", m.SelectedTaskUUID)
	}

	m.MoveTaskSelection(-1)
	if m.SelectedTaskUUID != "backlog-1" {
		t.Errorf("expected selection to wrap up to backlog-1, got %s", m.SelectedTaskUUID)
	}
}

func TestRenderTodoShelfSections(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout: viewmodel.Layout{
			TodoW: 30,
		},
		Tasks: []model.Task{
			{
				UUID:           "rem",
				Title:          "Clean room",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: time.Now()},
			},
			{
				UUID:           "back",
				Title:          "Read book",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
		},
	}

	rendered := components.RenderTodoShelf(m, th, 20)
	if !strings.Contains(rendered, "REMINDERS") {
		t.Error("expected shelf rendering to contain 'REMINDERS'")
	}
	if !strings.Contains(rendered, "BACKLOG") {
		t.Error("expected shelf rendering to contain 'BACKLOG'")
	}
	if !strings.Contains(rendered, "Clean room") {
		t.Error("expected shelf rendering to contain reminder title 'Clean room'")
	}
	if !strings.Contains(rendered, "Read book") {
		t.Error("expected shelf rendering to contain backlog title 'Read book'")
	}
}

func TestRenderTodoShelfHabits(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout: viewmodel.Layout{
			TodoW: 30,
		},
		SelectedDay: time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
		Tasks: []model.Task{
			{
				UUID:           "habit-1",
				Title:          "Drink water",
				SchedulingType: model.Habit,
				LifecycleState: model.StateCompleted,
				CompletedDates: []string{"2026-06-06"},
				UpdatedAt:      time.Date(2026, 6, 6, 10, 0, 0, 0, time.Local), // Completed today
			},
			{
				UUID:           "habit-2",
				Title:          "Stretch",
				SchedulingType: model.Habit,
				LifecycleState: model.StateCompleted,
				CompletedDates: []string{"2026-06-05"},
				UpdatedAt:      time.Date(2026, 6, 5, 10, 0, 0, 0, time.Local), // Completed yesterday
			},
		},
	}

	rendered := components.RenderTodoShelf(m, th, 40)
	if !strings.Contains(rendered, "HABITS") {
		t.Error("expected shelf rendering to contain 'HABITS'")
	}
	if !strings.Contains(rendered, "☑ Drink water") {
		t.Error("expected habit completed today to render as ☑ Drink water")
	}
	if !strings.Contains(rendered, "☐ Stretch") {
		t.Error("expected habit completed yesterday to render as ☐ Stretch")
	}
}

func TestReminderRemainingDays(t *testing.T) {
	th := theme.NewTheme()
	now := time.Now()
	
	m := &viewmodel.Model{
		Layout: viewmodel.Layout{
			TodoW: 45,
		},
		Tasks: []model.Task{
			{
				UUID:           "rem-today",
				Title:          "Call doctor",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: now},
			},
			{
				UUID:           "rem-future",
				Title:          "Submit tax",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: now.Add(48 * time.Hour)},
			},
			{
				UUID:           "rem-past",
				Title:          "Pay bills",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: now.Add(-24 * time.Hour)},
			},
		},
	}

	rendered := components.RenderTodoShelf(m, th, 30)
	if !strings.Contains(rendered, "due today") {
		t.Error("expected rendering to contain 'due today'")
	}
	if !strings.Contains(rendered, "2 days remaining") {
		t.Error("expected rendering to contain '2 days remaining'")
	}
	if !strings.Contains(rendered, "overdue by 1 day") {
		t.Error("expected rendering to contain 'overdue by 1 day'")
	}
}

func TestTodoShelfCompletedSectionRendering(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout: viewmodel.Layout{
			TodoW: 30,
		},
		Tasks: []model.Task{
			{
				UUID:           "backlog-active",
				Title:          "Active backlog",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "backlog-completed",
				Title:          "Done backlog",
				SchedulingType: model.Floating,
				LifecycleState: model.StateCompleted,
			},
		},
	}

	shelfTasks := m.GetTodoShelfTasks()
	if len(shelfTasks) != 2 {
		t.Fatalf("expected 2 shelf tasks, got %d", len(shelfTasks))
	}
	// Verify that the completed task is placed at the end (very bottom)
	if shelfTasks[0].UUID != "backlog-active" {
		t.Errorf("expected active task to be first, got %s", shelfTasks[0].UUID)
	}
	if shelfTasks[1].UUID != "backlog-completed" {
		t.Errorf("expected completed task to be last, got %s", shelfTasks[1].UUID)
	}

	rendered := components.RenderTodoShelf(m, th, 40)
	if !strings.Contains(rendered, "COMPLETED") {
		t.Error("expected rendering to contain 'COMPLETED' section")
	}
	if !strings.Contains(rendered, "☑ Done backlog") {
		t.Error("expected rendering to contain '☑ Done backlog'")
	}
}

func TestTodoShelfFocusRecall(t *testing.T) {
	m := &viewmodel.Model{
		CurrentView:    viewmodel.DayView,
		TodoShelfFocus: true,
		Tasks: []model.Task{
			{
				UUID:           "task-1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "task-2",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
		},
		SelectedTaskUUID: "task-2",
	}

	// 1. Defocus the todo shelf (TodoShelfFocus -> SidebarFocus)
	m.CycleFocus()
	if m.TodoShelfFocus {
		t.Fatal("expected todo shelf to be defocused")
	}
	if m.LastTodoShelfTaskUUID != "task-2" {
		t.Fatalf("expected last focused task UUID to be task-2, got %s", m.LastTodoShelfTaskUUID)
	}

	// 2. Cycle again (SidebarFocus -> TimelineFocus)
	m.CycleFocus()

	// 3. Cycle again (TimelineFocus -> TodoShelfFocus)
	m.CycleFocus()
	if !m.TodoShelfFocus {
		t.Fatal("expected todo shelf to be refocused")
	}
	if m.SelectedTaskUUID != "task-2" {
		t.Fatalf("expected selection to be restored to task-2, got %s", m.SelectedTaskUUID)
	}

	// 4. Defocus again
	m.CycleFocus()

	// 5. Remove task-2 from Tasks list (simulate deletion/completion shift)
	m.Tasks = []model.Task{m.Tasks[0]} // only task-1 remains

	// 6. Refocus the shelf again (SidebarFocus -> Timeline -> TodoShelfFocus)
	m.CycleFocus()
	m.CycleFocus()
	if !m.TodoShelfFocus {
		t.Fatal("expected shelf to be refocused")
	}
	if m.SelectedTaskUUID != "task-1" {
		t.Fatalf("expected selection to fall back to task-1, got %s", m.SelectedTaskUUID)
	}
}

func TestTodoShelfDeleteSelectionAdjustment(t *testing.T) {
	m := &viewmodel.Model{
		CurrentView:           viewmodel.DayView,
		TodoShelfFocus:        true,
		SelectedTaskUUID:      "task-2",
		LastTodoShelfTaskUUID: "task-2",
		Tasks: []model.Task{
			{
				UUID:           "task-1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "task-2",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "task-3",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
		},
	}

	// Test 1: Deleting middle task (task-2) shifts selection to task-3 (next sibling)
	m.AdjustSelectionBeforeDeletion("task-2")
	if m.SelectedTaskUUID != "task-3" {
		t.Fatalf("expected SelectedTaskUUID to be 'task-3', got '%s'", m.SelectedTaskUUID)
	}
	if m.LastTodoShelfTaskUUID != "task-3" {
		t.Fatalf("expected LastTodoShelfTaskUUID to be 'task-3', got '%s'", m.LastTodoShelfTaskUUID)
	}

	// Test 2: Deleting last task (task-3) shifts selection to task-2 (previous sibling)
	m.Tasks = []model.Task{
		{
			UUID:           "task-1",
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		},
		{
			UUID:           "task-3",
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		},
	}
	m.SelectedTaskUUID = "task-3"
	m.LastTodoShelfTaskUUID = "task-3"
	m.AdjustSelectionBeforeDeletion("task-3")
	if m.SelectedTaskUUID != "task-1" {
		t.Fatalf("expected SelectedTaskUUID to be 'task-1', got '%s'", m.SelectedTaskUUID)
	}
	if m.LastTodoShelfTaskUUID != "task-1" {
		t.Fatalf("expected LastTodoShelfTaskUUID to be 'task-1', got '%s'", m.LastTodoShelfTaskUUID)
	}

	// Test 3: Deleting only task (task-1) clears selection
	m.Tasks = []model.Task{
		{
			UUID:           "task-1",
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		},
	}
	m.SelectedTaskUUID = "task-1"
	m.LastTodoShelfTaskUUID = "task-1"
	m.AdjustSelectionBeforeDeletion("task-1")
	if m.SelectedTaskUUID != "" {
		t.Fatalf("expected SelectedTaskUUID to be empty, got '%s'", m.SelectedTaskUUID)
	}
	if m.LastTodoShelfTaskUUID != "" {
		t.Fatalf("expected LastTodoShelfTaskUUID to be empty, got '%s'", m.LastTodoShelfTaskUUID)
	}
}

func TestTimelineDeleteSelectionAdjustment(t *testing.T) {
	now := time.Now()
	m := &viewmodel.Model{
		CurrentView:      viewmodel.DayView,
		TodoShelfFocus:   false,
		SelectedTaskUUID: "task-2",
		SelectedDay:      now,
		Tasks: []model.Task{
			{
				UUID:           "task-1",
				SchedulingType: model.Anchored,
				TimeWindow: model.TimeWindow{
					Start: now.Add(-time.Hour),
					End:   now,
				},
			},
			{
				UUID:           "task-2",
				SchedulingType: model.Anchored,
				TimeWindow: model.TimeWindow{
					Start: now,
					End:   now.Add(time.Hour),
				},
			},
			{
				UUID:           "task-3",
				SchedulingType: model.Anchored,
				TimeWindow: model.TimeWindow{
					Start: now.Add(time.Hour),
					End:   now.Add(2 * time.Hour),
				},
			},
		},
	}

	// Test 1: Deleting middle task (task-2) shifts selection to task-3 (next sibling)
	m.AdjustSelectionBeforeDeletion("task-2")
	if m.SelectedTaskUUID != "task-3" {
		t.Fatalf("expected SelectedTaskUUID to be 'task-3', got '%s'", m.SelectedTaskUUID)
	}
	if m.TimelineHour != now.Add(time.Hour).Hour() {
		t.Fatalf("expected TimelineHour to be %d, got %d", now.Add(time.Hour).Hour(), m.TimelineHour)
	}
}

func TestRecurringAndHabitShelfBehavior(t *testing.T) {
	day1 := time.Date(2026, 6, 14, 0, 0, 0, 0, time.Local)
	day2 := day1.AddDate(0, 0, 1)

	// 1. Habit repeatable/anchored test
	m := &viewmodel.Model{
		SelectedDay: day1,
		Tasks: []model.Task{
			{
				UUID:           "habit-1",
				Title:          "Drink Water (Anchored)",
				SchedulingType: model.Habit,
				TimeWindow: model.TimeWindow{
					Start: day1.Add(9 * time.Hour),
					End:   day1.Add(10 * time.Hour),
				},
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "habit-2",
				Title:          "Stretch (De-anchored)",
				SchedulingType: model.Habit,
				TimeWindow:     model.TimeWindow{},
				LifecycleState: model.StateReady,
			},
		},
	}

	// On Day 1: habit-1 is anchored, so it shouldn't be on the shelf. habit-2 is de-anchored, so it should.
	shelf1 := m.GetTodoShelfTasks()
	foundHabit1OnDay1 := false
	foundHabit2OnDay1 := false
	for _, task := range shelf1 {
		if task.UUID == "habit-1" {
			foundHabit1OnDay1 = true
		}
		if task.UUID == "habit-2" {
			foundHabit2OnDay1 = true
		}
	}
	if foundHabit1OnDay1 {
		t.Errorf("expected anchored habit-1 to NOT appear on Day 1 shelf")
	}
	if !foundHabit2OnDay1 {
		t.Errorf("expected de-anchored habit-2 to appear on Day 1 shelf")
	}

	// Move to Day 2: habit-1 is anchored, so it should NOT appear on the shelf either. habit-2 is de-anchored, so it should.
	m.SelectedDay = day2
	shelf2 := m.GetTodoShelfTasks()
	foundHabit1OnDay2 := false
	foundHabit2OnDay2 := false
	for _, task := range shelf2 {
		if task.UUID == "habit-1" {
			foundHabit1OnDay2 = true
		}
		if task.UUID == "habit-2" {
			foundHabit2OnDay2 = true
		}
	}
	if foundHabit1OnDay2 {
		t.Errorf("expected anchored habit-1 to NOT appear on Day 2 shelf")
	}
	if !foundHabit2OnDay2 {
		t.Errorf("expected de-anchored habit-2 to appear on Day 2 shelf")
	}

	// 2. Grouping de-anchored recurring tasks test
	m.Tasks = []model.Task{
		{
			UUID:                "rec-1",
			Title:               "Gym",
			SchedulingType:      model.Floating,
			RecurringParentUUID: "gym-parent",
			LifecycleState:      model.StateReady,
		},
		{
			UUID:                "rec-2",
			Title:               "Gym",
			SchedulingType:      model.Floating,
			RecurringParentUUID: "gym-parent",
			LifecycleState:      model.StateReady,
		},
		{
			UUID:                "rec-3",
			Title:               "Gym",
			SchedulingType:      model.Floating,
			RecurringParentUUID: "gym-parent",
			LifecycleState:      model.StateReady,
		},
	}

	shelfTasks := m.GetTodoShelfTasks()
	// Should be grouped into a single item
	if len(shelfTasks) != 1 {
		t.Fatalf("expected 1 task on the shelf (grouped), got %d", len(shelfTasks))
	}
	if shelfTasks[0].Title != "Gym (3)" {
		t.Errorf("expected grouped title to be 'Gym (3)', got '%s'", shelfTasks[0].Title)
	}

	// 3. Recurring habit test: anchored on Day 2, should NOT appear on shelf on Day 1
	m.SelectedDay = day1
	m.Tasks = []model.Task{
		{
			UUID:                "rec-habit-1",
			Title:               "Gym Habit",
			SchedulingType:      model.Habit,
			RecurringParentUUID: "gym-habit-parent",
			TimeWindow: model.TimeWindow{
				Start: day2.Add(9 * time.Hour),
				End:   day2.Add(10 * time.Hour),
			},
			LifecycleState: model.StateReady,
		},
	}
	shelfDay1 := m.GetTodoShelfTasks()
	for _, task := range shelfDay1 {
		if task.UUID == "rec-habit-1" {
			t.Errorf("expected recurring habit anchored on Day 2 to NOT appear on Day 1 shelf")
		}
	}
}

func TestGlobalBacklogMoveTaskSelection(t *testing.T) {
	m := &viewmodel.Model{
		CurrentView:      viewmodel.SprintView,
		TodoShelfFocus:   true,
		SelectedTaskUUID: "gb-1",
		Tasks: []model.Task{
			{
				UUID:           "gb-1",
				Title:          "Backlog Task 1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "gb-2",
				Title:          "Backlog Task 2",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "gb-3",
				Title:          "Backlog Task 3",
				SchedulingType: model.Floating,
				LifecycleState: model.StateCompleted,
			},
		},
	}

	// 1. Move down from gb-1 -> gb-2
	m.MoveTaskSelection(1)
	if m.SelectedTaskUUID != "gb-2" {
		t.Errorf("expected selection to move to gb-2, got %s", m.SelectedTaskUUID)
	}

	// 2. Move down from gb-2 -> gb-3 (completed)
	m.MoveTaskSelection(1)
	if m.SelectedTaskUUID != "gb-3" {
		t.Errorf("expected selection to move to gb-3, got %s", m.SelectedTaskUUID)
	}

	// 3. Move down from gb-3 -> wrap to gb-1
	m.MoveTaskSelection(1)
	if m.SelectedTaskUUID != "gb-1" {
		t.Errorf("expected selection to wrap to gb-1, got %s", m.SelectedTaskUUID)
	}

	// 4. Move up from gb-1 -> wrap to gb-3
	m.MoveTaskSelection(-1)
	if m.SelectedTaskUUID != "gb-3" {
		t.Errorf("expected selection to wrap backwards to gb-3, got %s", m.SelectedTaskUUID)
	}
}

func TestTodoShelfMoveShelfSectionJK(t *testing.T) {
	now := time.Now()
	m := &viewmodel.Model{
		CurrentView:      viewmodel.DayView,
		TodoShelfFocus:   true,
		SelectedDay:      now,
		SelectedTaskUUID: "rem-1",
		Tasks: []model.Task{
			{
				UUID:           "rem-1",
				Title:          "Reminder 1",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: now},
			},
			{
				UUID:           "rem-2",
				Title:          "Reminder 2",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: now.Add(time.Hour)},
			},
			// Habits empty!
			{
				UUID:           "back-1",
				Title:          "Backlog 1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "back-2",
				Title:          "Backlog 2",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "comp-1",
				Title:          "Completed 1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateCompleted,
			},
		},
	}

	// 1. Initial selection is in Reminders (rem-1).
	// Pressing J (MoveShelfSection(1)) should skip empty Habits and jump to Backlog (back-1)!
	m.MoveShelfSection(1)
	if m.SelectedTaskUUID != "back-1" {
		t.Errorf("expected J to jump to back-1 (Backlog), got %s", m.SelectedTaskUUID)
	}

	// 2. Pressing J again should jump to Completed (comp-1)
	m.MoveShelfSection(1)
	if m.SelectedTaskUUID != "comp-1" {
		t.Errorf("expected J to jump to comp-1 (Completed), got %s", m.SelectedTaskUUID)
	}

	// 3. Pressing J again should wrap to Reminders (rem-1)
	m.MoveShelfSection(1)
	if m.SelectedTaskUUID != "rem-1" {
		t.Errorf("expected J to wrap back to rem-1 (Reminders), got %s", m.SelectedTaskUUID)
	}

	// 4. Pressing K (MoveShelfSection(-1)) should jump backwards to Completed (comp-1)
	m.MoveShelfSection(-1)
	if m.SelectedTaskUUID != "comp-1" {
		t.Errorf("expected K to jump backwards to comp-1 (Completed), got %s", m.SelectedTaskUUID)
	}

	// 5. Pressing K again should jump backwards to Backlog (back-1)
	m.MoveShelfSection(-1)
	if m.SelectedTaskUUID != "back-1" {
		t.Errorf("expected K to jump backwards to back-1 (Backlog), got %s", m.SelectedTaskUUID)
	}
}

func TestGlobalBacklogMoveShelfSectionJK(t *testing.T) {
	m := &viewmodel.Model{
		CurrentView:      viewmodel.SprintView,
		TodoShelfFocus:   true,
		SelectedTaskUUID: "gb-back-1",
		Tasks: []model.Task{
			{
				UUID:           "gb-back-1",
				Title:          "Global Backlog 1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
			{
				UUID:           "gb-comp-1",
				Title:          "Global Completed 1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateCompleted,
			},
		},
	}

	// J from Backlog -> Completed
	m.MoveShelfSection(1)
	if m.SelectedTaskUUID != "gb-comp-1" {
		t.Errorf("expected J to jump to gb-comp-1, got %s", m.SelectedTaskUUID)
	}

	// J from Completed -> Backlog
	m.MoveShelfSection(1)
	if m.SelectedTaskUUID != "gb-back-1" {
		t.Errorf("expected J to wrap back to gb-back-1, got %s", m.SelectedTaskUUID)
	}

	// K from Backlog -> Completed
	m.MoveShelfSection(-1)
	if m.SelectedTaskUUID != "gb-comp-1" {
		t.Errorf("expected K to jump backwards to gb-comp-1, got %s", m.SelectedTaskUUID)
	}
}

func TestClearShelfSectionWithConfirmation(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	task1 := model.Task{
		UUID:           "task-back-1",
		Title:          "Backlog Task 1",
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	task2 := model.Task{
		UUID:           "task-back-2",
		Title:          "Backlog Task 2",
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	taskComp := model.Task{
		UUID:           "task-done-1",
		Title:          "Done Task 1",
		SchedulingType: model.Floating,
		LifecycleState: model.StateCompleted,
	}

	database.AddTask(task1)
	database.AddTask(task2)
	database.AddTask(taskComp)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.DayView
	m.TodoShelfFocus = true
	m.SidebarFocus = false
	m.SelectedTaskUUID = "task-back-1" // Focused on BACKLOG section

	// 1. Press Shift+D while focused on BACKLOG section
	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if !m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be true when pressing Shift+D on shelf section")
	}
	if m.ConfirmActionType != "clear_shelf_section" {
		t.Fatalf("expected ConfirmActionType to be 'clear_shelf_section', got '%s'", m.ConfirmActionType)
	}
	if m.ConfirmShelfSection.Type != viewmodel.SectionBacklog {
		t.Fatalf("expected ConfirmShelfSection.Type to be BACKLOG, got %s", m.ConfirmShelfSection.Type)
	}
	if len(m.ConfirmShelfSection.Tasks) != 2 {
		t.Fatalf("expected 2 tasks to be cleared in BACKLOG, got %d", len(m.ConfirmShelfSection.Tasks))
	}

	// 2. Canceling with 'n' should preserve all tasks
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be false after cancel")
	}
	if len(database.GetTasks()) != 3 {
		t.Fatalf("expected 3 tasks to remain in database, got %d", len(database.GetTasks()))
	}

	// 3. Confirming with 'y' should delete the tasks in the BACKLOG section but keep COMPLETED task
	m.HandleNormalKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if !m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be true on second Shift+D")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.ConfirmOpen {
		t.Fatal("expected ConfirmOpen to be false after confirm")
	}

	tasksAfter := database.GetTasks()
	if len(tasksAfter) != 1 {
		t.Fatalf("expected only 1 task remaining after clearing BACKLOG, got %d", len(tasksAfter))
	}
	if tasksAfter[0].UUID != "task-done-1" {
		t.Fatalf("expected remaining task to be task-done-1, got %s", tasksAfter[0].UUID)
	}
}

func TestUndoneTasksCarryOverToNextDay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	now := time.Now()
	day1 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day2 := day1.AddDate(0, 0, 1)
	day3 := day1.AddDate(0, 0, 2)

	task1 := model.Task{
		UUID:           "task-day1-undone",
		Title:          "Undone Day 1 Task",
		InitiateDate:   day1,
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	task2 := model.Task{
		UUID:           "task-day3-future",
		Title:          "Future Day 3 Task",
		InitiateDate:   day3,
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}
	database.AddTask(task1)
	database.AddTask(task2)

	m := viewmodel.NewModel(database, nil)

	// Day 1: task1 should appear on shelf, task2 should not
	m.SelectedDay = day1
	shelfDay1 := m.GetTodoShelfTasks()
	hasTask1 := false
	hasTask2 := false
	for _, tk := range shelfDay1 {
		if tk.UUID == "task-day1-undone" {
			hasTask1 = true
		}
		if tk.UUID == "task-day3-future" {
			hasTask2 = true
		}
	}
	if !hasTask1 || hasTask2 {
		t.Fatalf("Day 1: expected task1=true, task2=false, got task1=%v, task2=%v", hasTask1, hasTask2)
	}

	// Day 2: task1 was not done on Day 1, so it MUST carry onto Day 2
	m.SelectedDay = day2
	shelfDay2 := m.GetTodoShelfTasks()
	hasTask1 = false
	hasTask2 = false
	for _, tk := range shelfDay2 {
		if tk.UUID == "task-day1-undone" {
			hasTask1 = true
		}
		if tk.UUID == "task-day3-future" {
			hasTask2 = true
		}
	}
	if !hasTask1 {
		t.Fatalf("Day 2: expected undone task1 to carry over onto Day 2 shelf")
	}
	if hasTask2 {
		t.Fatalf("Day 2: did not expect future task2 on Day 2 shelf")
	}

	// Day 3: both task1 (carried over) and task2 (initiated on Day 3) should appear
	m.SelectedDay = day3
	shelfDay3 := m.GetTodoShelfTasks()
	hasTask1 = false
	hasTask2 = false
	for _, tk := range shelfDay3 {
		if tk.UUID == "task-day1-undone" {
			hasTask1 = true
		}
		if tk.UUID == "task-day3-future" {
			hasTask2 = true
		}
	}
	if !hasTask1 || !hasTask2 {
		t.Fatalf("Day 3: expected both task1 (carried over) and task2 on Day 3 shelf, got task1=%v, task2=%v", hasTask1, hasTask2)
	}

	// Now mark task1 completed on Day 3
	task1.LifecycleState = model.StateCompleted
	task1.UpdatedAt = day3
	database.UpdateTask(task1)
	m.RefreshTasks()

	// On Day 3: task1 appears in COMPLETED section
	shelfDay3Completed := m.GetTodoShelfTasks()
	hasTask1Completed := false
	for _, tk := range shelfDay3Completed {
		if tk.UUID == "task-day1-undone" && tk.LifecycleState == model.StateCompleted {
			hasTask1Completed = true
		}
	}
	if !hasTask1Completed {
		t.Fatalf("Day 3: expected completed task1 in Completed section")
	}

	// On Day 4: completed task1 should NOT carry over
	day4 := day1.AddDate(0, 0, 3)
	m.SelectedDay = day4
	shelfDay4 := m.GetTodoShelfTasks()
	hasTask1OnDay4 := false
	for _, tk := range shelfDay4 {
		if tk.UUID == "task-day1-undone" {
			hasTask1OnDay4 = true
		}
	}
	if hasTask1OnDay4 {
		t.Fatalf("Day 4: completed task1 should not carry over onto Day 4 shelf")
	}
}

func TestSprintFeaturesWithStatusExcludedFromDayTodoShelfBacklog(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	sprint := model.Sprint{
		UUID:      uuid.New().String(),
		Name:      "Sprint 1",
		StartDate: today,
		EndDate:   today.AddDate(0, 0, 13),
	}
	database.AddSprint(sprint)

	// 1. Feature in sprint with status READY (Defined) - should NOT show in Day Backlog
	feat1 := model.Task{
		UUID:           "feat-1",
		ID:             "FEAT-1",
		WorkItemType:   model.WorkItemFeature,
		Title:          "Commercialise Kratos",
		SprintUUID:     sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
		InitiateDate:   today,
	}

	// 2. Feature in sprint with AddedToToday = true - SHOULD show on Today Shelf
	feat2 := model.Task{
		UUID:           "feat-2",
		ID:             "FEAT-2",
		WorkItemType:   model.WorkItemFeature,
		Title:          "Added To Today Feature",
		SprintUUID:     sprint.UUID,
		LifecycleState: model.StateReady,
		SchedulingType: model.Floating,
		AddedToToday:   true,
		InitiateDate:   today,
	}

	// 3. Regular Task without sprint / status - SHOULD show in Day Backlog
	regularTask := model.Task{
		UUID:           "task-regular",
		Title:          "Regular Task",
		SchedulingType: model.Floating,
		InitiateDate:   today,
	}

	database.AddTask(feat1)
	database.AddTask(feat2)
	database.AddTask(regularTask)

	m := viewmodel.NewModel(database, nil)
	m.CurrentView = viewmodel.DayView
	m.SelectedDay = today

	shelfData := m.GetShelfData()

	var backlogSection viewmodel.ShelfSection
	for _, sec := range shelfData.Sections {
		if sec.Type == viewmodel.SectionBacklog {
			backlogSection = sec
			break
		}
	}

	hasFeat1 := false
	hasFeat2 := false
	hasRegularTask := false

	for _, tk := range backlogSection.Tasks {
		if tk.UUID == "feat-1" {
			hasFeat1 = true
		}
		if tk.UUID == "feat-2" {
			hasFeat2 = true
		}
		if tk.UUID == "task-regular" {
			hasRegularTask = true
		}
	}

	if hasFeat1 {
		t.Fatalf("FEAT-1 with status READY in sprint should NOT be in Day View Backlog")
	}
	if !hasFeat2 {
		t.Fatalf("FEAT-2 with AddedToToday=true should be in Day View shelf")
	}
	if !hasRegularTask {
		t.Fatalf("Regular task without sprint status should be in Day View Backlog")
	}
}




