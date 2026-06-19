package pages

import (
	"math"
	"strings"
	"time"
	"sort"
	"fmt"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderDailyAllocationPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
	maxDaySecs := 1
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -6+i)
		daysList[i] = day
		s := stats.DailyFocusSecs[day.Format("2006-01-02")]
		daySecsList[i] = s
		if s > maxDaySecs {
			maxDaySecs = s
		}
	}

	targetHrs := 8.0
	barTotalW := innerW - 22
	if barTotalW < 4 {
		barTotalW = 4
	}

	var lines []string
	for i := 0; i < 7; i++ {
		day := daysList[i]
		daySecs := daySecsList[i]
		dayHrs := float64(daySecs) / 3600.0

		solidW := int(math.Round(dayHrs / targetHrs * float64(barTotalW)))
		if solidW > barTotalW {
			solidW = barTotalW
		}
		if solidW == 0 && daySecs > 0 {
			solidW = 1
		}
		mutedW := barTotalW - solidW
		if mutedW < 0 {
			mutedW = 0
		}

		solidBar := strings.Repeat("█", solidW)
		mutedBar := strings.Repeat("░", mutedW)

		solidStyled := lipgloss.NewStyle().Foreground(t.Accent).Render(solidBar)
		mutedStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(mutedBar)
		barStr := solidStyled + mutedStyled

		isToday := day.Format("2006-01-02") == today.Format("2006-01-02")
		dayLabel := day.Format("Mon _2")
		if isToday {
			dayLabel = lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(dayLabel)
		} else {
			dayLabel = lipgloss.NewStyle().Foreground(t.Muted).Render(dayLabel)
		}

		hrsStr := fmt.Sprintf("%4.1fh", dayHrs)
		row := fmt.Sprintf("  %-8s  %s  %s", dayLabel, barStr, hrsStr)
		if lipgloss.Width(row) > innerW {
			row = string([]rune(row)[:innerW])
		}
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 0
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📅 DAILY FOCUS & ALLOCATION", lines, w, h, borderCol)
}

func renderFocusSessionTimelinePanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6

	var lines []string
	today := time.Now()

	var timelineTasks []model.Task
	for _, task := range m.Tasks {
		if task.SchedulingType == model.Anchored && viewmodel.SameDay(task.TimeWindow.Start, today) {
			timelineTasks = append(timelineTasks, task)
		}
	}
	sort.Slice(timelineTasks, func(i, j int) bool {
		return timelineTasks[i].TimeWindow.Start.Before(timelineTasks[j].TimeWindow.Start)
	})

	if len(timelineTasks) == 0 {
		lines = []string{
			lipgloss.NewStyle().Foreground(t.Muted).Render("  No focus blocks scheduled for today."),
		}
	} else {
		for idx, task := range timelineTasks {
			startStr := task.TimeWindow.Start.Format("15:04")
			durMin := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())

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

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "󱎫 FOCUS SESSION TIMELINE", lines, w, h, borderCol)
}

func renderActivationTrendPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
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
		secs := stats.DailyFocusSecs[date.Format("2006-01-02")]
		hrs := float64(secs) / 3600.0
		var cellColor lipgloss.Color
		char := "■"
		switch {
		case hrs == 0:
			cellColor = t.Muted
			char = "░"
		case hrs <= 1.5:
			cellColor = t.P2Color
		case hrs <= 4.0:
			cellColor = t.Accent
		default:
			cellColor = t.SuccessColor
		}
		q4Blocks = append(q4Blocks, lipgloss.NewStyle().Foreground(cellColor).Render(char))
	}
	blocksStr := strings.Join(q4Blocks, " ")

	legend := fmt.Sprintf("Less  %s  %s  %s  %s  More",
		lipgloss.NewStyle().Foreground(t.Muted).Render("░"),
		lipgloss.NewStyle().Foreground(t.P2Color).Render("■"),
		lipgloss.NewStyle().Foreground(t.Accent).Render("■"),
		lipgloss.NewStyle().Foreground(t.SuccessColor).Render("■"),
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

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🧱 30-DAY ACTIVATION TREND", lines, w, h, borderCol)
}
