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

func (m Model) renderCapacityPanel(w, h int) string {
	innerW := w - 6
	innerH := h - 2

	today := time.Now()
	weeklyPoints := make(map[time.Weekday]int)
	weeklyCompletedPoints := make(map[time.Weekday]int)
	weeklyCount := make(map[time.Weekday]int)
	
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()))
	for i := 0; i < 7; i++ {
		day := startOfWeek.AddDate(0, 0, i)
		for _, t := range m.Tasks {
			if t.TimeWindow.Start.Year() == day.Year() &&
				t.TimeWindow.Start.Month() == day.Month() &&
				t.TimeWindow.Start.Day() == day.Day() {
				weeklyPoints[day.Weekday()] += t.StoryPoints
				weeklyCount[day.Weekday()]++
				if t.LifecycleState == model.StateCompleted {
					weeklyCompletedPoints[day.Weekday()] += t.StoryPoints
				}
			}
		}
	}

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}

	maxPoints := 1
	for _, wd := range weekdays {
		if weeklyPoints[wd] > maxPoints {
			maxPoints = weeklyPoints[wd]
		}
	}

	var lines []string
	isDetailed := innerH >= 10
	isExpanded := innerH >= 7 && !isDetailed

	barMaxW := innerW - 32
	if barMaxW < 4 {
		barMaxW = 4
	}

	for idx, wd := range weekdays {
		pts := weeklyPoints[wd]
		compPts := weeklyCompletedPoints[wd]
		count := weeklyCount[wd]

		solidW := int(math.Round(float64(pts) * float64(barMaxW) / float64(maxPoints)))
		if solidW > barMaxW {
			solidW = barMaxW
		}
		if solidW == 0 && pts > 0 {
			solidW = 1
		}
		mutedW := barMaxW - solidW
		if mutedW < 0 {
			mutedW = 0
		}

		isToday := wd == today.Weekday()
		nameColor := m.Theme.Muted
		if isToday {
			nameColor = m.Theme.Fg
		}
		nameStr := lipgloss.NewStyle().Foreground(nameColor).Bold(isToday).Render(weekdayNames[idx])

		solidBar := strings.Repeat("█", solidW)
		mutedBar := strings.Repeat("░", mutedW)

		barColor := m.Theme.Accent
		if pts >= 9 {
			barColor = m.Theme.P0Color
		} else if pts == 0 {
			barColor = m.Theme.Muted
		}

		solidStr := lipgloss.NewStyle().Foreground(barColor).Render(solidBar)
		mutedStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(mutedBar)
		barStr := solidStr + mutedStr

		var ptStr string
		if isDetailed {
			ptStr = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%2d/%2d SP (%d tasks)", compPts, pts, count))
		} else if isExpanded {
			ptStr = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%2d/%2d SP", compPts, pts))
		} else {
			ptStr = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%dSP", pts))
		}

		rowContent := fmt.Sprintf("  %-5s %s  %s", nameStr, barStr, ptStr)
		lines = append(lines, rowContent)
	}

	remainingLines := innerH - 2 - len(lines)
	if remainingLines > 2 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", innerW)))
		totalWeeklySP := 0
		totalWeeklyCompSP := 0
		for _, wd := range weekdays {
			totalWeeklySP += weeklyPoints[wd]
			totalWeeklyCompSP += weeklyCompletedPoints[wd]
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("WEEKLY CAPACITY SNAPSHOT:"),
			fmt.Sprintf(" • Total Velocity:    %d / %d SP completed", totalWeeklyCompSP, totalWeeklySP),
		)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 1 && m.DashboardFocusRow == 0
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("📊 WEEKLY CAPACITY UTILIZATION", lines, w, h, borderCol)
}

func (m Model) renderUpcomingPanel(w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string
	today := time.Now()

	var upcoming []model.Task
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			continue
		}
		isFuture := false
		if t.SchedulingType == model.Anchored {
			isFuture = t.TimeWindow.Start.After(today) && !sameDay(t.TimeWindow.Start, today)
		} else {
			isFuture = !sameDay(t.CreatedAt, today)
		}
		if isFuture {
			upcoming = append(upcoming, t)
		}
	}

	sort.Slice(upcoming, func(i, j int) bool {
		if upcoming[i].Priority != upcoming[j].Priority {
			return upcoming[i].Priority < upcoming[j].Priority
		}
		return upcoming[i].Title < upcoming[j].Title
	})

	if len(upcoming) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(" • No future tasks scheduled."))
	} else {
		maxCount := innerH / 2 - 2
		if maxCount < 2 {
			maxCount = 2
		}
		for idx, t := range upcoming {
			if idx >= maxCount {
				break
			}
			pColor := m.priorityColor(t.Priority)
			pBadge := lipgloss.NewStyle().Foreground(pColor).Render(fmt.Sprintf("[%s]", string(t.Priority)))
			
			fixedW := 8
			suffixStr := ""
			if t.SchedulingType == model.Anchored {
				suffixStr = fmt.Sprintf(" (%s)", t.TimeWindow.Start.Format("Mon Jan _2"))
				fixedW = 21
			}
			
			title := sentenceCase(t.Title)
			maxTitleW := innerW - fixedW
			if maxTitleW < 5 {
				maxTitleW = 5
			}
			
			titleRunes := []rune(title)
			if len(titleRunes) > maxTitleW {
				title = string(titleRunes[:maxTitleW-1]) + "…"
			}
			
			row := fmt.Sprintf(" • %s %s%s", pBadge, title, suffixStr)
			lines = append(lines, row)
		}
	}

	remaining := innerH - len(lines) - 2
	if remaining > 5 {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", innerW)),
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("🔥 DAILY LOAD DISTRIBUTION:"),
		)
		
		pCounts := make(map[model.Priority]int)
		for _, t := range m.Tasks {
			if t.LifecycleState != model.StateCompleted {
				pCounts[t.Priority]++
			}
		}

		priorities := []model.Priority{model.P0, model.P1, model.P2, model.P3}
		pNames := []string{"P0 Critical", "P1 High    ", "P2 Medium  ", "P3 Low     "}
		pColors := []lipgloss.Color{m.Theme.P0Color, m.Theme.P1Color, m.Theme.P2Color, m.Theme.P3Color}

		maxVal := 1
		for _, p := range priorities {
			if pCounts[p] > maxVal {
				maxVal = pCounts[p]
			}
		}

		barMax := innerW - 18
		if barMax < 5 {
			barMax = 5
		}

		for idx, p := range priorities {
			cnt := pCounts[p]
			fillW := int(math.Round(float64(cnt) * float64(barMax) / float64(maxVal)))
			if fillW > barMax {
				fillW = barMax
			}
			if fillW == 0 && cnt > 0 {
				fillW = 1
			}
			bar := strings.Repeat("█", fillW) + strings.Repeat("░", barMax-fillW)
			barStyled := lipgloss.NewStyle().Foreground(pColors[idx]).Render(bar)
			row := fmt.Sprintf("  %s %s %2d tasks", pNames[idx], barStyled, cnt)
			lines = append(lines, row)
		}
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 0 && m.DashboardFocusRow == 1
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🎯 TARGETS & LOAD DISTRIBUTION", lines, w, h, borderCol)
}

func (m Model) getRecommendedCapacity() int {
	return 15
}

func (m Model) priorityColor(p model.Priority) lipgloss.Color {
	switch p {
	case model.P0:
		return m.Theme.P0Color
	case model.P1:
		return m.Theme.P1Color
	case model.P2:
		return m.Theme.P2Color
	default:
		return m.Theme.P3Color
	}
}
