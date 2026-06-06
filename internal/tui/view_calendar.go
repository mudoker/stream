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

	// Squeeze day blocks to 3 characters wide, use single-spaced rows, and stack as many as workspace height/width allows.
	maxLeftW := workspaceWidth - 35
	if maxLeftW < 33 {
		maxLeftW = 33 // At least 1 month
	}

	innerLeftW := maxLeftW - 6 // content + padding (4) + borders (2)
	colsFit := innerLeftW / 29
	if colsFit < 1 {
		colsFit = 1
	}

	// Calculate exact layout widths to eliminate gaps!
	monthGridW := colsFit*29 + (colsFit-1)*2
	leftColWidth := monthGridW + 6 // contentW + 4 padding + 2 borders
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

		// Title for this month block
		title := fmt.Sprintf("◀  %s %d  ▶", strings.ToUpper(month.String()), year)
		if mIdx > 0 {
			title = fmt.Sprintf("   %s %d", strings.ToUpper(month.String()), year)
		}

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

				// Format cell contents to be exactly 3 character cells wide
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
				rowBlocks = append(rowBlocks, "  ") // 2 spaces gutter
			}
			rowBlocks = append(rowBlocks, monthBlocks[mIdx])
		}
		if len(rowBlocks) > 0 {
			monthRows = append(monthRows, lipgloss.JoinHorizontal(lipgloss.Top, rowBlocks...))
		}
	}

	// Agenda Inspector (Right Column) tasks list
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

			timeText := fmt.Sprintf("● %s - %s", t.TimeWindow.Start.Format("15:04"), t.TimeWindow.End.Format("15:04"))
			titleText := sentenceCase(t.Title)
			metaText := fmt.Sprintf("Priority: %s • %d SP", t.Priority, t.StoryPoints)

			timeStyle := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
			titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
			metaStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)

			if isSelected {
				timeStyle = timeStyle.Foreground(lipgloss.Color("#ff8700"))
				titleStyle = titleStyle.Foreground(lipgloss.Color("#ff8700")).Bold(true)
				titleText = "👉 " + titleText
			}

			timeStyled := timeStyle.Render(timeText)
			titleStyled := titleStyle.Render("  " + titleText)
			metaStyled := metaStyle.Render("  " + metaText)

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



func (m Model) renderWeekView(height int) string {
	today := time.Now()
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Week title calculation
	_, weekNum := m.SelectedDay.ISOWeek()
	startStr := strings.ToUpper(weekStart.Format("June 2")) // Wait, let's use standard format
	startStr = strings.ToUpper(weekStart.Format("January 2"))
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

	// Split the central canvas width equally into 7 columns (accounting for vertical separators)
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

		// Clean header row
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

		// Horizontal rule divider below header
		dayContent = append(dayContent, lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", colWidth)))

		// Render stacked cards
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

	// Join with vertical separator lines
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
