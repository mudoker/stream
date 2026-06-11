package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/modals"
	"stream/internal/view/theme"

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
	startStr := strings.ToUpper(weekStart.Format("January 2"))
	endStr := strings.ToUpper(weekEnd.Format("January 2, 2006"))

	weekTitle := fmt.Sprintf("◀  WEEK %d (%s - %s)  ▶", weekNum, startStr, endStr)
	titleColor := t.Fg
	if m.SidebarFocus {
		titleColor = t.Muted
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

	availLaneH := laneHeight - 2
	if availLaneH < 1 {
		availLaneH = 1
	}

	allDaysLines, maxLinesCount := getWeekViewLines(m, t, colWidth)

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

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		isToday := viewmodel.SameDay(day, today)
		isSelected := viewmodel.SameDay(day, m.SelectedDay)

		colStyle := lipgloss.NewStyle().Width(colWidth)

		headerText := fmt.Sprintf("%s %02d", weekdayNames[i], day.Day())
		if isToday {
			headerText += " [ACT]"
		}

		var headerStyle lipgloss.Style
		if isToday {
			headerStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
		} else if isSelected {
			headerStyle = lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true)
		} else {
			headerStyle = lipgloss.NewStyle().Foreground(t.Muted)
		}

		lines := allDaysLines[i]
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
		columnRows = append(columnRows, lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", colWidth)))
		columnRows = append(columnRows, visibleLines...)

		colRendered = append(colRendered, colStyle.Render(strings.Join(columnRows, "\n")))
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

func getWeekViewLines(m *viewmodel.Model, t theme.Theme, colWidth int) ([][]string, int) {
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)

	var allDaysLines [][]string
	maxLinesCount := 0

	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		var dayTasks []model.Task
		for _, task := range m.Tasks {
			if task.SchedulingType == model.Anchored && viewmodel.SameDay(task.TimeWindow.Start, day) {
				dayTasks = append(dayTasks, task)
			}
		}

		var cardsContent []string
		resolved := viewmodel.ResolveOverlaps(dayTasks)
		if len(resolved) == 0 {
			cardsContent = append(cardsContent, "", lipgloss.NewStyle().Foreground(t.Muted).Render("  (No work)"))
		} else {
			for _, rc := range resolved {
				timeText := fmt.Sprintf("%s-%s", rc.Task.TimeWindow.Start.Format("15:04"), rc.Task.TimeWindow.End.Format("15:04"))
				blockColor := t.PriorityColor(rc.Task.Priority)
				if rc.Task.UUID == m.SelectedTaskUUID {
					blockColor = t.FocusPurple
				}

				cardW := colWidth - 4
				if cardW < 6 {
					cardW = 6
				}

				timeLine := lipgloss.NewStyle().Foreground(t.Muted).Render(timeText)
				titleText := theme.SentenceCase(rc.Task.Title)
				maxTitleW := cardW - 2
				titleRunes := []rune(titleText)
				if len(titleRunes) > maxTitleW {
					if maxTitleW > 1 {
						titleText = string(titleRunes[:maxTitleW-1]) + "…"
					} else {
						titleText = string(titleRunes[:maxTitleW])
					}
				}
				titleLine := lipgloss.NewStyle().Foreground(t.Fg).Render(titleText)

				cardContent := timeLine + "\n" + titleLine

				cardStr := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(blockColor).
					Padding(1, 1).
					Width(cardW).
					Render(cardContent)

				cardLines := strings.Split(cardStr, "\n")
				cardsContent = append(cardsContent, "") // empty line spacing
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

	contentW := m.Layout.WorkspaceW - 4
	colWidth := (contentW - 6) / 7
	if colWidth < 12 {
		colWidth = 12
	}

	_, maxLinesCount := getWeekViewLines(m, t, colWidth)

	maxScroll := maxLinesCount - availLaneH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
