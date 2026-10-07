package pages

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"
	"stream/internal/viewmodel/tasks"

	"github.com/charmbracelet/lipgloss"
)

func renderCapacityPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6
	innerH := h - 2

	today := time.Now()
	weeklyPoints := make(map[time.Weekday]int)
	weeklyCompletedPoints := make(map[time.Weekday]int)
	weeklyCount := make(map[time.Weekday]int)

	offset := int(today.Weekday()) - 1
	if offset < 0 {
		offset = 6
	}
	weekStart := today.AddDate(0, 0, -offset)
	for i := 0; i < 7; i++ {
		day := weekStart.AddDate(0, 0, i)
		for _, task := range m.Tasks {
			if (m.ActiveWorkspaceUUID == "ALL_WORKSPACES" || task.WorkspaceUUID == m.ActiveWorkspaceUUID) &&
				task.TimeWindow.Start.Year() == day.Year() &&
				task.TimeWindow.Start.Month() == day.Month() &&
				task.TimeWindow.Start.Day() == day.Day() {
				weeklyPoints[day.Weekday()] += task.StoryPoints
				weeklyCount[day.Weekday()]++
				if task.LifecycleState == "completed" {
					weeklyCompletedPoints[day.Weekday()] += task.StoryPoints
				}
			}
		}
	}

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}

	maxPoints := 1
	for _, wd := range weekdays {
		if weeklyPoints[wd] > maxPoints {
			maxPoints = weeklyPoints[wd]
		}
	}

	var lines []string
	isDetailed := innerH >= 10
	isExpanded := innerH >= 7 && !isDetailed

	barMaxW := innerW - 32
	if barMaxW < 4 {
		barMaxW = 4
	}

	for idx, wd := range weekdays {
		pts := weeklyPoints[wd]
		compPts := weeklyCompletedPoints[wd]
		count := weeklyCount[wd]

		solidW := int(math.Round(float64(pts) * float64(barMaxW) / float64(maxPoints)))
		if solidW > barMaxW {
			solidW = barMaxW
		}
		if solidW == 0 && pts > 0 {
			solidW = 1
		}
		mutedW := barMaxW - solidW
		if mutedW < 0 {
			mutedW = 0
		}

		isToday := wd == today.Weekday()
		nameColor := t.Muted
		if isToday {
			nameColor = t.Fg
		}
		nameStr := lipgloss.NewStyle().Foreground(nameColor).Bold(isToday).Render(weekdayNames[idx])

		solidBar := strings.Repeat("█", solidW)
		mutedBar := strings.Repeat("░", mutedW)

		barColor := t.Accent
		if pts >= 9 {
			barColor = t.P0Color
		} else if pts == 0 {
			barColor = t.Muted
		}

		solidStr := lipgloss.NewStyle().Foreground(barColor).Render(solidBar)
		mutedStr := lipgloss.NewStyle().Foreground(t.Muted).Render(mutedBar)
		barStr := solidStr + mutedStr

		var ptStr string
		if isDetailed {
			ptStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2d/%2d SP (%d tasks)", compPts, pts, count))
		} else if isExpanded {
			ptStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2d/%2d SP", compPts, pts))
		} else {
			ptStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%dSP", pts))
		}

		rowContent := fmt.Sprintf("  %-5s %s  %s", nameStr, barStr, ptStr)
		lines = append(lines, rowContent)
	}

	remainingLines := innerH - len(lines) - 2
	if remainingLines > 2 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW)))
		totalWeeklySP := 0
		totalWeeklyCompSP := 0
		totalWeeklyTasks := 0
		peakDayIdx := 0
		peakDaySP := 0
		for idx, wd := range weekdays {
			pts := weeklyPoints[wd]
			totalWeeklySP += pts
			totalWeeklyCompSP += weeklyCompletedPoints[wd]
			totalWeeklyTasks += weeklyCount[wd]
			if pts > peakDaySP {
				peakDaySP = pts
				peakDayIdx = idx
			}
		}

		lines = append(lines,
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("WEEKLY CAPACITY SNAPSHOT:"),
			fmt.Sprintf(" • Total Velocity:    %d / %d SP completed", totalWeeklyCompSP, totalWeeklySP),
		)

		if innerH-len(lines) > 2 {
			pct := 0.0
			if totalWeeklySP > 0 {
				pct = float64(totalWeeklyCompSP) / float64(totalWeeklySP) * 100
			}
			lines = append(lines,
				fmt.Sprintf(" • Completion Rate:   %.0f%% (%d total tasks)", pct, totalWeeklyTasks),
			)
		}

		if innerH-len(lines) > 2 && peakDaySP > 0 {
			lines = append(lines,
				fmt.Sprintf(" • Peak Load Day:     %s (%d SP scheduled)", weekdayNames[peakDayIdx], peakDaySP),
			)
		}

		if innerH-len(lines) > 2 {
			activeSprint, hasSprint := m.GetActiveSprint()
			if hasSprint {
				defined, inProg, rev, tst, comp := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
				sprintTotalSP := 0
				for _, tList := range [][]model.Task{defined, inProg, rev, tst, comp} {
					for _, tsk := range tList {
						sprintTotalSP += tsk.StoryPoints
					}
				}
				sprintPct := 0.0
				if sprintTotalSP > 0 {
					sprintPct = float64(totalWeeklySP) / float64(sprintTotalSP) * 100
				}
				lines = append(lines,
					fmt.Sprintf(" • Sprint Allocation: %.0f%% of active sprint", sprintPct),
				)
			}
		}

		if innerH-len(lines) > 1 {
			paceStr := "✓ OPTIMAL"
			if totalWeeklySP > 40 {
				paceStr = "⚡ HIGH LOAD"
			} else if totalWeeklySP == 0 {
				paceStr = "○ LIGHT LOAD"
			}
			lines = append(lines,
				fmt.Sprintf(" • Workload Pace:     %s", paceStr),
			)
		}
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.DashboardFocusCol == 1 && m.DashboardFocusRow == 0
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📊 WEEKLY CAPACITY UTILIZATION", lines, w, h, borderCol)
}
