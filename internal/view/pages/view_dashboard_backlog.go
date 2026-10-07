package pages

import (
	"fmt"
	"math"
	"strings"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderBacklogHealthPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string

	totalBacklog := 0
	readyCount := 0
	overdueCount := 0
	blockedCount := 0
	totalBacklogSP := 0
	inSprintCount := 0
	unassignedCount := 0
	wsCounts := make(map[string]int)
	wsCompCounts := make(map[string]int)

	pCounts := make(map[model.Priority]int)

	for _, task := range m.Tasks {
		if m.ActiveWorkspaceUUID == "ALL_WORKSPACES" || task.WorkspaceUUID == m.ActiveWorkspaceUUID {
			if task.SchedulingType == model.Floating && task.LifecycleState != model.StateCompleted {
				totalBacklog++
				totalBacklogSP += task.StoryPoints
				pCounts[task.Priority]++
				if task.SprintUUID != "" {
					inSprintCount++
				} else {
					unassignedCount++
				}
				if task.LifecycleState == model.StateReady {
					readyCount++
				}
				if task.LifecycleState == model.StateOverdue {
					overdueCount++
				}
			}
			if task.LifecycleState == model.StateOverdue {
				overdueCount++
			}
		}
		for _, ws := range m.Workspaces {
			if task.WorkspaceUUID == ws.UUID {
				wsCounts[ws.Name]++
				if task.LifecycleState == model.StateCompleted {
					wsCompCounts[ws.Name]++
				}
			}
		}
	}

	avgSP := 0.0
	if totalBacklog > 0 {
		avgSP = float64(totalBacklogSP) / float64(totalBacklog)
	}

	lines = append(lines,
		fmt.Sprintf(" • Total Backlog Size:   %d Floating Tasks (%d SP)", totalBacklog, totalBacklogSP),
		fmt.Sprintf(" • Sprint Allocation:    %d Sprint / %d Unassigned", inSprintCount, unassignedCount),
		fmt.Sprintf(" • Ready to Pull:        %d Tasks (Avg %.1f SP/task)", readyCount, avgSP),
		fmt.Sprintf(" • Overdue / Blocked:    %d Overdue, %d Blocked", overdueCount, blockedCount),
	)

	remaining := innerH - len(lines) - 2
	if remaining > 4 {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW)),
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("💼 WORKSPACE DISTRIBUTION:"),
		)

		for _, ws := range m.Workspaces {
			tot := wsCounts[ws.Name]
			comp := wsCompCounts[ws.Name]
			pct := 0.0
			if tot > 0 {
				pct = float64(comp) / float64(tot) * 100
			}

			barW := innerW - 36
			if barW < 4 {
				barW = 4
			}
			fillW := int(math.Round(pct * float64(barW) / 100.0))
			bar := strings.Repeat("█", fillW) + strings.Repeat("░", barW-fillW)
			barStyled := lipgloss.NewStyle().Foreground(t.Accent).Render(bar)

			row := fmt.Sprintf("  %s %-12s %s  %d/%d (%2.0f%%)", ws.Icon, ws.Name, barStyled, comp, tot, pct)
			lines = append(lines, row)
		}

		if innerH-len(lines) >= 3 {
			lines = append(lines,
				"",
				lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  • Priority Mix: P0:%d | P1:%d | P2:%d | P3:%d",
					pCounts[model.P0], pCounts[model.P1], pCounts[model.P2], pCounts[model.P3])),
			)
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 1 && m.DashboardFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📋 BACKLOG & CATEGORIES", lines, w, h, borderCol)
}
