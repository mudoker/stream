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
	workspaceWidth := m.Layout.WorkspaceW - 4
	laneHeight := height - 4
	if laneHeight < 10 {
		laneHeight = 10
	}

	maxLeftW := workspaceWidth - 35
	if maxLeftW < 33 {
		maxLeftW = 33
	}

	innerLeftW := maxLeftW - 6
	colsFit := innerLeftW / 29
	if colsFit < 1 {
		colsFit = 1
	}

	monthGridW := colsFit*29 + (colsFit-1)*2
	leftColWidth := monthGridW + 6
	rightColWidth := workspaceWidth - leftColWidth
	if rightColWidth < 30 {
		rightColWidth = 30
	}

	rowsFit := 1
	if laneHeight > 9 {
		rowsFit = 1 + (laneHeight-9)/10
	}
	if rowsFit < 1 {
		rowsFit = 1
	}

	numMonths := colsFit * rowsFit

	var monthBlocks []string
	startMonth := m.SelectedDay.AddDate(0, m.ScrollOffset, 0)

	for mIdx := 0; mIdx < numMonths; mIdx++ {
		curMonth := startMonth.AddDate(0, mIdx, 0)
		year, month, _ := curMonth.Date()

		firstOfCur := time.Date(year, month, 1, 0, 0, 0, 0, today.Location())
		offset := int(firstOfCur.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		gridStart := firstOfCur.AddDate(0, 0, -offset)

		title := fmt.Sprintf("   %s %d", strings.ToUpper(month.String()), year)

		var gridRows []string
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
				if isSelected {
					if dayNum < 10 {
						valStr = fmt.Sprintf("[%d]", dayNum)
					} else {
						valStr = fmt.Sprintf(" %d", dayNum)
					}
				} else {
					if dayNum < 10 {
						valStr = fmt.Sprintf(" 0%d", dayNum)
					} else {
						valStr = fmt.Sprintf(" %d", dayNum)
					}
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
				} else {
					dayColor = lipgloss.Color("#45475a")
				}

				style := lipgloss.NewStyle().Foreground(dayColor)
				if isToday {
					style = style.Foreground(m.Theme.Accent).Bold(true)
				} else if isSelected {
					style = style.Foreground(m.Theme.FocusPurple).Bold(true)
				}

				rowDays = append(rowDays, style.Render(valStr))
				cellDay = cellDay.AddDate(0, 0, 1)
			}
			gridRows = append(gridRows, "  "+strings.Join(rowDays, " "))
		}
		gridContent := strings.Join(gridRows, "\n")

		titleColor := m.Theme.Accent
		if m.SidebarFocus {
			titleColor = m.Theme.Muted
		}
		monthBlock := fmt.Sprintf(
			"  %s\n  Mo  Tu  We  Th  Fr  Sa  Su\n  ───────────────────────────\n%s",
			lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render(title),
			gridContent,
		)
		monthBlocks = append(monthBlocks, monthBlock)
	}

	var monthRows []string
	for r := 0; r < rowsFit; r++ {
		var rowBlocks []string
		for c := 0; c < colsFit; c++ {
			mIdx := r*colsFit + c
			if mIdx >= len(monthBlocks) {
				break
			}
			if c > 0 {
				rowBlocks = append(rowBlocks, "  ")
			}
			rowBlocks = append(rowBlocks, monthBlocks[mIdx])
		}
		if len(rowBlocks) > 0 {
			monthRows = append(monthRows, lipgloss.JoinHorizontal(lipgloss.Top, rowBlocks...))
		}
	}

	var inspectorLines []string
	var selectedDayTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, m.SelectedDay) {
			selectedDayTasks = append(selectedDayTasks, t)
		}
	}

	availW := rightColWidth - 6
	if availW < 10 {
		availW = 10
	}

	if len(selectedDayTasks) == 0 {
		emptyMsg := "No tasks scheduled for this date"
		paddingTop := (laneHeight - 6) / 2
		if paddingTop < 0 {
			paddingTop = 0
		}
		inspectorLines = append(inspectorLines, strings.Repeat("\n", paddingTop))
		inspectorLines = append(inspectorLines, lipgloss.NewStyle().
			Width(availW).
			Align(lipgloss.Center).
			Foreground(m.Theme.Muted).
			Render(emptyMsg))
	} else {
		for _, t := range selectedDayTasks {
			isSelected := t.UUID == m.SelectedTaskUUID

			pColor := m.priorityColor(t.Priority)
			pStyle := lipgloss.NewStyle().Foreground(pColor).Bold(true)

			bullet := pStyle.Render("●")
			titleText := sentenceCase(t.Title)
			timeRangeText := fmt.Sprintf(" %s - %s", t.TimeWindow.Start.Format("15:04"), t.TimeWindow.End.Format("15:04"))

			timeStyle := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
			titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)

			if isSelected {
				timeStyle = timeStyle.Foreground(lipgloss.Color("#ff8700"))
				titleStyle = titleStyle.Foreground(lipgloss.Color("#ff8700")).Bold(true)
				titleText = "👉 " + titleText
			} else {
				titleStyle = titleStyle.Foreground(pColor)
			}

			timeStyled := bullet + timeStyle.Render(timeRangeText)
			titleStyled := titleStyle.Render("  " + titleText)

			pBadge := pStyle.Render(string(t.Priority))
			spText := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%d SP", t.StoryPoints))
			metaStyled := fmt.Sprintf("  Priority: %s  •  %s", pBadge, spText)

			inspectorLines = append(inspectorLines, timeStyled, titleStyled, metaStyled, "")
		}
	}

	leftPanel := lipgloss.NewStyle().
		Width(leftColWidth - 2).
		Height(laneHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Padding(1, 2).
		Render(
			strings.Join(monthRows, "\n\n"),
		)

	rightPanel := lipgloss.NewStyle().
		Width(rightColWidth - 2).
		Height(laneHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(fmt.Sprintf("📅 SELECTED DATE: %s", strings.ToUpper(m.SelectedDay.Format("Monday, Jun 2, 2006")))) + "\n" +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", availW)) + "\n\n" +
			strings.Join(inspectorLines, "\n"),
		)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}
