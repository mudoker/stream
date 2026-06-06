package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// View is the root Bubble Tea render function.
// It assembles the full-screen layout using JoinHorizontal with explicit Width()
// on every column so Lipgloss clips and pads deterministically.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing workspace..."
	}

	// Zen Mode is a full-screen takeover
	if m.CurrentMode == ModeZen {
		return m.renderZenMode()
	}

	// Calculate dynamic workspace height based on cmd palette presence to prevent bottom overflow
	cmdPaletteH := 0
	var cmdPaletteStr string
	if m.CurrentMode == ModeCommand {
		cmdPaletteStr = m.renderCommandPalette()
		cmdPaletteH = lipgloss.Height(cmdPaletteStr)
	}

	appContentHeight := m.Height - cmdPaletteH
	if appContentHeight < 10 {
		appContentHeight = 10
	}

	l := m.Layout

	// ── Sidebar ─────────────────────────────────────────────────────
	sidebarBorderCol := lipgloss.Color("#2a2c37")
	if m.SidebarFocus {
		sidebarBorderCol = m.Theme.Accent
	} else if m.CurrentView == DayView && !m.TodoShelfFocus { // Timeline is focused
		sidebarBorderCol = m.Theme.Accent
	}
	sidebarStyle := lipgloss.NewStyle().
		Width(l.SidebarW).
		Height(appContentHeight).
		MaxHeight(appContentHeight).
		BorderRight(true).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderForeground(sidebarBorderCol).
		Padding(1, 1)

	// ── Workspace (non-day views use the full workspace width) ────────
	workspaceStyle := lipgloss.NewStyle().
		Width(l.WorkspaceW).
		Height(appContentHeight).
		MaxHeight(appContentHeight).
		Padding(1, 2)

	// ── Day View: three-column layout ────────────────────────────────
	timelineStyle := lipgloss.NewStyle().
		Width(l.TimelineW).
		Height(appContentHeight).
		MaxHeight(appContentHeight)

	todoBorderCol := lipgloss.Color("#2a2c37")
	if m.TodoShelfFocus && !m.SidebarFocus {
		todoBorderCol = m.Theme.Accent
	} else if !m.SidebarFocus && !m.TodoShelfFocus { // Timeline is focused
		todoBorderCol = m.Theme.Accent
	}
	todoStyle := lipgloss.NewStyle().
		Width(l.TodoW).
		Height(appContentHeight).
		MaxHeight(appContentHeight).
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderForeground(todoBorderCol)

	var canvas string

	if m.CurrentView == DayView {
		// Three-column layout: sidebar | timeline | todo shelf
		timelineContent := m.renderDayTimeline(appContentHeight)
		todoContent := m.renderTodoShelf(appContentHeight)

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(m.renderArcSidebar(appContentHeight)),
			timelineStyle.Render(timelineContent),
			todoStyle.Render(todoContent),
		)
	} else {
		// Two-column layout: sidebar | full workspace
		var workspaceContent string
		switch m.CurrentView {
		case DashboardView:
			workspaceContent = m.renderDashboard(appContentHeight)
		case MonthView:
			workspaceContent = m.renderMonthView(appContentHeight)
		case WeekView:
			workspaceContent = m.renderWeekView(appContentHeight)
		case AnalyticsView:
			workspaceContent = m.renderAnalyticsView(appContentHeight)
		case SettingsView:
			workspaceContent = m.renderSettingsView(appContentHeight)
		}

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(m.renderArcSidebar(appContentHeight)),
			workspaceStyle.Render(workspaceContent),
		)
	}

	// Command Palette overlaid at the bottom
	if m.CurrentMode == ModeCommand {
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdPaletteStr)
	}

	// Centered floating modal over the full canvas
	if m.CurrentMode == ModeForm || m.CurrentMode == ModeWorkspaceForm || m.CurrentMode == ModeWorkspacePicker || m.PromptOpen || m.ReviewOpen || m.HelpOpen || m.DetailOpen || m.ConfirmOpen || m.AnchorPromptOpen {
		var modalStr string
		switch {
		case m.ConfirmOpen:
			modalStr = m.renderConfirmModal()
		case m.AnchorPromptOpen:
			modalStr = m.renderAnchorPromptModal()
		case m.CurrentMode == ModeForm:
			modalStr = m.renderFormModal()
		case m.CurrentMode == ModeWorkspaceForm:
			modalStr = m.renderWorkspaceFormModal()
		case m.CurrentMode == ModeWorkspacePicker:
			modalStr = m.renderWorkspacePickerModal()
		case m.PromptOpen:
			modalStr = m.renderPromptModal()
		case m.ReviewOpen:
			modalStr = m.renderReviewModal()
		case m.HelpOpen:
			modalStr = m.renderHelpModal()
		case m.DetailOpen:
			modalStr = m.renderDetailModal()
		}

		modalW := lipgloss.Width(modalStr)
		modalH := lipgloss.Height(modalStr)
		topPad := (m.Height - modalH) / 2
		leftPad := (m.Width - modalW) / 2
		if topPad < 0 {
			topPad = 0
		}
		if leftPad < 0 {
			leftPad = 0
		}
		canvas = overlayString(canvas, modalStr, leftPad, topPad, m.Width)
	}

	return canvas
}

// renderArcSidebar renders the left navigation panel.
func (m Model) renderArcSidebar(appContentHeight int) string {
	l := m.Layout
	innerW := l.SidebarW - 2 // account for Padding(1,1)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))
	muted := lipgloss.NewStyle().Foreground(m.Theme.Muted)

	// Calculate counts dynamically
	today := time.Now()
	todayCount := 0
	p0Count := 0
	activeCount := 0
	for _, t := range m.Tasks {
		isToday := false
		if t.SchedulingType == model.Anchored {
			isToday = sameDay(t.TimeWindow.Start, today)
		} else {
			isToday = sameDay(t.CreatedAt, today)
		}

		if t.LifecycleState != model.StateCompleted {
			if isToday {
				todayCount++
			}
			if t.Priority == model.P0 {
				p0Count++
			}
			if t.LifecycleState == model.StateActive {
				activeCount++
			}
		}
	}

	var rows []string

	brand := "▲ s t r e a m"
	ver := "v0.4.2"
	brandLine := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(brand)
	verLine := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(ver)
	rows = append(rows, brandLine, verLine, sep)

	// User Context Node (with exactly 1 cell padding)
	userName := "Doan Huu Quoc"
	profileText := "👤 " + userName
	if len([]rune(profileText)) > innerW {
		profileText = "👤 " + string([]rune(userName)[:innerW-5]) + "..."
	}
	profileCard := lipgloss.NewStyle().
		Foreground(m.Theme.Fg).
		Padding(1, 1).
		Width(innerW).
		Render(profileText)
	rows = append(rows, profileCard, sep, "")

	// WORKSPACES Section Divider (1 leading space)
	// Show index counter when multiple workspaces exist
	wsHeaderText := " WORKSPACES"
	wsHeaderColor := m.Theme.Muted
	if m.SidebarFocus {
		wsHeaderColor = m.Theme.Accent
	}
	if len(m.Workspaces) > 1 {
		activeIdx := 0
		for i, ws := range m.Workspaces {
			if ws.UUID == m.ActiveWorkspaceUUID {
				activeIdx = i
				break
			}
		}
		counterStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%d/%d  w/W", activeIdx+1, len(m.Workspaces)))
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

		// Active cursor: ›, inactive: space-aligned
		cursor := "  "
		var rowStyle lipgloss.Style
		if isActive {
			cursorCol := m.Theme.Accent
			if !m.SidebarFocus {
				cursorCol = m.Theme.Muted
			}
			cursor = lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus).Render("›")
			rowStyle = lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus)
		} else {
			cursor = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%d", i+1))
			rowStyle = lipgloss.NewStyle().Foreground(m.Theme.Fg)
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

	// VIEWS Category Divider (1 leading space)
	viewsHeaderColor := m.Theme.Muted
	if m.SidebarFocus {
		viewsHeaderColor = m.Theme.Accent
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(viewsHeaderColor).Bold(m.SidebarFocus).Render(" VIEWS"), "")

	type navItem struct {
		label string
		icon  string
		view  ViewType
	}
	items := []navItem{
		{"Dashboard", "", DashboardView},
		{"Month Grid", "󰸗", MonthView},
		{"Week Lanes", "󰸶", WeekView},
		{"Day Timeline", "󰸴", DayView},
		{"Analytics", "󰄫", AnalyticsView},
		{"Settings", "⚙️", SettingsView},
	}

	for _, item := range items {
		isSelected := m.CurrentView == item.view
		badgeStr := ""
		if item.view == DayView && todayCount > 0 {
			badgeStr = fmt.Sprintf("[%d]", todayCount)
		}

		// Selected prefix is 1 char: "┃", unselected is " " (1 space) for 1-char left margin
		var leftText string
		if isSelected {
			cursorCol := m.Theme.Accent
			if !m.SidebarFocus {
				cursorCol = m.Theme.Muted
			}
			cursor := lipgloss.NewStyle().Foreground(cursorCol).Bold(m.SidebarFocus).Render("┃")
			leftText = fmt.Sprintf("%s%s %s", cursor, item.icon, item.label)
		} else {
			leftText = fmt.Sprintf(" %s %s", item.icon, item.label)
		}

		leftW := lipgloss.Width(leftText)
		rightW := lipgloss.Width(badgeStr)
		if leftW+rightW+1 > innerW {
			// Truncate menu label to prevent wrapping
			maxLabelW := innerW - lipgloss.Width("┃  ") - rightW - 2
			if maxLabelW > 3 {
				item.label = string([]rune(item.label)[:maxLabelW-3]) + "..."
			} else {
				item.label = string([]rune(item.label)[:maxLabelW])
			}
			if isSelected {
				cursorCol := m.Theme.Accent
				if !m.SidebarFocus {
					cursorCol = m.Theme.Muted
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
				fgColor = m.Theme.Fg
			}
			renderedRow = lipgloss.NewStyle().
				Foreground(fgColor).
				Bold(m.SidebarFocus).
				Width(innerW).
				Render(rowText)
		} else {
			renderedRow = lipgloss.NewStyle().
				Foreground(m.Theme.Muted). // Dimmed 40% contrast opacity step
				Width(innerW).
				Render(rowText)
		}
		rows = append(rows, renderedRow)
	}

	rows = append(rows, "", sep, "")

	// LIFECYCLE Category Divider (1 leading space)
	lifecycleHeaderColor := m.Theme.Muted
	if m.SidebarFocus {
		lifecycleHeaderColor = m.Theme.Accent
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(lifecycleHeaderColor).Bold(m.SidebarFocus).Render(" LIFECYCLE"), "")

	p0Bullet := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Render("󰀦")
	p0TextLeft := fmt.Sprintf("  %s P0 Urgent", p0Bullet) // 2 leading spaces for icon alignment
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
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Fg).Width(innerW).Render(p0Row))

	ipBullet := lipgloss.NewStyle().Foreground(m.Theme.P1Color).Render("󰄬")
	ipTextLeft := fmt.Sprintf("  %s In Progress", ipBullet) // 2 leading spaces for icon alignment
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
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Fg).Width(innerW).Render(ipRow))

	// Push footer to the bottom
	occupied := len(rows) + 9
	if m.ZenTimer != nil && m.ZenTimer.Running {
		occupied++
	}
	remaining := appContentHeight - occupied
	if remaining > 0 {
		rows = append(rows, strings.Repeat("\n", remaining))
	}

	// System Resource Metrics Footer
	memPct := 45 + int(time.Now().Unix()%20)
	barW := innerW - 15
	if barW < 4 {
		barW = 4
	}
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
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(memRow))

	// Sticky Settings
	rows = append(rows, sep)

	// Footer: mode badge + gcal + clock
	syncColor := m.Theme.Muted
	if m.Sync.IsOnline() {
		syncColor = m.Theme.SuccessColor
	}
	gcal := lipgloss.NewStyle().Foreground(syncColor).Render("● gcal")

	modeColor := m.Theme.Muted
	switch m.CurrentMode {
	case ModeZen:
		modeColor = m.Theme.FocusPurple
	case ModeCommand:
		modeColor = m.Theme.P1Color
	case ModeForm:
		modeColor = m.Theme.Accent
	case ModeWorkspaceForm:
		modeColor = m.Theme.Accent
	case ModeWorkspacePicker:
		modeColor = m.Theme.Accent
	}
	modeBadge := lipgloss.NewStyle().Foreground(modeColor).Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))
	clock := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).
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
			Foreground(m.Theme.FocusPurple).
			Bold(true)
		rows = append(rows, "  "+focusStyle.Render(focusText))
	}

	rows = append(rows, modeBadge+"  "+gcal)
	rows = append(rows, clock)
	rows = append(rows, muted.Render("? help"))

	return strings.Join(rows, "\n")
}
