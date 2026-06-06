package tui

import (
	"sort"
	"time"

	"stream/internal/model"
)

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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (m Model) BuildDayTaskRects(tasks []model.Task) []TaskRect {
	resolved := ResolveOverlaps(tasks)
	if len(resolved) == 0 {
		return nil
	}

	const timestampLaneW = 7
	const leftSpacerW = 4
	const rightSpacerW = 2

	gridW := m.Layout.TimelineW - timestampLaneW
	if gridW < 10 {
		gridW = 10
	}

	colsAreaW := gridW - leftSpacerW - rightSpacerW
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
	if colW < 8 {
		colW = 8
	}

	var rects []TaskRect
	for _, rc := range resolved {
		startRow := timeToRow(rc.Task.TimeWindow.Start)
		endRow := timeToRow(rc.Task.TimeWindow.End)
		if endRow > totalRows {
			endRow = totalRows
		}
		h := endRow - startRow
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
	minDuration := 1 * time.Hour
	dur := t.TimeWindow.End.Sub(t.TimeWindow.Start)
	if dur < minDuration {
		return t.TimeWindow.Start.Add(minDuration)
	}
	return t.TimeWindow.End
}

// ResolveOverlaps processes a list of tasks for a single day and assigns columns to handle overlapping times
func ResolveOverlaps(tasks []model.Task) []ScheduledColumn {
	// Filter to Anchored tasks
	var anchored []model.Task
	for _, t := range tasks {
		if t.SchedulingType == model.Anchored {
			anchored = append(anchored, t)
		}
	}

	if len(anchored) == 0 {
		return nil
	}

	// Sort tasks by start time, then end time, then priority weight
	sort.Slice(anchored, func(i, j int) bool {
		startI := anchored[i].TimeWindow.Start
		startJ := anchored[j].TimeWindow.Start
		if !startI.Equal(startJ) {
			return startI.Before(startJ)
		}
		endI := getEffectiveEnd(anchored[i])
		endJ := getEffectiveEnd(anchored[j])
		if !endI.Equal(endJ) {
			return endI.Before(endJ)
		}
		return anchored[i].SortingWeight() > anchored[j].SortingWeight()
	})

	// Assign columns
	var cols []int // active tasks column indices
	var active []model.Task
	results := make([]ScheduledColumn, len(anchored))

	for i, task := range anchored {
		// Remove inactive tasks that don't overlap with current task start
		var nextActive []model.Task
		var nextCols []int
		for j, act := range active {
			effEnd := getEffectiveEnd(act)
			if effEnd.After(task.TimeWindow.Start) {
				nextActive = append(nextActive, act)
				nextCols = append(nextCols, cols[j])
			}
		}
		active = nextActive
		cols = nextCols

		// Find lowest available column index
		colIdx := 0
		for {
			used := false
			for _, c := range cols {
				if c == colIdx {
					used = true
					break
				}
			}
			if !used {
				break
			}
			colIdx++
		}

		cols = append(cols, colIdx)
		active = append(active, task)

		results[i] = ScheduledColumn{
			ColIndex: colIdx,
			Task:     task,
		}
	}

	// Find the max column index in any overlapping group to calculate TotalCol
	// We can group tasks into connected components of overlapping intervals
	for i := 0; i < len(results); i++ {
		maxCol := results[i].ColIndex

		// Look ahead and behind for overlapping tasks to find the max column count in this overlap cluster
		for j := 0; j < len(results); j++ {
			if i == j {
				continue
			}
			tI := results[i].Task
			tJ := results[j].Task

			// Check if intervals overlap
			startI := tI.TimeWindow.Start
			endI := getEffectiveEnd(tI)
			startJ := tJ.TimeWindow.Start
			endJ := getEffectiveEnd(tJ)

			overlap := startI.Before(endJ) && startJ.Before(endI)
			if overlap {
				if results[j].ColIndex > maxCol {
					maxCol = results[j].ColIndex
				}
			}
		}
		results[i].TotalCol = maxCol + 1
	}

	return results
}
