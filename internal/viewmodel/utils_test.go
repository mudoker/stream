package viewmodel

import (
	"testing"
	"time"

	"stream/internal/model"
)

func TestCalculateAnalyticsStats(t *testing.T) {
	now := time.Now()
	m := &Model{
		Tasks: []model.Task{
			{
				UUID:           "t-work",
				Title:          "Write code",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				Tags:           []string{"dev"},
				TimeWindow: model.TimeWindow{
					Start: now.Add(-1 * time.Hour),
					End:   now,
				},
				ExecutionMetrics: model.ExecutionMetrics{
					ElapsedFocusSeconds:     3600,
					TotalCompletedPomodoros: 2,
					InterruptionCount:       0,
				},
				UpdatedAt: now,
			},
			{
				UUID:           "t-personal",
				Title:          "Buy milk",
				SchedulingType: model.Floating,
				LifecycleState: model.StateCompleted,
				Tags:           []string{"personal"},
				StoryPoints:    1,
				ExecutionMetrics: model.ExecutionMetrics{
					ElapsedFocusSeconds:     1200,
					TotalCompletedPomodoros: 1,
					InterruptionCount:       1,
				},
				UpdatedAt: now,
			},
		},
	}

	stats := m.CalculateAnalyticsStats()
	if stats.TotalCount != 2 {
		t.Errorf("expected 2 tasks, got %d", stats.TotalCount)
	}
	if stats.CompletedCount != 2 {
		t.Errorf("expected 2 completed tasks, got %d", stats.CompletedCount)
	}
	if stats.Streak != 1 {
		t.Errorf("expected 1 streak, got %d", stats.Streak)
	}
	if stats.WorkHrs != 1.0 {
		t.Errorf("expected 1.0 work focus hour, got %f", stats.WorkHrs)
	}
	if stats.PersonalHrs != 1200.0/3600.0 {
		t.Errorf("expected personal focus hours matching 1200s, got %f", stats.PersonalHrs)
	}
	if stats.TotalFocusSecs != 4800 {
		t.Errorf("expected 4800 total focus seconds, got %d", stats.TotalFocusSecs)
	}
}

func TestUtilsRowAndOverlaps(t *testing.T) {
	// 1. TimeToRow
	row := TimeToRow(time.Date(2026, 6, 11, 9, 30, 0, 0, time.Local))
	if row != 114 { // (9*12) + (30*12/60) = 108 + 6 = 114
		t.Errorf("expected row 114, got %d", row)
	}

	// 2. PartitionHeights
	heights := PartitionHeights(30, 2)
	if len(heights) != 2 || heights[0] != 15 || heights[1] != 15 {
		t.Errorf("expected partitioned heights [15 15], got %v", heights)
	}

	// 3. ResolveOverlaps and BuildDayTaskRects
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tasks := []model.Task{
		{
			UUID:           "t-overlap-1",
			Title:          "Meeting 1",
			SchedulingType: model.Anchored,
			Priority:       model.P1,
			TimeWindow: model.TimeWindow{
				Start: dayStart.Add(10 * time.Hour),
				End:   dayStart.Add(11 * time.Hour),
			},
		},
		{
			UUID:           "t-overlap-2",
			Title:          "Meeting 2",
			SchedulingType: model.Anchored,
			Priority:       model.P2,
			TimeWindow: model.TimeWindow{
				Start: dayStart.Add(10 * time.Hour),
				End:   dayStart.Add(11 * time.Hour),
			},
		},
	}

	m := &Model{}
	m.Layout.TimelineW = 50
	rects := m.BuildDayTaskRects(tasks)
	if len(rects) != 2 {
		t.Fatalf("expected 2 task rects, got %d", len(rects))
	}
	if rects[0].TotalCol != 2 || rects[1].TotalCol != 2 {
		t.Errorf("expected overlaps columns partitioning to be 2")
	}

	// 4. CalculateTaskRestTime
	restTime := CalculateTaskRestTime(tasks[0])
	if restTime != 10*time.Minute {
		t.Errorf("expected 10m rest time, got %v", restTime)
	}

	// 5. HasPriorityOverlapCollision
	m.Tasks = tasks
	collision := m.HasPriorityOverlapCollision(tasks[0])
	if !collision {
		t.Errorf("expected priority collision with self/others at same time")
	}

	// 6. absInt
	if absInt(-5) != 5 || absInt(5) != 5 {
		t.Errorf("absInt failed")
	}

	// 7. getEffectiveEnd
	shortTask := model.Task{
		TimeWindow: model.TimeWindow{
			Start: dayStart,
			End:   dayStart.Add(15 * time.Minute),
		},
	}
	effEnd := getEffectiveEnd(shortTask)
	if effEnd.Sub(dayStart) != 1*time.Hour {
		t.Errorf("expected 1 hour minimum effective end for short task, got %v", effEnd.Sub(dayStart))
	}

	longTask := model.Task{
		TimeWindow: model.TimeWindow{
			Start: dayStart,
			End:   dayStart.Add(2 * time.Hour),
		},
	}
	if getEffectiveEnd(longTask) != longTask.TimeWindow.End {
		t.Errorf("expected getEffectiveEnd to preserve long task end time")
	}

	// 8. ParseFlexibleTime
	h1, m1 := ParseFlexibleTime("14:45", 9, 0)
	if h1 != 14 || m1 != 45 {
		t.Errorf("expected 14:45, got %02d:%02d", h1, m1)
	}
	h2, m2 := ParseFlexibleTime("invalid", 9, 0)
	if h2 != 9 || m2 != 0 {
		t.Errorf("expected fallback 9:00, got %02d:%02d", h2, m2)
	}
}
