package components

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

func RenderArcSidebar(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	innerW := l.SidebarW - 2
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))

	// Calculate counts dynamically
	today := time.Now()
	todayCount := 0
	p0Count := 0
	activeCount := 0
	for _, task := range m.Tasks {
		isToday := false
		if task.SchedulingType == model.Anchored {
			isToday = viewmodel.SameDay(task.TimeWindow.Start, today)
		} else {
			isToday = viewmodel.SameDay(task.CreatedAt, today)
		}

		if task.LifecycleState != model.StateCompleted {
			if isToday {
				todayCount++
			}
			if task.Priority == model.P0 {
				p0Count++
			}
			if task.LifecycleState == model.StateActive {
				activeCount++
			}
		}
	}

	var rows []string

	brand := "▲ s t r e a m"
	ver := "v1.0.0"
	brandLine := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(brand)
	verLine := lipgloss.NewStyle().Foreground(t.Muted).Render(ver)
	rows = append(rows, brandLine, verLine, sep)

	// User Context Node
	userName := "Doan Huu Quoc"
	if m.DB != nil {
		userName = m.DB.GetUserSettings().Username
	}
	profileText := "👤 " + userName
	if len([]rune(profileText)) > innerW {
		profileText = "👤 " + string([]rune(userName)[:innerW-5]) + "..."
	}
	profileCard := lipgloss.NewStyle().
		Foreground(t.Fg).
		Padding(1, 1).
		Width(innerW).
		Render(profileText)
	rows = append(rows, profileCard, sep, "")

	// Workspaces
	rows = append(rows, renderSidebarWorkspaces(m, t, innerW)...)

	// Views
	rows = append(rows, renderSidebarViews(m, t, innerW, todayCount)...)
	rows = append(rows, "", sep, "")

	// Lifecycle
	rows = append(rows, renderSidebarLifecycle(m, t, innerW, p0Count, activeCount)...)

	// Footer
	rows = append(rows, renderSidebarFooter(m, t, innerW, appContentHeight, len(rows))...)

	return strings.Join(rows, "\n")
}

func renderSidebarWorkspaces(m *viewmodel.Model, t theme.Theme, innerW int) []string {
	var rows []string
	wsHeaderText := " WORKSPACES"
	wsHeaderColor := t.Muted
	if m.SidebarFocus {
		wsHeaderColor = t.Accent
	}
	if len(m.Workspaces) > 1 {
		activeIdx := 0
		for i, ws := range m.Workspaces {
			if ws.UUID == m.ActiveWorkspaceUUID {
				activeIdx = i
				break
			}
		}
		counterStr := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%d/%d  w/W", activeIdx+1, len(m.Workspaces)))
		headerLeft := lipgloss.NewStyle().Foreground(wsHeaderColor).Bold(m.SidebarFocus).Render(wsHeaderText)
		heatWL := lipgloss.Width(headerLeft)
		heatWR := lipgloss.Width(counterStr)
		heatPad := innerW - heatWL - heatWR
		if heatPad < 0 {
			heatPad = 0
		}
		rows = append(rows, headerLeft+strings.Repeat(" ", heatPad)+counterStr)
	} else {
		rows = append(rows, lipgloss.NewStyle().Foreground(wsHeaderColor).Bold(m.SidebarFocus).Render(wsHeaderText))
	}

	for i, ws := range m.Workspaces {
		wsIcon := ws.Icon
		if wsIcon == "" {
			wsIcon = "💼"
		}
		wsLabel := ws.Name
		wsBadge := ws.Badge

		isActive := ws.UUID == m.ActiveWorkspaceUUID

		cursor := "  "
		var rowStyle lipgloss.Style
		if isActive {
			cursorCol := t.Accent
			if !m.SidebarFocus {
				cursorCol = t.Muted
			}
			cursor = lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus).Render("›")
			rowStyle = lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus)
		} else {
			cursor = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%d", i+1))
			rowStyle = lipgloss.NewStyle().Foreground(t.Fg)
		}

		wsTextLeft := fmt.Sprintf(" %s %s %s", cursor, wsIcon, wsLabel)
		wsTextRight := wsBadge
		wsLeftW := lipgloss.Width(wsTextLeft)
		wsRightW := lipgloss.Width(wsTextRight)
		if wsLeftW+wsRightW+1 > innerW {
			maxLabelW := innerW - lipgloss.Width(" › 💼 ") - wsRightW - 2
			if maxLabelW > 3 {
				wsLabel = string([]rune(wsLabel)[:maxLabelW-3]) + "..."
			} else if maxLabelW > 0 {
				wsLabel = string([]rune(wsLabel)[:maxLabelW])
			}
			wsTextLeft = fmt.Sprintf(" %s %s %s", cursor, wsIcon, wsLabel)
			wsLeftW = lipgloss.Width(wsTextLeft)
		}
		wsPad := innerW - wsLeftW - wsRightW - 1
		if wsPad < 0 {
			wsPad = 0
		}
		wsRow := wsTextLeft + strings.Repeat(" ", wsPad) + wsTextRight + " "
		rows = append(rows, rowStyle.Width(innerW).Render(wsRow))
	}
	rows = append(rows, "")
	return rows
}

func renderSidebarViews(m *viewmodel.Model, t theme.Theme, innerW int, todayCount int) []string {
	var rows []string
	viewsHeaderColor := t.Muted
	if m.SidebarFocus {
		viewsHeaderColor = t.Accent
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(viewsHeaderColor).Bold(m.SidebarFocus).Render(" VIEWS"), "")

	type navItem struct {
		label string
		icon  string
		view  viewmodel.ViewType
	}
	items := []navItem{
		{"Dashboard", "", viewmodel.DashboardView},
		{"Month Grid", "󰸗", viewmodel.MonthView},
		{"Week Lanes", "󰸶", viewmodel.WeekView},
		{"Day Timeline", "󰸴", viewmodel.DayView},
		{"Analytics", "󰄫", viewmodel.AnalyticsView},
	}

	for _, item := range items {
		isSelected := m.CurrentView == item.view
		badgeStr := ""
		if item.view == viewmodel.DayView && todayCount > 0 {
			badgeStr = fmt.Sprintf("[%d]", todayCount)
		}

		var leftText string
		if isSelected {
			cursorCol := t.Accent
			if !m.SidebarFocus {
				cursorCol = t.Muted
			}
			cursor := lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus).Render("┃")
			leftText = fmt.Sprintf("%s%s %s", cursor, item.icon, item.label)
		} else {
			leftText = fmt.Sprintf(" %s %s", item.icon, item.label)
		}

		leftW := lipgloss.Width(leftText)
		rightW := lipgloss.Width(badgeStr)
		if leftW+rightW+1 > innerW {
			maxLabelW := innerW - lipgloss.Width("┃  ") - rightW - 2
			if maxLabelW > 3 {
				item.label = string([]rune(item.label)[:maxLabelW-3]) + "..."
			} else {
				item.label = string([]rune(item.label)[:maxLabelW])
			}
			if isSelected {
				cursorCol := t.Accent
				if !m.SidebarFocus {
					cursorCol = t.Muted
				}
				cursor := lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus).Render("┃")
				leftText = fmt.Sprintf("%s%s %s", cursor, item.icon, item.label)
			} else {
				leftText = fmt.Sprintf(" %s %s", item.icon, item.label)
			}
			leftW = lipgloss.Width(leftText)
		}

		spaceCount := innerW - leftW - rightW - 1
		if spaceCount < 0 {
			spaceCount = 0
		}
		rowText := leftText + strings.Repeat(" ", spaceCount) + badgeStr + " "

		var renderedRow string
		if isSelected {
			fgColor := lipgloss.Color("#ffffff")
			if !m.SidebarFocus {
				fgColor = t.Fg
			}
			renderedRow = lipgloss.NewStyle().
				Foreground(fgColor).
				Bold(m.SidebarFocus).
				Width(innerW).
				Render(rowText)
		} else {
			renderedRow = lipgloss.NewStyle().
				Foreground(t.Muted).
				Width(innerW).
				Render(rowText)
		}
		rows = append(rows, renderedRow)
	}
	return rows
}

func renderSidebarLifecycle(m *viewmodel.Model, t theme.Theme, innerW int, p0Count int, activeCount int) []string {
	var rows []string
	lifecycleHeaderColor := t.Muted
	if m.SidebarFocus {
		lifecycleHeaderColor = t.Accent
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(lifecycleHeaderColor).Bold(m.SidebarFocus).Render(" LIFECYCLE"), "")

	p0Bullet := lipgloss.NewStyle().Foreground(t.P0Color).Render("󰀦")
	p0TextLeft := fmt.Sprintf("  %s P0 Urgent", p0Bullet)
	p0TextRight := fmt.Sprintf("[%d]", p0Count)
	p0LeftW := lipgloss.Width(p0TextLeft)
	p0RightW := lipgloss.Width(p0TextRight)
	if p0LeftW+p0RightW+1 > innerW {
		p0TextLeft = fmt.Sprintf("  %s Urgent", p0Bullet)
		p0LeftW = lipgloss.Width(p0TextLeft)
	}
	p0Pad := innerW - p0LeftW - p0RightW - 1
	if p0Pad < 0 {
		p0Pad = 0
	}
	p0Row := p0TextLeft + strings.Repeat(" ", p0Pad) + p0TextRight + " "
	rows = append(rows, lipgloss.NewStyle().Foreground(t.Fg).Width(innerW).Render(p0Row))

	ipBullet := lipgloss.NewStyle().Foreground(t.P1Color).Render("󰄬")
	ipTextLeft := fmt.Sprintf("  %s In Progress", ipBullet)
	ipTextRight := fmt.Sprintf("[%d]", activeCount)
	ipLeftW := lipgloss.Width(ipTextLeft)
	ipRightW := lipgloss.Width(ipTextRight)
	if ipLeftW+ipRightW+1 > innerW {
		ipTextLeft = fmt.Sprintf("  %s Active", ipBullet)
		ipLeftW = lipgloss.Width(ipTextLeft)
	}
	ipPad := innerW - ipLeftW - ipRightW - 1
	if ipPad < 0 {
		ipPad = 0
	}
	ipRow := ipTextLeft + strings.Repeat(" ", ipPad) + ipTextRight + " "
	rows = append(rows, lipgloss.NewStyle().Foreground(t.Fg).Width(innerW).Render(ipRow))
	return rows
}

func renderSidebarFooter(m *viewmodel.Model, t theme.Theme, innerW int, appContentHeight int, occupied int) []string {
	var rows []string
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))
	muted := lipgloss.NewStyle().Foreground(t.Muted)

	// Summary of the day
	today := time.Now()
	todayCompletedCount := 0
	todayTotalCount := 0
	todayCompletedSP := 0
	todayTotalSP := 0
	todayFocusSeconds := 0

	for _, task := range m.Tasks {
		isToday := false
		if task.SchedulingType == model.Anchored {
			isToday = viewmodel.SameDay(task.TimeWindow.Start, today)
		} else {
			isToday = viewmodel.SameDay(task.CreatedAt, today)
		}

		if isToday {
			todayTotalCount++
			todayTotalSP += task.StoryPoints
			todayFocusSeconds += task.ExecutionMetrics.ElapsedFocusSeconds
			if task.LifecycleState == model.StateCompleted {
				todayCompletedCount++
				todayCompletedSP += task.StoryPoints
			}
		}
	}

	focusMins := todayFocusSeconds / 60
	summaryTitle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("  TODAY SUMMARY")

	tasksLabel := "  Tasks Completed"
	tasksVal := fmt.Sprintf("%d/%d", todayCompletedCount, todayTotalCount)
	tasksPad := innerW - lipgloss.Width(tasksLabel) - lipgloss.Width(tasksVal) - 1
	if tasksPad < 0 {
		tasksPad = 0
	}
	tasksRow := tasksLabel + strings.Repeat(" ", tasksPad) + tasksVal

	spLabel := "  Story Points"
	spVal := fmt.Sprintf("%d/%d SP", todayCompletedSP, todayTotalSP)
	spPad := innerW - lipgloss.Width(spLabel) - lipgloss.Width(spVal) - 1
	if spPad < 0 {
		spPad = 0
	}
	spRow := spLabel + strings.Repeat(" ", spPad) + spVal

	focusLabel := "  Focus Time"
	focusVal := fmt.Sprintf("%dh %dm", focusMins/60, focusMins%60)
	if focusMins < 60 {
		focusVal = fmt.Sprintf("%dm", focusMins)
	}
	focusPad := innerW - lipgloss.Width(focusLabel) - lipgloss.Width(focusVal) - 1
	if focusPad < 0 {
		focusPad = 0
	}
	focusRow := focusLabel + strings.Repeat(" ", focusPad) + focusVal

	// CPU & RAM Bars
	cpuPct := 20 + int(time.Now().Unix()%35)
	memPct := 45 + int(time.Now().Unix()%20)
	barW := innerW - 15
	if barW < 4 {
		barW = 4
	}

	// CPU Bar
	cpuSolid := cpuPct * barW / 100
	if cpuSolid < 0 {
		cpuSolid = 0
	}
	if cpuSolid > barW {
		cpuSolid = barW
	}
	cpuEmpty := barW - cpuSolid
	cpuBarStr := strings.Repeat("█", cpuSolid) + strings.Repeat("░", cpuEmpty)

	cpuLeftText := fmt.Sprintf("  CPU  [%s]", cpuBarStr)
	cpuRightText := fmt.Sprintf("%d%%", cpuPct)
	cpuLeftW := lipgloss.Width(cpuLeftText)
	cpuRightW := lipgloss.Width(cpuRightText)
	cpuSpaceCount := innerW - cpuLeftW - cpuRightW - 1
	if cpuSpaceCount < 0 {
		cpuSpaceCount = 0
	}
	cpuRow := cpuLeftText + strings.Repeat(" ", cpuSpaceCount) + cpuRightText + " "

	// RAM Bar
	solidCount := memPct * barW / 100
	if solidCount < 0 {
		solidCount = 0
	}
	if solidCount > barW {
		solidCount = barW
	}
	emptyCount := barW - solidCount
	barStr := strings.Repeat("█", solidCount) + strings.Repeat("░", emptyCount)

	leftText := fmt.Sprintf("  RAM  [%s]", barStr)
	rightText := fmt.Sprintf("%d%%", memPct)
	leftW := lipgloss.Width(leftText)
	rightW := lipgloss.Width(rightText)
	spaceCount := innerW - leftW - rightW - 1
	if spaceCount < 0 {
		spaceCount = 0
	}
	memRow := leftText + strings.Repeat(" ", spaceCount) + rightText + " "

	// Dynamic spacer calculation
	footOccupied := 11
	if m.ZenTimer != nil && m.ZenTimer.Running {
		footOccupied++
	}
	remaining := appContentHeight - (occupied + footOccupied + 9)
	if remaining > 0 {
		rows = append(rows, strings.Repeat("\n", remaining))
	}

	// Append Today Summary
	rows = append(rows, summaryTitle)
	rows = append(rows, muted.Render(tasksRow))
	rows = append(rows, muted.Render(spRow))
	rows = append(rows, muted.Render(focusRow))
	rows = append(rows, "")

	// Append CPU & RAM Bars
	rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render(cpuRow))
	rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render(memRow))

	rows = append(rows, sep)

	syncColor := t.Muted
	if m.Sync.IsOnline() {
		syncColor = t.SuccessColor
	}
	gcal := lipgloss.NewStyle().Foreground(syncColor).Render("● gcal")

	modeColor := t.Muted
	switch m.CurrentMode {
	case viewmodel.ModeZen:
		modeColor = t.FocusPurple
	case viewmodel.ModeCommand:
		modeColor = t.P1Color
	case viewmodel.ModeForm:
		modeColor = t.Accent
	case viewmodel.ModeWorkspaceForm:
		modeColor = t.Accent
	case viewmodel.ModeWorkspacePicker:
		modeColor = t.Accent
	case viewmodel.ModeSyncForm:
		modeColor = t.Accent
	}
	modeBadge := lipgloss.NewStyle().Foreground(modeColor).Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))
	clock := lipgloss.NewStyle().Foreground(t.Fg).Bold(true).
		Render(time.Now().Format("15:04"))

	if m.ZenTimer != nil && m.ZenTimer.Running {
		zt := m.ZenTimer
		hVal := int(zt.TimeRemaining.Hours())
		mVal := int(zt.TimeRemaining.Minutes()) % 60
		sVal := int(zt.TimeRemaining.Seconds()) % 60
		focusText := fmt.Sprintf("󱎫 FOCUS (%02d:%02d:%02d)", hVal, mVal, sVal)
		if zt.IsPaused {
			focusText = fmt.Sprintf("󱎫 PAUSED (%02d:%02d:%02d)", hVal, mVal, sVal)
		}
		focusStyle := lipgloss.NewStyle().
			Foreground(t.FocusPurple).
			Bold(true)
		rows = append(rows, "  "+focusStyle.Render(focusText))
	}

	rows = append(rows, modeBadge+"  "+gcal)
	rows = append(rows, clock)
	rows = append(rows, muted.Render("? help"))
	return rows
}
