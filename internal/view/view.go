package view

import (
	"fmt"
	"strings"

	"stream/internal/view/components"
	"stream/internal/view/modals"
	"stream/internal/view/pages"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View struct {
	Model *viewmodel.Model
	Theme theme.Theme
}

func NewView(model *viewmodel.Model) *View {
	v := &View{
		Model: model,
		Theme: theme.NewTheme(),
	}
	model.ViewFunc = func(m *viewmodel.Model) string {
		return v.Render()
	}
	return v
}

func (v *View) Init() tea.Cmd {
	return v.Model.Init()
}

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := v.Model.Update(msg)
	if vm, ok := newModel.(*viewmodel.Model); ok {
		v.Model = vm
	}
	return v, cmd
}

func (v *View) View() string {
	return v.Render()
}

func (v *View) Render() string {
	m := v.Model

	if m.IsLocked {
		return modals.RenderLockScreen(m, v.Theme)
	}

	// Zen Mode is a full-screen takeover
	if m.CurrentMode == viewmodel.ModeZen {
		return pages.RenderZenMode(m, v.Theme)
	}

	// Calculate dynamic workspace height based on cmd palette presence to prevent bottom overflow
	cmdPaletteH := 0
	var cmdPaletteStr string
	if m.CurrentMode == viewmodel.ModeCommand {
		cmdPaletteStr = modals.RenderCommandPalette(m, v.Theme)
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
		sidebarBorderCol = v.Theme.Accent
	} else if m.CurrentView == viewmodel.DayView && !m.TodoShelfFocus { // Timeline is focused
		sidebarBorderCol = v.Theme.Accent
	}
	sidebarStyle := lipgloss.NewStyle().
		Width(l.SidebarW).
		Height(appContentHeight-2).
		MaxHeight(appContentHeight-2).
		BorderRight(true).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderForeground(sidebarBorderCol).
		Padding(1, 1)

	// ── Workspace (non-day views use the full workspace width) ────────
	workspaceStyle := lipgloss.NewStyle().
		Width(l.WorkspaceW).
		Height(appContentHeight-2).
		MaxHeight(appContentHeight-2).
		Padding(1, 2)

	// ── Day View: three-column layout ────────────────────────────────
	timelineStyle := lipgloss.NewStyle().
		Width(l.TimelineW).
		Height(appContentHeight).
		MaxHeight(appContentHeight)

	todoBorderCol := lipgloss.Color("#2a2c37")
	if m.TodoShelfFocus && !m.SidebarFocus {
		todoBorderCol = v.Theme.Accent
	} else if !m.SidebarFocus && !m.TodoShelfFocus { // Timeline is focused
		todoBorderCol = v.Theme.Accent
	}
	todoStyle := lipgloss.NewStyle().
		Width(l.TodoW).
		Height(appContentHeight - 2).
		MaxHeight(appContentHeight - 2).
		Padding(1, 1).
		BorderLeft(true).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderForeground(todoBorderCol)

	var canvas string

	if m.CurrentView == viewmodel.DayView {
		// Three-column layout: sidebar | timeline | todo shelf
		timelineContent := pages.RenderDayTimeline(m, v.Theme, appContentHeight)
		todoContent := components.RenderTodoShelf(m, v.Theme, appContentHeight)

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(components.RenderArcSidebar(m, v.Theme, appContentHeight-2)),
			timelineStyle.Render(timelineContent),
			todoStyle.Render(todoContent),
		)
	} else {
		// Two-column layout: sidebar | full workspace
		var workspaceContent string
		switch m.CurrentView {
		case viewmodel.DashboardView:
			workspaceContent = pages.RenderDashboard(m, v.Theme, appContentHeight-2)
		case viewmodel.MonthView:
			workspaceContent = pages.RenderMonthView(m, v.Theme, appContentHeight-2)
		case viewmodel.WeekView:
			workspaceContent = pages.RenderWeekView(m, v.Theme, appContentHeight-2)
		case viewmodel.AnalyticsView:
			workspaceContent = pages.RenderAnalyticsView(m, v.Theme, appContentHeight-2)
		}

		canvas = lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarStyle.Render(components.RenderArcSidebar(m, v.Theme, appContentHeight-2)),
			workspaceStyle.Render(workspaceContent),
		)
	}

	// Overlay mini-Zen timer if active
	canvas = v.overlayMiniZen(canvas, m.Width)

	// ── Status Bar ──────────────────────────────────────────────────
	statusBarW := m.Width
	modeStr := fmt.Sprintf(" %s ", m.CurrentMode)
	modeStyle := lipgloss.NewStyle().Background(v.Theme.Accent).Foreground(lipgloss.Color("#1e1e2e")).Bold(true)

	statusText := m.StatusMsg
	if statusText == "" {
		statusText = "Ready"
	}
	var statusStyle lipgloss.Style
	if strings.HasPrefix(statusText, "Go to:") {
		statusStyle = lipgloss.NewStyle().Foreground(v.Theme.Accent).Bold(true)
	} else if strings.Contains(statusText, "error") || strings.Contains(statusText, "Error") || strings.Contains(statusText, "fail") {
		statusStyle = lipgloss.NewStyle().Foreground(v.Theme.P0Color).Bold(true)
	} else {
		statusStyle = lipgloss.NewStyle().Foreground(v.Theme.Fg)
	}
	statusStr := statusStyle.Render("  " + statusText)

	onlineStr := " ○ Local "
	onlineColor := v.Theme.Muted
	if m.Sync != nil && m.Sync.IsOnline() {
		onlineStr = " ● GCal Sync "
		onlineColor = v.Theme.SuccessColor
	}
	syncStr := lipgloss.NewStyle().Foreground(onlineColor).Bold(true).Render(onlineStr)

	modeW := lipgloss.Width(modeStr)
	syncW := lipgloss.Width(onlineStr)
	msgW := statusBarW - modeW - syncW - 4
	if msgW < 10 {
		msgW = 10
	}

	statusRendered := statusStr
	statusPlainW := lipgloss.Width(statusRendered)
	if statusPlainW > msgW {
		statusRendered = theme.SliceAnsi(statusRendered, 0, msgW)
		statusPlainW = msgW
	}

	paddingW := statusBarW - modeW - syncW - statusPlainW
	if paddingW < 0 {
		paddingW = 0
	}

	statusBarStr := modeStyle.Render(modeStr) + statusRendered + strings.Repeat(" ", paddingW) + syncStr
	canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, statusBarStr)

	// Command Palette overlaid below status bar
	if m.CurrentMode == viewmodel.ModeCommand {
		canvas = lipgloss.JoinVertical(lipgloss.Left, canvas, cmdPaletteStr)
	}

	// Centered floating modal over the full canvas
	if m.WarningOpen || m.AuthNoticeOpen || m.CurrentMode == viewmodel.ModeForm || m.CurrentMode == viewmodel.ModeWorkspaceForm || m.CurrentMode == viewmodel.ModeWorkspacePicker || m.PromptOpen || m.ReviewOpen || m.HelpOpen || m.DetailOpen || m.ConfirmOpen || m.AnchorPromptOpen || m.CurrentMode == viewmodel.ModeProfileForm || m.CurrentMode == viewmodel.ModeSyncForm || m.SessionExpiryPromptOpen {
		var modalStr string
		switch {
		case m.WarningOpen:
			modalStr = modals.RenderWarningModal(m, v.Theme)
		case m.AuthNoticeOpen:
			modalStr = modals.RenderAuthNoticeModal(m, v.Theme)
		case m.SessionExpiryPromptOpen:
			modalStr = modals.RenderSessionExpiryModal(m, v.Theme)
		case m.ConfirmOpen:
			modalStr = modals.RenderConfirmModal(m, v.Theme)
		case m.AnchorPromptOpen:
			modalStr = modals.RenderAnchorPromptModal(m, v.Theme)
		case m.CurrentMode == viewmodel.ModeForm:
			modalStr = modals.RenderFormModal(m, v.Theme)
		case m.CurrentMode == viewmodel.ModeWorkspaceForm:
			modalStr = modals.RenderWorkspaceFormModal(m, v.Theme)
		case m.CurrentMode == viewmodel.ModeWorkspacePicker:
			modalStr = modals.RenderWorkspacePickerModal(m, v.Theme)
		case m.CurrentMode == viewmodel.ModeProfileForm:
			modalStr = modals.RenderProfileFormModal(m, v.Theme)
		case m.CurrentMode == viewmodel.ModeSyncForm:
			modalStr = modals.RenderSyncFormModal(m, v.Theme)
		case m.PromptOpen:
			modalStr = modals.RenderPromptModal(m, v.Theme)
		case m.ReviewOpen:
			modalStr = modals.RenderReviewModal(m, v.Theme)
		case m.HelpOpen:
			modalStr = modals.RenderHelpModal(m, v.Theme)
		case m.DetailOpen:
			modalStr = modals.RenderDetailModal(m, v.Theme)
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
