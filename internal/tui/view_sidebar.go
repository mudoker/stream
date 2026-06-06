package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderArcSidebar(appContentHeight int) string {
	l := m.Layout
	innerW := l.SidebarW - 2
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))

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
		Foreground(m.Theme.Fg).
		Padding(1, 1).
		Width(innerW).
		Render(profileText)
	rows = append(rows, profileCard, sep, "")

	// Workspaces
	rows = append(rows, m.renderSidebarWorkspaces(innerW)...)

	// Views
	rows = append(rows, m.renderSidebarViews(innerW, todayCount)...)
	rows = append(rows, "", sep, "")

	// Lifecycle
	rows = append(rows, m.renderSidebarLifecycle(innerW, p0Count, activeCount)...)

	// Footer
	rows = append(rows, m.renderSidebarFooter(innerW, appContentHeight, len(rows))...)

	return strings.Join(rows, "\n")
}

func (m Model) renderSidebarWorkspaces(innerW int) []string {
	var rows []string
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
	return rows
}

func (m Model) renderSidebarViews(innerW int, todayCount int) []string {
	var rows []string
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
	}

	for _, item := range items {
		isSelected := m.CurrentView == item.view
		badgeStr := ""
		if item.view == DayView && todayCount > 0 {
			badgeStr = fmt.Sprintf("[%d]", todayCount)
		}

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
				Foreground(m.Theme.Muted).
				Width(innerW).
				Render(rowText)
		}
		rows = append(rows, renderedRow)
	}
	return rows
}

func (m Model) renderSidebarLifecycle(innerW int, p0Count int, activeCount int) []string {
	var rows []string
	lifecycleHeaderColor := m.Theme.Muted
	if m.SidebarFocus {
		lifecycleHeaderColor = m.Theme.Accent
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(lifecycleHeaderColor).Bold(m.SidebarFocus).Render(" LIFECYCLE"), "")

	p0Bullet := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Render("󰀦")
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
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Fg).Width(innerW).Render(p0Row))

	ipBullet := lipgloss.NewStyle().Foreground(m.Theme.P1Color).Render("󰄬")
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
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Fg).Width(innerW).Render(ipRow))
	return rows
}


