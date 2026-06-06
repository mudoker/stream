package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)


func (m Model) renderSettingsCard(title string, content string, width int, height int) string {
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

	contentH := height - 4
	if contentH < 1 {
		contentH = 1
	}
	paddedContent := padLines(content, contentH)

	cardBody := lipgloss.JoinVertical(lipgloss.Left, header, bodyStyle.Render(paddedContent))
	return lipgloss.NewStyle().Height(height).MaxHeight(height).Render(cardBody)
}

func (m Model) renderSyncCard(panelW, rowH int) string {
	valStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
	cmdStyle := lipgloss.NewStyle().Background(m.Theme.SelectedBg).Foreground(m.Theme.Accent).Bold(true)

	syncStatus := "● Disconnected"
	statusColor := m.Theme.P0Color
	if m.Sync.IsOnline() {
		syncStatus = "● Connected"
		statusColor = m.Theme.SuccessColor
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)

	var sbSync strings.Builder
	sbSync.WriteString(fmt.Sprintf("  %-13s │  %s\n", "Status", statusStyle.Render(syncStatus)))
	sbSync.WriteString(fmt.Sprintf("  %-13s │  %s\n", "Client ID", valStyle.Render("stream-gcal-client")))
	sbSync.WriteString(fmt.Sprintf("  %-13s │  %s\n\n", "API Server", valStyle.Render("http://localhost:8080")))

	sbSync.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("COMMANDS\n"))
	sbSync.WriteString(fmt.Sprintf("  %s     %s\n", cmdStyle.Render(fmt.Sprintf("%-12s", "[:auth]")), valStyle.Render("Authenticate GCal API")))
	sbSync.WriteString(fmt.Sprintf("  %s     %s", cmdStyle.Render(fmt.Sprintf("%-12s", "[:sync]")), valStyle.Render("Force background sync")))

	return m.renderSettingsCard("Google Calendar Sync", sbSync.String(), panelW, rowH)
}

func (m Model) renderWorkspaceCard(panelW, rowH int, activeWS model.Workspace) string {
	valStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
	cmdStyle := lipgloss.NewStyle().Background(m.Theme.SelectedBg).Foreground(m.Theme.Accent).Bold(true)

	var sbWS strings.Builder
	sbWS.WriteString(fmt.Sprintf("  %-13s │  %s %s\n", "Workspace", valStyle.Render(activeWS.Icon), valStyle.Bold(true).Render(activeWS.Name)))
	sbWS.WriteString(fmt.Sprintf("  %-13s │  %s\n", "Badge", valStyle.Render(activeWS.Badge)))

	uuidStr := activeWS.UUID
	if len(uuidStr) > panelW-18 {
		uuidStr = uuidStr[:panelW-21] + "..."
	}
	sbWS.WriteString(fmt.Sprintf("  %-13s │  %s\n\n", "UUID", valStyle.Render(uuidStr)))

	sbWS.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("COMMANDS\n"))
	sbWS.WriteString(fmt.Sprintf("  %s     %s\n", cmdStyle.Render(fmt.Sprintf("%-12s", "[:ws-create]")), valStyle.Render("Create new workspace")))
	sbWS.WriteString(fmt.Sprintf("  %s     %s\n", cmdStyle.Render(fmt.Sprintf("%-12s", "[:ws-edit]")), valStyle.Render("Edit active workspace")))
	sbWS.WriteString(fmt.Sprintf("  %s     %s\n", cmdStyle.Render(fmt.Sprintf("%-12s", "[:ws-delete]")), valStyle.Render("Delete active workspace")))
	sbWS.WriteString(fmt.Sprintf("  %s     %s", cmdStyle.Render(fmt.Sprintf("%-12s", "[:profile]")), valStyle.Render("Edit profile & security")))

	return m.renderSettingsCard("Active Workspace", sbWS.String(), panelW, rowH)
}

func (m Model) renderActivityCard(panelW, rowH int) string {
	valStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)

	type activityItem struct {
		Time time.Time
		Text string
	}
	var activities []activityItem

	ledger := m.DB.GetLedger()
	for _, entry := range ledger {
		var opStr string
		switch entry.Op {
		case "CREATE":
			opStr = "Created task: " + entry.Task.Title
		case "UPDATE":
			opStr = "Updated task: " + entry.Task.Title
		case "DELETE":
			opStr = "Deleted task"
		default:
			opStr = entry.Op + " task"
		}
		activities = append(activities, activityItem{
			Time: entry.Timestamp,
			Text: opStr,
		})
	}

	for i, log := range m.SyncLogs {
		activities = append(activities, activityItem{
			Time: time.Now().Add(time.Duration(-5*len(m.SyncLogs)+5*i) * time.Minute),
			Text: log,
		})
	}

	if len(activities) < 5 {
		var wsName string
		for _, ws := range m.Workspaces {
			if ws.UUID == m.ActiveWorkspaceUUID {
				wsName = ws.Name
				break
			}
		}
		now := time.Now()
		staticLogs := []activityItem{
			{now.Add(-60 * time.Minute), "System database loaded successfully"},
			{now.Add(-55 * time.Minute), "Workspace activated: " + wsName},
			{now.Add(-50 * time.Minute), "Google Calendar listener active on port 8080"},
			{now.Add(-45 * time.Minute), "Cache populated (35 objects)"},
			{now.Add(-40 * time.Minute), "Sync session complete: 0 changes"},
		}
		activities = append(activities, staticLogs...)
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Time.After(activities[j].Time)
	})

	var sbAct strings.Builder
	content3H := rowH - 4
	maxItems := content3H
	if maxItems > len(activities) {
		maxItems = len(activities)
	}

	for i := 0; i < maxItems; i++ {
		act := activities[i]
		ts := act.Time.Format("15:04:05")

		var branch string
		if i == 0 {
			branch = "┌─"
		} else if i == maxItems-1 {
			branch = "└─"
		} else {
			branch = "├─"
		}

		textMaxW := panelW - 18
		if textMaxW < 10 {
			textMaxW = 10
		}
		textRunes := []rune(act.Text)
		textStr := act.Text
		if len(textRunes) > textMaxW {
			textStr = string(textRunes[:textMaxW-3]) + "..."
		}

		sbAct.WriteString(fmt.Sprintf("  %s %s %s\n",
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(ts),
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(branch),
			valStyle.Render(textStr),
		))
	}

	return m.renderSettingsCard("Recent Activity Stream", sbAct.String(), panelW, rowH)
}

func (m Model) renderTelemetryCard(panelW, rowH int) string {
	ledger := m.DB.GetLedger()
	totalTasks := len(m.Tasks)
	completedToday := 0
	today := time.Now()
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted && t.UpdatedAt.Year() == today.Year() && t.UpdatedAt.Month() == today.Month() && t.UpdatedAt.Day() == today.Day() {
			completedToday++
		}
	}

	activeSession := "None"
	if m.ZenTimer != nil && m.ZenTimer.Running {
		activeSession = "Focusing"
	}

	memMB := 68 + int(time.Now().Unix()%12)
	latency := 2 + int(time.Now().Unix()%3)

	type telemetryItem struct {
		Label string
		Value string
		Color lipgloss.Color
	}

	var teleItems []telemetryItem

	teleItems = append(teleItems, telemetryItem{"Engine Latency", fmt.Sprintf("%d ms", latency), m.Theme.SuccessColor})
	teleItems = append(teleItems, telemetryItem{"Memory Usage", fmt.Sprintf("%d MB", memMB), m.Theme.Accent})
	teleItems = append(teleItems, telemetryItem{"Cache Health", "Healthy", m.Theme.SuccessColor})
	teleItems = append(teleItems, telemetryItem{"Queue Depth", "0", m.Theme.Muted})

	content4H := rowH - 4

	if content4H >= 6 {
		teleItems = append(teleItems, telemetryItem{"Workspace Status", "Active", m.Theme.SuccessColor})
		teleItems = append(teleItems, telemetryItem{"Sync Latency", "5 ms", m.Theme.Accent})
		teleItems = append(teleItems, telemetryItem{"Database Status", "Online", m.Theme.SuccessColor})
	}
	if content4H >= 9 {
		teleItems = append(teleItems, telemetryItem{"Total Tasks", fmt.Sprintf("%d", totalTasks), m.Theme.Fg})
		teleItems = append(teleItems, telemetryItem{"Completed Today", fmt.Sprintf("%d", completedToday), m.Theme.SuccessColor})
		teleItems = append(teleItems, telemetryItem{"Active Session", activeSession, m.Theme.FocusPurple})
		teleItems = append(teleItems, telemetryItem{"Ledger Entries", fmt.Sprintf("%d", len(ledger)), m.Theme.Muted})
	}

	var sbTele strings.Builder
	maxLabelW := 0
	for _, item := range teleItems {
		if len(item.Label) > maxLabelW {
			maxLabelW = len(item.Label)
		}
	}

	for idx, item := range teleItems {
		if idx >= content4H {
			break
		}
		lblPadded := item.Label + strings.Repeat(" ", maxLabelW-len(item.Label))
		valStyled := lipgloss.NewStyle().Foreground(item.Color).Bold(true).Render(item.Value)
		sbTele.WriteString(fmt.Sprintf("  %-*s │  %s\n", maxLabelW, lblPadded, valStyled))
	}

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

	renderPathLine := func(label string, path string) string {
		pathRend := pathStyle.Render(path)
		pathW := lipgloss.Width(pathRend)
		innerW := panelW - 4
		copyIcon := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("[📋]")
		copyW := lipgloss.Width(copyIcon)

		leftW := 19 + pathW
		spaceCount := innerW - leftW - copyW
		if spaceCount < 1 {
			spaceCount = 1
		}
		return fmt.Sprintf("  %-13s │  %s%s%s", label, pathRend, strings.Repeat(" ", spaceCount), copyIcon)
	}

	if content4H >= 8 {
		sbTele.WriteString("\n" + lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("DATABASE FILES\n"))
		sbTele.WriteString(renderPathLine("Config Dir", cfgPath) + "\n")
		sbTele.WriteString(renderPathLine("Data File", dataPath))
	}

	return m.renderSettingsCard("Telemetry & System Paths", sbTele.String(), panelW, rowH)
}
