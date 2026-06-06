package tui

import (
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSettingsView(height int) string {
	workspaceWidth := m.Layout.WorkspaceW - 4

	// 1. Page Header
	var headerTitle string
	if !m.SidebarFocus {
		headerTitle = lipgloss.NewStyle().
			Foreground(m.Theme.Accent).
			Bold(true).
			Render("⚙️  SETTINGS & CONFIGURATION CONSOLE")
	} else {
		headerTitle = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render("⚙️  SETTINGS & CONFIGURATION CONSOLE")
	}

	headerLine := headerTitle + "\n" + lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", workspaceWidth))

	// Find active workspace
	var activeWS model.Workspace
	for _, ws := range m.Workspaces {
		if ws.UUID == m.ActiveWorkspaceUUID {
			activeWS = ws
			break
		}
	}

	// 2. Budget Grid Heights mathematically
	// Total available visual height of the main window area is height - 4 (reserves space for header + spacing)
	gridH := height - 6
	if gridH < 12 {
		gridH = 12
	}
	row1H := gridH / 2
	row2H := gridH - row1H

	panelW := (workspaceWidth - 6) / 2
	if panelW < 24 {
		panelW = 24
	}

	// 3. Assemble Grid
	card1 := m.renderSyncCard(panelW, row1H)
	card2 := m.renderWorkspaceCard(panelW, row1H, activeWS)
	card3 := m.renderActivityCard(panelW, row2H)
	card4 := m.renderTelemetryCard(panelW, row2H)

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, card3, "  ", card4)

	content := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		"",
		row1,
		"",
		row2,
	)

	return content
}

func padLines(s string, count int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < count {
		lines = append(lines, "")
	}
	if len(lines) > count {
		lines = lines[:count]
	}
	return strings.Join(lines, "\n")
}

