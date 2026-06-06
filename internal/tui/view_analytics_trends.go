package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderActivationTrendPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6
	innerH := h - 2

	today := time.Now()
	var q4Blocks []string

	maxBlocks := innerW / 2
	if maxBlocks < 7 {
		maxBlocks = 7
	}
	if maxBlocks > 30 {
		maxBlocks = 30
	}

	for i := maxBlocks - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		secs := stats.dailyFocusSecs[date.Format("2006-01-02")]
		hrs := float64(secs) / 3600.0
		var cellColor lipgloss.Color
		char := "■"
		switch {
		case hrs == 0:
			cellColor = m.Theme.Muted
			char = "░"
		case hrs <= 1.5:
			cellColor = m.Theme.P2Color
		case hrs <= 4.0:
			cellColor = m.Theme.Accent
		default:
			cellColor = m.Theme.SuccessColor
		}
		q4Blocks = append(q4Blocks, lipgloss.NewStyle().Foreground(cellColor).Render(char))
	}
	blocksStr := strings.Join(q4Blocks, " ")

	legend := fmt.Sprintf("Less  %s  %s  %s  %s  More",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("░"),
		lipgloss.NewStyle().Foreground(m.Theme.P2Color).Render("■"),
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("■"),
		lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("■"),
	)

	centeredBlocks := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(blocksStr)
	centeredLegend := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(legend)
	
	var lines []string
	
	paddingTop := (innerH - 2 - 3) / 2
	if paddingTop < 0 {
		paddingTop = 0
	}
	for i := 0; i < paddingTop; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, centeredBlocks, "", centeredLegend)

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🧱 30-DAY ACTIVATION TREND", lines, w, h, borderCol)
}

func (m Model) renderVelocityTrendPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -7)
	fourteenDaysAgo := today.AddDate(0, 0, -14)

	compThisWeek := 0
	compLastWeek := 0

	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			if t.UpdatedAt.After(sevenDaysAgo) && t.UpdatedAt.Before(today) {
				compThisWeek++
			} else if t.UpdatedAt.After(fourteenDaysAgo) && t.UpdatedAt.Before(sevenDaysAgo) {
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
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))
		return fmt.Sprintf("  %s %s  %d tasks", label, bar, val)
	}

	var lines []string
	lines = append(lines,
		renderRow("This Week:", compThisWeek, m.Theme.Accent),
		"",
		renderRow("Last Week:", compLastWeek, m.Theme.Muted),
		"",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("  Average weekly output: %.1f tasks completed", float64(compThisWeek+compLastWeek)/2.0)),
	)

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🔥 FOCUS FOCUS VELOCITY TREND", lines, w, h, borderCol)
}

func (m Model) renderStreakPerformancePanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	consistency := "NONE"
	if stats.weeklySuccessRate >= 80 {
		consistency = "EXCELLENT"
	} else if stats.weeklySuccessRate >= 50 {
		consistency = "GOOD"
	} else if stats.weeklySuccessRate > 0 {
		consistency = "FAIR"
	}

	var lines []string
	lines = append(lines,
		renderRow("Current Focus Streak:", fmt.Sprintf("%d Days", stats.streak)),
		"",
		renderRow("All-Time Longest Streak:", fmt.Sprintf("%d Days", stats.longestStreak)),
		"",
		renderRow("Weekly Success Rate:", fmt.Sprintf("%.0f%%", stats.weeklySuccessRate)),
		"",
		renderRow("Operational consistency:", consistency),
	)

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("⚡ STREAK PERFORMANCE", lines, w, h, borderCol)
}

func (m Model) renderMonthOverMonthPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	monthlySecs := make(map[time.Month]int)
	for dateStr, secs := range stats.dailyFocusSecs {
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
		
		barStr := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %s %s  %3.0fh", monthNames[idx], barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 4
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("📅 MONTH-OVER-MONTH SUMMARY", lines, w, h, borderCol)
}
