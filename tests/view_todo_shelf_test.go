package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/components"
	"stream/internal/view/theme"
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
