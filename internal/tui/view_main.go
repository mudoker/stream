package tui

import (
	"strings"
	"time"

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

	l := m.Layout

	// ── Sidebar ─────────────────────────────────────────────────────
	sidebarStyle := lipgloss.NewStyle().
		Width(l.SidebarW).
		Height(l.Height).
		Background(m.Theme.PanelBg).
		BorderRight(true).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderForeground(lipgloss.Color("#2a2c37")).
		Padding(1, 1)

	// ── Workspace (non-day views use the full workspace width) ────────
	workspaceStyle := lipgloss.NewStyle().
		Width(l.WorkspaceW).
		Height(l.Height).
		Background(m.Theme.CanvasBg).
		Padding(1, 2)

	// ── Day View: three-column layout ────────────────────────────────
	timelineStyle := lipgloss.NewStyle().
		Width(l.TimelineW).
		Height(l.Height).
		Background(m.Theme.CanvasBg)

	todoStyle := lipgloss.NewStyle().
		Width(l.TodoW).
		Height(l.Height).
		Background(m.Theme.PanelBg).
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderForeground(lipgloss.Color("#2a2c37"))

	var canvas string

	if m.CurrentView == DayView {
		// Three-column layout: sidebar | timeline | todo shelf
		timelineContent := m.renderDayTimeline()
		todoContent := m.renderTodoShelf()

		// Overlay mini zen widget onto timeline if running
		if m.ZenTimer != nil && m.ZenTimer.Running {
			timelineContent = m.overlayMiniZen(timelineContent, l.TimelineW)
		}

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(m.renderArcSidebar()),
			timelineStyle.Render(timelineContent),
			todoStyle.Render(todoContent),
		)
	} else {
		// Two-column layout: sidebar | full workspace
		var workspaceContent string
		switch m.CurrentView {
		case DashboardView:
			workspaceContent = m.renderDashboard()
		case MonthView:
			workspaceContent = m.renderMonthView(l.Height)
		case WeekView:
			workspaceContent = m.renderWeekView(l.Height)
		case AnalyticsView:
			workspaceContent = m.renderAnalyticsView()
		}

		if m.ZenTimer != nil && m.ZenTimer.Running {
			workspaceContent = m.overlayMiniZen(workspaceContent, l.WorkspaceW)
		}

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(m.renderArcSidebar()),
			workspaceStyle.Render(workspaceContent),
		)
	}

	// Command Palette overlaid at the bottom
	if m.CurrentMode == ModeCommand {
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, m.renderCommandPalette())
	}

	// Centered floating modal over the full canvas
	if m.CurrentMode == ModeForm || m.PromptOpen || m.ReviewOpen || m.HelpOpen || m.DetailOpen {
		var modalStr string
		switch {
		case m.CurrentMode == ModeForm:
			modalStr = m.renderFormModal()
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
func (m Model) renderArcSidebar() string {
	l := m.Layout
	innerW := l.SidebarW - 2 // account for Padding(1,1)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", innerW))
	muted := lipgloss.NewStyle().Foreground(m.Theme.Muted)

	var rows []string

	// Wordmark
	rows = append(rows,
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲ stream"),
		muted.Render("workspace"),
		"",
		sep,
		"",
		muted.Bold(true).Render("VIEWS"),
		"",
	)

	type navItem struct {
		label string
		key   string
		view  ViewType
	}
	items := []navItem{
		{"Dashboard", "1", DashboardView},
		{"Month", "2", MonthView},
		{"Week", "3", WeekView},
		{"Day", "4", DayView},
		{"Analytics", "5", AnalyticsView},
	}

	for _, item := range items {
		if m.CurrentView == item.view {
			bar := lipgloss.Border{Left: "▎"}
			style := lipgloss.NewStyle().
				Background(m.Theme.SelectedBg).
				Foreground(m.Theme.Fg).
				Bold(true).
				Border(bar, false, false, false, true).
				BorderForeground(m.Theme.Accent).
				Padding(0, 1).
				Width(innerW - 1)
			keyStr := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(item.key)
			rows = append(rows, style.Render(keyStr+" "+item.label))
		} else {
			style := lipgloss.NewStyle().
				Foreground(m.Theme.Muted).
				Padding(0, 2).
				Width(innerW)
			rows = append(rows, style.Render(item.key+" "+item.label))
		}
	}

	rows = append(rows, "", sep, "")

	// Push footer to the bottom
	occupied := len(rows) + 4
	remaining := m.Height - occupied
	if remaining > 0 {
		rows = append(rows, strings.Repeat("\n", remaining))
	}

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
	}
	modeBadge := lipgloss.NewStyle().Foreground(modeColor).Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))
	clock := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).
		Render(time.Now().Format("15:04"))

	rows = append(rows, sep)
	rows = append(rows, modeBadge+"  "+gcal)
	rows = append(rows, clock)
	rows = append(rows, muted.Render("? help"))

	return strings.Join(rows, "\n")
}
