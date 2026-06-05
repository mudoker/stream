package tui

import (
	"sort"

	"tuical/internal/model"
)

type ScheduledColumn struct {
	ColIndex int
	TotalCol int
	Task     model.Task
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
			if act.TimeWindow.End.After(task.TimeWindow.Start) {
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
			overlap := tI.TimeWindow.Start.Before(tJ.TimeWindow.End) && tJ.TimeWindow.Start.Before(tI.TimeWindow.End)
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
