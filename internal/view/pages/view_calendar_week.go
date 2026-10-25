package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/modals"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

// RenderWeekView renders the 7-day week schedule view.
func RenderWeekView(m *viewmodel.Model, t theme.Theme, height int) string {
	today := time.Now()
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	_, weekNum := m.SelectedDay.ISOWeek()

	workspaceW := m.Layout.WorkspaceW
	if workspaceW < 40 {
		workspaceW = 40
	}
	availW := workspaceW - 4
	if availW < 14 {
		availW = 14
	}

	isWeekFocused := !m.SidebarFocus
	var titleColor lipgloss.Color
	if isWeekFocused {
		titleColor = t.Accent
	} else {
		titleColor = t.Muted
	}

	weekTitle := formatWeekTitle(weekNum, weekStart, weekEnd, isWeekFocused, availW)
	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(isWeekFocused).
		Align(lipgloss.Center).
		Width(availW)
	renderedTitle := titleStyle.Render(weekTitle)

	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}
	var colRendered []string

	colWidths, _, startCol, endCol := computeWeekColWidths(m)

	laneHeight := height - 4
	if laneHeight < 10 {
		laneHeight = 10
	}

	availLaneH := laneHeight - 2
	if availLaneH < 1 {
		availLaneH = 1
	}

	allDaysLines, maxLinesCount := getWeekViewLines(m, t, colWidths)

	maxScroll := maxLinesCount - availLaneH
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Dynamic clamp in view to support resizing cleanly
	scrollOffset := m.ScrollOffset
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
		m.ScrollOffset = maxScroll // write back to model
	}
	if scrollOffset < 0 {
		scrollOffset = 0
		m.ScrollOffset = 0
	}

	for idx := startCol; idx < endCol; idx++ {
		day := weekStart.AddDate(0, 0, idx)
		isToday := viewmodel.SameDay(day, today)
		isSelected := viewmodel.SameDay(day, m.SelectedDay)
		colWidth := colWidths[idx]

		isFirstScrolled := (idx == startCol && startCol > 0)
		isLastScrolled := (idx == endCol-1 && endCol < 7)

		headerText := formatWeekHeader(weekdayNames[idx], day.Day(), isToday, isFirstScrolled, isLastScrolled, colWidth)

		var headerStyle lipgloss.Style
		var colSepColor lipgloss.Color
		if isToday {
			headerStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Width(colWidth).Align(lipgloss.Center)
			colSepColor = t.Accent
		} else if isSelected {
			headerStyle = lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true).Width(colWidth).Align(lipgloss.Center)
			colSepColor = t.FocusPurple
		} else {
			headerStyle = lipgloss.NewStyle().Foreground(t.Muted).Width(colWidth).Align(lipgloss.Center)
			colSepColor = lipgloss.Color("#45475a")
		}

		lines := allDaysLines[idx]
		var visibleLines []string
		if scrollOffset < len(lines) {
			visibleLines = lines[scrollOffset:]
		}

		if len(visibleLines) > availLaneH {
			visibleLines = visibleLines[:availLaneH]
		}
		for len(visibleLines) < availLaneH {
			visibleLines = append(visibleLines, "")
		}

		var columnRows []string
		columnRows = append(columnRows, headerStyle.Render(headerText))
		columnRows = append(columnRows, lipgloss.NewStyle().Foreground(colSepColor).Render(strings.Repeat("─", colWidth)))
		columnRows = append(columnRows, visibleLines...)

		colBox := lipgloss.NewStyle().
			Width(colWidth).
			Height(laneHeight).
			MaxHeight(laneHeight).
			Render(strings.Join(columnRows, "\n"))

		colRendered = append(colRendered, colBox)
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

func computeWeekColWidths(m *viewmodel.Model) (colWidths []int, visibleCols int, startCol int, endCol int) {
	workspaceW := m.Layout.WorkspaceW
	if workspaceW < 40 {
		workspaceW = 40
	}
	availW := workspaceW - 4
	minColW := 14
	if availW < minColW {
		availW = minColW
	}
	numCols := 7

	visibleCols = (availW + 1) / (minColW + 1)
	if visibleCols < 1 {
		visibleCols = 1
	}
	if visibleCols > numCols {
		visibleCols = numCols
	}

	m.AutoScrollWeekCol()
	startCol = m.WeekScrollColOffset
	endCol = startCol + visibleCols
	if endCol > numCols {
		endCol = numCols
		startCol = endCol - visibleCols
		if startCol < 0 {
			startCol = 0
		}
	}

	// Distribute available width across visible columns to fill 100% of horizontal space
	availForCols := availW - (visibleCols - 1)
	baseColW := availForCols / visibleCols
	remColW := availForCols % visibleCols

	colWidths = make([]int, numCols)
	for i := 0; i < numCols; i++ {
		if i >= startCol && i < endCol {
			visIdx := i - startCol
			w := baseColW
			if visIdx < remColW {
				w++
			}
			if w < minColW {
				w = minColW
			}
			colWidths[i] = w
		} else {
			colWidths[i] = baseColW
			if colWidths[i] < minColW {
				colWidths[i] = minColW
			}
		}
	}
	return colWidths, visibleCols, startCol, endCol
}

func formatWeekHeader(dayName string, dayNum int, isToday bool, isFirstScrolled bool, isLastScrolled bool, maxW int) string {
	scrollPrefix := ""
	if isFirstScrolled {
		scrollPrefix = "◂ "
	}
	scrollSuffix := ""
	if isLastScrolled {
		scrollSuffix = " ▸"
	}

	if isToday {
		opt1 := fmt.Sprintf("%s%s %02d [ACT]%s", scrollPrefix, dayName, dayNum, scrollSuffix)
		if lipgloss.Width(opt1) <= maxW {
			return opt1
		}
		opt1NoScroll := fmt.Sprintf("%s %02d [ACT]", dayName, dayNum)
		if lipgloss.Width(opt1NoScroll) <= maxW {
			return opt1NoScroll
		}

		opt2 := fmt.Sprintf("%s%s %02d*%s", scrollPrefix, dayName, dayNum, scrollSuffix)
		if lipgloss.Width(opt2) <= maxW {
			return opt2
		}
		opt2NoScroll := fmt.Sprintf("%s %02d*", dayName, dayNum)
		if lipgloss.Width(opt2NoScroll) <= maxW {
			return opt2NoScroll
		}

		opt3 := fmt.Sprintf("%s%02d*", string([]rune(dayName)[:1]), dayNum)
		if lipgloss.Width(opt3) <= maxW {
			return opt3
		}

		opt4 := fmt.Sprintf("%02d*", dayNum)
		if lipgloss.Width(opt4) <= maxW {
			return opt4
		}
	} else {
		opt1 := fmt.Sprintf("%s%s %02d%s", scrollPrefix, dayName, dayNum, scrollSuffix)
		if lipgloss.Width(opt1) <= maxW {
			return opt1
		}
		opt1NoScroll := fmt.Sprintf("%s %02d", dayName, dayNum)
		if lipgloss.Width(opt1NoScroll) <= maxW {
			return opt1NoScroll
		}

		opt2 := fmt.Sprintf("%s%02d", string([]rune(dayName)[:1]), dayNum)
		if lipgloss.Width(opt2) <= maxW {
			return opt2
		}

		opt3 := fmt.Sprintf("%02d", dayNum)
		if lipgloss.Width(opt3) <= maxW {
			return opt3
		}
	}

	optEnd := fmt.Sprintf("%02d", dayNum)
	if lipgloss.Width(optEnd) <= maxW {
		return optEnd
	}
	if maxW > 0 && len(optEnd) >= maxW {
		return optEnd[:maxW]
	}
	return ""
}

func formatWeekTitle(weekNum int, weekStart, weekEnd time.Time, isFocused bool, maxW int) string {
	startStr := strings.ToUpper(weekStart.Format("January 2"))
	endStr := strings.ToUpper(weekEnd.Format("January 2, 2006"))

	prefix := "  "
	if isFocused {
		prefix = "● "
	}

	opt1 := fmt.Sprintf("%s◀  WEEK %d (%s - %s)  ▶", prefix, weekNum, startStr, endStr)
	if lipgloss.Width(opt1) <= maxW {
		return opt1
	}

	startShort := strings.ToUpper(weekStart.Format("Jan 02"))
	endShort := strings.ToUpper(weekEnd.Format("Jan 02, 2006"))
	opt2 := fmt.Sprintf("%s◀  WEEK %d (%s - %s)  ▶", prefix, weekNum, startShort, endShort)
	if lipgloss.Width(opt2) <= maxW {
		return opt2
	}

	endShortNoYear := strings.ToUpper(weekEnd.Format("Jan 02"))
	opt3 := fmt.Sprintf("%s◀  WEEK %d (%s - %s)  ▶", prefix, weekNum, startShort, endShortNoYear)
	if lipgloss.Width(opt3) <= maxW {
		return opt3
	}

	opt4 := fmt.Sprintf("%s◀  WEEK %d  ▶", prefix, weekNum)
	if lipgloss.Width(opt4) <= maxW {
		return opt4
	}

	opt5 := fmt.Sprintf("%sW%d", prefix, weekNum)
	if lipgloss.Width(opt5) <= maxW {
		return opt5
	}

	return fmt.Sprintf("W%d", weekNum)
}

func getWeekViewLines(m *viewmodel.Model, t theme.Theme, colWidths []int) ([][]string, int) {
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)

	var allDaysLines [][]string
	maxLinesCount := 0

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		colWidth := 14
		if i < len(colWidths) && colWidths[i] > 0 {
			colWidth = colWidths[i]
		}

		var dayTasks []model.Task
		for _, task := range m.Tasks {
			if model.IsTaskAnchored(task) && viewmodel.SameDay(task.TimeWindow.Start, day) {
				dayTasks = append(dayTasks, task)
			}
		}

		var cardsContent []string
		resolved := viewmodel.ResolveOverlaps(dayTasks)
		if len(resolved) == 0 {
			emptyText := "(No work)"
			if colWidth < 11 {
				emptyText = "(-)"
			}
			cardsContent = append(cardsContent, "", lipgloss.NewStyle().Foreground(t.Muted).Width(colWidth).Align(lipgloss.Center).Render(emptyText))
		} else {
			for idx, rc := range resolved {
				var timeText string
				padV := 0
				padH := 0
				if colWidth >= 16 {
					padH = 1
				}

				dur := rc.Task.TimeWindow.End.Sub(rc.Task.TimeWindow.Start)
				durMins := int(dur.Minutes())
				outerH := (durMins * 6) / 60
				if outerH < 4 {
					outerH = 4
				}
				if outerH >= 6 {
					padV = 1
				}

				cardW := colWidth - 2 - (padH * 2)
				if cardW < 4 {
					cardW = 4
				}

				if rc.Task.IsAllDay {
					timeText = "All Day"
				} else {
					fullTime := fmt.Sprintf("%s-%s", rc.Task.TimeWindow.Start.Format("15:04"), rc.Task.TimeWindow.End.Format("15:04"))
					if lipgloss.Width(fullTime) <= cardW {
						timeText = fullTime
					} else {
						startTime := rc.Task.TimeWindow.Start.Format("15:04")
						if lipgloss.Width(startTime) <= cardW {
							timeText = startTime
						} else {
							timeText = ""
						}
					}
				}

				blockColor := t.PriorityColor(rc.Task.Priority)
				if rc.Task.UUID == m.SelectedTaskUUID {
					blockColor = t.FocusPurple
				}

				timeLine := lipgloss.NewStyle().Foreground(t.Muted).Render(timeText)
				titleText := theme.SentenceCase(rc.Task.Title)
				maxTitleW := cardW
				titleRunes := []rune(titleText)
				if len(titleRunes) > maxTitleW {
					if maxTitleW > 1 {
						titleText = string(titleRunes[:maxTitleW-1]) + "…"
					} else if maxTitleW == 1 {
						titleText = string(titleRunes[:1])
					} else {
						titleText = ""
					}
				}
				titleLine := lipgloss.NewStyle().Foreground(t.Fg).Render(titleText)

				var cardContent string
				if timeText != "" {
					cardContent = timeLine + "\n" + titleLine
				} else {
					cardContent = titleLine
				}

				cardContentH := outerH - 2 - (padV * 2)
				if cardContentH < 1 {
					cardContentH = 1
				}

				cardStr := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(blockColor).
					Padding(padV, padH).
					Width(cardW).
					Height(cardContentH).
					MaxHeight(cardContentH).
					Render(cardContent)

				cardLines := strings.Split(cardStr, "\n")
				if idx > 0 {
					prevTask := resolved[idx-1].Task
					if rc.Task.TimeWindow.Start.Equal(prevTask.TimeWindow.End) {
						// Contiguous - do not add empty line spacing
					} else {
						cardsContent = append(cardsContent, "") // empty line spacing
					}
				}
				cardsContent = append(cardsContent, cardLines...)
			}
		}

		flatContent := strings.Join(cardsContent, "\n")
		lines := strings.Split(flatContent, "\n")
		allDaysLines = append(allDaysLines, lines)
		if len(lines) > maxLinesCount {
			maxLinesCount = len(lines)
		}
	}
	return allDaysLines, maxLinesCount
}

func GetWeekViewMaxScroll(m *viewmodel.Model, t theme.Theme) int {
	cmdPaletteH := 0
	if m.CurrentMode == viewmodel.ModeCommand {
		cmdPaletteStr := modals.RenderCommandPalette(m, t)
		cmdPaletteH = lipgloss.Height(cmdPaletteStr)
	}

	appContentHeight := m.Height - cmdPaletteH - 1
	if appContentHeight < 10 {
		appContentHeight = 10
	}

	height := appContentHeight - 2
	laneHeight := height - 4
	if laneHeight < 10 {
		laneHeight = 10
	}

	availLaneH := laneHeight - 2
	if availLaneH < 1 {
		availLaneH = 1
	}

	colWidths, _, _, _ := computeWeekColWidths(m)
	_, maxLinesCount := getWeekViewLines(m, t, colWidths)

	maxScroll := maxLinesCount - availLaneH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
