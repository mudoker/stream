package pages

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func renderAgendaPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	var lines []string
	agendaTasks := m.GetAgendaTasks()

	isDetailed := innerH >= 12
	isExpanded := innerH >= 7 && !isDetailed

	if len(agendaTasks) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("No tasks scheduled for today."))
	} else {
		for _, task := range agendaTasks {
			chk := "[ ]"
			if task.LifecycleState == model.StateCompleted {
				chk = "[✓]"
			}

			title := theme.SentenceCase(task.Title)
			var timeStr string
			if task.SchedulingType == model.Anchored || task.SchedulingType == model.Event {
				if task.IsAllDay {
					timeStr = "ALL DAY"
				} else {
					timeStr = task.TimeWindow.Start.Format("15:04")
				}
			} else if task.SchedulingType == model.Reminder {
				if task.TimeWindow.Start.Second() == 1 {
					timeStr = "REM"
				} else {
					timeStr = fmt.Sprintf("REM %s", task.TimeWindow.Start.Format("15:04"))
				}
			} else {
				timeStr = "FLOAT"
			}

			var line string
			pColor := t.PriorityColor(task.Priority)

			if isDetailed {
				pBadge := fmt.Sprintf("[%s]", string(task.Priority))
				spBadge := fmt.Sprintf("%dSP", task.StoryPoints)
				if task.SchedulingType == model.Reminder {
					spBadge = ""
				}
				stateStr := string(task.LifecycleState)

				leftSide := fmt.Sprintf("%s %-5s %s", chk, timeStr, title)
				var rightSide string
				if spBadge != "" {
					rightSide = fmt.Sprintf("%s %s %s", pBadge, spBadge, stateStr)
				} else {
					rightSide = fmt.Sprintf("%s %s", pBadge, stateStr)
				}

				leftW := lipgloss.Width(leftSide)
				rightW := lipgloss.Width(rightSide)
				pad := innerW - leftW - rightW
				if pad < 1 {
					pad = 1
				}
				if leftW+rightW > innerW {
					maxLeft := innerW - rightW - 2
					leftSideRunes := []rune(leftSide)
					if maxLeft > 3 {
						leftSide = string(leftSideRunes[:maxLeft-1]) + "…"
					} else {
						leftSide = string(leftSideRunes[:maxLeft])
					}
					leftW = lipgloss.Width(leftSide)
					pad = innerW - leftW - rightW
				}
				line = leftSide + strings.Repeat(" ", pad) + rightSide
			} else if isExpanded {
				pBadge := fmt.Sprintf("[%s]", string(task.Priority))
				leftSide := fmt.Sprintf("%s %-5s %s", chk, timeStr, title)
				leftW := lipgloss.Width(leftSide)
				rightW := lipgloss.Width(pBadge)
				pad := innerW - leftW - rightW
				if pad < 1 {
					pad = 1
				}
				if leftW+rightW > innerW {
					maxLeft := innerW - rightW - 2
					leftSideRunes := []rune(leftSide)
					if maxLeft > 3 {
						leftSide = string(leftSideRunes[:maxLeft-1]) + "…"
					} else {
						leftSide = string(leftSideRunes[:maxLeft])
					}
					leftW = lipgloss.Width(leftSide)
					pad = innerW - leftW - rightW
				}
				line = leftSide + strings.Repeat(" ", pad) + pBadge
			} else {
				line = fmt.Sprintf("%s %s", chk, title)
				if lipgloss.Width(line) > innerW {
					runes := []rune(line)
					line = string(runes[:innerW-1]) + "…"
				}
			}

			if task.LifecycleState == model.StateCompleted {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true).Render(line)
			} else if task.LifecycleState == model.StateActive {
				line = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(pColor).Render(line)
			}
			lines = append(lines, line)
		}
	}

	remainingLines := innerH - 2 - len(lines)
	if remainingLines > 3 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW)))
		compCount := 0
		totCount := len(agendaTasks)
		totSP := 0
		compSP := 0
		for _, task := range agendaTasks {
			totSP += task.StoryPoints
			if task.LifecycleState == model.StateCompleted {
				compCount++
				compSP += task.StoryPoints
			}
		}

		pct := 0.0
		if totCount > 0 {
			pct = float64(compCount) / float64(totCount) * 100
		}

		lines = append(lines,
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("AGENDA OPERATIONS STATUS:"),
			fmt.Sprintf(" • Tasks Cleared:     %d / %d  (%.0f%%)", compCount, totCount, pct),
			fmt.Sprintf(" • Story Points:      %d / %d completed", compSP, totSP),
		)

		if innerH-len(lines)-2 > 1 {
			lines = append(lines,
				fmt.Sprintf(" • Health Status:     %s", getAgendaHealthStatus(t, agendaTasks)),
				fmt.Sprintf(" • Target Capacity:   %d SP daily load", m.GetRecommendedCapacity()),
			)
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 0 && m.DashboardFocusRow == 0
	if isFocused {
		borderCol = t.Accent
	}

	return renderPanel(t, "⚡ ACTIVE AGENDA INBOX", lines, w, h, borderCol)
}

func getAgendaHealthStatus(t theme.Theme, tasks []model.Task) string {
	overdueCount := 0
	p0Count := 0
	for _, task := range tasks {
		if task.LifecycleState == model.StateOverdue {
			overdueCount++
		}
		if task.Priority == model.P0 && task.LifecycleState != model.StateCompleted {
			p0Count++
		}
	}
	if overdueCount > 0 {
		return lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("⚠️ OVERDUE CRITICAL")
	}
	if p0Count > 0 {
		return lipgloss.NewStyle().Foreground(t.P1Color).Bold(true).Render("⚡ HIGH LOAD")
	}
	return lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("✓ OPTIMAL")
}
