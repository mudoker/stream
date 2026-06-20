package viewmodel

import (
	"strings"
	"testing"
	"time"

	"stream/internal/model"
)

func TestResolveOverlaps_MovingOverlay(t *testing.T) {
	// A: 10:00-11:00, B: 11:00-12:00, C: 12:00-13:00 (all consecutive normal events)
	// D: 10:30-11:30 (moving clone event, overlaps with A and B)
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	taskA := model.Task{
		UUID:           "task-A",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(10 * time.Hour),
			End:   dayStart.Add(11 * time.Hour),
		},
	}
	taskB := model.Task{
		UUID:           "task-B",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(11 * time.Hour),
			End:   dayStart.Add(12 * time.Hour),
		},
	}
	taskC := model.Task{
		UUID:           "task-C",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(12 * time.Hour),
			End:   dayStart.Add(13 * time.Hour),
		},
	}
	taskDMoving := model.Task{
		UUID:           "task-D_moving",
		SchedulingType: model.Event,
		TimeWindow: model.TimeWindow{
			Start: dayStart.Add(10*time.Hour + 30*time.Minute),
			End:   dayStart.Add(11*time.Hour + 30*time.Minute),
		},
	}

	tasks := []model.Task{taskA, taskB, taskC, taskDMoving}

	// 1. ResolveOverlaps test
	cols := ResolveOverlaps(tasks)
	if len(cols) != 4 {
		t.Fatalf("expected 4 column resolutions, got %d", len(cols))
	}

	// Ensure normal tasks (A, B, C) are not influenced by the moving clone
	for _, col := range cols {
		if strings.HasSuffix(col.Task.UUID, "_moving") {
			// Special moving task must have ColIndex = 0, TotalCol = 1
			if col.ColIndex != 0 || col.TotalCol != 1 {
				t.Errorf("special moving task should have ColIndex = 0 and TotalCol = 1, got ColIndex=%d TotalCol=%d", col.ColIndex, col.TotalCol)
			}
		} else {
			// Normal tasks do not overlap with each other, so they must have ColIndex = 0, TotalCol = 1
			if col.ColIndex != 0 || col.TotalCol != 1 {
				t.Errorf("normal task %s should have ColIndex = 0 and TotalCol = 1, got ColIndex=%d TotalCol=%d", col.Task.UUID, col.ColIndex, col.TotalCol)
			}
		}
	}

	// 2. BuildDayTaskRects test (shifting logic bypass)
	m := &Model{}
	m.Layout.TimelineW = 50
	rects := m.BuildDayTaskRects(tasks)
	if len(rects) != 4 {
		t.Fatalf("expected 4 task rects, got %d", len(rects))
	}

	for _, r := range rects {
		expectedStartRow := TimeToRow(r.Task.TimeWindow.Start) / 5
		expectedEndRow := TimeToRow(r.Task.TimeWindow.End) / 5
		expectedHeight := expectedEndRow - expectedStartRow

		if r.Top != expectedStartRow {
			t.Errorf("task %s rect top was shifted: expected %d, got %d", r.Task.UUID, expectedStartRow, r.Top)
		}
		if r.Height != expectedHeight {
			t.Errorf("task %s rect height was altered: expected %d, got %d", r.Task.UUID, expectedHeight, r.Height)
		}
	}
}
