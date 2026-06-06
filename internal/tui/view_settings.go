package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSettingsView(height int) string {
	workspaceWidth := m.Layout.WorkspaceW - 4
	appContentHeight := height - 4 // spacing for header and footer
	if appContentHeight < 10 {
		appContentHeight = 10
	}

	// 1. Page Header
	var headerTitle string
	if !m.SidebarFocus {
		headerTitle = lipgloss.NewStyle().
			Foreground(m.Theme.Accent).
			Bold(true).
			Render("⚙️  SETTINGS & CONFIGURATION")
	} else {
		headerTitle = lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Bold(true).
			Render("⚙️  SETTINGS & CONFIGURATION")
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

	// 2. Prepare Panels
	panelW := (workspaceWidth - 2) / 2
	if panelW < 20 {
		panelW = 20
	}

	// ── PANEL 1: Google Calendar Sync ──
	syncStatus := "Disconnected"
	statusColor := m.Theme.P0Color
	if m.Sync.IsOnline() {
		syncStatus = "Connected (Online)"
		statusColor = m.Theme.SuccessColor
	}
	
	var sbSync strings.Builder
	sbSync.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("GOOGLE CALENDAR SYNC") + "\n\n")
	sbSync.WriteString(fmt.Sprintf("  Status:     %s\n", lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(syncStatus)))
	sbSync.WriteString(fmt.Sprintf("  Client ID:  %s\n", "stream-gcal-client"))
	sbSync.WriteString(fmt.Sprintf("  API Server: http://localhost:8080\n\n"))
	sbSync.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  Commands:\n"))
	sbSync.WriteString("   • :auth  - Authenticate with GCal\n")
	sbSync.WriteString("   • :sync  - Force background sync\n")
	
	panel1 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.SelectedBg).
		Padding(1, 2).
		Width(panelW).
		Height(10).
		Render(sbSync.String())

	// ── PANEL 2: Active Workspace ──
	var sbWS strings.Builder
	sbWS.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("ACTIVE WORKSPACE") + "\n\n")
	sbWS.WriteString(fmt.Sprintf("  Name:   %s %s\n", activeWS.Icon, activeWS.Name))
	sbWS.WriteString(fmt.Sprintf("  Badge:  %s\n", activeWS.Badge))
	sbWS.WriteString(fmt.Sprintf("  UUID:   %s\n\n", activeWS.UUID))
	sbWS.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  Commands:\n"))
	sbWS.WriteString("   • :ws-create - Add workspace\n")
	sbWS.WriteString("   • :ws-edit   - Edit workspace\n")
	sbWS.WriteString("   • :ws-delete - Delete workspace\n")

	panel2 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.SelectedBg).
		Padding(1, 2).
		Width(panelW).
		Height(10).
		Render(sbWS.String())

	// ── PANEL 3: Theme & UI Styling ──
	var sbTheme strings.Builder
	sbTheme.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("THEME & COLOR SCHEME") + "\n\n")
	sbTheme.WriteString("  Active Theme: Catppuccin Mocha\n")
	sbTheme.WriteString(fmt.Sprintf("  Palette Preview:\n"))
	sbTheme.WriteString(fmt.Sprintf("   • Success:  %s\n", lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("████")))
	sbTheme.WriteString(fmt.Sprintf("   • Accent:   %s\n", lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("████")))
	sbTheme.WriteString(fmt.Sprintf("   • P0/P1:    %s  %s\n", 
		lipgloss.NewStyle().Foreground(m.Theme.P0Color).Render("████"),
		lipgloss.NewStyle().Foreground(m.Theme.P1Color).Render("████")))

	panel3 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.SelectedBg).
		Padding(1, 2).
		Width(panelW).
		Height(10).
		Render(sbTheme.String())

	// ── PANEL 4: Database & Paths ──
	configDir := m.DB.GetConfigDir()
	var sbDB strings.Builder
	sbDB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("DATABASE & LEDGER PATHS") + "\n\n")
	
	// Truncate paths if they are too long for panel width
	truncPath := func(path string, maxLen int) string {
		if len(path) > maxLen {
			return "..." + path[len(path)-maxLen+3:]
		}
		return path
	}
	pathMaxW := panelW - 16
	sbDB.WriteString(fmt.Sprintf("  Config Dir: %s\n", truncPath(configDir, pathMaxW)))
	sbDB.WriteString(fmt.Sprintf("  Data File:  %s\n", truncPath(filepath.Join(configDir, "data.json"), pathMaxW)))
	sbDB.WriteString(fmt.Sprintf("  Workspaces: %s\n", truncPath(filepath.Join(configDir, "workspaces.json"), pathMaxW)))
	sbDB.WriteString(fmt.Sprintf("  Ledger:     %s\n", truncPath(filepath.Join(configDir, "ledger.json"), pathMaxW)))

	panel4 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.SelectedBg).
		Padding(1, 2).
		Width(panelW).
		Height(10).
		Render(sbDB.String())

	// Assemble rows
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, panel1, "  ", panel2)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, panel3, "  ", panel4)

	// Combine everything
	content := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		"",
		row1,
		"",
		row2,
	)

	return content
}
