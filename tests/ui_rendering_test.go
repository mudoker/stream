package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/view/components"
	"stream/internal/view/modals"
	"stream/internal/view/pages"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"
)

func TestRenderCardLayout(t *testing.T) {
	th := theme.NewTheme()
	m := &viewmodel.Model{}

	task := model.Task{
		UUID:           "test-task-1",
		Title:          "Write Antigravity Tests",
		Priority:       model.P1,
		SchedulingType: model.Anchored,
		StoryPoints:    3,
		TimeWindow: model.TimeWindow{
			Start: time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 6, 19, 10, 30, 0, 0, time.UTC),
		},
		LifecycleState: model.StateReady,
	}

	// 1. Test standard rendering (height >= 3)
	card := components.RenderCard(m, th, task, 40, 5, false, false)
	cleaned := cleanAnsi(card)

	if !strings.Contains(cleaned, "Write Antigravity Tests") {
		t.Errorf("Expected card content to contain sentence-case title, got:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, "09:00 → 10:30") {
		t.Errorf("Expected card content to contain time window, got:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, "3 SP") {
		t.Errorf("Expected card content to contain story points, got:\n%s", cleaned)
	}

	// 2. Test short rendering (height < 3)
	shortCard := components.RenderCard(m, th, task, 40, 2, false, false)
	cleanedShort := cleanAnsi(shortCard)

	if !strings.Contains(cleanedShort, "Write Antigravity Tests") {
		t.Errorf("Expected short card to contain title, got:\n%s", cleanedShort)
	}
	if !strings.Contains(cleanedShort, "▲ P1") {
		t.Errorf("Expected short card to contain priority, got:\n%s", cleanedShort)
	}
}

func TestRenderExecutionMetricsFormatter(t *testing.T) {
	task := model.Task{
		Title:          "Metrics Testing",
		SchedulingType: model.Anchored,
		StoryPoints:    2,
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 3600,
			ElapsedBreakSeconds: 600,
			InterruptionCount:   2,
		},
	}

	m := &viewmodel.Model{}
	th := theme.NewTheme()
	info := modals.ComputeTaskMetricsInfo(m, th, task)

	// Test with no indent, without pomodoros
	res := cleanAnsi(modals.RenderExecutionMetrics(task, info, "", " ", false))

	if !strings.Contains(res, "EXECUTION METRICS") {
		t.Errorf("Expected header, got:\n%s", res)
	}
	if !strings.Contains(res, "Planned Time:") {
		t.Errorf("Expected Planned Time, got:\n%s", res)
	}
	if !strings.Contains(res, "Focus/Rest:") {
		t.Errorf("Expected Focus/Rest ratio, got:\n%s", res)
	}
	if strings.Contains(res, "Pomodoros:") {
		t.Errorf("Did not expect Pomodoros line when includePomodoros is false, got:\n%s", res)
	}

	// Test Event task - should not render rest time
	eventTask := model.Task{
		Title:          "Commute Meeting",
		SchedulingType: model.Event,
		ExecutionMetrics: model.ExecutionMetrics{
			ElapsedFocusSeconds: 1800,
			ElapsedBreakSeconds: 180,
		},
	}
	eventInfo := modals.ComputeTaskMetricsInfo(m, th, eventTask)
	eventRes := cleanAnsi(modals.RenderExecutionMetrics(eventTask, eventInfo, "", " ", false))

	if strings.Contains(eventRes, "Rest Logged:") {
		t.Errorf("Expected Event task to omit Rest Logged, got:\n%s", eventRes)
	}
	if strings.Contains(eventRes, "Focus/Rest:") {
		t.Errorf("Expected Event task to omit Focus/Rest ratio, got:\n%s", eventRes)
	}
}

func TestDetailModalAndPanel(t *testing.T) {
	m := &viewmodel.Model{
		DetailTask: model.Task{
			UUID:           "inspect-task",
			Title:          "Inspect Me Please",
			Priority:       model.P0,
			StoryPoints:    5,
			SchedulingType: model.Anchored,
			TimeWindow: model.TimeWindow{
				Start: time.Date(2026, 6, 19, 14, 0, 0, 0, time.UTC),
				End:   time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC),
			},
			Description: "This is a detailed description of the task.",
		},
	}
	th := theme.NewTheme()

	// 1. Test Detail Modal
	modalOutput := modals.RenderDetailModal(m, th)
	cleanedModal := cleanAnsi(modalOutput)

	if !strings.Contains(cleanedModal, "Task Inspector") {
		t.Errorf("Expected detail modal header, got:\n%s", cleanedModal)
	}
	if !strings.Contains(cleanedModal, "Inspect Me Please") {
		t.Errorf("Expected detail modal to contain sentence-case title, got:\n%s", cleanedModal)
	}
	if !strings.Contains(cleanedModal, "This is a detailed description") {
		t.Errorf("Expected detail modal to contain description, got:\n%s", cleanedModal)
	}

	// 2. Test Detail Panel
	panelOutput := modals.RenderDetailPanel(m, th, 40)
	cleanedPanel := cleanAnsi(panelOutput)

	if !strings.Contains(cleanedPanel, "INSPECT ME PLEASE") {
		t.Errorf("Expected detail panel title in uppercase, got:\n%s", cleanedPanel)
	}
}

func TestPagesSimpleRenders(t *testing.T) {
	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Task 1",
				Priority:       model.P1,
				SchedulingType: model.Anchored,
				StoryPoints:    2,
				TimeWindow: model.TimeWindow{
					Start: time.Now(),
					End:   time.Now().Add(1 * time.Hour),
				},
				LifecycleState: model.StateReady,
			},
		},
		Workspaces: []model.Workspace{
			{
				UUID: "default-ws",
				Name: "General",
				Icon: "📂",
			},
		},
	}
	m.Layout.WorkspaceW = 80
	th := theme.NewTheme()

	// 1. Render Dashboard
	dashboardRes := pages.RenderDashboard(m, th, 40)
	if len(dashboardRes) == 0 {
		t.Error("RenderDashboard returned empty string")
	}
	if !strings.Contains(cleanAnsi(dashboardRes), "PLANNED") {
		t.Error("RenderDashboard does not contain banner markers")
	}

	// 2. Render Analytics
	analyticsRes := pages.RenderAnalyticsView(m, th, 40)
	if len(analyticsRes) == 0 {
		t.Error("RenderAnalyticsView returned empty string")
	}
	if !strings.Contains(cleanAnsi(analyticsRes), "STREAK") {
		t.Error("RenderAnalyticsView does not contain Streak metric")
	}
}

func TestCommandPaletteFiltering(t *testing.T) {
	m := &viewmodel.Model{}
	m.Width = 60
	m.CommandInput.SetValue("Quit")
	th := theme.NewTheme()

	paletteOutput := modals.RenderCommandPalette(m, th)
	cleanedPalette := cleanAnsi(paletteOutput)

	if !strings.Contains(cleanedPalette, "quit") && !strings.Contains(cleanedPalette, "Quit") {
		t.Errorf("Expected palette to contain filtered command matching 'Quit', got:\n%s", cleanedPalette)
	}
}

func TestConsecutiveTasksTimelineRendering(t *testing.T) {
	today := time.Now()
	t1 := model.Task{
		UUID:           "task-1",
		Title:          "Task One",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, time.Local),
			End:   time.Date(today.Year(), today.Month(), today.Day(), 11, 0, 0, 0, time.Local),
		},
	}
	t2 := model.Task{
		UUID:           "task-2",
		Title:          "Task Two",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: time.Date(today.Year(), today.Month(), today.Day(), 11, 0, 0, 0, time.Local),
			End:   time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, time.Local),
		},
	}

	m := &viewmodel.Model{
		Tasks:            []model.Task{t1, t2},
		SelectedDay:      today,
		TimelineHour:     11,
		SelectedTaskUUID: "task-1",
	}
	m.Layout.TimelineW = 40
	m.Layout.WorkspaceW = 80
	th := theme.NewTheme()

	timelineOut := pages.RenderDayTimeline(m, th, 40)
	cleanedTimeline := cleanAnsi(timelineOut)

	// Verify both task titles are rendered in the day view timeline output
	if !strings.Contains(cleanedTimeline, "Task One") {
		t.Error("Expected consecutive Task One to be rendered on the timeline")
	}
	if !strings.Contains(cleanedTimeline, "Task Two") {
		t.Error("Expected consecutive Task Two to be rendered on the timeline")
	}

	lines := strings.Split(cleanedTimeline, "\n")
	idx1, idx2 := -1, -1
	for idx, line := range lines {
		if strings.Contains(line, "Task One") {
			idx1 = idx
		}
		if strings.Contains(line, "Task Two") {
			idx2 = idx
		}
	}

	if idx1 == -1 || idx2 == -1 {
		t.Fatalf("Could not find both task titles. Task One index: %d, Task Two index: %d", idx1, idx2)
	}

	// Verify they are rendered on adjacent rows with zero blank lines/shared lines.
	// Task One ends at 11:00 (represented by row 88). Task Two starts at 11:00, but is shifted to 89.
	// Task One title is on row 82 (idx1). Task One bottom border is on row 88 (idx1 + 6).
	// Task Two top border is on row 89 (idx1 + 7). Task Two title is on row 91 (idx1 + 9 = idx2).
	if idx2-idx1 != 9 {
		t.Errorf("Expected Task Two title line to be exactly 9 lines after Task One title line, got %d difference", idx2-idx1)
	}

	bottomBorderLine := lines[idx1+6]
	topBorderLine := lines[idx1+7]

	// Verify both borders are fully boxed (contain closed corner characters: └/╰/╯/┘ for bottom, ┌/╭/╮/┐ for top).
	// They must not contain ├ or ┤ since they are not sharing/merging borders.
	if strings.Contains(bottomBorderLine, "├") || strings.Contains(bottomBorderLine, "┤") {
		t.Errorf("Expected Task One bottom border to be closed, but found corner override characters in: %q", bottomBorderLine)
	}
	if strings.Contains(topBorderLine, "├") || strings.Contains(topBorderLine, "┤") {
		t.Errorf("Expected Task Two top border to be closed, but found corner override characters in: %q", topBorderLine)
	}

	// Verify standard corner characters exist in bottom border line
	if !strings.ContainsAny(bottomBorderLine, "└╰╯┘") {
		t.Errorf("Expected bottom border corner characters in bottom border line, got: %q", bottomBorderLine)
	}

	// Verify standard corner characters exist in top border line
	if !strings.ContainsAny(topBorderLine, "┌╭╮┐") {
		t.Errorf("Expected top border corner characters in top border line, got: %q", topBorderLine)
	}
}
