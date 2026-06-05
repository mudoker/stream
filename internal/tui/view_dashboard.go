package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard() string {
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

	workspaceWidth := m.Layout.WorkspaceW - 4 // 4 for Padding(1,2) on each side
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37"))

	// ── Header bar ──────────────────────────────────────────────────
	headerDate := lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Bold(true).
		Render(today.Format("Monday, January 2"))
	subDate := lipgloss.NewStyle().
		Foreground(m.Theme.Muted).
		Render(today.Format("2006"))

	headerLine := headerDate + "  " + subDate
	divider := sepStyle.Render(strings.Repeat("─", workspaceWidth-4))

	// ── Today KPI Strip ─────────────────────────────────────────────
	remaining := len(todayTasks) - completedCount
	completionPct := 0.0
	if len(todayTasks) > 0 {
		completionPct = float64(completedCount) / float64(len(todayTasks)) * 100
	}

	kpi := m.renderDashKPI(workspaceWidth,
		planned(plannedFocusSecs), elapsed(elapsedFocusSecs),
		completedCount, remaining, completionPct,
	)

	// ── Left: Today tasks list ────────────────────────────────────
	todayLines := []string{
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TODAY"),
		"",
	}
	if len(todayTasks) == 0 {
		todayLines = append(todayLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("No tasks scheduled for today."))
	} else {
		for _, t := range todayTasks {
			dot := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("○")
			if t.LifecycleState == model.StateCompleted {
				dot = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("✓")
			} else if t.LifecycleState == model.StateActive {
				dot = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("●")
			}
			pColor := m.priorityColor(t.Priority)
			pBadge := lipgloss.NewStyle().Foreground(pColor).Render("▲")
			title := sentenceCase(t.Title)
			if len(title) > 28 {
				title = title[:26] + "…"
			}
			timeLabel := ""
			if t.SchedulingType == model.Anchored {
				timeLabel = "  " + lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(t.TimeWindow.Start.Format("15:04"))
			}
			todayLines = append(todayLines, fmt.Sprintf(" %s %s  %s%s", dot, pBadge, title, timeLabel))
		}
	}

	// ── Right: Upcoming tasks ─────────────────────────────────────
	upcomingLines := []string{
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("UPCOMING"),
		"",
	}
	var upcomingTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && t.TimeWindow.Start.After(today) && t.LifecycleState != model.StateCompleted {
			upcomingTasks = append(upcomingTasks, t)
		}
	}
	if len(upcomingTasks) == 0 {
		upcomingLines = append(upcomingLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("No upcoming tasks."))
	} else {
		for i, t := range upcomingTasks {
			if i >= 5 {
				more := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("+ %d more…", len(upcomingTasks)-5))
				upcomingLines = append(upcomingLines, " "+more)
				break
			}
			timeStr := t.TimeWindow.Start.Format("Mon Jan _2  15:04")
			tLabel := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeStr)
			title := sentenceCase(t.Title)
			if len(title) > 22 {
				title = title[:20] + "…"
			}
			upcomingLines = append(upcomingLines, fmt.Sprintf(" %s  %s", tLabel, title))
		}
	}

	// ── Weekly capacity bar chart ─────────────────────────────────
	weekChart := m.renderWeeklyCapacityChart(workspaceWidth)

	// ── Assemble layout ──────────────────────────────────────────
	leftColW := (workspaceWidth / 2) - 4
	rightColW := workspaceWidth - leftColW - 8
	if leftColW < 20 {
		leftColW = 20
	}
	if rightColW < 20 {
		rightColW = 20
	}

	leftCol := lipgloss.NewStyle().
		Width(leftColW).
		Render(strings.Join(todayLines, "\n"))
	rightCol := lipgloss.NewStyle().
		Width(rightColW).
		Render(strings.Join(upcomingLines, "\n"))

	twoCol := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "    ", rightCol)

	var out strings.Builder
	out.WriteString(headerLine + "\n")
	out.WriteString(divider + "\n\n")
	out.WriteString(kpi + "\n\n")
	out.WriteString(divider + "\n\n")
	out.WriteString(twoCol + "\n\n")
	out.WriteString(divider + "\n\n")
	out.WriteString(weekChart)

	return out.String()
}

// renderDashKPI renders a horizontal KPI strip across the top of the dashboard.
func (m Model) renderDashKPI(w int, plannedHr, elapsedHr string, done, remaining int, pct float64) string {
	cardBg := lipgloss.NewStyle().Background(m.Theme.PanelBg)
	kpis := []struct {
		label string
		value string
		color lipgloss.Color
	}{
		{"Planned", plannedHr, m.Theme.Muted},
		{"Logged", elapsedHr, m.Theme.Accent},
		{"Done", fmt.Sprintf("%d tasks", done), m.Theme.SuccessColor},
		{"Remaining", fmt.Sprintf("%d tasks", remaining), m.Theme.P1Color},
		{"Completion", fmt.Sprintf("%.0f%%", pct), m.Theme.FocusPurple},
	}

	cardWidth := (w - 4) / len(kpis)
	if cardWidth < 12 {
		cardWidth = 12
	}
	var cards []string
	for _, k := range kpis {
		label := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render(k.label)
		value := lipgloss.NewStyle().Foreground(k.color).Bold(true).Render(k.value)
		card := cardBg.
			Width(cardWidth - 1).
			Padding(0, 1).
			Render(label + "\n" + value)
		cards = append(cards, card)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
}

// renderWeeklyCapacityChart renders a horizontal bar chart of weekly task load.
func (m Model) renderWeeklyCapacityChart(w int) string {
	today := time.Now()
	weeklyPoints := make(map[time.Weekday]int)
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()))
	for i := 0; i < 7; i++ {
		day := startOfWeek.AddDate(0, 0, i)
		for _, t := range m.Tasks {
			if t.TimeWindow.Start.Year() == day.Year() &&
				t.TimeWindow.Start.Month() == day.Month() &&
				t.TimeWindow.Start.Day() == day.Day() {
				weeklyPoints[day.Weekday()] += t.StoryPoints
			}
		}
	}

	chartMaxW := w - 16
	if chartMaxW < 10 {
		chartMaxW = 10
	}
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	maxPoints := 1
	for _, wd := range weekdays {
		if weeklyPoints[wd] > maxPoints {
			maxPoints = weeklyPoints[wd]
		}
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("WEEKLY CAPACITY"))
	lines = append(lines, "")

	for idx, wd := range weekdays {
		pts := weeklyPoints[wd]
		barWidth := int(math.Round(float64(pts) * float64(chartMaxW) / float64(maxPoints)))
		bar := strings.Repeat("█", barWidth)
		if bar == "" && pts > 0 {
			bar = "▏"
		}

		isToday := wd == today.Weekday()
		nameColor := m.Theme.Muted
		barColor := m.Theme.Accent
		if isToday {
			nameColor = m.Theme.Fg
			barColor = m.Theme.FocusPurple
		}
		if pts >= 9 {
			barColor = m.Theme.P0Color
		} else if pts == 0 {
			barColor = m.Theme.Muted
		}

		nameStr := lipgloss.NewStyle().Foreground(nameColor).Bold(isToday).Render(weekdayNames[idx])
		barStr := lipgloss.NewStyle().Foreground(barColor).Render(bar)
		ptStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%d SP", pts))
		lines = append(lines, fmt.Sprintf("  %s  │ %s %s", nameStr, barStr, ptStr))
	}

	return strings.Join(lines, "\n")
}


func (m Model) priorityColor(p model.Priority) lipgloss.Color {
	switch p {
	case model.P0:
		return m.Theme.P0Color
	case model.P1:
		return m.Theme.P1Color
	case model.P3:
		return m.Theme.P3Color
	default:
		return m.Theme.P2Color
	}
}

func planned(secs int) string {
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, min)
	}
	return fmt.Sprintf("%dm", min)
}

func elapsed(secs int) string {
	return planned(secs)
}
