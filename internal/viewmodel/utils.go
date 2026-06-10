package viewmodel

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/timer"
)

const (
	RowsPerHour = 8   // 7.5-minute slots per hour (8 rows/hour)
	TotalRows   = 192 // 24h * 8 rows
	GutterWidth = 11  // " HH:MM ───┼" timestamp gutter
)

// TimeToRow converts a time.Time to its local day row index (0 to TotalRows-1).
func TimeToRow(t time.Time) int {
	local := t.Local()
	return (local.Hour() * RowsPerHour) + (local.Minute() * RowsPerHour / 60)
}

// SameDay returns true if a and b are on the same calendar day in local time.
func SameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}

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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (m *Model) BuildDayTaskRects(tasks []model.Task) []TaskRect {
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
		startRow := TimeToRow(rc.Task.TimeWindow.Start)
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

	// Sort tasks by start time, then actual end time + rest time, then priority weight.
	sort.Slice(anchored, func(i, j int) bool {
		startI := anchored[i].TimeWindow.Start
		startJ := anchored[j].TimeWindow.Start
		if !startI.Equal(startJ) {
			return startI.Before(startJ)
		}
		endI := anchored[i].TimeWindow.End.Add(CalculateTaskRestTime(anchored[i]))
		endJ := anchored[j].TimeWindow.End.Add(CalculateTaskRestTime(anchored[j]))
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
			effEnd := act.TimeWindow.End.Add(CalculateTaskRestTime(act))
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
			endI := tI.TimeWindow.End.Add(CalculateTaskRestTime(tI))
			startJ := tJ.TimeWindow.Start
			endJ := tJ.TimeWindow.End.Add(CalculateTaskRestTime(tJ))

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

func CalculateTaskRestTime(t model.Task) time.Duration {
	workDur := t.TimeWindow.End.Sub(t.TimeWindow.Start)
	if t.SchedulingType != model.Anchored {
		workDur = time.Duration(t.StoryPoints) * 45 * time.Minute
	}
	sessions := timer.PartitionTask(workDur)
	var rest time.Duration
	for _, s := range sessions {
		if s.Type == timer.BreakSession {
			rest += s.Duration
		}
	}
	return rest
}

func (m *Model) HasPriorityOverlapCollision(t model.Task) bool {
	if m.CurrentMode == ModeTaskMove || strings.HasSuffix(t.UUID, "_moving") {
		return false
	}
	if t.SchedulingType != model.Anchored {
		return false
	}
	taskUUID := movingTaskBaseUUID(t.UUID)
	for _, t2 := range m.Tasks {
		if movingTaskBaseUUID(t2.UUID) == taskUUID || t2.SchedulingType != model.Anchored {
			continue
		}
		if !SameDay(t.TimeWindow.Start, t2.TimeWindow.Start) {
			continue
		}
		overlap := t.TimeWindow.Start.Before(t2.TimeWindow.End) && t2.TimeWindow.Start.Before(t.TimeWindow.End)
		if overlap {
			if t.Priority == model.P0 || t.Priority == model.P1 || t2.Priority == model.P0 || t2.Priority == model.P1 {
				return true
			}
		}
	}
	return false
}

func movingTaskBaseUUID(uuid string) string {
	return strings.TrimSuffix(uuid, "_moving")
}

type TagVal struct {
	Tag  string
	Secs int
}

type AnalyticsStats struct {
	Streak                           int
	LongestStreak                    int
	WeeklySuccessRate                float64
	EffectiveSessions                int
	TotalHrs                         float64
	WorkHrs                          float64
	PersonalHrs                      float64
	WorkPct                          float64
	PersonalPct                      float64
	PurityPct                        float64
	TotalFocusSecs                   int
	TotalInterruptions               int
	CompletedCount                   int
	TotalCount                       int
	Rate                             float64
	Tags                             []TagVal
	DailyFocusSecs                   map[string]int
}

func (m *Model) CalculateAnalyticsStats() AnalyticsStats {
	today := time.Now()
	completionsByDate := make(map[string]bool)
	var totalFocusSecs int
	var totalInterruptions int
	var completedCount int
	var totalCount int
	var workSecs int
	var personalSecs int
	var effectiveSessions int
	var completedWithNoInterruptionCount int
	tagSecs := make(map[string]int)

	for _, t := range m.Tasks {
		totalCount++
		if t.LifecycleState == model.StateCompleted {
			completedCount++
			dateStr := t.UpdatedAt.Format("2006-01-02")
			completionsByDate[dateStr] = true

			if t.ExecutionMetrics.InterruptionCount == 0 {
				completedWithNoInterruptionCount++
			}

			isPersonal := false
			for _, tag := range t.Tags {
				if strings.ToLower(tag) == "personal" {
					isPersonal = true
					break
				}
			}
			if strings.Contains(strings.ToLower(t.Title), "personal") || strings.Contains(strings.ToLower(t.Description), "personal") {
				isPersonal = true
			}

			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && t.SchedulingType == model.Anchored {
				dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}

			if isPersonal {
				personalSecs += dur
			} else {
				workSecs += dur
			}

			effectiveSessions += t.ExecutionMetrics.TotalCompletedPomodoros

			for _, tag := range t.Tags {
				normalized := strings.TrimSpace(tag)
				if normalized != "" {
					tagSecs[normalized] += dur
				}
			}
		}
		totalFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
		totalInterruptions += t.ExecutionMetrics.InterruptionCount
	}

	// Calculate Streak
	streak := 0
	checkDate := today
	todayStr := today.Format("2006-01-02")
	if completionsByDate[todayStr] {
		for {
			dateStr := checkDate.Format("2006-01-02")
			if completionsByDate[dateStr] {
				streak++
				checkDate = checkDate.AddDate(0, 0, -1)
			} else {
				break
			}
		}
	} else {
		yesterday := today.AddDate(0, 0, -1)
		yesterdayStr := yesterday.Format("2006-01-02")
		if completionsByDate[yesterdayStr] {
			checkDate = yesterday
			for {
				dateStr := checkDate.Format("2006-01-02")
				if completionsByDate[dateStr] {
					streak++
					checkDate = checkDate.AddDate(0, 0, -1)
				} else {
					break
				}
			}
		}
	}

	// Longest Streak calculation
	longestStreak := 0
	if len(completionsByDate) > 0 {
		var dates []time.Time
		for dateStr := range completionsByDate {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dates = append(dates, t)
			}
		}
		sort.Slice(dates, func(i, j int) bool {
			return dates[i].Before(dates[j])
		})

		currentLongest := 1
		longestStreak = 1
		for i := 1; i < len(dates); i++ {
			daysDiff := int(dates[i].Sub(dates[i-1]).Hours() / 24)
			if daysDiff == 1 {
				currentLongest++
			} else if daysDiff > 1 {
				if currentLongest > longestStreak {
					longestStreak = currentLongest
				}
				currentLongest = 1
			}
		}
		if currentLongest > longestStreak {
			longestStreak = currentLongest
		}
	}

	// Weekly Success Rate
	sevenDaysAgo := today.AddDate(0, 0, -7)
	completedInLast7Days := 0
	totalInLast7Days := 0
	for _, t := range m.Tasks {
		inLast7Days := false
		if t.SchedulingType == model.Anchored {
			inLast7Days = t.TimeWindow.Start.After(sevenDaysAgo)
		} else {
			inLast7Days = t.CreatedAt.After(sevenDaysAgo)
		}
		if inLast7Days {
			totalInLast7Days++
			if t.LifecycleState == model.StateCompleted {
				completedInLast7Days++
			}
		}
	}
	weeklySuccessRate := 100.0
	if totalInLast7Days > 0 {
		weeklySuccessRate = float64(completedInLast7Days) / float64(totalInLast7Days) * 100
	}

	rate := 0.0
	if totalCount > 0 {
		rate = float64(completedCount) / float64(totalCount) * 100
	}

	workHrs := float64(workSecs) / 3600.0
	personalHrs := float64(personalSecs) / 3600.0
	totalHrs := workHrs + personalHrs

	totalHrsForRatio := totalHrs
	if totalHrsForRatio == 0 {
		totalHrsForRatio = 1.0
	}
	workPct := workHrs / totalHrsForRatio
	personalPct := personalHrs / totalHrsForRatio

	purityPct := 0.0
	if completedCount > 0 {
		purityPct = (float64(completedWithNoInterruptionCount) / float64(completedCount)) * 100
	}

	var sortedTags []TagVal
	for k, v := range tagSecs {
		sortedTags = append(sortedTags, TagVal{Tag: k, Secs: v})
	}
	sort.Slice(sortedTags, func(i, j int) bool {
		if sortedTags[i].Secs == sortedTags[j].Secs {
			return sortedTags[i].Tag < sortedTags[j].Tag
		}
		return sortedTags[i].Secs > sortedTags[j].Secs
	})

	dailyFocusSecs := make(map[string]int)
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			dateStr := t.UpdatedAt.Format("2006-01-02")
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && t.SchedulingType == model.Anchored {
				dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}
			dailyFocusSecs[dateStr] += dur
		}
	}

	return AnalyticsStats{
		Streak:             streak,
		LongestStreak:      longestStreak,
		WeeklySuccessRate:  weeklySuccessRate,
		EffectiveSessions:  effectiveSessions,
		TotalHrs:           totalHrs,
		WorkHrs:            workHrs,
		PersonalHrs:        personalHrs,
		WorkPct:            workPct,
		PersonalPct:        personalPct,
		PurityPct:          purityPct,
		TotalFocusSecs:     totalFocusSecs,
		TotalInterruptions: totalInterruptions,
		CompletedCount:     completedCount,
		TotalCount:         totalCount,
		Rate:               rate,
		Tags:               sortedTags,
		DailyFocusSecs:     dailyFocusSecs,
	}
}

