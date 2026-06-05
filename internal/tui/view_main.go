package tui

import (
	"strings"
	"time"

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
