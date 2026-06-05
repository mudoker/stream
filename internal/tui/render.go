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
		return "Initializing workspace..."
	}

	// 1. Zen Mode takes over full canvas
	if m.CurrentMode == ModeZen {
		return m.renderZenMode()
	}

	// 2. Main structure: Header + Workspace Content + Footer
	header := m.renderHeader()
	footer := m.renderFooter()

	mainHeight := m.Height - lipgloss.Height(header) - lipgloss.Height(footer)
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

	// Drawer slide-over on the right (Linear style inspector)
	if m.DetailOpen {
		detailPanel := m.renderDetailPanel(mainHeight)
		drawerWidth := int(float64(m.Width) * 0.35)
		if drawerWidth < 25 {
			drawerWidth = 25
		}
		leftWidth := m.Width - drawerWidth - 3
		if leftWidth < 20 {
			leftWidth = 20
		}
		leftContent := lipgloss.NewStyle().Width(leftWidth).Render(content)
		rightContent := lipgloss.NewStyle().Width(drawerWidth).Render(detailPanel)
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftContent, "   ", rightContent)
	}

	// Task Creation Wizard Overlay
	if m.CurrentMode == ModeForm {
		formModal := m.renderFormModal()
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
	} else if m.PromptOpen {
		promptModal := m.renderPromptModal()
		contentHeight := lipgloss.Height(content)
		formHeight := lipgloss.Height(promptModal)
		paddingTop := (contentHeight - formHeight) / 2
		if paddingTop < 0 {
			paddingTop = 0
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			strings.Repeat("\n", paddingTop),
			promptModal,
			strings.Repeat("\n", max(0, contentHeight-formHeight-paddingTop)),
		)
	} else if m.ReviewOpen {
		reviewModal := m.renderReviewModal()
		contentHeight := lipgloss.Height(content)
		formHeight := lipgloss.Height(reviewModal)
		paddingTop := (contentHeight - formHeight) / 2
		if paddingTop < 0 {
			paddingTop = 0
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			strings.Repeat("\n", paddingTop),
			reviewModal,
			strings.Repeat("\n", max(0, contentHeight-formHeight-paddingTop)),
		)
	}

	canvas := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)

	// Floating Command Palette Overlay at the bottom
	if m.CurrentMode == ModeCommand {
		cmdPalette := m.renderCommandPalette()
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdPalette)
	}

	return canvas
}

func (m Model) renderHeader() string {
	viewNames := []string{"dashboard", "month grid", "weekly lanes", "daily timeline", "analytics"}
	var tabs []string

	for i, name := range viewNames {
		if int(m.CurrentView) == i {
			// Linear-style soft indigo active tab
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(m.Theme.Accent).
				Bold(true).
				Render(name))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(m.Theme.Muted).
				Render(name))
		}
	}

	tabsJoined := strings.Join(tabs, "  /  ")

	// Arc-style minimalist mode indicator
	modeColor := m.Theme.Accent
	if m.CurrentMode == ModeZen {
		modeColor = m.Theme.P0Color
	} else if m.CurrentMode == ModeForm {
		modeColor = m.Theme.P1Color
	}
	modeLabel := lipgloss.NewStyle().
		Foreground(modeColor).
		Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))

	timeStr := time.Now().Format("Jan _2 15:04")
	timeDisplay := lipgloss.NewStyle().
		Foreground(m.Theme.SuccessColor).
		Render(timeStr)

	gapWidth := m.Width - lipgloss.Width(tabsJoined) - lipgloss.Width(modeLabel) - lipgloss.Width(timeDisplay) - 10
	if gapWidth < 2 {
		gapWidth = 2
	}
	gap := strings.Repeat(" ", gapWidth)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Center, tabsJoined, gap, modeLabel, "   ", timeDisplay)

	return lipgloss.NewStyle().
		Background(m.Theme.CanvasBg).
		Width(m.Width).
		Padding(1, 2).
		Render(headerLine) + "\n"
}

func (m Model) renderFooter() string {
	shortcuts := "1-5 view switch  •  i new task  •  enter details  •  z zen mode  •  : commands"
	status := m.StatusMsg
	if status == "" {
		status = "idle"
	}
	status = strings.ToLower(status)
	if len(status) > 35 {
		status = status[:32] + "..."
	}

	syncState := "offline"
	if m.Sync.IsOnline() {
		syncState = "online"
	}
	syncDisplay := lipgloss.NewStyle().
		Foreground(m.Theme.SuccessColor).
		Render(fmt.Sprintf("gcal: %s", syncState))

	left := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(shortcuts)
	center := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("   " + status)

	gapWidth := m.Width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(syncDisplay) - 4
	if gapWidth < 2 {
		gapWidth = 2
	}
	gap := strings.Repeat(" ", gapWidth)

	footerLine := lipgloss.JoinHorizontal(lipgloss.Center, left, center, gap, syncDisplay)

	return "\n" + lipgloss.NewStyle().
		Background(m.Theme.CanvasBg).
		Width(m.Width).
		Padding(0, 2).
		Render(footerLine)
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

	// Linear style section headers
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

	// Workload Chart Widget
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

	rightWidth := m.Width - 46
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
		Width(m.Width - 4).
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

	colWidth := (m.Width - 10) / 7
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

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderDayView(height int) string {
	timelineWidth := int(float64(m.Width) * 0.73)
	shelfWidth := m.Width - timelineWidth - 4
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

	var timelineLines []string
	headerText := fmt.Sprintf("DAILY WORKSPACE  /  %s", strings.ToUpper(m.SelectedDay.Format("Monday, Jan _2")))
	timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)+"\n")

	now := time.Now()
	isToday := m.SelectedDay.Year() == now.Year() && m.SelectedDay.Month() == now.Month() && m.SelectedDay.Day() == now.Day()

	for h := 8; h <= 20; h++ {
		// Clean visual timeline cursor slicing schedule
		if isToday && now.Hour() == h && now.Minute() < 30 && h > 8 {
			lineText := fmt.Sprintf("───────────────────── %02d:%02d NOW ─────────────────────", now.Hour(), now.Minute())
			timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(lineText))
		}

		var tasksAtHour []ScheduledColumn
		for _, rc := range cols {
			startH := rc.Task.TimeWindow.Start.Hour()
			endH := rc.Task.TimeWindow.End.Hour()
			if h >= startH && h < endH {
				tasksAtHour = append(tasksAtHour, rc)
			}
		}

		isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == h
		hourLabel := fmt.Sprintf("%02d:00", h)
		if isSelectedHour {
			hourLabel = lipgloss.NewStyle().Background(m.Theme.Accent).Foreground(m.Theme.CanvasBg).Bold(true).Render(hourLabel)
		} else {
			hourLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(hourLabel)
		}

		if len(tasksAtHour) == 0 {
			timelineLines = append(timelineLines, fmt.Sprintf("  %s  │", hourLabel))
		} else {
			var colBlocks []string
			cellWidth := (timelineWidth - 14) / len(tasksAtHour)
			if cellWidth < 12 {
				cellWidth = 12
			}

			for _, col := range tasksAtHour {
				t := col.Task
				title := strings.ToUpper(t.Title)
				if len(title) > cellWidth-6 {
					title = title[:cellWidth-9] + "..."
				}

				isActiveBlock := isToday && now.Hour() >= t.TimeWindow.Start.Hour() && now.Hour() < t.TimeWindow.End.Hour()

				var pBarColor lipgloss.Color = m.Theme.P2Color
				if t.Priority == model.P0 {
					pBarColor = m.Theme.P0Color
				} else if t.Priority == model.P1 {
					pBarColor = m.Theme.P1Color
				} else if t.Priority == model.P3 {
					pBarColor = m.Theme.P3Color
				}

				cardStyle := lipgloss.NewStyle().
					Background(m.Theme.PanelBg).
					Foreground(m.Theme.Fg).
					Padding(1, 2)

				if isActiveBlock {
					cardStyle = cardStyle.Background(m.Theme.SelectedBg)
				}

				var cardLines []string
				pBadge := lipgloss.NewStyle().Foreground(pBarColor).Bold(true).Render("▲ " + string(t.Priority))
				spBadge := fmt.Sprintf("• %d SP", t.StoryPoints)
				cardLines = append(cardLines, fmt.Sprintf("%s %s", pBadge, spBadge))

				if isActiveBlock {
					activeTimer := ""
					if m.CurrentMode == ModeZen && m.ZenTimer != nil {
						hVal := int(m.ZenTimer.TimeRemaining.Hours())
						mVal := int(m.ZenTimer.TimeRemaining.Minutes()) % 60
						sVal := int(m.ZenTimer.TimeRemaining.Seconds()) % 60
						activeTimer = fmt.Sprintf(" • %02d:%02d:%02d", hVal, mVal, sVal)
					}
					cardLines = append(cardLines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▌ ACTIVE"+activeTimer))
				}

				cardLines = append(cardLines, "", lipgloss.NewStyle().Bold(true).Render(title))
				cardLines = append(cardLines, fmt.Sprintf("%s → %s", t.TimeWindow.Start.Format("15:04"), t.TimeWindow.End.Format("15:04")))

				renderedCard := cardStyle.Width(cellWidth - 4).Render(strings.Join(cardLines, "\n"))

				leftBar := lipgloss.NewStyle().
					Foreground(pBarColor).
					Background(m.Theme.PanelBg).
					Height(lipgloss.Height(renderedCard)).
					Render("▌")

				colBlocks = append(colBlocks, lipgloss.JoinHorizontal(lipgloss.Top, leftBar, renderedCard))
			}

			joinedBlocks := lipgloss.JoinHorizontal(lipgloss.Top, colBlocks...)
			timelineLines = append(timelineLines, fmt.Sprintf("  %s  │ %s", hourLabel, joinedBlocks))
		}
	}

	leftBox := m.Theme.PanelStyle.
		Width(timelineWidth).
		Height(height).
		Render(strings.Join(timelineLines, "\n"))

	// Todo Shelf (Command Center)
	var shelfLines []string
	shelfLines = append(shelfLines, "TODO COMMAND CENTER\n")

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

			title := strings.ToUpper(t.Title)
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

	rightBox := m.Theme.PanelStyle.
		Width(shelfWidth).
		Height(height).
		Render(strings.Join(shelfLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "    ", rightBox)
}

func (m Model) renderZenMode() string {
	if m.ZenTimer == nil {
		return "No focus timer running."
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
	sb.WriteString("EXECUTION ANALYTICS & PRODUCTIVITY DATA\n")
	sb.WriteString("───────────────────────────────────────\n\n")
	sb.WriteString(fmt.Sprintf("  Total Focus Logged:      %s\n", time.Duration(totalFocusSecs)*time.Second))
	sb.WriteString(fmt.Sprintf("  Total Interruptions:     %d\n", totalInterruptions))
	sb.WriteString(fmt.Sprintf("  Task Completion Rate:    %.1f%% (%d/%d Tasks)\n\n", rate, completedCount, totalCount))

	sb.WriteString("  Productivity Heatmap Preview (Story Points Completed):\n")
	sb.WriteString("  Mon ██████████ 10 SP\n")
	sb.WriteString("  Tue ██████ 6 SP\n")
	sb.WriteString("  Wed ████████████ 12 SP\n")
	sb.WriteString("  Thu ████ 4 SP\n")
	sb.WriteString("  Fri ████████ 8 SP\n")

	return m.Theme.PanelStyle.
		Width(m.Width - 4).
		Height(height).
		Render(sb.String())
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
