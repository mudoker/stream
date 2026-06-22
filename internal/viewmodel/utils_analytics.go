package viewmodel

import (
	"sort"
	"strings"
	"time"

	"stream/internal/model"
)

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
		if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
			continue
		}
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
			if dur == 0 && model.IsTaskAnchored(t) {
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
		if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
			continue
		}
		inLast7Days := false
		if model.IsTaskAnchored(t) {
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
		if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
			continue
		}
		if t.LifecycleState == model.StateCompleted {
			dateStr := t.UpdatedAt.Format("2006-01-02")
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && model.IsTaskAnchored(t) {
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
