package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAgendaPanel(w, h int) string {
	innerW := w - 6
	innerH := h - 2
	
	var lines []string
	agendaTasks := m.getAgendaTasks()

	isDetailed := innerH >= 12
	isExpanded := innerH >= 7 && !isDetailed

	if len(agendaTasks) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("No tasks scheduled for today."))
	} else {
		for _, t := range agendaTasks {
			chk := "[ ]"
			if t.LifecycleState == model.StateCompleted {
				chk = "[✓]"
			}

			title := sentenceCase(t.Title)
			var timeStr string
			if t.SchedulingType == model.Anchored {
				timeStr = t.TimeWindow.Start.Format("15:04")
			} else {
				timeStr = "FLOAT"
			}

			var line string
			pColor := m.priorityColor(t.Priority)
			
			if isDetailed {
				pBadge := fmt.Sprintf("[%s]", string(t.Priority))
				spBadge := fmt.Sprintf("%dSP", t.StoryPoints)
				stateStr := string(t.LifecycleState)
				
				leftSide := fmt.Sprintf("%s %-5s %s", chk, timeStr, title)
				rightSide := fmt.Sprintf("%s %s %s", pBadge, spBadge, stateStr)
				
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
				pBadge := fmt.Sprintf("[%s]", string(t.Priority))
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

			if t.LifecycleState == model.StateCompleted {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Faint(true).Render(line)
			} else if t.LifecycleState == model.StateActive {
				line = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(pColor).Render(line)
			}
			lines = append(lines, line)
		}
	}

	remainingLines := innerH - 2 - len(lines)
	if remainingLines > 3 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", innerW)))
		compCount := 0
		totCount := len(agendaTasks)
		totSP := 0
		compSP := 0
		for _, t := range agendaTasks {
			totSP += t.StoryPoints
			if t.LifecycleState == model.StateCompleted {
				compCount++
				compSP += t.StoryPoints
			}
		}

		pct := 0.0
		if totCount > 0 {
			pct = float64(compCount) / float64(totCount) * 100
		}
		
		lines = append(lines,
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("AGENDA OPERATIONS STATUS:"),
			fmt.Sprintf(" • Tasks Cleared:     %d / %d  (%.0f%%)", compCount, totCount, pct),
			fmt.Sprintf(" • Story Points:      %d / %d completed", compSP, totSP),
		)
		
		if innerH - len(lines) - 2 > 1 {
			lines = append(lines,
				fmt.Sprintf(" • Health Status:     %s", m.getAgendaHealthStatus(agendaTasks)),
				fmt.Sprintf(" • Target Capacity:   %d SP daily load", m.getRecommendedCapacity()),
			)
		}
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 0 && m.DashboardFocusRow == 0
	if isFocused {
		borderCol = m.Theme.Accent
	}

	return m.renderPanel("⚡ ACTIVE AGENDA INBOX", lines, w, h, borderCol)
}

func (m Model) getAgendaHealthStatus(tasks []model.Task) string {
	overdueCount := 0
	p0Count := 0
	for _, t := range tasks {
		if t.LifecycleState == model.StateOverdue {
			overdueCount++
		}
		if t.Priority == model.P0 && t.LifecycleState != model.StateCompleted {
			p0Count++
		}
	}
	if overdueCount > 0 {
		return lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("⚠️ OVERDUE CRITICAL")
	}
	if p0Count > 0 {
		return lipgloss.NewStyle().Foreground(m.Theme.P1Color).Bold(true).Render("⚡ HIGH LOAD")
	}
	return lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("✓ OPTIMAL")
}

func (m Model) getAgendaTasks() []model.Task {
	today := time.Now()
	var agendaTasks []model.Task
	for _, t := range m.Tasks {
		isTodayOrUpcoming := false
		if t.SchedulingType == model.Anchored {
			isTodayOrUpcoming = t.TimeWindow.Start.Year() == today.Year() &&
				t.TimeWindow.Start.Month() == today.Month() &&
				t.TimeWindow.Start.Day() == today.Day() || t.TimeWindow.Start.After(today)
		} else {
			isTodayOrUpcoming = t.CreatedAt.Year() == today.Year() &&
				t.CreatedAt.Month() == today.Month() &&
				t.CreatedAt.Day() == today.Day() && t.LifecycleState != model.StateCompleted
		}
		if isTodayOrUpcoming {
			agendaTasks = append(agendaTasks, t)
		}
	}
	sort.Slice(agendaTasks, func(i, j int) bool {
		if agendaTasks[i].SchedulingType == model.Anchored && agendaTasks[j].SchedulingType == model.Anchored {
			return agendaTasks[i].TimeWindow.Start.Before(agendaTasks[j].TimeWindow.Start)
		}
		if agendaTasks[i].SchedulingType == model.Anchored {
			return true
		}
		if agendaTasks[j].SchedulingType == model.Anchored {
			return false
		}
		return agendaTasks[i].CreatedAt.Before(agendaTasks[j].CreatedAt)
	})
	return agendaTasks
}
