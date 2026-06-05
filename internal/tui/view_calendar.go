package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderMonthView(height int) string {
	today := time.Now()
	year, month, _ := m.SelectedDay.Date()

	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, today.Location())
	offset := int(firstOfMonth.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	gridStart := firstOfMonth.AddDate(0, 0, -offset)

	var sb strings.Builder
	title := fmt.Sprintf("%s %d", strings.ToUpper(month.String()), year)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("  "+title) + "\n\n")

	sb.WriteString("  Mon    Tue    Wed    Thu    Fri    Sat    Sun\n")
	sb.WriteString("  ─────────────────────────────────────────────\n")

	cellDay := gridStart
	for week := 0; week < 6; week++ {
		var rowDays []string
		for wday := 0; wday < 7; wday++ {
			dayNum := cellDay.Day()
			isCurrentMonth := cellDay.Month() == month
			isToday := cellDay.Year() == today.Year() && cellDay.Month() == today.Month() && cellDay.Day() == today.Day()
			isSelected := cellDay.Year() == m.SelectedDay.Year() && cellDay.Month() == m.SelectedDay.Month() && cellDay.Day() == m.SelectedDay.Day()

			dailySP := 0
			for _, t := range m.Tasks {
				if t.TimeWindow.Start.Year() == cellDay.Year() &&
					t.TimeWindow.Start.Month() == cellDay.Month() &&
					t.TimeWindow.Start.Day() == cellDay.Day() {
					dailySP += t.StoryPoints
				}
			}

			var valStr string
			if isToday {
				valStr = fmt.Sprintf("[%2d]", dayNum)
			} else if isSelected {
				valStr = fmt.Sprintf("❮%2d❯", dayNum)
			} else {
				valStr = fmt.Sprintf(" %2d ", dayNum)
			}

			var dayColor lipgloss.TerminalColor = m.Theme.Muted
			if isCurrentMonth {
				dayColor = m.Theme.Fg
				if dailySP >= 9 {
					dayColor = m.Theme.P0Color
				} else if dailySP >= 6 {
					dayColor = m.Theme.P1Color
				} else if dailySP >= 3 {
					dayColor = m.Theme.SuccessColor
				}
			}

			style := lipgloss.NewStyle().Foreground(dayColor)
			if isToday {
				style = style.Foreground(m.Theme.Accent).Bold(true)
			} else if isSelected {
				style = style.Foreground(m.Theme.FocusPurple).Bold(true)
			}

			rowDays = append(rowDays, fmt.Sprintf("%-6s", style.Render(valStr)))
			cellDay = cellDay.AddDate(0, 0, 1)
		}
		sb.WriteString("  " + strings.Join(rowDays, "") + "\n\n")
	}

	return m.Theme.PanelStyle.
		Width(m.Layout.WorkspaceW - 4).
		Height(height).
		Render(sb.String())
}

func (m Model) renderWeekView(height int) string {
	today := time.Now()
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)

	weekdayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var columns []string

	colWidth := (m.Layout.WorkspaceW - 4) / 7
	if colWidth < 12 {
		colWidth = 12
	}

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		isToday := day.Year() == today.Year() && day.Month() == today.Month() && day.Day() == today.Day()
		isSelected := day.Year() == m.SelectedDay.Year() && day.Month() == m.SelectedDay.Month() && day.Day() == m.SelectedDay.Day()

		var dayTasks []model.Task
		for _, t := range m.Tasks {
			if t.SchedulingType == model.Anchored &&
				t.TimeWindow.Start.Year() == day.Year() &&
				t.TimeWindow.Start.Month() == day.Month() &&
				t.TimeWindow.Start.Day() == day.Day() {
				dayTasks = append(dayTasks, t)
			}
		}

		colStyle := lipgloss.NewStyle().
			Width(colWidth).
			Height(height - 2).
			Background(m.Theme.PanelBg).
			Padding(1, 1)

		if isToday {
			colStyle = colStyle.Background(m.Theme.SelectedBg)
		}

		var dayContent []string
		headerText := fmt.Sprintf("%s %02d/%02d", weekdayNames[i], day.Month(), day.Day())
		if isToday {
			headerText = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)
		} else if isSelected {
			headerText = lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render(headerText)
		} else {
			headerText = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(headerText)
		}
		dayContent = append(dayContent, headerText, "")

		resolved := ResolveOverlaps(dayTasks)
		if len(resolved) == 0 {
			dayContent = append(dayContent, "\nno scheduled work")
		} else {
			for _, rc := range resolved {
				timeText := fmt.Sprintf("%s-%s", rc.Task.TimeWindow.Start.Format("15:04"), rc.Task.TimeWindow.End.Format("15:04"))

				var blockColor lipgloss.Color = m.Theme.P2Color
				if rc.Task.Priority == model.P0 {
					blockColor = m.Theme.P0Color
				} else if rc.Task.Priority == model.P1 {
					blockColor = m.Theme.P1Color
				} else if rc.Task.Priority == model.P3 {
					blockColor = m.Theme.P3Color
				}

				title := strings.ToUpper(rc.Task.Title)
				if len(title) > colWidth-3 {
					title = title[:colWidth-5] + ".."
				}

				block := lipgloss.NewStyle().
					Background(blockColor).
					Foreground(m.Theme.CanvasBg).
					Padding(0, 1).
					Bold(true).
					Render(fmt.Sprintf("%s\n%s", timeText, title))

				dayContent = append(dayContent, "\n"+block)
			}
		}

		columns = append(columns, colStyle.Render(strings.Join(dayContent, "\n")))
	}

	joined := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	lines := strings.Split(joined, "\n")

	if m.ScrollOffset >= len(lines) {
		m.ScrollOffset = len(lines) - 1
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}

	var visibleList []string
	if m.ScrollOffset > 0 {
		visibleList = append(visibleList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("▲ ", m.Width/4)))
	}
	visibleList = append(visibleList, lines[m.ScrollOffset:]...)

	if len(visibleList) > height {
		visibleList = visibleList[:height-1]
		visibleList = append(visibleList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("▼ ", m.Width/4)))
	}

	return strings.Join(visibleList, "\n")
}
