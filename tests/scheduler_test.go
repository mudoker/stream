package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"
	"stream/internal/view/components"
	"stream/internal/view/pages"

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
			if len(runes) < 2 || (runes[0] != '┌' && runes[0] != '╭') || (runes[len(runes)-1] != '┐' && runes[len(runes)-1] != '╮') {
				t.Fatalf("top border malformed on row %d: %q", row, line)
			}
			for i := 1; i < len(runes)-1; i++ {
				if runes[i] != '─' {
					t.Fatalf("top border broken at column %d: %q", i, line)
				}
			}
		} else if row == height-1 {
			if len(runes) < 2 || (runes[0] != '└' && runes[0] != '╰' && runes[0] != '├') || (runes[len(runes)-1] != '┘' && runes[len(runes)-1] != '╯' && runes[len(runes)-1] != '┤') {
				t.Fatalf("bottom border malformed on row %d: %q", row, line)
			}
			for i := 1; i < len(runes)-1; i++ {
				if runes[i] != '─' && runes[i] != '╌' {
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

func TestRenderCardMaintainsClosedRectangle(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{}
	task := model.Task{
		UUID: "task-1",
		Title: "Border Test",
		Priority: model.P2,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.Local),
			End:   time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local),
		},
		StoryPoints: 3,
	}

	card := components.RenderCard(m, th, task, 18, 4, false, false)
	assertClosedRectangle(t, card, 18, 4)

	cardSmall := components.RenderCard(m, th, task, 18, 3, false, false)
	assertClosedRectangle(t, cardSmall, 18, 3)
}

func TestDayViewSpatialNavigation(t *testing.T) {
	m := &viewmodel.Model{
		Layout:      viewmodel.ComputeLayout(120, 30),
		SelectedDay: time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	tasks := []model.Task{
		{UUID: "A", Title: "A", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local)}},
		{UUID: "B", Title: "B", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local)}},
		{UUID: "C", Title: "C", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.Local)}},
		{UUID: "D", Title: "D", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.Local)}},
	}
	m.Tasks = tasks

	m.SelectedTaskUUID = "A"
	m.NavigateHorizontal(1)
	if m.SelectedTaskUUID != "B" {
		t.Fatalf("expected right navigation from A to B, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "A"
	m.NavigateVertical(1)
	if m.SelectedTaskUUID != "C" {
		t.Fatalf("expected down navigation from A to C, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "D"
	m.NavigateHorizontal(-1)
	if m.SelectedTaskUUID != "C" {
		t.Fatalf("expected left navigation from D to C, got %s", m.SelectedTaskUUID)
	}

	m.SelectedTaskUUID = "D"
	m.NavigateVertical(-1)
	if m.SelectedTaskUUID != "B" {
		t.Fatalf("expected up navigation from D to B, got %s", m.SelectedTaskUUID)
	}
}

func TestRegressionOverlappingScheduleBorderIntegrity(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout:      viewmodel.ComputeLayout(120, 40),
		SelectedDay: time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
	}

	tasks := []model.Task{
		{UUID: "A", Title: "A", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 17, 0, 0, 0, time.Local)}},
		{UUID: "B", Title: "B", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 30, 0, 0, time.Local), End: time.Date(2026, 6, 6, 14, 30, 0, 0, time.Local)}},
		{UUID: "C", Title: "C", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.Local)}},
		{UUID: "D", Title: "D", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 15, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 16, 0, 0, 0, time.Local)}},
		{UUID: "E", Title: "E", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 15, 30, 0, 0, time.Local), End: time.Date(2026, 6, 6, 16, 15, 0, 0, time.Local)}},
		{UUID: "F", Title: "F", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 16, 30, 0, 0, time.Local), End: time.Date(2026, 6, 6, 17, 30, 0, 0, time.Local)}},
	}
	m.Tasks = tasks

	rects := m.BuildDayTaskRects(tasks)
	if len(rects) != len(tasks) {
		t.Fatalf("expected %d task rects, got %d", len(tasks), len(rects))
	}

	for _, rect := range rects {
		card := components.RenderCard(m, th, rect.Task, rect.Width, rect.Height, false, false)
		assertClosedRectangle(t, card, rect.Width, rect.Height)
	}
}

func TestDayTimelineRenderingBorderIntegrity(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout:       viewmodel.ComputeLayout(120, 40),
		SelectedDay:  time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
		TimelineHour: 14,
	}

	// Create adjacent/overlapping tasks that share boundary rows
	// Task A ends at 14:00, Task C starts at 14:00.
	// Task A overlaps with Task B (13:30 - 14:30), meaning column calculations are triggered.
	// Task C starts at 14:00, ends at 15:00, and is non-overlapping in its later block.
	tasks := []model.Task{
		{UUID: "A", Title: "A", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local)}},
		{UUID: "B", Title: "B", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 13, 30, 0, 0, time.Local), End: time.Date(2026, 6, 6, 14, 30, 0, 0, time.Local)}},
		{UUID: "C", Title: "C", SchedulingType: model.Anchored, Priority: model.P2, TimeWindow: model.TimeWindow{Start: time.Date(2026, 6, 6, 14, 0, 0, 0, time.Local), End: time.Date(2026, 6, 6, 15, 0, 0, 0, time.Local)}},
	}
	m.Tasks = tasks

	rendered := pages.RenderDayTimeline(m, th, 30)

	lines := strings.Split(rendered, "\n")
	expectedWidth := m.Layout.TimelineW

	t.Logf("Verifying all timeline lines match expected width %d", expectedWidth)

	// Check timeline rows (header and separator are on first few lines, start checking from index 3)
	for i := 3; i < len(lines); i++ {
		line := stripAnsi(lines[i])
		if line == "" {
			continue
		}
		width := lipgloss.Width(line)
		if width != expectedWidth {
			t.Errorf("line %d has width %d, expected consistent width of %d. Line: %q", i, width, expectedWidth, line)
		}
	}
}

func TestDayTimelineSameBlockOverlappingTasksBorderIntegrity(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{
		Layout:       viewmodel.ComputeLayout(120, 40),
		SelectedDay:  time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
		TimelineHour: 10,
	}

	// Two tasks on the exact same block (10:00 - 11:30)
	tasks := []model.Task{
		{
			UUID:           "task-same-1",
			Title:          "Drink Water Float",
			SchedulingType: model.Anchored,
			Priority:       model.P1,
			TimeWindow: model.TimeWindow{
				Start: time.Date(2026, 6, 6, 10, 0, 0, 0, time.Local),
				End:   time.Date(2026, 6, 6, 11, 30, 0, 0, time.Local),
			},
		},
		{
			UUID:           "task-same-2",
			Title:          "Read Book",
			SchedulingType: model.Anchored,
			Priority:       model.P2,
			TimeWindow: model.TimeWindow{
				Start: time.Date(2026, 6, 6, 10, 0, 0, 0, time.Local),
				End:   time.Date(2026, 6, 6, 11, 30, 0, 0, time.Local),
			},
		},
	}
	m.Tasks = tasks

	rendered := pages.RenderDayTimeline(m, th, 30)
	lines := strings.Split(rendered, "\n")
	expectedWidth := m.Layout.TimelineW

	for i := 3; i < len(lines); i++ {
		line := stripAnsi(lines[i])
		if line == "" {
			continue
		}
		width := lipgloss.Width(line)
		if width != expectedWidth {
			t.Errorf("line %d has width %d, expected consistent width of %d. Line: %q", i, width, expectedWidth, line)
		}
	}
}

