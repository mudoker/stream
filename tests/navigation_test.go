package tests

import (
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWeekNavigation(t *testing.T) {
	// Initialize tasks on different weekdays
	monday := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) // June 15, 2026 is Monday
	t1 := model.Task{
		UUID:           "task-monday",
		Title:          "Monday Task",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: monday,
			End:   monday.Add(1 * time.Hour),
		},
	}
	t2 := model.Task{
		UUID:           "task-tuesday",
		Title:          "Tuesday Task",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: monday.AddDate(0, 0, 1),
			End:   monday.AddDate(0, 0, 1).Add(1 * time.Hour),
		},
	}

	m := &viewmodel.Model{
		CurrentView:      viewmodel.WeekView,
		CurrentMode:      viewmodel.ModeNormal,
		Tasks:            []model.Task{t1, t2},
		SelectedDay:      monday,
		SelectedTaskUUID: "task-monday",
	}

	// 1. Move right (l) -> changes selected day to Tuesday and selects task-tuesday
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !m.SelectedDay.Equal(monday.AddDate(0, 0, 1)) {
		t.Errorf("Expected SelectedDay to be Tuesday, got %s", m.SelectedDay)
	}
	if m.SelectedTaskUUID != "task-tuesday" {
		t.Errorf("Expected SelectedTaskUUID to be task-tuesday, got %s", m.SelectedTaskUUID)
	}

	// 2. Move down (j) on Tuesday -> stays on Tuesday and still selects task-tuesday (since it cycles within Tuesday)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if !m.SelectedDay.Equal(monday.AddDate(0, 0, 1)) {
		t.Errorf("Expected SelectedDay to remain Tuesday, got %s", m.SelectedDay)
	}
	if m.SelectedTaskUUID != "task-tuesday" {
		t.Errorf("Expected SelectedTaskUUID to remain task-tuesday, got %s", m.SelectedTaskUUID)
	}

	// 3. Move left (h) -> shifts back to Monday and selects task-monday
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if !m.SelectedDay.Equal(monday) {
		t.Errorf("Expected SelectedDay to be Monday, got %s", m.SelectedDay)
	}
	if m.SelectedTaskUUID != "task-monday" {
		t.Errorf("Expected SelectedTaskUUID to be task-monday, got %s", m.SelectedTaskUUID)
	}

	// 4. Shift week time frame backward (H)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	if !m.SelectedDay.Equal(monday.AddDate(0, 0, -7)) {
		t.Errorf("Expected SelectedDay to shift back 7 days, got %s", m.SelectedDay)
	}

	// 5. Shift week time frame forward using K (or L)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	if !m.SelectedDay.Equal(monday) {
		t.Errorf("Expected SelectedDay to return to Monday, got %s", m.SelectedDay)
	}
}

func TestDayNavigation(t *testing.T) {
	today := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	t1 := model.Task{
		UUID:           "task-today",
		Title:          "Today Task",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: today,
			End:   today.Add(1 * time.Hour),
		},
	}

	m := &viewmodel.Model{
		CurrentView:      viewmodel.DayView,
		CurrentMode:      viewmodel.ModeNormal,
		Tasks:            []model.Task{t1},
		SelectedDay:      today,
		SelectedTaskUUID: "task-today",
		TimelineHour:     12,
	}

	// 1. Shift Day Right (L)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if !m.SelectedDay.Equal(today.AddDate(0, 0, 1)) {
		t.Errorf("Expected SelectedDay to shift to tomorrow, got %v", m.SelectedDay)
	}

	// Record the timeline hour after navigation (might fall back to current local hour)
	currentHour := m.TimelineHour

	// 2. Scroll Timeline Hour Down (J)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	expectedHourJ := (currentHour + 1) % 24
	if m.TimelineHour != expectedHourJ {
		t.Errorf("Expected TimelineHour to increment to %d, got %d", expectedHourJ, m.TimelineHour)
	}

	// 3. Scroll Timeline Hour Up (K)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	if m.TimelineHour != currentHour {
		t.Errorf("Expected TimelineHour to return to %d, got %d", currentHour, m.TimelineHour)
	}
}

func TestMonthNavigation(t *testing.T) {
	today := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	m := &viewmodel.Model{
		CurrentView: viewmodel.MonthView,
		CurrentMode: viewmodel.ModeNormal,
		SelectedDay: today,
	}

	// 1. Move down (j) -> shifts day by 7 days
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if !m.SelectedDay.Equal(today.AddDate(0, 0, 7)) {
		t.Errorf("Expected selected day to increase by 7 days, got %v", m.SelectedDay)
	}

	// 2. Move up (k) -> returns to today
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if !m.SelectedDay.Equal(today) {
		t.Errorf("Expected selected day to return to original, got %v", m.SelectedDay)
	}

	// 3. Press Enter -> transitions to DayView
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.CurrentView != viewmodel.DayView {
		t.Errorf("Expected current view to be DayView, got %v", m.CurrentView)
	}
}

func TestDashboardNavigation(t *testing.T) {
	m := &viewmodel.Model{
		CurrentView:       viewmodel.DashboardView,
		CurrentMode:       viewmodel.ModeNormal,
		DashboardFocusCol: 0,
		DashboardFocusRow: 0,
		Height:            40,
	}

	// 1. Move right (l) -> focuses col 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.DashboardFocusCol != 1 {
		t.Errorf("Expected column 1 to be focused, got %d", m.DashboardFocusCol)
	}

	// 2. Move down (j) -> focuses row 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.DashboardFocusRow != 1 {
		t.Errorf("Expected row 1 to be focused, got %d", m.DashboardFocusRow)
	}

	// 3. Move up (k) -> focuses row 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.DashboardFocusRow != 0 {
		t.Errorf("Expected row 0 to be focused, got %d", m.DashboardFocusRow)
	}
}

func TestWeekScrollingAndTaskAutoScroll(t *testing.T) {
	monday := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	t1 := model.Task{
		UUID:           "task-monday-1",
		Title:          "Monday Task 1",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: monday,
			End:   monday.Add(1 * time.Hour), // height = 6 lines
		},
	}
	t2 := model.Task{
		UUID:           "task-monday-2",
		Title:          "Monday Task 2",
		SchedulingType: model.Anchored,
		TimeWindow: model.TimeWindow{
			Start: monday.Add(2 * time.Hour),
			End:   monday.Add(4 * time.Hour), // height = 12 lines
		},
	}

	m := &viewmodel.Model{
		CurrentView:      viewmodel.WeekView,
		CurrentMode:      viewmodel.ModeNormal,
		Tasks:            []model.Task{t1, t2},
		SelectedDay:      monday,
		SelectedTaskUUID: "task-monday-1",
		ScrollOffset:     0,
		Height:           20, // results in availLaneH = 11
	}

	// 1. With tasks: pressing j cycles to the next task and triggers auto-scroll.
	// task 1: start=0, height=6.
	// task 2: start=7, height=12.
	// Moving to task 2 sets taskStart = 7, taskEnd = 19.
	// availLaneH = 11.
	// Since taskEnd (19) > ScrollOffset(0) + availLaneH(11), it scrolls to taskEnd - availLaneH = 8.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.SelectedTaskUUID != "task-monday-2" {
		t.Errorf("Expected task-monday-2 to be selected, got %s", m.SelectedTaskUUID)
	}
	if m.ScrollOffset != 8 {
		t.Errorf("Expected ScrollOffset to be 8, got %d", m.ScrollOffset)
	}

	// 2. Pressing k goes back to task 1.
	// task 1: start=0, height=6.
	// Since taskStart (0) < ScrollOffset(8), it scrolls to taskStart = 0.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.SelectedTaskUUID != "task-monday-1" {
		t.Errorf("Expected task-monday-1 to be selected, got %s", m.SelectedTaskUUID)
	}
	if m.ScrollOffset != 0 {
		t.Errorf("Expected ScrollOffset to be 0, got %d", m.ScrollOffset)
	}

	// 3. Clear all tasks on Monday to test manual scrolling.
	m.Tasks = nil
	m.SelectedTaskUUID = ""
	
	// Scroll offset should increment/decrement directly when there are no tasks
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ScrollOffset != 1 {
		t.Errorf("Expected manual scroll down ScrollOffset to be 1, got %d", m.ScrollOffset)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.ScrollOffset != 0 {
		t.Errorf("Expected manual scroll up ScrollOffset to be 0, got %d", m.ScrollOffset)
	}
}

func TestConfirmModalKeyNavigation(t *testing.T) {
	task := model.Task{
		UUID:           "task-1",
		Title:          "Task to Delete",
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}

	m := &viewmodel.Model{
		CurrentMode:      viewmodel.ModeNormal,
		Tasks:            []model.Task{task},
		SelectedTaskUUID: "task-1",
	}

	// 1. Open delete modal using 'd'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if !m.ConfirmOpen || m.ConfirmActionType != "delete" {
		t.Fatalf("Expected delete confirm modal to open, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}

	// 2. Default selection index should be 0 (Yes, Delete)
	if m.ConfirmSelectedIndex != 0 {
		t.Errorf("Expected default selected index to be 0, got %d", m.ConfirmSelectedIndex)
	}

	// 3. Move down/right to option 1 (No, Cancel)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ConfirmSelectedIndex != 1 {
		t.Errorf("Expected selected index to be 1 after pressing 'j', got %d", m.ConfirmSelectedIndex)
	}

	// 4. Move up/left back to option 0 (Yes, Delete)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.ConfirmSelectedIndex != 0 {
		t.Errorf("Expected selected index to be 0 after pressing 'k', got %d", m.ConfirmSelectedIndex)
	}

	// 5. Navigate to option 1 and press enter to cancel deletion
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close after canceling deletion")
	}
	if len(m.Tasks) == 0 {
		t.Error("Expected task to NOT be deleted after canceling")
	}

	// 6. Re-open delete modal, stay on option 0 (Yes), and press enter to delete
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close after deletion")
	}
	if len(m.Tasks) != 0 {
		t.Error("Expected task to be deleted after confirming")
	}
}

func TestCompleteReminderAndLogSessionModals(t *testing.T) {
	reminder := model.Task{
		UUID:           "reminder-1",
		Title:          "A Reminder Task",
		SchedulingType: model.Reminder,
		LifecycleState: model.StateReady,
	}

	floating := model.Task{
		UUID:           "floating-1",
		Title:          "A Floating Task",
		SchedulingType: model.Floating,
		LifecycleState: model.StateReady,
	}

	m := &viewmodel.Model{
		CurrentMode:      viewmodel.ModeNormal,
		Tasks:            []model.Task{reminder, floating},
		SelectedTaskUUID: "reminder-1",
	}

	// Test 1: Complete Reminder Modal Key Navigation and Y/N
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.ConfirmOpen || m.ConfirmActionType != "complete_reminder" {
		t.Fatalf("Expected complete_reminder confirm modal to open, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.ConfirmSelectedIndex != 0 {
		t.Errorf("Expected default selected index to be 0, got %d", m.ConfirmSelectedIndex)
	}

	// Move to Cancel option using 'j'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ConfirmSelectedIndex != 1 {
		t.Errorf("Expected selected index to be 1, got %d", m.ConfirmSelectedIndex)
	}

	// Press Enter to cancel
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close after canceling")
	}
	if m.Tasks[0].LifecycleState != model.StateReady {
		t.Error("Expected reminder state to still be Ready")
	}

	// Re-open and use 'y' to complete
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close after completing")
	}
	if m.Tasks[0].LifecycleState != model.StateCompleted {
		t.Error("Expected reminder state to be Completed")
	}

	// Test 2: Log Focus Session Modal Key Navigation and Y/N
	m.SelectedTaskUUID = "floating-1"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.ConfirmOpen || m.ConfirmActionType != "log_session_confirm" {
		t.Fatalf("Expected log_session_confirm confirm modal to open, got ConfirmOpen=%t ConfirmActionType=%s", m.ConfirmOpen, m.ConfirmActionType)
	}
	if m.ConfirmSelectedIndex != 0 {
		t.Errorf("Expected default selected index to be 0, got %d", m.ConfirmSelectedIndex)
	}

	// Move to No option using 'l' (or 'right')
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.ConfirmSelectedIndex != 1 {
		t.Errorf("Expected selected index to be 1, got %d", m.ConfirmSelectedIndex)
	}

	// Press Enter to complete task without logging hours
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close")
	}
	if m.Tasks[1].LifecycleState != model.StateCompleted {
		t.Error("Expected task state to be Completed")
	}

	// Re-open (first mark incomplete) and log hours option
	m.Tasks[1].LifecycleState = model.StateReady
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.ConfirmOpen || m.ConfirmActionType != "log_session_confirm" {
		t.Fatal("Expected log_session_confirm to open again")
	}
	// Select Yes option (0) and press Enter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConfirmOpen {
		t.Error("Expected confirm modal to close")
	}
	if !m.LogSessionPromptOpen {
		t.Error("Expected log hours input prompt to open")
	}
}

