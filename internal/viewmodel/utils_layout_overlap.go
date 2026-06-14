package viewmodel

import (
	"sort"
	"strings"
	"time"

	"stream/internal/model"
)

// taskEffectiveInterval computes the start and end boundaries of a task,
// including any rest buffers or commute buffers.
func taskEffectiveInterval(t model.Task) (time.Time, time.Time) {
	start := t.TimeWindow.Start
	end := t.TimeWindow.End.Add(CalculateTaskRestTime(t))

	if t.SchedulingType == model.Event && strings.TrimSpace(t.Location) != "" && t.CommuteBuffer > 0 {
		commute := time.Duration(t.CommuteBuffer) * time.Minute
		start = start.Add(-commute)
		end = end.Add(commute)
	}
	return start, end
}

// ResolveOverlaps processes a list of tasks for a single day and assigns columns to handle overlapping times
func ResolveOverlaps(tasks []model.Task) []ScheduledColumn {
	var anchored []model.Task
	for _, t := range tasks {
		if model.IsTaskAnchored(t) {
			anchored = append(anchored, t)
		}
	}

	if len(anchored) == 0 {
		return nil
	}

	// Sort tasks by effective start time, then effective end time, then priority weight.
	sort.Slice(anchored, func(i, j int) bool {
		startI, endI := taskEffectiveInterval(anchored[i])
		startJ, endJ := taskEffectiveInterval(anchored[j])
		if !startI.Equal(startJ) {
			return startI.Before(startJ)
		}
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
		taskStart, _ := taskEffectiveInterval(task)

		// Remove inactive tasks that don't overlap with current task start
		var nextActive []model.Task
		var nextCols []int
		for j, act := range active {
			_, effEnd := taskEffectiveInterval(act)
			if effEnd.After(taskStart) {
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
	for i := 0; i < len(results); i++ {
		maxCol := results[i].ColIndex

		for j := 0; j < len(results); j++ {
			if i == j {
				continue
			}
			tI := results[i].Task
			tJ := results[j].Task

			startI, endI := taskEffectiveInterval(tI)
			startJ, endJ := taskEffectiveInterval(tJ)

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
