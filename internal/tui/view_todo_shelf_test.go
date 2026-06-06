package tui

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
)

func TestGetTodoShelfTasksSorting(t *testing.T) {
	now := time.Now()
	m := &Model{
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

	shelfTasks := m.getTodoShelfTasks()
	if len(shelfTasks) != 4 {
		t.Fatalf("expected 4 tasks on the shelf, got %d", len(shelfTasks))
	}

	// First two should be reminders sorted by time: reminder-1 (Early), reminder-2 (Late)
	if shelfTasks[0].UUID != "reminder-1" {
		t.Errorf("expected first shelf task to be reminder-1, got %s", shelfTasks[0].UUID)
	}
	if shelfTasks[1].UUID != "reminder-2" {
		t.Errorf("expected second shelf task to be reminder-2, got %s", shelfTasks[1].UUID)
	}

	// Last two should be backlog (Floating) tasks sorted by priority (P0 before P3)
	if shelfTasks[2].UUID != "backlog-2" {
		t.Errorf("expected third shelf task to be backlog-2 (P0), got %s", shelfTasks[2].UUID)
	}
	if shelfTasks[3].UUID != "backlog-1" {
		t.Errorf("expected fourth shelf task to be backlog-1 (P3), got %s", shelfTasks[3].UUID)
	}
}

func TestTodoShelfMoveTaskSelection(t *testing.T) {
	now := time.Now()
	m := &Model{
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

	// Move selection down
	m.moveTaskSelection(1)
	if m.SelectedTaskUUID != "backlog-1" {
		t.Errorf("expected selection to move down to backlog-1, got %s", m.SelectedTaskUUID)
	}

	// Move selection down again (wrapping to top)
	m.moveTaskSelection(1)
	if m.SelectedTaskUUID != "reminder-1" {
		t.Errorf("expected selection to wrap to reminder-1, got %s", m.SelectedTaskUUID)
	}

	// Move selection up (wrapping to bottom)
	m.moveTaskSelection(-1)
	if m.SelectedTaskUUID != "backlog-1" {
		t.Errorf("expected selection to wrap up to backlog-1, got %s", m.SelectedTaskUUID)
	}
}

func TestRenderTodoShelfSections(t *testing.T) {
	m := &Model{
		Layout: Layout{
			TodoW: 30,
		},
		Theme: NewTheme(),
		Tasks: []model.Task{
			{
				UUID:           "rem",
				Title:          "Clean Room",
				SchedulingType: model.Reminder,
				LifecycleState: model.StateReady,
				TimeWindow:     model.TimeWindow{Start: time.Now()},
			},
			{
				UUID:           "back",
				Title:          "Read Book",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
			},
		},
	}

	rendered := m.renderTodoShelf(20)
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
