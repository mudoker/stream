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
