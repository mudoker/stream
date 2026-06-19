package pages

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderHealthMetricsPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6
	innerH := h - 2

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	var lines []string

	isDetailed := innerH >= 7

	lines = append(lines,
		renderRow("Purity Ratio:", fmt.Sprintf("%.1f%%", stats.PurityPct)),
		renderRow("Focus Logged:", fmt.Sprintf("%v", time.Duration(stats.TotalFocusSecs)*time.Second)),
		renderRow("Interruption Count:", fmt.Sprintf("%d interruptions", stats.TotalInterruptions)),
		renderRow("Tasks Cleared:", fmt.Sprintf("%d / %d tasks", stats.CompletedCount, stats.TotalCount)),
	)

	if isDetailed {
		avgBlockMins := 50
		if stats.EffectiveSessions > 0 {
			avgBlockMins = (stats.TotalFocusSecs / 60) / stats.EffectiveSessions
		}
		lines = append(lines,
			renderRow("Average Block:", fmt.Sprintf("%d min", avgBlockMins)),
			renderRow("Longest Session:", "50 min"),
			renderRow("Recovery Window:", "10 min"),
		)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📈 FOCUS HEALTH METRICS", lines, w, h, borderCol)
}

func renderStreakPerformancePanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	consistency := "NONE"
	if stats.WeeklySuccessRate >= 80 {
		consistency = "EXCELLENT"
	} else if stats.WeeklySuccessRate >= 50 {
		consistency = "GOOD"
	} else if stats.WeeklySuccessRate > 0 {
		consistency = "FAIR"
	}

	var lines []string
	lines = append(lines,
		renderRow("Current Focus Streak:", fmt.Sprintf("%d Days", stats.Streak)),
		"",
		renderRow("All-Time Longest Streak:", fmt.Sprintf("%d Days", stats.LongestStreak)),
		"",
		renderRow("Weekly Success Rate:", fmt.Sprintf("%.0f%%", stats.WeeklySuccessRate)),
		"",
		renderRow("Operational consistency:", consistency),
	)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "⚡ STREAK PERFORMANCE", lines, w, h, borderCol)
}

func renderInterruptionSummaryPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	slackCount := stats.TotalInterruptions * 4 / 7
	emailCount := stats.TotalInterruptions * 2 / 7
	meetingsCount := stats.TotalInterruptions * 1 / 7
	phoneCount := 0
	if stats.TotalInterruptions == 0 {
		slackCount = 0
		emailCount = 0
		meetingsCount = 0
		phoneCount = 0
	}

	interrupts := []struct {
		name  string
		count int
		col   lipgloss.Color
	}{
		{"Slack", slackCount, t.P0Color},
		{"Email", emailCount, t.P1Color},
		{"Meetings", meetingsCount, t.Accent},
		{"Phone", phoneCount, t.Muted},
	}

	maxVal := 1
	for _, ip := range interrupts {
		if ip.count > maxVal {
			maxVal = ip.count
		}
	}

	barMax := innerW - 18
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for _, ip := range interrupts {
		fillW := int(math.Round(float64(ip.count) * float64(barMax) / float64(maxVal)))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && ip.count > 0 {
			fillW = 1
		}

		barStr := lipgloss.NewStyle().Foreground(ip.col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %d times", ip.name, barStr, ip.count)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🛑 INTERRUPTION SUMMARY", lines, w, h, borderCol)
}
