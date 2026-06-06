package tui

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func stripAnsi(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func assertClosedRectangle(t *testing.T, card string, width, height int) {
	lines := strings.Split(card, "\n")
	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d\n%s", height, len(lines), card)
	}

	for row, raw := range lines {
		line := stripAnsi(raw)
		if lipgloss.Width(line) != width {
			t.Fatalf("line %d width mismatch: expected %d, got %d\n%q", row, width, lipgloss.Width(line), line)
		}
		runes := []rune(line)
		if row == 0 {
			if len(runes) < 2 || runes[0] != '┌' || runes[len(runes)-1] != '┐' {
				t.Fatalf("top border malformed on row %d: %q", row, line)
			}
			for i := 1; i < len(runes)-1; i++ {
				if runes[i] != '─' {
					t.Fatalf("top border broken at column %d: %q", i, line)
				}
			}
		} else if row == height-1 {
			if len(runes) < 2 || runes[0] != '└' || runes[len(runes)-1] != '┘' {
				t.Fatalf("bottom border malformed on row %d: %q", row, line)
			}
			for i := 1; i < len(runes)-1; i++ {
				if runes[i] != '─' {
					t.Fatalf("bottom border broken at column %d: %q", i, line)
				}
			}
		} else {
			if len(runes) < 2 || runes[0] != '│' || runes[len(runes)-1] != '│' {
				t.Fatalf("vertical border missing on row %d: %q", row, line)
			}
		}
	}
}

func TestRenderTaskCardMaintainsClosedRectangle(t *testing.T) {
	theme := NewTheme()
	m := Model{Theme: theme}
	task := model.Task{
		UUID: "task-1",
		Title: "Border Test",
		Priority: model.P2,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC),
		},
		StoryPoints: 3,
	}

	card := m.renderTaskCard(task, 18, 4, false, false)
	assertClosedRectangle(t, card, 18, 4)

	cardSmall := m.renderTaskCard(task, 18, 3, false, false)
	assertClosedRectangle(t, cardSmall, 18, 3)
}

func TestDayViewSpatialNavigation(t *testing.T) {
	m := Model{
		Layout:     computeLayout(120, 30),
		Theme:      NewTheme(),
		SelectedDay: time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
	}

	tasks := []model.Task{
		{UUID: "A", Title: "A", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC)}},
		{UUID: "B", Title: "B", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC)}},
		{UUID: "C", Title: "C", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)}},
		{UUID: "D", Title: "D", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)}},
	}
	m.Tasks = tasks

	m.SelectedTaskUUID = "A"
	m.navigateHorizontal(1)
	if m.SelectedTaskUUID != "B" {
		t.Fatalf("expected right navigation from A to B, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "A"
	m.navigateVertical(1)
	if m.SelectedTaskUUID != "C" {
		t.Fatalf("expected down navigation from A to C, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "D"
	m.navigateHorizontal(-1)
	if m.SelectedTaskUUID != "C" {
		t.Fatalf("expected left navigation from D to C, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "D"
	m.navigateVertical(-1)
	if m.SelectedTaskUUID != "B" {
		t.Fatalf("expected up navigation from D to B, got %s", m.SelectedTaskUUID)
	}
}

func TestRegressionOverlappingScheduleBorderIntegrity(t *testing.T) {
	m := Model{
		Layout:      computeLayout(120, 40),
		Theme:       NewTheme(),
		SelectedDay: time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
	}

	tasks := []model.Task{
		{UUID: "A", Title: "A", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 17, 0, 0, 0, time.UTC)}},
		{UUID: "B", Title: "B", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 30, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 14, 30, 0, 0, time.UTC)}},
		{UUID: "C", Title: "C", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)}},
		{UUID: "D", Title: "D", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 16, 0, 0, 0, time.UTC)}},
		{UUID: "E", Title: "E", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 15, 30, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 16, 15, 0, 0, time.UTC)}},
		{UUID: "F", Title: "F", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 16, 30, 0, 0, time.UTC), End: time.Date(2026, 6, 6, 17, 30, 0, 0, time.UTC)}},
	}
	m.Tasks = tasks

	rects := m.BuildDayTaskRects(tasks)
	if len(rects) != len(tasks) {
		t.Fatalf("expected %d task rects, got %d", len(tasks), len(rects))
	}

	for _, rect := range rects {
		card := m.renderTaskCard(rect.Task, rect.Width, rect.Height, false, false)
		assertClosedRectangle(t, card, rect.Width, rect.Height)
	}
}
