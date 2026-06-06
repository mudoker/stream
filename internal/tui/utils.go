package tui

import (
	"sort"
	"strconv"
	"strings"
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
		durationMinutes := int(rc.Task.TimeWindow.End.Sub(rc.Task.TimeWindow.Start).Minutes())
		h := (durationMinutes * rowsPerHour + 59) / 60
		if startRow+h > totalRows {
			h = totalRows - startRow
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

	// Sort tasks by start time, then actual end time, then priority weight.
	// Use the real end time here so short tasks ( < 1h ) don't artificially
	// expand for overlap grouping and force unrelated tasks into the same
	// columns. getEffectiveEnd() is intentionally avoided here.
	sort.Slice(anchored, func(i, j int) bool {
		startI := anchored[i].TimeWindow.Start
		startJ := anchored[j].TimeWindow.Start
		if !startI.Equal(startJ) {
			return startI.Before(startJ)
		}
		endI := anchored[i].TimeWindow.End
		endJ := anchored[j].TimeWindow.End
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
			// Use the actual end time when deciding whether an active task
			// still overlaps the current task's start. Using getEffectiveEnd
			// here caused short tasks to be treated as long and incorrectly
			// kept in the active set.
			effEnd := act.TimeWindow.End
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
			endI := tI.TimeWindow.End
			startJ := tJ.TimeWindow.Start
			endJ := tJ.TimeWindow.End

			// Two intervals overlap if each starts before the other ends.
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

// ParseFlexibleTime parses time input in multiple formats:
// - "14" -> 14:00
// - "14:30" -> 14:30
// - "9" -> 09:00
// - "09:30" -> 09:30
// If parsing fails, returns the provided default values.
func ParseFlexibleTime(timeStr string, defaultHour, defaultMin int) (int, int) {
	hour, min := defaultHour, defaultMin

	if strings.Contains(timeStr, ":") {
		// Format: HH:MM - only apply if both hour and minute are valid
		parts := strings.Split(timeStr, ":")
		if len(parts) == 2 {
			h, errH := strconv.Atoi(parts[0])
			m, errM := strconv.Atoi(parts[1])
			// Only update if both parse successfully and are in valid ranges
			if errH == nil && errM == nil && h >= 0 && h < 24 && m >= 0 && m < 60 {
				hour = h
				min = m
			}
		}
	} else {
		// Format: H or HH
		if h, err := strconv.Atoi(strings.TrimSpace(timeStr)); err == nil && h >= 0 && h < 24 {
			hour = h
			min = 0
		}
	}

	return hour, min
}
