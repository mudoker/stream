package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderWeekView(height int) string {
	today := time.Now()
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	_, weekNum := m.SelectedDay.ISOWeek()
	startStr := strings.ToUpper(weekStart.Format("January 2"))
	endStr := strings.ToUpper(weekEnd.Format("January 2, 2006"))

	weekTitle := fmt.Sprintf("◀  WEEK %d (%s - %s)  ▶", weekNum, startStr, endStr)
	titleColor := m.Theme.Fg
	if m.SidebarFocus {
		titleColor = m.Theme.Muted
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(!m.SidebarFocus).
		Align(lipgloss.Center).
		Width(m.Layout.WorkspaceW - 4)
	renderedTitle := titleStyle.Render(weekTitle)

	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}
	var colRendered []string

	contentW := m.Layout.WorkspaceW - 4
	colWidth := (contentW - 6) / 7
	if colWidth < 12 {
		colWidth = 12
	}

	laneHeight := height - 4
	if laneHeight < 10 {
		laneHeight = 10
	}

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		isToday := sameDay(day, today)
		isSelected := sameDay(day, m.SelectedDay)

		var dayTasks []model.Task
		for _, t := range m.Tasks {
			if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, day) {
				dayTasks = append(dayTasks, t)
			}
		}

		colStyle := lipgloss.NewStyle().
			Width(colWidth).
			Height(laneHeight)

		var dayContent []string

		headerText := fmt.Sprintf("%s %02d", weekdayNames[i], day.Day())
		if isToday {
			headerText += " [ACT]"
		}

		var headerStyle lipgloss.Style
		if isToday {
			headerStyle = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
		} else if isSelected {
			headerStyle = lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Bold(true)
		} else {
			headerStyle = lipgloss.NewStyle().Foreground(m.Theme.Muted)
		}
		dayContent = append(dayContent, headerStyle.Render(headerText))

		dayContent = append(dayContent, lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", colWidth)))

		resolved := ResolveOverlaps(dayTasks)
		if len(resolved) == 0 {
			dayContent = append(dayContent, "", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  (No work)"))
		} else {
			for _, rc := range resolved {
				timeText := fmt.Sprintf("%s-%s", rc.Task.TimeWindow.Start.Format("15:04"), rc.Task.TimeWindow.End.Format("15:04"))
				blockColor := m.getTaskCardColor(rc.Task)
				if rc.Task.UUID == m.SelectedTaskUUID {
					blockColor = lipgloss.Color("#ff8700")
				}

				cardW := colWidth - 4
				if cardW < 6 {
					cardW = 6
				}

				timeLine := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeText)
				titleText := sentenceCase(rc.Task.Title)
				maxTitleW := cardW - 2
				titleRunes := []rune(titleText)
				if len(titleRunes) > maxTitleW {
					if maxTitleW > 1 {
						titleText = string(titleRunes[:maxTitleW-1]) + "…"
					} else {
						titleText = string(titleRunes[:maxTitleW])
					}
				}
				titleLine := lipgloss.NewStyle().Foreground(m.Theme.Fg).Render(titleText)

				cardContent := timeLine + "\n" + titleLine

				cardStr := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(blockColor).
					Padding(1, 1).
					Width(cardW).
					Render(cardContent)

				dayContent = append(dayContent, "", cardStr)
			}
		}

		colRendered = append(colRendered, colStyle.Render(strings.Join(dayContent, "\n")))
	}

	var sepLines []string
	for h := 0; h < laneHeight; h++ {
		sepLines = append(sepLines, "│")
	}
	sepStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#45475a")).
		Render(strings.Join(sepLines, "\n"))

	var colsToJoin []string
	for idx, col := range colRendered {
		if idx > 0 {
			colsToJoin = append(colsToJoin, sepStr)
		}
		colsToJoin = append(colsToJoin, col)
	}

	joinedLanes := lipgloss.JoinHorizontal(lipgloss.Top, colsToJoin...)

	var out strings.Builder
	out.WriteString(renderedTitle + "\n\n")
	out.WriteString(joinedLanes)

	return out.String()
}

func (m Model) getTaskCardColor(t model.Task) lipgloss.Color {
	switch t.Priority {
	case model.P0:
		return m.Theme.P0Color
	case model.P1:
		return m.Theme.P1Color
	default:
		return m.Theme.P2Color
	}
}
