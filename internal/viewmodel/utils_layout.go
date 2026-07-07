package viewmodel

import (
	"strings"
	"time"

	"stream/internal/viewmodel/common/constants"
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

	gridW := m.Layout.TimelineW - constants.TimelineTimestampLaneW
	if gridW < 10 {
		gridW = 10
	}

	colsAreaW := gridW - constants.TimelineLeftSpacerW - constants.TimelineRightSpacerW
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
	if colW < constants.TimelineMinColW {
		colW = constants.TimelineMinColW
	}

	lastOccupiedRow := make([]int, numCols)
	for c := 0; c < numCols; c++ {
		lastOccupiedRow[c] = -1
	}

	var rects []TaskRect
	for _, rc := range resolved {
		isSpecial := strings.HasSuffix(rc.Task.UUID, "_moving") || strings.HasSuffix(rc.Task.UUID, "_adjusting")

		startRow := TimeToRow(rc.Task.TimeWindow.Start) / 5

		colIndex := rc.ColIndex
		if colIndex >= numCols {
			colIndex = numCols - 1
		}

		// Determine visual start row based on commute buffer
		commuteRows := 0
		if rc.Task.HasCommuteBuffer() {
			commuteRows = (rc.Task.CommuteBuffer*(RowsPerHour/5) + 59) / 60
		}

		topStartRow := startRow - commuteRows
		if topStartRow < 0 {
			topStartRow = 0
		}

		// Prevent visual overlap in the same column by ensuring the task starts after the predecessor's visual block
		if !isSpecial && lastOccupiedRow[colIndex] != -1 && topStartRow < lastOccupiedRow[colIndex] {
			topStartRow = lastOccupiedRow[colIndex]
			startRow = topStartRow + commuteRows
		}

		durationMinutes := int(rc.Task.TimeWindow.End.Sub(rc.Task.TimeWindow.Start).Minutes())
		h := (durationMinutes*(RowsPerHour/5) + 59) / 60
		if startRow+h > TotalRows/5 {
			h = TotalRows/5 - startRow
		}
		if h < 1 {
			h = 1
		}

		// Track the actual final row occupied by this visual block
		maxRowOccupied := startRow + h - 1

		// Add commute buffer bottom
		if rc.Task.HasCommuteBuffer() {
			maxRowOccupied += commuteRows
		}

		// Add rest buffer
		restDur := CalculateTaskRestTime(rc.Task)
		restMins := int(restDur.Minutes())
		if restMins > 0 {
			restRows := (restMins*(RowsPerHour/5) + 59) / 60
			maxRowOccupied += restRows
		}

		if !isSpecial {
			lastOccupiedRow[colIndex] = maxRowOccupied
		}

		x := rc.ColIndex * colW
		w := colW
		if isSpecial {
			x = 0
			w = colsAreaW
		}
		y := startRow
		rects = append(rects, TaskRect{
			ScheduledColumn: rc,
			Left:            x,
			Right:           x + w,
			Top:             y,
			Bottom:          y + h,
			CenterX:         x + w/2,
			CenterY:         y + h/2,
			Width:           w,
			Height:          h,
		})
	}

	return rects
}

func getEffectiveEnd(t model.Task) time.Time {
	dur := t.TimeWindow.End.Sub(t.TimeWindow.Start)
	if dur < constants.MinTaskEffectiveDuration {
		return t.TimeWindow.Start.Add(constants.MinTaskEffectiveDuration)
	}
	return t.TimeWindow.End
}
