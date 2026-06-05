package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard(height int) string {
	today := time.Now()
	var todayTasks []model.Task
	var completedCount int
	var plannedFocusSecs int
	var elapsedFocusSecs int

	for _, t := range m.Tasks {
		isToday := false
		if t.SchedulingType == model.Anchored {
			isToday = t.TimeWindow.Start.Year() == today.Year() &&
				t.TimeWindow.Start.Month() == today.Month() &&
				t.TimeWindow.Start.Day() == today.Day()
		} else {
			isToday = t.CreatedAt.Year() == today.Year() &&
				t.CreatedAt.Month() == today.Month() &&
				t.CreatedAt.Day() == today.Day()
		}

		if isToday {
			todayTasks = append(todayTasks, t)
			if t.LifecycleState == model.StateCompleted {
				completedCount++
			}
			plannedFocusSecs += t.StoryPoints * 45 * 60
			elapsedFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
		}
	}

	hdrStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)

	todayWidgetContent := fmt.Sprintf(
		"planned focus        %-8s\n"+
			"completed focus      %-8s\n"+
			"remaining tasks      %-8d\n",
		time.Duration(plannedFocusSecs)*time.Second,
		time.Duration(elapsedFocusSecs)*time.Second,
		len(todayTasks)-completedCount,
	)

	todayWidget := m.Theme.PanelStyle.
		Width(38).
		Render(hdrStyle.Render("T O D A Y   S U M M A R Y") + "\n\n" + todayWidgetContent)

	upcomingLines := []string{hdrStyle.Render("U P C O M I N G   T A S K S") + "\n"}
	var upcomingTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && t.TimeWindow.Start.After(today) && t.LifecycleState != model.StateCompleted {
			upcomingTasks = append(upcomingTasks, t)
		}
	}

	if len(upcomingTasks) == 0 {
		upcomingLines = append(upcomingLines, "no upcoming tasks scheduled.")
	} else {
		for i, t := range upcomingTasks {
			if i >= 3 {
				break
			}
			upcomingLines = append(upcomingLines, fmt.Sprintf("%-5s   %s", t.TimeWindow.Start.Format("15:04"), strings.ToUpper(t.Title)))
		}
	}

	upcomingWidget := m.Theme.PanelStyle.
		Width(38).
		Render(strings.Join(upcomingLines, "\n"))

	leftPane := lipgloss.JoinVertical(lipgloss.Left, todayWidget, "\n", upcomingWidget)

	// Capacity widget
	weeklyPoints := make(map[time.Weekday]int)
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()))
	for i := 0; i < 7; i++ {
		day := startOfWeek.AddDate(0, 0, i)
		for _, t := range m.Tasks {
			if t.TimeWindow.Start.Year() == day.Year() && t.TimeWindow.Start.Month() == day.Month() && t.TimeWindow.Start.Day() == day.Day() {
				weeklyPoints[day.Weekday()] += t.StoryPoints
			}
		}
	}

	chartLines := []string{hdrStyle.Render("W E E K L Y   C A P A C I T Y") + "\n"}
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	maxPoints := 0
	for _, wd := range weekdays {
		if weeklyPoints[wd] > maxPoints {
			maxPoints = weeklyPoints[wd]
		}
	}
	if maxPoints == 0 {
		maxPoints = 1
	}

	for idx, wd := range weekdays {
		pts := weeklyPoints[wd]
		barWidth := int(math.Round(float64(pts) * 18.0 / float64(maxPoints)))
		bar := strings.Repeat("█", barWidth)
		if bar == "" && pts > 0 {
			bar = "▏"
		}
		color := m.Theme.Accent
		if pts >= 9 {
			color = m.Theme.P0Color
		} else if pts <= 2 {
			color = m.Theme.Muted
		}

		coloredBar := lipgloss.NewStyle().Foreground(color).Render(bar)
		chartLines = append(chartLines, fmt.Sprintf("%s   │ %s (%d SP)", weekdayNames[idx], coloredBar, pts))
	}

	sidebarWidth := int(float64(m.Width) * 0.13)
	if sidebarWidth < 18 {
		sidebarWidth = 18
	} else if sidebarWidth > 26 {
		sidebarWidth = 26
	}
	workspaceWidth := m.Width - sidebarWidth - 3
	if workspaceWidth < 30 {
		workspaceWidth = 30
	}

	rightWidth := workspaceWidth - 48
	if rightWidth < 20 {
		rightWidth = 20
	}

	chartWidget := m.Theme.PanelStyle.
		Width(rightWidth).
		Height(12).
		Render(strings.Join(chartLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "   ", chartWidget)
}
