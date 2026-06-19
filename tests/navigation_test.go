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

	// 1. Move right (l) -> changes selected day to Tuesday
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !m.SelectedDay.Equal(monday.AddDate(0, 0, 1)) {
		t.Errorf("Expected SelectedDay to be Tuesday, got %s", m.SelectedDay)
	}

	// 2. Move down (j) -> Focuses on tasks chronologically in the week
	m.SelectedTaskUUID = ""
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.SelectedTaskUUID != "task-monday" {
		t.Errorf("Expected SelectedTaskUUID to shift to task-monday, got %s", m.SelectedTaskUUID)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.SelectedTaskUUID != "task-tuesday" {
		t.Errorf("Expected SelectedTaskUUID to shift to task-tuesday, got %s", m.SelectedTaskUUID)
	}

	// 3. Move up (k) -> moves back to task-monday
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.SelectedTaskUUID != "task-monday" {
		t.Errorf("Expected SelectedTaskUUID to shift back to task-monday, got %s", m.SelectedTaskUUID)
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
