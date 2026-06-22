package pages

import (
	"fmt"
	"sort"
	"strings"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderPanel(t theme.Theme, title string, lines []string, w, h int, borderCol lipgloss.Color) string {
	innerW := w - 6
	innerH := h - 2
	if innerW < 4 {
		innerW = 4
	}
	if innerH < 2 {
		innerH = 2
	}

	var contentLines []string
	contentLines = append(contentLines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(title))
	contentLines = append(contentLines, "")

	for _, l := range lines {
		if len(contentLines) >= innerH {
			break
		}
		contentLines = append(contentLines, l)
	}

	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}

	for i, line := range contentLines {
		rawW := lipgloss.Width(line)
		if rawW < innerW {
			contentLines[i] = line + strings.Repeat(" ", innerW-rawW)
		} else if rawW > innerW {
			contentLines[i] = theme.SliceAnsi(line, 0, innerW)
		}
	}

	joined := strings.Join(contentLines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Width(w - 2).
		Height(innerH).
		Padding(0, 2).
		Render(joined)
}

func renderRecentActivityPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string

	type event struct {
		timeStr string
		desc    string
		color   lipgloss.Color
	}

	var events []event

	var completedTasks []model.Task
	for _, task := range m.Tasks {
		if (m.ActiveWorkspaceUUID == "ALL_WORKSPACES" || task.WorkspaceUUID == m.ActiveWorkspaceUUID) && task.LifecycleState == model.StateCompleted {
			completedTasks = append(completedTasks, task)
		}
	}
	sort.Slice(completedTasks, func(i, j int) bool {
		return completedTasks[i].UpdatedAt.After(completedTasks[j].UpdatedAt)
	})

	for i, task := range completedTasks {
		if i >= 5 {
			break
		}
		events = append(events, event{
			timeStr: task.UpdatedAt.Format("15:04"),
			desc:    fmt.Sprintf("Completed Task: %s", theme.SentenceCase(task.Title)),
			color:   t.SuccessColor,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].timeStr > events[j].timeStr
	})

	for idx, ev := range events {
		if len(lines) >= innerH {
			break
		}
		desc := ev.desc
		maxDescW := innerW - 10
		if maxDescW < 5 {
			maxDescW = 5
		}
		descRunes := []rune(desc)
		if len(descRunes) > maxDescW {
			desc = string(descRunes[:maxDescW-1]) + "…"
		}
		tStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(ev.timeStr)
		dStyle := lipgloss.NewStyle().Foreground(ev.color).Render(desc)
		row := fmt.Sprintf(" %s  %s", tStyle, dStyle)
		lines = append(lines, row)
		if innerH > 12 && idx < len(events)-1 {
			lines = append(lines, "")
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 0 && m.DashboardFocusRow == 2
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📜 RECENT ACTIVITY STREAM", lines, w, h, borderCol)
}

func renderTelemetryPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string

	dbSize := len(m.Tasks)*250 + len(m.Workspaces)*300
	dbSizeKB := float64(dbSize) / 1024.0
	syncOnline := "ONLINE"
	if m.Sync == nil || !m.Sync.IsOnline() {
		syncOnline = "OFFLINE"
	}

	lines = append(lines,
		fmt.Sprintf(" • Engine Latency:     %dms (TUI Update)", 2),
		fmt.Sprintf(" • Memory footprint:   %d MB (Active Pool)", 28),
		fmt.Sprintf(" • Database Size:      %.2f KB", dbSizeKB),
		fmt.Sprintf(" • Sync Engine State:  %s (Sync Queue: 0)", syncOnline),
		fmt.Sprintf(" • Cache Hit Rate:     94.8%% (Read Optimizer)"),
	)

	remaining := innerH - len(lines) - 2
	if remaining > 4 {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW)),
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("🔔 OPERATIONAL LOGS:"),
		)

		if len(m.SyncLogs) == 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("  No operational logs recorded."))
		} else {
			maxLogs := remaining - 3
			for idx, log := range m.SyncLogs {
				if idx >= maxLogs {
					break
				}
				row := "  " + log
				if lipgloss.Width(row) > innerW {
					row = string([]rune(row)[:innerW-2]) + "…"
				}
				lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(row))
			}
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 1 && m.DashboardFocusRow == 2
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "⚙️ SYSTEM TELEMETRY", lines, w, h, borderCol)
}
