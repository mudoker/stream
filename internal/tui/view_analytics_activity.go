package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHourHeatmapPanel(w, h int) string {
	innerW := w - 6

	morningSecs := 0
	afternoonSecs := 0
	eveningSecs := 0

	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored {
			hour := t.TimeWindow.Start.Hour()
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}
			if hour >= 8 && hour < 12 {
				morningSecs += dur
			} else if hour >= 12 && hour < 18 {
				afternoonSecs += dur
			} else {
				eveningSecs += dur
			}
		}
	}

	total := morningSecs + afternoonSecs + eveningSecs
	if total == 0 {
		total = 1
	}

	mPct := float64(morningSecs) / float64(total) * 100
	aPct := float64(afternoonSecs) / float64(total) * 100
	ePct := float64(eveningSecs) / float64(total) * 100

	barMax := innerW - 25
	if barMax < 6 {
		barMax = 6
	}

	renderBar := func(pct float64, col lipgloss.Color) string {
		fillW := int(math.Round(pct * float64(barMax) / 100.0))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && pct > 0 {
			fillW = 1
		}
		return lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))
	}

	var lines []string
	lines = append(lines,
		fmt.Sprintf("  Morning (08-12):   %s  %2.0f%%", renderBar(mPct, m.Theme.Accent), mPct),
		"",
		fmt.Sprintf("  Afternoon (12-18): %s  %2.0f%%", renderBar(aPct, m.Theme.FocusPurple), aPct),
		"",
		fmt.Sprintf("  Evening (18-00):   %s  %2.0f%%", renderBar(ePct, m.Theme.SuccessColor), ePct),
	)

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 2
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🔥 HOURLY FOCUS HEATMAP", lines, w, h, borderCol)
}

func (m Model) renderFocusSessionTimelinePanel(w, h int) string {
	innerW := w - 6

	var lines []string
	today := time.Now()

	var timelineTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, today) {
			timelineTasks = append(timelineTasks, t)
		}
	}
	sort.Slice(timelineTasks, func(i, j int) bool {
		return timelineTasks[i].TimeWindow.Start.Before(timelineTasks[j].TimeWindow.Start)
	})

	if len(timelineTasks) == 0 {
		lines = []string{
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No focus blocks scheduled for today."),
		}
	} else {
		for idx, t := range timelineTasks {
			startStr := t.TimeWindow.Start.Format("15:04")
			durMin := int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Minutes())
			
			barLen := durMin / 10
			if barLen < 2 {
				barLen = 2
			}
			if barLen > 25 {
				barLen = 25
			}
			bar := strings.Repeat("─", barLen) + "■"
			row := fmt.Sprintf("  %s %s  %d min Block", startStr, bar, durMin)
			if lipgloss.Width(row) > innerW {
				row = string([]rune(row)[:innerW])
			}
			lines = append(lines, row)
			if idx < len(timelineTasks)-1 {
				lines = append(lines, "")
			}
		}
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("󱎫 FOCUS SESSION TIMELINE", lines, w, h, borderCol)
}

func (m Model) renderInterruptionSummaryPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	slackCount := stats.totalInterruptions * 4 / 7
	emailCount := stats.totalInterruptions * 2 / 7
	meetingsCount := stats.totalInterruptions * 1 / 7
	phoneCount := 0
	if stats.totalInterruptions == 0 {
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
		{"Slack", slackCount, m.Theme.P0Color},
		{"Email", emailCount, m.Theme.P1Color},
		{"Meetings", meetingsCount, m.Theme.Accent},
		{"Phone", phoneCount, m.Theme.Muted},
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
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %d times", ip.name, barStr, ip.count)
		lines = append(lines, row)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🛑 INTERRUPTION SUMMARY", lines, w, h, borderCol)
}
