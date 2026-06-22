package tests

import (
	"testing"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
)

func TestCalculateAnalyticsStats_Streak(t *testing.T) {
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Completed today",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
			},
			{
				UUID:           "t2",
				Title:          "Completed yesterday",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      yesterday,
			},
			{
				UUID:           "t3",
				Title:          "Completed day before yesterday",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      dayBefore,
			},
		},
	}

	stats := m.CalculateAnalyticsStats()
	if stats.Streak != 3 {
		t.Errorf("Expected current streak to be 3, got %d", stats.Streak)
	}
	if stats.LongestStreak != 3 {
		t.Errorf("Expected longest streak to be 3, got %d", stats.LongestStreak)
	}

	// 2. Test a streak gap: today has no completions, yesterday has, day before has
	m2 := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t2",
				Title:          "Completed yesterday",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      yesterday,
			},
			{
				UUID:           "t3",
				Title:          "Completed day before",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      dayBefore,
			},
		},
	}
	stats2 := m2.CalculateAnalyticsStats()
	if stats2.Streak != 2 {
		t.Errorf("Expected streak to be maintained at 2 when yesterday was completed but today is not yet, got %d", stats2.Streak)
	}

	// 3. Test absolute gap (e.g. today and yesterday empty, day before completed)
	m3 := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t3",
				Title:          "Completed day before",
				LifecycleState: model.StateCompleted,
				UpdatedAt:      dayBefore,
			},
		},
	}
	stats3 := m3.CalculateAnalyticsStats()
	if stats3.Streak != 0 {
		t.Errorf("Expected streak to be broken (0) when yesterday was not completed, got %d", stats3.Streak)
	}
}

func TestCalculateAnalyticsStats_WorkVsPersonal(t *testing.T) {
	today := time.Now()

	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Work task",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(2 * time.Hour), // 2 hours
				},
			},
			{
				UUID:           "t2",
				Title:          "Buy personal groceries",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(1 * time.Hour), // 1 hour
				},
			},
			{
				UUID:           "t3",
				Title:          "Call friend",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				Tags:           []string{"Personal"},
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(30 * time.Minute), // 0.5 hours
				},
			},
		},
	}

	stats := m.CalculateAnalyticsStats()
	if stats.WorkHrs != 2.0 {
		t.Errorf("Expected 2.0 work hours, got %.1f", stats.WorkHrs)
	}
	if stats.PersonalHrs != 1.5 {
		t.Errorf("Expected 1.5 personal hours, got %.1f", stats.PersonalHrs)
	}
	if stats.TotalHrs != 3.5 {
		t.Errorf("Expected 3.5 total hours, got %.1f", stats.TotalHrs)
	}
}

func TestCalculateAnalyticsStats_PurityAndTags(t *testing.T) {
	today := time.Now()

	m := &viewmodel.Model{
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Coding",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				Tags:           []string{"development", "go"},
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(1 * time.Hour),
				},
				ExecutionMetrics: model.ExecutionMetrics{
					InterruptionCount: 0,
				},
			},
			{
				UUID:           "t2",
				Title:          "Email",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				Tags:           []string{"development"},
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(30 * time.Minute),
				},
				ExecutionMetrics: model.ExecutionMetrics{
					InterruptionCount: 1, // Interrupted
				},
			},
		},
	}

	stats := m.CalculateAnalyticsStats()

	// Purity: 1 out of 2 completed tasks had 0 interruptions = 50%
	if stats.PurityPct != 50.0 {
		t.Errorf("Expected purity pct 50.0%%, got %.1f%%", stats.PurityPct)
	}

	// Tags: "development" is in both (1h + 30m = 1.5h = 5400s), "go" is in one (1h = 3600s)
	if len(stats.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(stats.Tags))
	}
	if stats.Tags[0].Tag != "development" || stats.Tags[0].Secs != 5400 {
		t.Errorf("Expected development tag first with 5400s, got %v", stats.Tags[0])
	}
	if stats.Tags[1].Tag != "go" || stats.Tags[1].Secs != 3600 {
		t.Errorf("Expected go tag second with 3600s, got %v", stats.Tags[1])
	}
}

func TestCalculateAnalyticsStats_WorkspaceFilter(t *testing.T) {
	today := time.Now()

	m := &viewmodel.Model{
		ActiveWorkspaceUUID: "ws-1",
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Task in ws-1",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				WorkspaceUUID:  "ws-1",
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(1 * time.Hour),
				},
			},
			{
				UUID:           "t2",
				Title:          "Task in ws-2",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				WorkspaceUUID:  "ws-2",
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(2 * time.Hour),
				},
			},
		},
	}

	stats := m.CalculateAnalyticsStats()

	// Only t1 should be counted, so total hours should be 1.0, and total count/completed count should be 1.
	if stats.TotalHrs != 1.0 {
		t.Errorf("Expected 1.0 total hours (only ws-1 counted), got %.1f", stats.TotalHrs)
	}
	if stats.CompletedCount != 1 {
		t.Errorf("Expected 1 completed count, got %d", stats.CompletedCount)
	}
	if stats.TotalCount != 1 {
		t.Errorf("Expected 1 total count, got %d", stats.TotalCount)
	}
}

func TestCalculateAnalyticsStats_AllWorkspaceAggregation(t *testing.T) {
	today := time.Now()

	m := &viewmodel.Model{
		ActiveWorkspaceUUID: "ALL_WORKSPACES",
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Task in ws-1",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				WorkspaceUUID:  "ws-1",
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(1 * time.Hour),
				},
			},
			{
				UUID:           "t2",
				Title:          "Task in ws-2",
				SchedulingType: model.Anchored,
				LifecycleState: model.StateCompleted,
				UpdatedAt:      today,
				WorkspaceUUID:  "ws-2",
				TimeWindow: model.TimeWindow{
					Start: today,
					End:   today.Add(2 * time.Hour),
				},
			},
		},
	}

	stats := m.CalculateAnalyticsStats()

	// Both tasks should be counted in ALL_WORKSPACES, so total hours should be 3.0, and completed count/total count should be 2.
	if stats.TotalHrs != 3.0 {
		t.Errorf("Expected 3.0 total hours (both workspaces counted), got %.1f", stats.TotalHrs)
	}
	if stats.CompletedCount != 2 {
		t.Errorf("Expected 2 completed count, got %d", stats.CompletedCount)
	}
	if stats.TotalCount != 2 {
		t.Errorf("Expected 2 total count, got %d", stats.TotalCount)
	}
}

func TestGetTodoShelfTasks_AllWorkspaceAggregation(t *testing.T) {
	today := time.Now()

	m := &viewmodel.Model{
		ActiveWorkspaceUUID: "ALL_WORKSPACES",
		SelectedDay:         today,
		Tasks: []model.Task{
			{
				UUID:           "t1",
				Title:          "Task in ws-1",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
				WorkspaceUUID:  "ws-1",
				CreatedAt:      today,
			},
			{
				UUID:           "t2",
				Title:          "Task in ws-2",
				SchedulingType: model.Floating,
				LifecycleState: model.StateReady,
				WorkspaceUUID:  "ws-2",
				CreatedAt:      today,
			},
		},
	}

	shelf := m.GetTodoShelfTasks()
	if len(shelf) != 2 {
		t.Errorf("Expected 2 tasks in todo shelf under All workspace, got %d", len(shelf))
	}
}


