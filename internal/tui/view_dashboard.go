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

func (m Model) renderDashboard(height int) string {
	today := time.Now()
	var agendaTasks []model.Task
	var completedCount int
	var plannedFocusSecs int
	var elapsedFocusSecs int

	for _, t := range m.Tasks {
		isTodayOrUpcoming := false
		if t.SchedulingType == model.Anchored {
			isTodayOrUpcoming = t.TimeWindow.Start.Year() == today.Year() &&
				t.TimeWindow.Start.Month() == today.Month() &&
				t.TimeWindow.Start.Day() == today.Day() || t.TimeWindow.Start.After(today)
		} else {
			isTodayOrUpcoming = t.CreatedAt.Year() == today.Year() &&
				t.CreatedAt.Month() == today.Month() &&
				t.CreatedAt.Day() == today.Day() && t.LifecycleState != model.StateCompleted
		}

		if isTodayOrUpcoming {
			agendaTasks = append(agendaTasks, t)
			if t.LifecycleState == model.StateCompleted {
				completedCount++
			}
			plannedFocusSecs += t.StoryPoints * 45 * 60
			elapsedFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
		}
	}

	workspaceWidth := m.Layout.WorkspaceW - 4

	// ── Page Header ──────────────────────────────────────────────
	var headerDate, subDate string
	if !m.SidebarFocus {
		headerDate = lipgloss.NewStyle().
			Foreground(m.Theme.Accent).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(m.Theme.Fg).
			Bold(true).
			Render(today.Format("2006"))
	} else {
		headerDate = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render(today.Format("Monday, January 2"))
		subDate = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render(today.Format("2006"))
	}
	headerLine := headerDate + "  " + subDate

	// ── High-Fidelity Performance Banner ──────────────────────────
	completionPct := 0.0
	if len(agendaTasks) > 0 {
		completionPct = float64(completedCount) / float64(len(agendaTasks)) * 100
	}

	bannerItems := []string{
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("PLANNED"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(planned(plannedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("LOGGED"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(elapsed(elapsedFocusSecs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("DONE"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%d Tasks", completedCount))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("COMPLETION"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", completionPct))),
	}
	bullet := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("   •   ")
	bannerStr := strings.Join(bannerItems, bullet)

	bannerContainer := lipgloss.NewStyle().
		Width(workspaceWidth).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(bannerStr)

	// ── Dual-Column Width Layout ──────────────────────────────────
	leftColW := (workspaceWidth * 6) / 10
	rightColW := workspaceWidth - leftColW

	cardH := height - 7
	if cardH < 8 {
		cardH = 8
	}

	// ── Left Column: Active Agenda Inbox ──────────────────────────
	// Sort: Anchored first by start time, then floating
	sort.Slice(agendaTasks, func(i, j int) bool {
		if agendaTasks[i].SchedulingType == model.Anchored && agendaTasks[j].SchedulingType == model.Anchored {
			return agendaTasks[i].TimeWindow.Start.Before(agendaTasks[j].TimeWindow.Start)
		}
		if agendaTasks[i].SchedulingType == model.Anchored {
			return true
		}
		if agendaTasks[j].SchedulingType == model.Anchored {
			return false
		}
		return agendaTasks[i].CreatedAt.Before(agendaTasks[j].CreatedAt)
	})

	var agendaRows []string
	availLeftW := leftColW - 6
	if availLeftW < 15 {
		availLeftW = 15
	}

	if len(agendaTasks) == 0 {
		agendaRows = append(agendaRows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("No tasks scheduled for today."))
	} else {
		for _, t := range agendaTasks {
			chk := "[ ]"
			if t.LifecycleState == model.StateCompleted {
				chk = "[✓]"
			}

			title := sentenceCase(t.Title)
			dateStr := ""
			if t.SchedulingType == model.Anchored {
				if sameDay(t.TimeWindow.Start, today) {
					dateStr = t.TimeWindow.Start.Format("15:04")
				} else {
					dateStr = t.TimeWindow.Start.Format("Mon Jan _2")
				}
			}

			leftPart := fmt.Sprintf("%s %s", chk, title)
			leftRunes := []rune(leftPart)
			rightRunes := []rune(dateStr)

			if len(leftRunes)+len(rightRunes) > availLeftW {
				maxLeftLen := availLeftW - len(rightRunes) - 1
				if maxLeftLen > 4 {
					leftPart = string(leftRunes[:maxLeftLen-1]) + "…"
				} else {
					leftPart = string(leftRunes[:maxLeftLen])
				}
				leftRunes = []rune(leftPart)
			}

			padSize := availLeftW - len(leftRunes) - len(rightRunes)
			if padSize < 0 {
				padSize = 0
			}

			rowStr := leftPart + strings.Repeat(" ", padSize) + dateStr
			if t.LifecycleState == model.StateCompleted {
				rowStr = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(rowStr)
			} else if t.LifecycleState == model.StateActive {
				rowStr = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(rowStr)
			} else {
				rowStr = lipgloss.NewStyle().Foreground(m.Theme.Fg).Render(rowStr)
			}

			agendaRows = append(agendaRows, rowStr)
		}
	}

	leftCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Width(leftColW - 2).
		Height(cardH).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("⚡ ACTIVE AGENDA INBOX") + "\n\n" +
			strings.Join(agendaRows, "\n"),
		)

	// ── Right Column: Weekly Capacity Utilization ──────────────────
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

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}

	maxPoints := 1
	for _, wd := range weekdays {
		if weeklyPoints[wd] > maxPoints {
			maxPoints = weeklyPoints[wd]
		}
	}

	var rightRows []string
	availRightW := rightColW - 6
	barMaxW := availRightW - 14
	if barMaxW < 8 {
		barMaxW = 8
	}

	for idx, wd := range weekdays {
		pts := weeklyPoints[wd]
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
		rowStyle := lipgloss.NewStyle()
		if isToday {
			// Transparent active row highlight: no background color variables
		}

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

		ptStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%2d SP", pts))
		rowContent := fmt.Sprintf("  %-5s %s  %s", nameStr, barStr, ptStr)

		renderedRow := rowStyle.Width(availRightW).Render(rowContent)
		rightRows = append(rightRows, renderedRow)
	}

	rightCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Width(rightColW - 2).
		Height(cardH).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("📊 WEEKLY CAPACITY UTILIZATION") + "\n\n" +
			strings.Join(rightRows, "\n"),
		)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCard, rightCard)

	var out strings.Builder
	out.WriteString(headerLine + "\n\n")
	out.WriteString(bannerContainer + "\n\n")
	out.WriteString(columns)

	return out.String()
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
