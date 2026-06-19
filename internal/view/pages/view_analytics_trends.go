package pages

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderVelocityTrendPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -7)
	fourteenDaysAgo := today.AddDate(0, 0, -14)

	compThisWeek := 0
	compLastWeek := 0

	for _, task := range m.Tasks {
		if task.LifecycleState == model.StateCompleted {
			if task.UpdatedAt.After(sevenDaysAgo) && task.UpdatedAt.Before(today) {
				compThisWeek++
			} else if task.UpdatedAt.After(fourteenDaysAgo) && task.UpdatedAt.Before(sevenDaysAgo) {
				compLastWeek++
			}
		}
	}

	barMax := innerW - 18
	if barMax < 6 {
		barMax = 6
	}

	maxVal := compThisWeek
	if compLastWeek > maxVal {
		maxVal = compLastWeek
	}
	if maxVal == 0 {
		maxVal = 1
	}

	renderRow := func(label string, val int, col lipgloss.Color) string {
		fillW := int(math.Round(float64(val) * float64(barMax) / float64(maxVal)))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && val > 0 {
			fillW = 1
		}
		bar := lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))
		return fmt.Sprintf("  %s %s  %d tasks", label, bar, val)
	}

	var lines []string
	lines = append(lines,
		renderRow("This Week:", compThisWeek, t.Accent),
		"",
		renderRow("Last Week:", compLastWeek, t.Muted),
		"",
		lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  Average weekly output: %.1f tasks completed", float64(compThisWeek+compLastWeek)/2.0)),
	)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🔥 FOCUS FOCUS VELOCITY TREND", lines, w, h, borderCol)
}

func renderWeekdayAnalysisPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	weekdaySecs := make(map[time.Weekday]int)
	for dateStr, secs := range stats.DailyFocusSecs {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			weekdaySecs[parsed.Weekday()] += secs
		}
	}

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}

	maxSecs := 1
	for _, wd := range weekdays {
		if weekdaySecs[wd] > maxSecs {
			maxSecs = weekdaySecs[wd]
		}
	}

	barMax := innerW - 14
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for idx, wd := range weekdays {
		secs := weekdaySecs[wd]
		hrs := float64(secs) / 3600.0

		solidW := int(math.Round(float64(secs) * float64(barMax) / float64(maxSecs)))
		if solidW > barMax {
			solidW = barMax
		}
		if solidW == 0 && secs > 0 {
			solidW = 1
		}

		barStr := lipgloss.NewStyle().Foreground(t.Accent).Render(strings.Repeat("█", solidW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-solidW))

		isToday := wd == today.Weekday()
		nameStyle := lipgloss.NewStyle().Foreground(t.Muted)
		if isToday {
			nameStyle = nameStyle.Foreground(t.Fg).Bold(true)
		}

		row := fmt.Sprintf("  %s %s  %4.1fh", nameStyle.Render(weekdayNames[idx]), barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 2
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📊 WEEKDAY ANALYSIS", lines, w, h, borderCol)
}

func renderMonthOverMonthPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	monthlySecs := make(map[time.Month]int)
	for dateStr, secs := range stats.DailyFocusSecs {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil && parsed.Year() == today.Year() {
			monthlySecs[parsed.Month()] += secs
		}
	}

	months := []time.Month{time.January, time.February, time.March, time.April, time.May, time.June}
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}

	maxHrs := 1.0
	for _, mth := range months {
		hrs := float64(monthlySecs[mth]) / 3600.0
		if hrs > maxHrs {
			maxHrs = hrs
		}
	}

	barMax := innerW - 14
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for idx, mth := range months {
		hrs := float64(monthlySecs[mth]) / 3600.0
		fillW := int(math.Round(hrs * float64(barMax) / maxHrs))
		if fillW > barMax {
			fillW = barMax
		}

		barStr := lipgloss.NewStyle().Foreground(t.SuccessColor).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %s %s  %3.0fh", monthNames[idx], barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 4
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📅 MONTH-OVER-MONTH SUMMARY", lines, w, h, borderCol)
}
