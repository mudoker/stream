package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"tuical/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing canvas..."
	}

	// 1. Zen Mode takes over full viewport
	if m.CurrentMode == ModeZen {
		return m.renderZenMode()
	}

	// 2. Normal View layout construction
	header := m.renderHeader()
	footer := m.renderFooter()

	// Available height for main content
	mainHeight := m.Height - lipgloss.Height(header) - lipgloss.Height(footer) - 1
	if mainHeight < 5 {
		mainHeight = 5
	}

	var content string
	switch m.CurrentView {
	case DashboardView:
		content = m.renderDashboard(mainHeight)
	case MonthView:
		content = m.renderMonthView(mainHeight)
	case WeekView:
		content = m.renderWeekView(mainHeight)
	case DayView:
		content = m.renderDayView(mainHeight)
	case AnalyticsView:
		content = m.renderAnalyticsView(mainHeight)
	}

	// If Detail panel is open, slide it in on the right
	if m.DetailOpen {
		detailPanel := m.renderDetailPanel(mainHeight)
		rightWidth := 35
		leftWidth := m.Width - rightWidth - 2
		if leftWidth < 20 {
			leftWidth = 20
		}
		leftContent := lipgloss.NewStyle().Width(leftWidth).Render(content)
		rightContent := lipgloss.NewStyle().Width(rightWidth).Render(detailPanel)
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftContent, "  ", rightContent)
	}

	// If Wizard is active, draw it as a modal overlay on top of main content
	if m.CurrentMode == ModeForm {
		formModal := m.renderFormModal()
		// Simple mock of modal overlay center alignment
		contentHeight := lipgloss.Height(content)
		formHeight := lipgloss.Height(formModal)
		paddingTop := (contentHeight - formHeight) / 2
		if paddingTop < 0 {
			paddingTop = 0
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			strings.Repeat("\n", paddingTop),
			formModal,
			strings.Repeat("\n", max(0, contentHeight-formHeight-paddingTop)),
		)
	}

	// Join all parts vertically
	canvas := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

	// Add Command Palette at the very bottom if active
	if m.CurrentMode == ModeCommand {
		cmdBar := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.Theme.Focus).
			Width(m.Width - 2).
			Render(m.CommandInput.View())
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdBar)
	}

	return canvas
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) renderHeader() string {
	viewNames := []string{" Dashboard ", " Month ", " Week ", " Day ", " Analytics "}
	var tabs []string

	for i, name := range viewNames {
		if int(m.CurrentView) == i {
			// Highlighted active tab
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(m.Theme.Bg).
				Background(m.Theme.Focus).
				Bold(true).
				Render(name))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(m.Theme.Muted).
				Background(m.Theme.PanelBg).
				Render(name))
		}
	}

	tabsJoined := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Mode badge
	modeColor := m.Theme.Accent
	if m.CurrentMode == ModeZen {
		modeColor = m.Theme.Critical
	} else if m.CurrentMode == ModeForm {
		modeColor = m.Theme.Warning
	}
	modeBadge := lipgloss.NewStyle().
		Foreground(m.Theme.Bg).
		Background(modeColor).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf(" %s ", m.CurrentMode))

	timeStr := time.Now().Format("2006-01-02 15:04:05")
	timeDisplay := lipgloss.NewStyle().
		Foreground(m.Theme.Success).
		Render(timeStr)

	// Combine components
	gapWidth := m.Width - lipgloss.Width(tabsJoined) - lipgloss.Width(modeBadge) - lipgloss.Width(timeDisplay) - 2
	if gapWidth < 2 {
		gapWidth = 2
	}
	gap := strings.Repeat(" ", gapWidth)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Center, tabsJoined, gap, modeBadge, " ", timeDisplay)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(m.Theme.Muted).
		Width(m.Width).
		Render(headerLine)
}

func (m Model) renderFooter() string {
	shortcuts := "1-5: Views | i: New Task | Enter: Detail | z: Zen Mode | : CMD | Ctrl+C: Quit"
	status := m.StatusMsg
	if status == "" {
		status = "Ready."
	}
	if len(status) > 40 {
		status = status[:37] + "..."
	}

	syncState := "GCal: Offline"
	if m.Sync.IsOnline() {
		syncState = "GCal: Online ✓"
	}

	left := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(shortcuts)
	center := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("  " + status)
	right := lipgloss.NewStyle().Foreground(m.Theme.Success).Render(syncState)

	gapWidth := m.Width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right) - 4
	if gapWidth < 2 {
		gapWidth = 2
	}
	gap := strings.Repeat(" ", gapWidth)

	footerLine := lipgloss.JoinHorizontal(lipgloss.Center, left, center, gap, right)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(m.Theme.Muted).
		Width(m.Width).
		Render(footerLine)
}

func (m Model) renderDashboard(height int) string {
	// 1. Calculate statistics
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
			// Unscheduled but created today
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

	// 2. Render Left Panel (Summary & Next Scheduled Tasks)
	summary := fmt.Sprintf(
		"Tasks Today:       %d\n"+
			"Completed:         %d\n"+
			"Planned Focus:     %s\n"+
			"Completed Focus:   %s\n",
		len(todayTasks),
		completedCount,
		time.Duration(plannedFocusSecs)*time.Second,
		time.Duration(elapsedFocusSecs)*time.Second,
	)

	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Padding(1, 2).
		Width(35).
		Render(" Today Summary \n\n" + summary)

	upcomingLines := []string{" Upcoming Tasks:\n"}
	var upcomingTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && t.TimeWindow.Start.After(today) && t.LifecycleState != model.StateCompleted {
			upcomingTasks = append(upcomingTasks, t)
		}
	}

	if len(upcomingTasks) == 0 {
		upcomingLines = append(upcomingLines, "  No upcoming tasks scheduled.")
	} else {
		for i, t := range upcomingTasks {
			if i >= 4 {
				break
			}
			upcomingLines = append(upcomingLines, fmt.Sprintf("  • %s (%s)", t.Title, t.TimeWindow.Start.Format("15:04")))
		}
	}

	upcomingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Padding(1, 2).
		Width(35).
		Render(strings.Join(upcomingLines, "\n"))

	leftPane := lipgloss.JoinVertical(lipgloss.Left, summaryBox, "\n", upcomingBox)

	// 3. Render Right Panel (Weekly Workload Chart)
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

	chartLines := []string{" Weekly Workload (Story Points)\n"}
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
		barWidth := int(math.Round(float64(pts) * 15.0 / float64(maxPoints)))
		bar := strings.Repeat("█", barWidth)
		if bar == "" && pts > 0 {
			bar = "▏"
		}
		color := m.Theme.Success
		if pts >= 9 {
			color = m.Theme.Critical
		} else if pts >= 6 {
			color = m.Theme.Warning
		} else if pts <= 2 {
			color = m.Theme.Muted
		}

		coloredBar := lipgloss.NewStyle().Foreground(color).Render(bar)
		chartLines = append(chartLines, fmt.Sprintf("  %s │ %s (%d SP)", weekdayNames[idx], coloredBar, pts))
	}

	rightWidth := m.Width - 42
	if rightWidth < 20 {
		rightWidth = 20
	}

	chartBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Focus).
		Padding(1, 2).
		Width(rightWidth).
		Height(12).
		Render(strings.Join(chartLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "   ", chartBox)
}

func (m Model) renderMonthView(height int) string {
	today := time.Now()
	year, month, _ := m.SelectedDay.Date()

	// Find the start date of the monthly calendar grid (Monday of the first week of the month)
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, today.Location())
	offset := int(firstOfMonth.Weekday()) - 1 // Monday offset
	if offset < 0 {
		offset = 6
	}
	gridStart := firstOfMonth.AddDate(0, 0, -offset)

	var sb strings.Builder
	title := fmt.Sprintf("   Strategic Calendar Grid: %s %d   ", month.String(), year)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Focus).Bold(true).Render(title) + "\n\n")

	// Print Weekday headers
	sb.WriteString("  Mon   Tue   Wed   Thu   Fri   Sat   Sun  \n")
	sb.WriteString(" ╭─────┬─────┬─────┬─────┬─────┬─────┬─────╮\n")

	cellDay := gridStart
	for week := 0; week < 6; week++ {
		var rowDays []string
		var rowMetrics []string

		for wday := 0; wday < 7; wday++ {
			dayNum := cellDay.Day()
			isCurrentMonth := cellDay.Month() == month
			isToday := cellDay.Year() == today.Year() && cellDay.Month() == today.Month() && cellDay.Day() == today.Day()
			isSelected := cellDay.Year() == m.SelectedDay.Year() && cellDay.Month() == m.SelectedDay.Month() && cellDay.Day() == m.SelectedDay.Day()

			// Get daily SP workload
			dailySP := 0
			for _, t := range m.Tasks {
				if t.TimeWindow.Start.Year() == cellDay.Year() &&
					t.TimeWindow.Start.Month() == cellDay.Month() &&
					t.TimeWindow.Start.Day() == cellDay.Day() {
					dailySP += t.StoryPoints
				}
			}

			// Color coding according to SP load
			var numColor lipgloss.TerminalColor = m.Theme.Muted
			if isCurrentMonth {
				numColor = m.Theme.Fg
				if dailySP >= 9 {
					numColor = m.Theme.Critical
				} else if dailySP >= 6 {
					numColor = m.Theme.Warning
				} else if dailySP >= 3 {
					numColor = m.Theme.Success
				}
			}

			numStyle := lipgloss.NewStyle().Foreground(numColor)
			if isToday {
				numStyle = numStyle.Background(m.Theme.Accent).Foreground(m.Theme.Bg).Bold(true)
			} else if isSelected {
				numStyle = numStyle.Background(m.Theme.Focus).Foreground(m.Theme.Bg).Bold(true)
			}

			rowDays = append(rowDays, fmt.Sprintf(" %-3s", numStyle.Render(fmt.Sprintf("%2d", dayNum))))

			metricStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
			if dailySP > 0 {
				metricColor := m.Theme.Success
				if dailySP >= 9 {
					metricColor = m.Theme.Critical
				} else if dailySP >= 6 {
					metricColor = m.Theme.Warning
				}
				metricStyle = lipgloss.NewStyle().Foreground(metricColor)
			}
			rowMetrics = append(rowMetrics, metricStyle.Render(fmt.Sprintf("%2dSP", dailySP)))

			cellDay = cellDay.AddDate(0, 0, 1)
		}

		sb.WriteString(" │" + strings.Join(rowDays, "│") + "│\n")
		sb.WriteString(" │" + strings.Join(rowMetrics, "│") + "│\n")

		if week < 5 {
			sb.WriteString(" ├─────┼─────┼─────┼─────┼─────┼─────┼─────┤\n")
		}
	}
	sb.WriteString(" ╰─────┴─────┴─────┴─────┴─────┴─────┴─────╯\n")

	return sb.String()
}

func (m Model) renderWeekView(height int) string {
	today := time.Now()
	// Find starting Monday of SelectedDay's week
	offset := int(m.SelectedDay.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := m.SelectedDay.AddDate(0, 0, -offset)

	weekdayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var columns []string

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

		// Header for the column
		colStyle := lipgloss.NewStyle().Width(18).Height(height - 2).Border(lipgloss.RoundedBorder())
		if isToday {
			colStyle = colStyle.BorderForeground(m.Theme.Accent)
		} else if isSelected {
			colStyle = colStyle.BorderForeground(m.Theme.Focus)
		} else {
			colStyle = colStyle.BorderForeground(m.Theme.Muted)
		}

		var dayContent []string
		headerText := fmt.Sprintf("%s %02d/%02d", weekdayNames[i], day.Month(), day.Day())
		if isToday {
			headerText = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)
		}
		dayContent = append(dayContent, headerText, strings.Repeat("─", 16))

		// Render overlap splits
		resolved := ResolveOverlaps(dayTasks)
		if len(resolved) == 0 {
			dayContent = append(dayContent, "\n  No Events")
		} else {
			// Print event blocks inside the vertical column
			for _, rc := range resolved {
				timeText := fmt.Sprintf("%s-%s", rc.Task.TimeWindow.Start.Format("15:04"), rc.Task.TimeWindow.End.Format("15:04"))
				card := fmt.Sprintf("[%s]\n%s", timeText, rc.Task.Title)
				if len(card) > 16 {
					card = card[:13] + "..."
				}
				cardStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg).Padding(0, 1).Border(lipgloss.NormalBorder())
				if rc.Task.Priority == model.P0 {
					cardStyle = cardStyle.BorderForeground(m.Theme.Critical)
				} else {
					cardStyle = cardStyle.BorderForeground(m.Theme.Muted)
				}
				dayContent = append(dayContent, cardStyle.Render(card))
			}
		}

		columns = append(columns, colStyle.Render(strings.Join(dayContent, "\n")))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderDayView(height int) string {
	// Day timeline on left, Todo Shelf on right
	leftWidth := m.Width - 30
	if leftWidth < 30 {
		leftWidth = 30
	}

	// 1. Fetch tasks for selected day
	var anchoredTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored &&
			t.TimeWindow.Start.Year() == m.SelectedDay.Year() &&
			t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
			t.TimeWindow.Start.Day() == m.SelectedDay.Day() {
			anchoredTasks = append(anchoredTasks, t)
		}
	}

	// Resolve overlaps
	cols := ResolveOverlaps(anchoredTasks)

	// Create hourly schedule content
	var timelineLines []string
	timelineLines = append(timelineLines, fmt.Sprintf("   Timeline for %s\n", m.SelectedDay.Format("Monday, 2006-01-02")))

	// Current local time
	now := time.Now()
	isToday := m.SelectedDay.Year() == now.Year() && m.SelectedDay.Month() == now.Month() && m.SelectedDay.Day() == now.Day()

	for h := 8; h <= 20; h++ {
		// If live cursor matches this hour slot
		if isToday && now.Hour() == h && now.Minute() < 30 && (h > 8) {
			// Display live cursor line
			line := fmt.Sprintf("─────────── %02d:%02d LIVE NOW ───────────", now.Hour(), now.Minute())
			timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Success).Bold(true).Render(line))
		}

		// Render task cells at hour 'h'
		var activeTasksAtHour []ScheduledColumn
		for _, rc := range cols {
			startH := rc.Task.TimeWindow.Start.Hour()
			endH := rc.Task.TimeWindow.End.Hour()
			if h >= startH && h < endH {
				activeTasksAtHour = append(activeTasksAtHour, rc)
			}
		}

		isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == h
		hourLabel := fmt.Sprintf("%02d:00", h)
		if isSelectedHour {
			hourLabel = lipgloss.NewStyle().Background(m.Theme.Focus).Foreground(m.Theme.Bg).Bold(true).Render(hourLabel)
		} else {
			hourLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(hourLabel)
		}

		if len(activeTasksAtHour) == 0 {
			timelineLines = append(timelineLines, fmt.Sprintf("  %s │", hourLabel))
		} else {
			// Draw tasks split into columns
			var colBlocks []string
			cellWidth := (leftWidth - 12) / len(activeTasksAtHour)
			if cellWidth < 10 {
				cellWidth = 10
			}

			for _, col := range activeTasksAtHour {
				t := col.Task
				title := t.Title
				if len(title) > cellWidth-4 {
					title = title[:cellWidth-7] + "..."
				}

				isActiveBlock := isToday && now.Hour() >= t.TimeWindow.Start.Hour() && now.Hour() < t.TimeWindow.End.Hour()

				var blockStyle lipgloss.Style
				if isActiveBlock {
					blockStyle = m.Theme.ActiveCard
				} else {
					blockStyle = m.Theme.PanelStyle
				}

				// Color-code based on priority
				if t.Priority == model.P0 {
					blockStyle = blockStyle.BorderForeground(m.Theme.Critical)
				} else if t.Priority == model.P1 {
					blockStyle = blockStyle.BorderForeground(m.Theme.Warning)
				}

				cardText := fmt.Sprintf("❪%dSP❫ %s", t.StoryPoints, title)
				if isActiveBlock {
					cardText = "▌ ACTIVE NOW\n" + cardText
				}
				colBlocks = append(colBlocks, blockStyle.Width(cellWidth).Render(cardText))
			}

			joinedBlocks := lipgloss.JoinHorizontal(lipgloss.Top, colBlocks...)
			timelineLines = append(timelineLines, fmt.Sprintf("  %s │ %s", hourLabel, joinedBlocks))
		}
	}

	leftBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Width(leftWidth).
		Height(height).
		Render(strings.Join(timelineLines, "\n"))

	// 2. Render Todo Shelf
	var shelfLines []string
	shelfLines = append(shelfLines, " Todo Shelf (Floating) \n")

	shelfTasks := m.getTodoShelfTasks()
	if len(shelfTasks) == 0 {
		shelfLines = append(shelfLines, "  No floating tasks.")
	} else {
		for _, t := range shelfTasks {
			isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID

			var pMark string
			pColor := m.Theme.Muted
			switch t.Priority {
			case model.P0:
				pMark = "▲ P0"
				pColor = m.Theme.Critical
			case model.P1:
				pMark = "▲ P1"
				pColor = m.Theme.Warning
			case model.P2:
				pMark = "■ P2"
				pColor = m.Theme.Accent
			case model.P3:
				pMark = "▼ P3"
				pColor = m.Theme.Muted
			}

			title := t.Title
			if len(title) > 18 {
				title = title[:15] + "..."
			}

			pBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render(pMark)
			line := fmt.Sprintf("%s ❪%d SP❫ %s", pBadge, t.StoryPoints, title)

			cardStyle := m.Theme.PanelStyle
			if isSelected {
				cardStyle = m.Theme.SelectedStyle
			}

			shelfLines = append(shelfLines, cardStyle.Width(24).Render(line))
		}
	}

	rightBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Width(28).
		Height(height).
		Render(strings.Join(shelfLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
}

func (m Model) renderZenMode() string {
	if m.ZenTimer == nil {
		return "No Active Focus timer."
	}

	t := m.ZenTimer.Task
	sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Critical).Bold(true).Render(fmt.Sprintf(" [Zen Mode] Active Task: %s (%s)", t.Title, t.Priority)) + "\n\n")

	// Render Large Clock
	clockStr := RenderLargeTime(m.ZenTimer.TimeRemaining)
	clockBox := lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(clockStr)
	sb.WriteString(clockBox + "\n\n")

	// Render Session Info
	sessInfo := fmt.Sprintf("[ Session %d / %d: %s ]", m.ZenTimer.CurrentSessionIdx+1, len(m.ZenTimer.Sessions), sess.Type)
	sessInfoBox := lipgloss.NewStyle().
		Foreground(m.Theme.Success).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(sessInfo)
	sb.WriteString(sessInfoBox + "\n\n")

	// Render Progress Bar
	totalSecs := sess.Duration.Seconds()
	remSecs := m.ZenTimer.TimeRemaining.Seconds()
	percent := 1.0 - (remSecs / totalSecs)
	if percent < 0 {
		percent = 0
	}

	barWidth := m.Width - 10
	if barWidth < 20 {
		barWidth = 20
	}
	progBar := RenderProgressBar(barWidth, percent)
	pctStr := fmt.Sprintf("%d%% Completed", int(percent*100))
	progBox := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(fmt.Sprintf("%s %s", progBar, pctStr))
	sb.WriteString(progBox + "\n\n\n")

	// Instructions
	instructions := "[Space] Pause/Resume   [+] Add 5m   [b] Force Break   [Esc] Terminate Session"
	instBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(instructions)

	instContainer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(instBox)
	sb.WriteString(instContainer + "\n")

	return sb.String()
}

func (m Model) renderAnalyticsView(height int) string {
	var totalFocusSecs int
	var totalInterruptions int
	var completedCount int
	var totalCount int

	for _, t := range m.Tasks {
		totalCount++
		if t.LifecycleState == model.StateCompleted {
			completedCount++
		}
		totalFocusSecs += t.ExecutionMetrics.ElapsedFocusSeconds
		totalInterruptions += t.ExecutionMetrics.InterruptionCount
	}

	rate := 0.0
	if totalCount > 0 {
		rate = float64(completedCount) / float64(totalCount) * 100
	}

	var sb strings.Builder
	sb.WriteString("   Focus Execution Analytics \n\n")
	sb.WriteString(fmt.Sprintf("  Total Focus Logged:      %s\n", time.Duration(totalFocusSecs)*time.Second))
	sb.WriteString(fmt.Sprintf("  Total Interruptions:     %d\n", totalInterruptions))
	sb.WriteString(fmt.Sprintf("  Task Completion Rate:    %.1f%% (%d/%d Tasks)\n\n", rate, completedCount, totalCount))

	// Simple Productivity heatmap preview
	sb.WriteString("  Productivity Heatmap Preview (Story Points Completed):\n")
	sb.WriteString("  Mon ██████████ 10 SP\n")
	sb.WriteString("  Tue ██████ 6 SP\n")
	sb.WriteString("  Wed ████████████ 12 SP\n")
	sb.WriteString("  Thu ████ 4 SP\n")
	sb.WriteString("  Fri ████████ 8 SP\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Muted).
		Padding(1, 2).
		Width(m.Width - 4).
		Height(height).
		Render(sb.String())
}

func (m Model) renderDetailPanel(height int) string {
	t := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Focus).Bold(true).Render(t.Title) + "\n")
	sb.WriteString(strings.Repeat("─", 30) + "\n\n")

	sb.WriteString(fmt.Sprintf("Priority:     %s\n", t.Priority))
	sb.WriteString(fmt.Sprintf("Story Points: %d\n", t.StoryPoints))
	sb.WriteString(fmt.Sprintf("State:        %s\n", t.LifecycleState))
	sb.WriteString(fmt.Sprintf("Type:         %s\n\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("Start:        %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("End:          %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("Description:\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(desc) + "\n\n")

	sb.WriteString("Execution Metrics:\n")
	sb.WriteString(fmt.Sprintf(" • Focus Logged:    %s\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf(" • Pomodoros:       %d/%d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf(" • Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Focus).
		Padding(1, 2).
		Height(height).
		Render(sb.String())
}

func (m Model) renderFormModal() string {
	f := m.Form

	var fields []string
	fields = append(fields, "  Create New Work Item \n")

	renderField := func(label string, input string, index int) string {
		style := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if f.ActiveField == index {
			style = style.Foreground(m.Theme.Focus).Bold(true)
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
		submitText = lipgloss.NewStyle().Background(m.Theme.Success).Foreground(m.Theme.Bg).Bold(true).Render(submitText)
	} else {
		submitText = lipgloss.NewStyle().Foreground(m.Theme.Success).Render(submitText)
	}
	fields = append(fields, "\n  "+submitText)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Warning).
		Padding(1, 2).
		Width(45).
		Render(strings.Join(fields, "\n"))
}
