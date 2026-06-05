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

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing workspace..."
	}

	// 1. Zen Mode takes over the workspace only if in active fullscreen focus
	if m.CurrentMode == ModeZen {
		return m.renderZenMode()
	}

	// 2. Calculate dynamic column dimensions
	sidebarWidth := int(float64(m.Width) * 0.13)
	if sidebarWidth < 18 {
		sidebarWidth = 18
	} else if sidebarWidth > 26 {
		sidebarWidth = 26
	}
	sidebarContentWidth := sidebarWidth - 2
	if sidebarContentWidth < 10 {
		sidebarContentWidth = 10
	}

	workspaceWidth := m.Width - sidebarWidth - 3
	if workspaceWidth < 30 {
		workspaceWidth = 30
	}
	workspaceHeight := m.Height

	// 3. Build the left Arc-style Sidebar
	sidebar := m.renderArcSidebar(sidebarContentWidth)

	var content string
	switch m.CurrentView {
	case DashboardView:
		content = m.renderDashboard(workspaceHeight)
	case MonthView:
		content = m.renderMonthView(workspaceHeight)
	case WeekView:
		content = m.renderWeekView(workspaceHeight)
	case DayView:
		content = m.renderDayView(workspaceHeight)
	case AnalyticsView:
		content = m.renderAnalyticsView(workspaceHeight)
	}

	// 4. Overlay mini Zen Mode in top-right of workspace content
	if m.ZenTimer != nil && m.ZenTimer.Running {
		content = m.overlayMiniZen(content, workspaceWidth)
	}

	// 5. Join Left Arc Sidebar and Right Workspace Content
	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(workspaceHeight).
		Background(m.Theme.PanelBg).
		Padding(1, 1)

	workspaceStyle := lipgloss.NewStyle().
		Width(workspaceWidth).
		Height(workspaceHeight).
		Background(m.Theme.CanvasBg)

	canvas := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebarStyle.Render(sidebar),
		"   ",
		workspaceStyle.Render(content),
	)

	// Command Palette Overlay at the bottom
	if m.CurrentMode == ModeCommand {
		cmdPalette := m.renderCommandPalette()
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdPalette)
	}

	// Centered floating modal overlay handling over the entire canvas (sidebar + content)
	if m.CurrentMode == ModeForm || m.PromptOpen || m.ReviewOpen || m.HelpOpen || m.DetailOpen {
		var modalStr string
		if m.CurrentMode == ModeForm {
			modalStr = m.renderFormModal()
		} else if m.PromptOpen {
			modalStr = m.renderPromptModal()
		} else if m.ReviewOpen {
			modalStr = m.renderReviewModal()
		} else if m.HelpOpen {
			modalStr = m.renderHelpModal()
		} else if m.DetailOpen {
			modalStr = m.renderDetailModal()
		}

		modalWidth := lipgloss.Width(modalStr)
		modalHeight := lipgloss.Height(modalStr)

		paddingTop := (m.Height - modalHeight) / 2
		if paddingTop < 0 {
			paddingTop = 0
		}
		paddingLeft := (m.Width - modalWidth) / 2
		if paddingLeft < 0 {
			paddingLeft = 0
		}

		canvas = overlayString(canvas, modalStr, paddingLeft, paddingTop, m.Width)
	}

	return canvas
}

func (m Model) renderArcSidebar(width int) string {
	var sb []string

	// 1. Logo
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Bold(true).
		Render("▲  s t r e a m"))
	sb = append(sb, "")

	// 2. Navigation Spaces (Tabs)
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Muted).
		Bold(true).
		Padding(0, 2).
		Render("SPACES"))

	viewNames := []string{"dashboard", "month grid", "week lanes", "day timeline", "analytics"}
	for i, name := range viewNames {
		if int(m.CurrentView) == i {
			activeBorder := lipgloss.Border{Left: "┃"}
			activeStyle := lipgloss.NewStyle().
				Background(m.Theme.SelectedBg).
				Foreground(m.Theme.Accent).
				Bold(true).
				Border(activeBorder, false, false, false, true).
				BorderForeground(m.Theme.Accent).
				Padding(0, 1).
				Width(width - 1)
			sb = append(sb, activeStyle.Render(strings.ToUpper(name)))
		} else {
			inactiveStyle := lipgloss.NewStyle().
				Foreground(m.Theme.Muted).
				Padding(0, 2). // Align text with active tab (border width 1 + padding 1)
				Width(width)
			sb = append(sb, inactiveStyle.Render(strings.ToUpper(name)))
		}
	}

	sb = append(sb, "")

	// 3. Fill spacing dynamically to push footer elements down
	occupiedRows := len(sb) + 2
	remainingRows := m.Height - occupiedRows - 4
	if remainingRows > 0 {
		sb = append(sb, strings.Repeat("\n", remainingRows))
	}

	// 4. Sidebar Status Utilities (Mode, GCal sync, time)
	syncColor := m.Theme.Muted
	if m.Sync.IsOnline() {
		syncColor = m.Theme.SuccessColor
	}
	gcalBadge := lipgloss.NewStyle().Foreground(syncColor).Render("● gcal")

	modeBadge := lipgloss.NewStyle().
		Foreground(m.Theme.FocusPurple).
		Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))

	timeStr := time.Now().Format("15:04")

	sb = append(sb, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", width)))
	sb = append(sb, modeBadge+"  •  "+gcalBadge)
	sb = append(sb, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeStr))

	return strings.Join(sb, "\n")
}

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

	rightWidth := m.Width - 48
	if rightWidth < 20 {
		rightWidth = 20
	}

	chartWidget := m.Theme.PanelStyle.
		Width(rightWidth).
		Height(12).
		Render(strings.Join(chartLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "   ", chartWidget)
}

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
		Width(m.Width - 28).
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

	colWidth := (m.Width - 32) / 7
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

func (m Model) renderDayView(height int) string {
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

	// 75% Timeline, 25% Todo Shelf
	timelineWidth := int(float64(workspaceWidth) * 0.75)
	shelfWidth := workspaceWidth - timelineWidth - 4
	if timelineWidth < 30 {
		timelineWidth = 30
	}
	if shelfWidth < 20 {
		shelfWidth = 20
	}

	var anchoredTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored &&
			t.TimeWindow.Start.Year() == m.SelectedDay.Year() &&
			t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
			t.TimeWindow.Start.Day() == m.SelectedDay.Day() {
			anchoredTasks = append(anchoredTasks, t)
		}
	}

	cols := ResolveOverlaps(anchoredTasks)

	now := time.Now()
	isToday := m.SelectedDay.Year() == now.Year() && m.SelectedDay.Month() == now.Month() && m.SelectedDay.Day() == now.Day()

	// Content area width for the timeline grid (excluding 6 char timestamp)
	W := timelineWidth - 6
	if W < 10 {
		W = 10
	}

	// Calculate 24 hours * 4 rows/hour = 96 rows total
	var timelineLines []string
	
	// Add title header
	headerText := fmt.Sprintf("DAILY TIMELINE  /  %s", strings.ToUpper(m.SelectedDay.Format("Monday, Jan _2")))
	timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)+"\n")

	// Calculate "NOW" indicator row index
	nowRow := -1
	if isToday {
		nowRow = (now.Hour() * 4) + (now.Minute() / 15)
	}

	// Render the 96 rows (00:00 to 23:45)
	for r := 0; r < 96; r++ {
		// Find all active tasks overlapping row r
		type ActiveTaskCol struct {
			ColIndex int
			TotalCol int
			Task     model.Task
		}
		var activeTasks []ActiveTaskCol
		for _, rc := range cols {
			startRow := (rc.Task.TimeWindow.Start.Hour() * 4) + (rc.Task.TimeWindow.Start.Minute() / 15)
			endRow := (rc.Task.TimeWindow.End.Hour() * 4) + (rc.Task.TimeWindow.End.Minute() / 15)
			if r >= startRow && r < endRow {
				activeTasks = append(activeTasks, ActiveTaskCol{
					ColIndex: rc.ColIndex,
					TotalCol: rc.TotalCol,
					Task:     rc.Task,
				})
			}
		}

		// Partition W into segments
		type RowSegment struct {
			Start int
			End   int
			Text  string
		}
		var segments []RowSegment

		for _, at := range activeTasks {
			startRow := (at.Task.TimeWindow.Start.Hour() * 4) + (at.Task.TimeWindow.Start.Minute() / 15)
			endRow := (at.Task.TimeWindow.End.Hour() * 4) + (at.Task.TimeWindow.End.Minute() / 15)
			h := endRow - startRow

			colStart := (at.ColIndex * W) / at.TotalCol
			colEnd := ((at.ColIndex + 1) * W) / at.TotalCol
			w := colEnd - colStart

			// Check if this task is active NOW (machine timestamp matches its window)
			isActiveBlock := isToday && now.After(at.Task.TimeWindow.Start) && now.Before(at.Task.TimeWindow.End)

			// Render the segment of the task card at line relative to startRow
			isNowRow := (r == nowRow)
			isLeftmost := (colStart == 0)
			lineText := m.renderTaskCardLine(at.Task, w, h, r-startRow, isActiveBlock, isNowRow, isLeftmost)
			segments = append(segments, RowSegment{Start: colStart, End: colEnd, Text: lineText})
		}

		// Sort segments by Start position
		sort.Slice(segments, func(i, j int) bool {
			return segments[i].Start < segments[j].Start
		})

		// Fill in the gaps with guides/empty space
		var rowParts []string
		curr := 0
		for _, seg := range segments {
			if seg.Start > curr {
				gapW := seg.Start - curr
				var gapText string
				if r == nowRow {
					if curr == 0 {
						badge := getNowBadge(gapW, now)
						gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(badge + strings.Repeat("─", gapW-len(badge)))
					} else {
						gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", gapW))
					}
				} else if r%4 == 0 {
					gapText = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", gapW))
				} else {
					gapText = strings.Repeat(" ", gapW)
				}
				rowParts = append(rowParts, gapText)
			}
			rowParts = append(rowParts, seg.Text)
			curr = seg.End
		}
		if curr < W {
			gapW := W - curr
			var gapText string
			if r == nowRow {
				if curr == 0 {
					badge := getNowBadge(gapW, now)
					gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(badge + strings.Repeat("─", gapW-len(badge)))
				} else {
					gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", gapW))
				}
			} else if r%4 == 0 {
				gapText = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", gapW))
			} else {
				gapText = strings.Repeat(" ", gapW)
			}
			rowParts = append(rowParts, gapText)
		}

		// Clean left-aligned timestamp (removing vertical │)
		var hourLabel string
		isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == (r/4) && (r%4 == 0)
		if r == nowRow {
			timeLabel := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
			hourLabel = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(timeLabel) + " "
		} else if r%4 == 0 {
			timeLabel := fmt.Sprintf("%02d:00", r/4)
			if isSelectedHour {
				hourLabel = lipgloss.NewStyle().Background(m.Theme.Accent).Foreground(m.Theme.CanvasBg).Bold(true).Render(timeLabel) + " "
			} else {
				hourLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeLabel) + " "
			}
		} else {
			hourLabel = "      "
		}

		timelineLines = append(timelineLines, hourLabel+strings.Join(rowParts, ""))
	}

	// Scroll timeline dynamically centering around m.TimelineHour
	timelineStartRow := 8 * 4 // Default start at 8:00 AM
	maxRowsVisible := height - 4
	if maxRowsVisible < 10 {
		maxRowsVisible = 10
	}
	
	// Center around m.TimelineHour
	targetCenterRow := m.TimelineHour * 4
	timelineStartRow = targetCenterRow - maxRowsVisible/2
	if timelineStartRow < 0 {
		timelineStartRow = 0
	}
	timelineEndRow := timelineStartRow + maxRowsVisible - 1
	if timelineEndRow > 95 {
		timelineEndRow = 95
		timelineStartRow = timelineEndRow - maxRowsVisible + 1
		if timelineStartRow < 0 {
			timelineStartRow = 0
		}
	}

	var visibleTimelineLines []string
	visibleTimelineLines = append(visibleTimelineLines, timelineLines[0]) // Header
	if timelineStartRow > 0 {
		visibleTimelineLines = append(visibleTimelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("      ▲  (scroll up)"))
	}
	
	for r := timelineStartRow; r <= timelineEndRow; r++ {
		visibleTimelineLines = append(visibleTimelineLines, timelineLines[r+1])
	}

	if timelineEndRow < 95 {
		visibleTimelineLines = append(visibleTimelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("      ▼  (scroll down)"))
	}

	leftBox := m.Theme.PanelStyle.
		Width(timelineWidth - 4).
		Height(height - 2).
		Render(strings.Join(visibleTimelineLines, "\n"))

	// Todo Shelf (Command Center)
	var shelfLines []string
	shelfLines = append(shelfLines, "TODO BACKLOG\n")

	shelfTasks := m.getTodoShelfTasks()

	priorities := []model.Priority{model.P0, model.P1, model.P2, model.P3}
	pTitles := []string{"P0 URGENT", "P1 HIGH", "P2 MEDIUM", "P3 LOW"}
	pColors := []lipgloss.Color{m.Theme.P0Color, m.Theme.P1Color, m.Theme.P2Color, m.Theme.P3Color}

	for idx, prio := range priorities {
		var list []model.Task
		for _, t := range shelfTasks {
			if t.Priority == prio {
				list = append(list, t)
			}
		}

		if len(list) == 0 && prio != model.P0 {
			continue
		}

		header := lipgloss.NewStyle().Foreground(pColors[idx]).Bold(true).Render(pTitles[idx])
		shelfLines = append(shelfLines, "\n"+header)

		if len(list) == 0 {
			shelfLines = append(shelfLines, "  ● no backlog items")
			continue
		}

		for _, t := range list {
			isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID
			bullet := lipgloss.NewStyle().Foreground(pColors[idx]).Render("●")

			title := sentenceCase(t.Title)
			if len(title) > shelfWidth-14 {
				title = title[:shelfWidth-16] + ".."
			}

			line := fmt.Sprintf("  %s %s", bullet, title)
			if isSelected {
				line = lipgloss.NewStyle().
					Background(m.Theme.SelectedBg).
					Foreground(m.Theme.FocusPurple).
					Bold(true).
					Padding(0, 1).
					Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(m.Theme.Fg).Render(line)
			}
			shelfLines = append(shelfLines, line)
		}
	}

	// Apply Shelf scroll offset
	shelfLinesRendered := strings.Join(shelfLines, "\n")
	shelfLinesList := strings.Split(shelfLinesRendered, "\n")

	if m.ShelfScrollOffset >= len(shelfLinesList) {
		m.ShelfScrollOffset = len(shelfLinesList) - 1
	}
	if m.ShelfScrollOffset < 0 {
		m.ShelfScrollOffset = 0
	}

	var visibleShelfList []string
	if m.ShelfScrollOffset > 0 {
		visibleShelfList = append(visibleShelfList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▲  (scroll up)"))
	}
	visibleShelfList = append(visibleShelfList, shelfLinesList[m.ShelfScrollOffset:]...)

	if len(visibleShelfList) > height-2 {
		visibleShelfList = visibleShelfList[:height-2]
		visibleShelfList = append(visibleShelfList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▼  (scroll down)"))
	}

	rightBox := m.Theme.PanelStyle.
		Width(shelfWidth - 4).
		Height(height - 2).
		Render(strings.Join(visibleShelfList, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "    ", rightBox)
}

func (m Model) renderTaskCardLine(task model.Task, w int, h int, lineIdx int, isActive bool, isNowRow bool, isLeftmost bool) string {
	var pBarColor lipgloss.Color = m.Theme.P2Color
	if task.Priority == model.P0 {
		pBarColor = m.Theme.P0Color
	} else if task.Priority == model.P1 {
		pBarColor = m.Theme.P1Color
	} else if task.Priority == model.P3 {
		pBarColor = m.Theme.P3Color
	}

	bgStyle := lipgloss.NewStyle().Background(m.Theme.PanelBg).Foreground(m.Theme.Fg)
	if isActive {
		bgStyle = bgStyle.Background(m.Theme.SelectedBg)
	}

	timeStr := fmt.Sprintf("%s–%s", task.TimeWindow.Start.Format("15:04"), task.TimeWindow.End.Format("15:04"))
	now := time.Now()
	if isActive {
		remaining := task.TimeWindow.End.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		hVal := int(remaining.Hours())
		mVal := int(remaining.Minutes()) % 60
		sVal := int(remaining.Seconds()) % 60
		timeStr = fmt.Sprintf("%02d:%02d:%02d Remaining", hVal, mVal, sVal)
	}

	// Handle short task blocks (h < 4) using the borderless side-strip block style
	if h < 4 {
		var lineText string
		if isNowRow {
			contentW := w - 1
			if contentW < 1 {
				contentW = 1
			}
			if isLeftmost {
				badge := getNowBadge(contentW, now)
				lineText = badge + strings.Repeat("─", contentW-len(badge))
			} else {
				lineText = strings.Repeat("─", contentW)
			}
			leftStrip := lipgloss.NewStyle().Foreground(pBarColor).Background(bgStyle.GetBackground()).Render("┃")
			return leftStrip + lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Background(bgStyle.GetBackground()).Bold(true).Render(lineText)
		}

		if h == 1 {
			// Single row representation: sentence-case Title (Priority)
			lineText = fmt.Sprintf(" %s (%s)", sentenceCase(task.Title), task.Priority)
		} else if h == 2 {
			if lineIdx == 0 {
				lineText = " " + sentenceCase(task.Title)
			} else {
				lineText = fmt.Sprintf(" ▲ %s • %d SP", task.Priority, task.StoryPoints)
			}
		} else { // h == 3
			if lineIdx == 0 {
				lineText = " " + sentenceCase(task.Title)
			} else if lineIdx == 1 {
				lineText = fmt.Sprintf(" ▲ %s • %d SP", task.Priority, task.StoryPoints)
			} else {
				lineText = " " + timeStr
			}
		}

		contentW := w - 1
		if contentW < 1 {
			contentW = 1
		}
		if len(lineText) > contentW {
			if contentW > 3 {
				lineText = lineText[:contentW-3] + "..."
			} else {
				lineText = lineText[:contentW]
			}
		} else {
			lineText = lineText + strings.Repeat(" ", contentW-len(lineText))
		}

		leftStrip := lipgloss.NewStyle().Foreground(pBarColor).Background(bgStyle.GetBackground()).Render("┃")
		return leftStrip + bgStyle.Render(lineText)
	}

	// Standardize task blocks as solid layout cards with custom borders (h >= 4)
	customBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "┃",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	cardStyle := lipgloss.NewStyle().
		Border(customBorder).
		BorderForeground(pBarColor).
		Background(bgStyle.GetBackground()).
		Width(w).
		Height(h)

	// Build card content lines
	var content []string
	content = append(content, sentenceCase(task.Title))

	// Active block real-time progress countdown pulse
	if isActive {
		timerText := "● ACTIVE"
		content = append(content, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(timerText))
	} else {
		content = append(content, "")
	}

	// Metadata grouping inline at the bottom
	pColorName := "P2"
	if task.Priority == model.P0 {
		pColorName = "P0"
	} else if task.Priority == model.P1 {
		pColorName = "P1"
	} else if task.Priority == model.P3 {
		pColorName = "P3"
	}
	metaRow := fmt.Sprintf("▲ %s • %d SP • %s",
		pColorName,
		task.StoryPoints,
		timeStr,
	)
	content = append(content, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(metaRow))

	// Join content and pad/border render
	var visibleContent []string
	maxLines := h - 2
	for i := 0; i < maxLines; i++ {
		if isNowRow && i == lineIdx-1 {
			innerWidth := w - 2
			if innerWidth < 1 {
				innerWidth = 1
			}
			var lineText string
			if isLeftmost {
				badge := getNowBadge(innerWidth, now)
				lineText = badge + strings.Repeat("─", innerWidth-len(badge))
			} else {
				lineText = strings.Repeat("─", innerWidth)
			}
			visibleContent = append(visibleContent, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(lineText))
		} else {
			val := ""
			if i < len(content) {
				val = content[i]
			}
			visibleContent = append(visibleContent, val)
		}
	}

	rendered := cardStyle.Render(strings.Join(visibleContent, "\n"))
	renderedLines := strings.Split(rendered, "\n")
	if lineIdx < len(renderedLines) {
		return renderedLines[lineIdx]
	}
	return strings.Repeat(" ", w)
}

func sentenceCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m Model) renderZenMode() string {
	if m.ZenTimer == nil {
		return "No focus session timer."
	}

	t := m.ZenTimer.Task
	sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]

	var sb []string
	sb = append(sb, "")
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(strings.ToUpper(t.Title)))

	pBadge := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("▲ " + string(t.Priority))
	spBadge := fmt.Sprintf("• %d SP", t.StoryPoints)
	sb = append(sb, lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(fmt.Sprintf("%s %s", pBadge, spBadge)))

	sb = append(sb, "")

	clockStr := RenderLargeTime(m.ZenTimer.TimeRemaining)
	clockBox := lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(clockStr)
	sb = append(sb, clockBox)

	sb = append(sb, "")
	sessInfo := fmt.Sprintf("[ Session %d / %d: %s ]", m.ZenTimer.CurrentSessionIdx+1, len(m.ZenTimer.Sessions), strings.ToUpper(string(sess.Type)))
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.SuccessColor).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(sessInfo))

	sb = append(sb, "")

	pct := 1.0 - (m.ZenTimer.TimeRemaining.Seconds() / sess.Duration.Seconds())
	barWidth := int(float64(m.Width) * 0.70)
	if barWidth < 20 {
		barWidth = 20
	}
	progBar := RenderProgressBar(barWidth, pct)
	sb = append(sb, lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(fmt.Sprintf("%s %d%%", progBar, int(pct*100))))

	sb = append(sb, "\n")
	instructions := "space pause/resume   + add 5m   b skip block   esc exit focus"
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Muted).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(instructions))

	return strings.Join(sb, "\n")
}

func (m Model) renderAnalyticsView(height int) string {
	today := time.Now()

	completionsByDate := make(map[string]bool)
	var totalFocusSecs int
	var totalInterruptions int
	var completedCount int
	var totalCount int
	var workSecs int
	var personalSecs int
	var effectiveSessions int
	var completedWithNoInterruptionCount int

	// Tag duration tracking
	tagSecs := make(map[string]int)

	for _, t := range m.Tasks {
		totalCount++
		if t.LifecycleState == model.StateCompleted {
			completedCount++
			dateStr := t.UpdatedAt.Format("2006-01-02")
			completionsByDate[dateStr] = true

			if t.ExecutionMetrics.InterruptionCount == 0 {
				completedWithNoInterruptionCount++
			}

			isPersonal := false
			for _, tag := range t.Tags {
				if strings.ToLower(tag) == "personal" {
					isPersonal = true
					break
				}
			}
			if strings.Contains(strings.ToLower(t.Title), "personal") || strings.Contains(strings.ToLower(t.Description), "personal") {
				isPersonal = true
			}

			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && t.SchedulingType == model.Anchored {
				dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}

			if isPersonal {
				personalSecs += dur
			} else {
				workSecs += dur
			}

			effectiveSessions += t.ExecutionMetrics.TotalCompletedPomodoros

			// Track tags
			for _, tag := range t.Tags {
				normalized := strings.TrimSpace(tag)
				if normalized != "" {
					tagSecs[normalized] += dur
				}
			}
		}
		totalFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
		totalInterruptions += t.ExecutionMetrics.InterruptionCount
	}

	// Calculate Streak
	streak := 0
	checkDate := today
	todayStr := today.Format("2006-01-02")
	if completionsByDate[todayStr] {
		for {
			dateStr := checkDate.Format("2006-01-02")
			if completionsByDate[dateStr] {
				streak++
				checkDate = checkDate.AddDate(0, 0, -1)
			} else {
				break
			}
		}
	} else {
		yesterday := today.AddDate(0, 0, -1)
		yesterdayStr := yesterday.Format("2006-01-02")
		if completionsByDate[yesterdayStr] {
			checkDate = yesterday
			for {
				dateStr := checkDate.Format("2006-01-02")
				if completionsByDate[dateStr] {
					streak++
					checkDate = checkDate.AddDate(0, 0, -1)
				} else {
					break
				}
			}
		}
	}

	rate := 0.0
	if totalCount > 0 {
		rate = float64(completedCount) / float64(totalCount) * 100
	}

	workHrs := float64(workSecs) / 3600.0
	personalHrs := float64(personalSecs) / 3600.0
	totalHrs := workHrs + personalHrs

	// 1. Top KPI Row Cards
	cardStyle := lipgloss.NewStyle().
		Background(m.Theme.PanelBg).
		Padding(0, 2).
		Height(3)

	card1 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("STREAK"),
		lipgloss.NewStyle().Foreground(m.Theme.P1Color).Bold(true).Render(fmt.Sprintf("🔥 %d DAYS", streak)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Consecutive"),
	))

	card2 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("SESSIONS"),
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(fmt.Sprintf("🎯 %d BLOCKS", effectiveSessions)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Pomodoros"),
	))

	card3 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("FOCUS TIME"),
		lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(fmt.Sprintf("⏱️ %.1f HRS", totalHrs)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Work + Pers"),
	))

	workspaceWidth := m.Width - 25
	if workspaceWidth < 40 {
		workspaceWidth = 40
	}

	var kpiRow string
	if workspaceWidth >= 60 {
		kpiRow = lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3)
	} else {
		kpiRow = lipgloss.JoinVertical(lipgloss.Left, card1, "\n", card2, "\n", card3)
	}

	// 2. Left Column: Daily timeline & focus purity
	var timelineLines []string
	timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("DAILY PRODUCTIVITY TIMELINE"))

	// Compute last 7 days daily focus
	maxDaySecs := 0
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -6+i)
		daysList[i] = day
		daySecs := 0
		for _, t := range m.Tasks {
			if t.LifecycleState == model.StateCompleted &&
				t.UpdatedAt.Year() == day.Year() && t.UpdatedAt.Month() == day.Month() && t.UpdatedAt.Day() == day.Day() {
				dur := t.ExecutionMetrics.ElapsedFocusSeconds
				if dur == 0 && t.SchedulingType == model.Anchored {
					dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
				} else if dur == 0 {
					dur = t.StoryPoints * 45 * 60
				}
				daySecs += dur
			}
		}
		daySecsList[i] = daySecs
		if daySecs > maxDaySecs {
			maxDaySecs = daySecs
		}
	}
	if maxDaySecs == 0 {
		maxDaySecs = 1
	}

	timelineBarWidth := 15
	for i := 0; i < 7; i++ {
		day := daysList[i]
		daySecs := daySecsList[i]
		dayHrs := float64(daySecs) / 3600.0

		pct := float64(daySecs) / float64(maxDaySecs)
		barLen := int(math.Round(pct * float64(timelineBarWidth)))
		barStr := strings.Repeat("█", barLen)
		if barStr == "" && dayHrs > 0 {
			barStr = "▏"
		}

		var barColor lipgloss.Color
		if dayHrs == 0 {
			barColor = m.Theme.Muted
		} else if dayHrs <= 2.0 {
			barColor = m.Theme.P2Color
		} else if dayHrs <= 5.0 {
			barColor = m.Theme.Accent
		} else {
			barColor = m.Theme.SuccessColor
		}

		coloredBar := lipgloss.NewStyle().Foreground(barColor).Render(barStr)
		timelineLines = append(timelineLines, fmt.Sprintf("  %s │ %s %.1fh", day.Format("Jan _2"), coloredBar, dayHrs))
	}

	purityPct := 0.0
	if completedCount > 0 {
		purityPct = (float64(completedWithNoInterruptionCount) / float64(completedCount)) * 100
	}

	var statsLines []string
	statsLines = append(statsLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("FOCUS HEALTH & DISTRACTION"))
	statsLines = append(statsLines, fmt.Sprintf("  Purity (No Interrupt): %.1f%%", purityPct))
	statsLines = append(statsLines, fmt.Sprintf("  Total Focus Logged:    %s", time.Duration(totalFocusSecs)*time.Second))
	statsLines = append(statsLines, fmt.Sprintf("  Total Interruptions:   %d times", totalInterruptions))
	statsLines = append(statsLines, fmt.Sprintf("  Completed Tasks Rate:  %d/%d (%.1f%%)", completedCount, totalCount, rate))

	// 3. Right Column: Time distribution ratios & Top tags & 30-Day Trend
	var ratioLines []string
	ratioLines = append(ratioLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TIME ALLOCATION RATIOS"))

	totalHrsForRatio := totalHrs
	if totalHrsForRatio == 0 {
		totalHrsForRatio = 1.0
	}
	workPct := workHrs / totalHrsForRatio
	personalPct := personalHrs / totalHrsForRatio

	ratioBarWidth := 20
	var coloredRatioBar string
	if totalHrs == 0 {
		coloredRatioBar = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("·", ratioBarWidth))
	} else {
		workBarLen := int(math.Round(workPct * float64(ratioBarWidth)))
		persBarLen := ratioBarWidth - workBarLen
		if workBarLen == 0 && workHrs > 0 {
			workBarLen = 1
		}
		if persBarLen == 0 && personalHrs > 0 {
			persBarLen = 1
		}
		
		// Adjust back to ratioBarWidth
		if workBarLen + persBarLen > ratioBarWidth {
			if workBarLen > persBarLen {
				workBarLen = ratioBarWidth - persBarLen
			} else {
				persBarLen = ratioBarWidth - workBarLen
			}
		}

		workBarStr := strings.Repeat("█", workBarLen)
		persBarStr := strings.Repeat("█", persBarLen)
		coloredRatioBar = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(workBarStr) +
			lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(persBarStr)
	}

	ratioLines = append(ratioLines, fmt.Sprintf("  %s", coloredRatioBar))
	ratioLines = append(ratioLines, fmt.Sprintf("  Work Focus     %.0f%% (%.1fh)", workPct*100, workHrs))
	ratioLines = append(ratioLines, fmt.Sprintf("  Personal Focus %.0f%% (%.1fh)", personalPct*100, personalHrs))

	// Top Tags
	type TagVal struct {
		Tag  string
		Secs int
	}
	var tags []TagVal
	for k, v := range tagSecs {
		tags = append(tags, TagVal{Tag: k, Secs: v})
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Secs > tags[j].Secs
	})

	var tagLines []string
	tagLines = append(tagLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TOP TAGS BY FOCUS TIME"))
	if len(tags) == 0 {
		tagLines = append(tagLines, "  No tagged sessions logged.")
	} else {
		maxSecs := tags[0].Secs
		if maxSecs == 0 {
			maxSecs = 1
		}
		for idx, tv := range tags {
			if idx >= 3 {
				break
			}
			hrs := float64(tv.Secs) / 3600.0
			pct := float64(tv.Secs) / float64(maxSecs)
			barLen := int(math.Round(pct * 12))
			barStr := strings.Repeat("█", barLen)
			if barStr == "" && tv.Secs > 0 {
				barStr = "▏"
			}
			coloredBar := lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render(barStr)

			tagName := tv.Tag
			if len(tagName) > 8 {
				tagName = tagName[:7] + "…"
			}
			tagLines = append(tagLines, fmt.Sprintf("  %-8s %s %.1fh", tagName, coloredBar, hrs))
		}
	}

	// 30-Day activity trend sparkline
	dailyFocusSecs := make(map[string]int)
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			dateStr := t.UpdatedAt.Format("2006-01-02")
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && t.SchedulingType == model.Anchored {
				dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}
			dailyFocusSecs[dateStr] += dur
		}
	}

	var heatmapLines []string
	heatmapLines = append(heatmapLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("30-DAY FOCUS TREND"))

	var trendSB strings.Builder
	trendSB.WriteString("  ")
	for i := 29; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		secs := dailyFocusSecs[dateStr]
		hrs := float64(secs) / 3600.0

		var cellColor lipgloss.Color
		if hrs == 0 {
			cellColor = m.Theme.Muted
		} else if hrs <= 1.5 {
			cellColor = m.Theme.P2Color
		} else if hrs <= 4.0 {
			cellColor = m.Theme.Accent
		} else {
			cellColor = m.Theme.SuccessColor
		}

		char := "■"
		if hrs == 0 {
			char = "·"
		}
		trendSB.WriteString(lipgloss.NewStyle().Foreground(cellColor).Render(char) + " ")
	}
	heatmapLines = append(heatmapLines, trendSB.String())

	legendSB := strings.Builder{}
	legendSB.WriteString("    Less ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("·") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.P2Color).Render("■") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("■") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("■") + " ")
	legendSB.WriteString(" More")
	heatmapLines = append(heatmapLines, legendSB.String())

	// Assemble final content layout
	var contentSB strings.Builder
	contentSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲  E X E C U T I O N   A N A L Y T I C S") + "\n")
	contentSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", workspaceWidth-4)) + "\n\n")

	contentSB.WriteString(kpiRow + "\n\n")

	if workspaceWidth >= 75 {
		leftColContent := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(timelineLines, "\n"),
			"\n",
			strings.Join(statsLines, "\n"),
		)

		rightColContent := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(ratioLines, "\n"),
			"\n",
			strings.Join(tagLines, "\n"),
			"\n",
			strings.Join(heatmapLines, "\n"),
		)

		leftStyled := lipgloss.NewStyle().Width(36).Render(leftColContent)
		rightStyled := lipgloss.NewStyle().Width(workspaceWidth - 40).Render(rightColContent)

		columns := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, "    ", rightStyled)
		contentSB.WriteString(columns)
	} else {
		stacked := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(timelineLines, "\n"),
			"\n",
			strings.Join(statsLines, "\n"),
			"\n",
			strings.Join(ratioLines, "\n"),
			"\n",
			strings.Join(tagLines, "\n"),
			"\n",
			strings.Join(heatmapLines, "\n"),
		)
		contentSB.WriteString(stacked)
	}

	return m.Theme.PanelStyle.
		Width(m.Width - 28).
		Height(height).
		Render(contentSB.String())
}

func (m Model) renderDetailPanel(height int) string {
	t := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(strings.ToUpper(t.Title)) + "\n")
	sb.WriteString(strings.Repeat("─", 32) + "\n\n")

	sb.WriteString(fmt.Sprintf("Priority      %s\n", t.Priority))
	sb.WriteString(fmt.Sprintf("Story Points  %d\n", t.StoryPoints))
	sb.WriteString(fmt.Sprintf("Lifecycle     %s\n", t.LifecycleState))
	sb.WriteString(fmt.Sprintf("Schedule      %s\n\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("Start Time    %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("End Time      %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("DESCRIPTION\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(desc) + "\n\n")

	sb.WriteString("EXECUTION METRICS\n")
	sb.WriteString(fmt.Sprintf(" • Focus Logged:    %s\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf(" • Pomodoros:       %d/%d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf(" • Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))

	return lipgloss.NewStyle().
		Background(m.Theme.SelectedBg).
		Foreground(m.Theme.Fg).
		Padding(1, 2).
		Height(height - 2).
		Render(sb.String())
}

func (m Model) renderDetailModal() string {
	t := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("ℹ  TASK INSPECTOR\n"))
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", 44)) + "\n\n")

	sb.WriteString(fmt.Sprintf("  Title:        %s\n", lipgloss.NewStyle().Bold(true).Render(sentenceCase(t.Title))))
	sb.WriteString(fmt.Sprintf("  Priority:     %s      •  Story Points:  %d SP\n", t.Priority, t.StoryPoints))
	sb.WriteString(fmt.Sprintf("  State:        %s  •  Schedule:      %s\n\n", t.LifecycleState, t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("  Start Time:   %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("  End Time:     %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("  DESCRIPTION:\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	wrappedDesc := wrapText(desc, 40)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(indentText(wrappedDesc, "    ")) + "\n\n")

	sb.WriteString("  EXECUTION METRICS:\n")
	sb.WriteString(fmt.Sprintf("   ● Focus Logged:    %v\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf("   ● Pomodoros:       %d/%d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf("   ● Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))

	sb.WriteString("\n  [z] Start Focus   [x] Complete   [d] Delete   [Esc/Enter] Close")

	return m.Theme.ModalStyle.
		Width(48).
		Render(sb.String())
}

func (m Model) renderFormModal() string {
	f := m.Form

	var fields []string
	fields = append(fields, "  CREATE WORK ITEM TASK\n  ─────────────────────\n")

	renderField := func(label string, input string, index int) string {
		style := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if f.ActiveField == index {
			style = style.Foreground(m.Theme.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-15s %s", style.Render(label), input)
	}

	fields = append(fields, renderField("1. Title:", f.TitleInput.View(), 0))
	fields = append(fields, renderField("2. Description:", f.DescInput.View(), 1))
	fields = append(fields, renderField("3. Priority:", f.PriorityInput.View(), 2))
	fields = append(fields, renderField("4. Story Points:", f.SPInput.View(), 3))
	fields = append(fields, renderField("5. Anchored (Y/N):", f.AnchorInput.View(), 4))
	fields = append(fields, renderField("6. Start Time:", f.StartTimeInput.View(), 5))
	fields = append(fields, renderField("7. Duration (m):", f.DurationInput.View(), 6))

	submitText := " [SUBMIT] "
	if f.ActiveField == 7 {
		submitText = lipgloss.NewStyle().Background(m.Theme.SuccessColor).Foreground(m.Theme.CanvasBg).Bold(true).Render(submitText)
	} else {
		submitText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(submitText)
	}
	fields = append(fields, "\n  "+submitText)

	return m.Theme.ModalStyle.
		Width(48).
		Render(strings.Join(fields, "\n"))
}

func (m Model) renderCommandPalette() string {
	var sb strings.Builder
	sb.WriteString(m.CommandInput.View() + "\n")
	sb.WriteString("  ────────────────────────────────────────────\n")

	val := strings.ToLower(m.CommandInput.Value())
	cmds := []string{"create", "todo", "complete", "delete", "sync", "auth", "dashboard", "month", "week", "day", "analytics", "quit"}

	count := 0
	for _, c := range cmds {
		if strings.Contains(c, val) {
			bullet := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("❯")
			sb.WriteString(fmt.Sprintf("  %s %-12s  %s\n", bullet, c, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("command action")))
			count++
			if count >= 4 {
				break
			}
		}
	}

	return lipgloss.NewStyle().
		Background(m.Theme.ModalBg).
		Foreground(m.Theme.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Width(m.Width - 4).
		Padding(1, 2).
		Render(sb.String())
}

func (m Model) renderPromptModal() string {
	var lines []string
	lines = append(lines, "  ⚡ TASK READY FOR FOCUS\n  ──────────────────────\n")
	lines = append(lines, fmt.Sprintf("  Title:    %s", lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(m.PromptTask.Title))))
	lines = append(lines, fmt.Sprintf("  Priority: %s  •  Story Points: %d", m.PromptTask.Priority, m.PromptTask.StoryPoints))
	lines = append(lines, fmt.Sprintf("  Time:     %s - %s", m.PromptTask.TimeWindow.Start.Format("15:04"), m.PromptTask.TimeWindow.End.Format("15:04")))
	lines = append(lines, "\n  [Enter] Start Focus   [s] Snooze 5m   [d/Esc] Dismiss")

	return m.Theme.ModalStyle.
		Width(48).
		BorderForeground(m.Theme.Accent).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderReviewModal() string {
	var lines []string
	lines = append(lines, "  📊 DAILY SHUTDOWN REVIEW\n  ────────────────────────\n")
	lines = append(lines, fmt.Sprintf("  Completed Tasks:   %d", m.ReviewTasksCompleted))
	lines = append(lines, fmt.Sprintf("  Deferred Tasks:    %d", m.ReviewTasksDeferred))
	lines = append(lines, fmt.Sprintf("  Total Focus Logged: %v", time.Duration(m.ReviewFocusSeconds)*time.Second))
	lines = append(lines, "\n  Move unfinished scheduled tasks to tomorrow?")
	lines = append(lines, "  [y] Yes, defer them   [n/Esc] No, leave as overdue")

	return m.Theme.ModalStyle.
		Width(48).
		BorderForeground(m.Theme.SuccessColor).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderHelpModal() string {
	var lines []string
	lines = append(lines, "  ▲ S T R E A M   C O M M A N D   R E F E R E N C E\n  ────────────────────────────────────────────────\n")
	lines = append(lines, "  KEYBOARD SHORTCUTS")
	lines = append(lines, "    1 - 5       Switch views (Dashboard, Month, Week, Day, Stats)")
	lines = append(lines, "    Tab / h / l Toggle Focus between Panels (Timeline / Shelf)")
	lines = append(lines, "    j / k       Navigate items or timeline hours")
	lines = append(lines, "    H / L       Navigate days backward / forward")
	lines = append(lines, "    ctrl+d / u  Scroll active pane down / up")
	lines = append(lines, "    i           Open task creation wizard form")
	lines = append(lines, "    x           Complete selected task")
	lines = append(lines, "    d           Delete selected task")
	lines = append(lines, "    z           Start Zen Mode focus session for task")
	lines = append(lines, "    :           Enter Command Palette mode")
	lines = append(lines, "    ?           Toggle this help documentation modal")
	lines = append(lines, "")
	lines = append(lines, "  COMMAND PALETTE (:command)")
	lines = append(lines, "    :create <t> Anchor a new task for today at 9:00 AM")
	lines = append(lines, "    :todo <t>   Add a floating task to the Backlog Shelf")
	lines = append(lines, "    :complete   Complete active task")
	lines = append(lines, "    :delete     Delete active task")
	lines = append(lines, "    :sync       Force Google Calendar sync")
	lines = append(lines, "    :review     Open Daily Shutdown Review checklist")
	lines = append(lines, "    :quit       Exit the stream application")
	lines = append(lines, "\n  Press [Esc / Enter / ?] to dismiss this help window")

	return m.Theme.ModalStyle.
		Width(54).
		BorderForeground(m.Theme.Accent).
		Render(strings.Join(lines, "\n"))
}

func sliceAnsi(s string, start, end int) string {
	var sb strings.Builder
	var runes = []rune(s)
	var inEscape = false
	var visualCount = 0

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			sb.WriteRune(r)
			continue
		}
		if inEscape {
			sb.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		if visualCount >= start && visualCount < end {
			sb.WriteRune(r)
		}
		visualCount++
	}
	return sb.String()
}

func overlayString(base string, overlay string, x int, y int, baseWidth int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := 0
	for _, l := range overlayLines {
		w := lipgloss.Width(l)
		if w > overlayWidth {
			overlayWidth = w
		}
	}

	for i, oLine := range overlayLines {
		targetY := y + i
		if targetY >= len(baseLines) {
			break
		}
		bLine := baseLines[targetY]

		leftPart := sliceAnsi(bLine, 0, x)
		rightPart := sliceAnsi(bLine, x+overlayWidth, baseWidth)

		leftVisualLen := lipgloss.Width(leftPart)
		if leftVisualLen < x {
			leftPart += strings.Repeat(" ", x-leftVisualLen)
		}

		baseLines[targetY] = leftPart + oLine + rightPart
	}
	return strings.Join(baseLines, "\n")
}

func wrapText(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	var words = strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var res []string
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > limit {
			res = append(res, currentLine)
			currentLine = word
		} else {
			if len(currentLine) > 0 {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if len(currentLine) > 0 {
		res = append(res, currentLine)
	}
	return strings.Join(res, "\n")
}

func indentText(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = indent + l
	}
	return strings.Join(lines, "\n")
}

func getNowBadge(width int, now time.Time) string {
	full := fmt.Sprintf("── NOW • %02d:%02d ──", now.Hour(), now.Minute())
	if len(full) <= width {
		return full
	}
	mid := fmt.Sprintf("NOW • %02d:%02d", now.Hour(), now.Minute())
	if len(mid) <= width {
		return mid
	}
	short := fmt.Sprintf("● %02d:%02d", now.Hour(), now.Minute())
	if len(short) <= width {
		return short
	}
	return "●"
}

func (m Model) overlayMiniZen(content string, workspaceWidth int) string {
	if m.ZenTimer == nil || !m.ZenTimer.Running {
		return content
	}

	zt := m.ZenTimer
	sess := zt.Sessions[zt.CurrentSessionIdx]
	hVal := int(zt.TimeRemaining.Hours())
	mVal := int(zt.TimeRemaining.Minutes()) % 60
	sVal := int(zt.TimeRemaining.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d Remaining", hVal, mVal, sVal)

	pct := 1.0 - (zt.TimeRemaining.Seconds() / sess.Duration.Seconds())
	bar := RenderProgressBar(18, pct)

	title := zt.Task.Title
	if len(title) > 20 {
		title = title[:17] + "..."
	}

	widgetWidth := 26
	widgetBg := m.Theme.SelectedBg
	if zt.IsPaused {
		widgetBg = m.Theme.PanelBg
	}

	sessionHeader := "● FOCUS RUNNING"
	if zt.IsPaused {
		sessionHeader = "● FOCUS PAUSED"
	}

	headerStyle := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true)
	if zt.IsPaused {
		headerStyle = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)
	}

	widgetStr := lipgloss.NewStyle().
		Background(widgetBg).
		Foreground(m.Theme.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Padding(0, 1).
		Width(widgetWidth).
		Render(fmt.Sprintf(
			"%s\n%s\n%s\n%s",
			headerStyle.Render(sessionHeader),
			sentenceCase(title),
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(timeStr),
			bar,
		))

	x := workspaceWidth - widgetWidth - 2
	if x < 0 {
		x = 0
	}
	return overlayString(content, widgetStr, x, 1, workspaceWidth)
}
