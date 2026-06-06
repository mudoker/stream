package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// View is the root Bubble Tea render function.
// It assembles the full-screen layout using JoinHorizontal with explicit Width()
// on every column so Lipgloss clips and pads deterministically.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing workspace..."
	}

	if m.IsLocked {
		return m.renderLockScreen()
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

	appContentHeight := m.Height - cmdPaletteH - 1
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
		Height(appContentHeight - 2).
		MaxHeight(appContentHeight - 2).
		BorderRight(true).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderForeground(sidebarBorderCol).
		Padding(1, 1)

	// ── Workspace (non-day views use the full workspace width) ────────
	workspaceStyle := lipgloss.NewStyle().
		Width(l.WorkspaceW).
		Height(appContentHeight - 2).
		MaxHeight(appContentHeight - 2).
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
			sidebarStyle.Render(m.renderArcSidebar(appContentHeight - 2)),
			timelineStyle.Render(timelineContent),
			todoStyle.Render(todoContent),
		)
	} else {
		// Two-column layout: sidebar | full workspace
		var workspaceContent string
		switch m.CurrentView {
		case DashboardView:
			workspaceContent = m.renderDashboard(appContentHeight - 2)
		case MonthView:
			workspaceContent = m.renderMonthView(appContentHeight - 2)
		case WeekView:
			workspaceContent = m.renderWeekView(appContentHeight - 2)
		case AnalyticsView:
			workspaceContent = m.renderAnalyticsView(appContentHeight - 2)
		case SettingsView:
			workspaceContent = m.renderSettingsView(appContentHeight - 2)
		}

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(m.renderArcSidebar(appContentHeight - 2)),
			workspaceStyle.Render(workspaceContent),
		)
	}

	// Command Palette overlaid at the bottom
	if m.CurrentMode == ModeCommand {
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdPaletteStr)
	}

	// Centered floating modal over the full canvas
	if m.CurrentMode == ModeForm || m.CurrentMode == ModeWorkspaceForm || m.CurrentMode == ModeWorkspacePicker || m.PromptOpen || m.ReviewOpen || m.HelpOpen || m.DetailOpen || m.ConfirmOpen || m.AnchorPromptOpen || m.CurrentMode == ModeProfileForm || m.SessionExpiryPromptOpen {
		var modalStr string
		switch {
		case m.SessionExpiryPromptOpen:
			modalStr = m.renderSessionExpiryModal()
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
		case m.CurrentMode == ModeProfileForm:
			modalStr = m.renderProfileFormModal()
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
