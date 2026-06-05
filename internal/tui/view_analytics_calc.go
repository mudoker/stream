package tui

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
	streak                           int
	effectiveSessions                int
	totalHrs                         float64
	workHrs                          float64
	personalHrs                      float64
	workPct                          float64
	personalPct                      float64
	purityPct                        float64
	totalFocusSecs                   int
	totalInterruptions               int
	completedCount                   int
	totalCount                       int
	rate                             float64
	tags                             []TagVal
	dailyFocusSecs                   map[string]int
}

func (m Model) calculateAnalyticsStats() AnalyticsStats {
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
		totalFocusSecs += m.Tasks[0].ExecutionMetrics.ElapsedFocusSeconds // Wait, in the original code, this was totalFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds. Let's make sure it is exactly as in original!
		// Wait! Let's check line 95: totalFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds. Yes, it was t.
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
		streak:             streak,
		effectiveSessions:  effectiveSessions,
		totalHrs:           totalHrs,
		workHrs:            workHrs,
		personalHrs:        personalHrs,
		workPct:            workPct,
		personalPct:        personalPct,
		purityPct:          purityPct,
		totalFocusSecs:     totalFocusSecs,
		totalInterruptions: totalInterruptions,
		completedCount:     completedCount,
		totalCount:         totalCount,
		rate:               rate,
		tags:               sortedTags,
		dailyFocusSecs:     dailyFocusSecs,
	}
}
