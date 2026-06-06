package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

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

func (m Model) renderSettingsCard(title string, content string, width int) string {
	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, false, true).
		BorderForeground(m.Theme.SelectedBg).
		Background(m.Theme.SelectedBg).
		Foreground(m.Theme.Accent).
		Bold(true).
		Width(width).
		Padding(0, 1)

	header := headerStyle.Render(" " + strings.ToUpper(title))

	bodyStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, true, true).
		BorderForeground(m.Theme.SelectedBg).
		Padding(1, 2).
		Width(width)

	return lipgloss.JoinVertical(lipgloss.Left, header, bodyStyle.Render(content))
}

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
	if panelW < 24 {
		panelW = 24
	}

	// Labels with aligned vertical separators
	lblStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted).Width(13)
	valStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
	cmdStyle := lipgloss.NewStyle().Background(m.Theme.SelectedBg).Foreground(m.Theme.Accent).Bold(true)

	// ── CARD 1: Google Calendar Sync ──
	syncStatus := "● Disconnected"
	statusColor := m.Theme.P0Color
	if m.Sync.IsOnline() {
		syncStatus = "● Connected"
		statusColor = m.Theme.SuccessColor
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
	
	var sbSync strings.Builder
	sbSync.WriteString(fmt.Sprintf("%s  %s\n", lblStyle.Render("Status        │"), statusStyle.Render(syncStatus)))
	sbSync.WriteString(fmt.Sprintf("%s  %s\n", lblStyle.Render("Client ID     │"), valStyle.Render("stream-gcal-client")))
	sbSync.WriteString(fmt.Sprintf("%s  %s\n\n", lblStyle.Render("API Server    │"), valStyle.Render("http://localhost:8080")))
	
	sbSync.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("COMMANDS\n"))
	sbSync.WriteString(fmt.Sprintf("  %-12s  %s\n", cmdStyle.Render(" :auth "), valStyle.Render("Authenticate GCal API")))
	sbSync.WriteString(fmt.Sprintf("  %-12s  %s", cmdStyle.Render(" :sync "), valStyle.Render("Force background sync")))

	// ── CARD 2: Active Workspace ──
	var sbWS strings.Builder
	sbWS.WriteString(fmt.Sprintf("%s  %s %s\n", lblStyle.Render("Workspace     │"), valStyle.Render(activeWS.Icon), valStyle.Bold(true).Render(activeWS.Name)))
	sbWS.WriteString(fmt.Sprintf("%s  %s\n", lblStyle.Render("Badge         │"), valStyle.Render(activeWS.Badge)))
	
	uuidStr := activeWS.UUID
	if len(uuidStr) > panelW-18 {
		uuidStr = uuidStr[:panelW-21] + "..."
	}
	sbWS.WriteString(fmt.Sprintf("%s  %s\n\n", lblStyle.Render("UUID          │"), valStyle.Render(uuidStr)))

	sbWS.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("COMMANDS\n"))
	sbWS.WriteString(fmt.Sprintf("  %-12s  %s\n", cmdStyle.Render(" :ws-create "), valStyle.Render("Create new workspace")))
	sbWS.WriteString(fmt.Sprintf("  %-12s  %s\n", cmdStyle.Render(" :ws-edit   "), valStyle.Render("Edit active workspace")))
	sbWS.WriteString(fmt.Sprintf("  %-12s  %s", cmdStyle.Render(" :ws-delete "), valStyle.Render("Delete active workspace")))

	// ── CARD 3: Theme & Color Scheme ──
	var sbTheme strings.Builder
	sbTheme.WriteString(fmt.Sprintf("%s  %s\n", lblStyle.Render("Active Theme  │"), valStyle.Render("Catppuccin Mocha")))
	sbTheme.WriteString(fmt.Sprintf("%s  %s  %s\n", lblStyle.Render("Accent Color  │"), lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("████"), valStyle.Render("#89b4fa")))
	sbTheme.WriteString(fmt.Sprintf("%s  %s  %s\n", lblStyle.Render("Success Color │"), lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("████"), valStyle.Render("#a6e3a1")))
	sbTheme.WriteString(fmt.Sprintf("%s  %s  %s\n", lblStyle.Render("Zen Purple    │"), lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render("████"), valStyle.Render("#b4befe")))
	sbTheme.WriteString(fmt.Sprintf("%s  %s  %s\n", lblStyle.Render("P0 Urgent     │"), lipgloss.NewStyle().Foreground(m.Theme.P0Color).Render("████"), valStyle.Render("#f38ba8")))
	sbTheme.WriteString(fmt.Sprintf("%s  %s  %s", lblStyle.Render("P1 High       │"), lipgloss.NewStyle().Foreground(m.Theme.P1Color).Render("████"), valStyle.Render("#fab387")))

	// ── CARD 4: Database & Paths ──
	configDir := m.DB.GetConfigDir()
	pathStyle := lipgloss.NewStyle().
		Background(m.Theme.SelectedBg).
		Foreground(m.Theme.Accent).
		Padding(0, 1)

	truncPath := func(path string, maxLen int) string {
		if len(path) > maxLen {
			return "..." + path[len(path)-maxLen+3:]
		}
		return path
	}

	pathMaxLen := panelW - 22
	if pathMaxLen < 15 {
		pathMaxLen = 15
	}

	cfgPath := truncPath(configDir, pathMaxLen)
	dataPath := truncPath(filepath.Join(configDir, "data.json"), pathMaxLen)
	wsPath := truncPath(filepath.Join(configDir, "workspaces.json"), pathMaxLen)
	ledgPath := truncPath(filepath.Join(configDir, "ledger.json"), pathMaxLen)

	renderPathLine := func(label string, path string) string {
		lbl := lblStyle.Render(label + "  │")
		pathRend := pathStyle.Render(path)
		pathW := lipgloss.Width(pathRend)
		
		// inner width is panelW - 4 (left/right padding of body)
		innerW := panelW - 4
		copyIcon := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("[📋]")
		copyW := lipgloss.Width(copyIcon)
		
		leftW := lipgloss.Width(lbl) + 2 + pathW
		spaceCount := innerW - leftW - copyW
		if spaceCount < 1 {
			spaceCount = 1
		}
		return fmt.Sprintf("%s  %s%s%s", lbl, pathRend, strings.Repeat(" ", spaceCount), copyIcon)
	}

	var sbDB strings.Builder
	sbDB.WriteString(renderPathLine("Config Dir", cfgPath) + "\n")
	sbDB.WriteString(renderPathLine("Data File", dataPath) + "\n")
	sbDB.WriteString(renderPathLine("Workspaces", wsPath) + "\n")
	sbDB.WriteString(renderPathLine("Ledger File", ledgPath))

	// 3. Assemble Grid with padded lines for perfect height matching
	card1 := m.renderSettingsCard("Google Calendar Sync", padLines(sbSync.String(), 7), panelW)
	card2 := m.renderSettingsCard("Active Workspace", padLines(sbWS.String(), 7), panelW)
	card3 := m.renderSettingsCard("Theme & Color Scheme", padLines(sbTheme.String(), 7), panelW)
	card4 := m.renderSettingsCard("Database & Ledger Paths", padLines(sbDB.String(), 7), panelW)

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
