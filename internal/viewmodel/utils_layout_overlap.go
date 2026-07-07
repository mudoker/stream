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

	if t.HasCommuteBuffer() {
		commute := time.Duration(t.CommuteBuffer) * time.Minute
		start = start.Add(-commute)
		end = end.Add(commute)
	}
	return start, end
}

// ResolveOverlaps processes a list of tasks for a single day and assigns columns to handle overlapping times
func ResolveOverlaps(tasks []model.Task) []ScheduledColumn {
	var normal []model.Task
	var special []model.Task
	for _, t := range tasks {
		if model.IsTaskAnchored(t) {
			if strings.HasSuffix(t.UUID, "_moving") || strings.HasSuffix(t.UUID, "_adjusting") {
				special = append(special, t)
			} else {
				normal = append(normal, t)
			}
		}
	}

	var results []ScheduledColumn

	if len(normal) > 0 {
		// Sort normal tasks by effective start time, then effective end time, then priority weight.
		sort.Slice(normal, func(i, j int) bool {
			startI, endI := taskEffectiveInterval(normal[i])
			startJ, endJ := taskEffectiveInterval(normal[j])
			if !startI.Equal(startJ) {
				return startI.Before(startJ)
			}
			if !endI.Equal(endJ) {
				return endI.Before(endJ)
			}
			return normal[i].SortingWeight() > normal[j].SortingWeight()
		})

		// Assign columns for normal tasks
		var cols []int // active tasks column indices
		var active []model.Task
		normalResults := make([]ScheduledColumn, len(normal))

		for i, task := range normal {
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

			normalResults[i] = ScheduledColumn{
				ColIndex: colIdx,
				Task:     task,
			}
		}

		// Find the max column index in any overlapping group to calculate TotalCol for normal tasks
		for i := 0; i < len(normalResults); i++ {
			maxCol := normalResults[i].ColIndex

			for j := 0; j < len(normalResults); j++ {
				if i == j {
					continue
				}
				tI := normalResults[i].Task
				tJ := normalResults[j].Task

				startI, endI := taskEffectiveInterval(tI)
				startJ, endJ := taskEffectiveInterval(tJ)

				// Two intervals overlap if each starts before the other ends.
				overlap := startI.Before(endJ) && startJ.Before(endI)
				if overlap {
					if normalResults[j].ColIndex > maxCol {
						maxCol = normalResults[j].ColIndex
					}
				}
			}
			normalResults[i].TotalCol = maxCol + 1
		}

		results = append(results, normalResults...)
	}

	// For each special task, assign it ColIndex = 0, TotalCol = 1
	for _, t := range special {
		results = append(results, ScheduledColumn{
			ColIndex: 0,
			TotalCol: 1,
			Task:     t,
		})
	}

	return results
}
