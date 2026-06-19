package viewmodel

import (
	"time"

	"stream/constant"
	"stream/internal/model"
)

func PartitionHeights(total int, parts int) []int {
	heights := make([]int, parts)
	base := total / parts
	rem := total % parts
	for i := 0; i < parts; i++ {
		heights[i] = base
		if i < rem {
			heights[i]++
		}
	}
	return heights
}

type ScheduledColumn struct {
	ColIndex int
	TotalCol int
	Task     model.Task
}

type TaskRect struct {
	ScheduledColumn
	Left    int
	Right   int
	Top     int
	Bottom  int
	CenterX int
	CenterY int
	Width   int
	Height  int
}

func (m *Model) BuildDayTaskRects(tasks []model.Task) []TaskRect {
	resolved := ResolveOverlaps(tasks)
	if len(resolved) == 0 {
		return nil
	}

	gridW := m.Layout.TimelineW - constant.TimelineTimestampLaneW
	if gridW < 10 {
		gridW = 10
	}

	colsAreaW := gridW - constant.TimelineLeftSpacerW - constant.TimelineRightSpacerW
	if colsAreaW < 1 {
		colsAreaW = 1
	}

	numCols := 1
	for _, rc := range resolved {
		if rc.TotalCol > numCols {
			numCols = rc.TotalCol
		}
	}

	colW := colsAreaW / numCols
	if colW < constant.TimelineMinColW {
		colW = constant.TimelineMinColW
	}

	var rects []TaskRect
	for _, rc := range resolved {
		startRow := TimeToRow(rc.Task.TimeWindow.Start.Add(1 * time.Minute))
		durationMinutes := int(rc.Task.TimeWindow.End.Sub(rc.Task.TimeWindow.Start).Minutes())
		h := (durationMinutes*RowsPerHour + 59) / 60
		if startRow+h > TotalRows {
			h = TotalRows - startRow
		}
		if h < 1 {
			h = 1
		}

		x := rc.ColIndex * colW
		y := startRow
		rects = append(rects, TaskRect{
			ScheduledColumn: rc,
			Left:            x,
			Right:           x + colW,
			Top:             y,
			Bottom:          y + h,
			CenterX:         x + colW/2,
			CenterY:         y + h/2,
			Width:           colW,
			Height:          h,
		})
	}

	return rects
}

func getEffectiveEnd(t model.Task) time.Time {
	dur := t.TimeWindow.End.Sub(t.TimeWindow.Start)
	if dur < constant.MinTaskEffectiveDuration {
		return t.TimeWindow.Start.Add(constant.MinTaskEffectiveDuration)
	}
	return t.TimeWindow.End
}
