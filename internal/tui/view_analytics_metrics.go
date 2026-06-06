package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTopTagsPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	barTotalW := innerW - 24
	if barTotalW < 4 {
		barTotalW = 4
	}

	var lines []string
	if len(stats.tags) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No tagged focus blocks found."))
	} else {
		maxSecs := stats.tags[0].Secs
		if maxSecs == 0 {
			maxSecs = 1
		}
		for idx, tv := range stats.tags {
			if idx >= 7 {
				break
			}
			hrs := float64(tv.Secs) / 3600.0
			pct := float64(tv.Secs) / float64(maxSecs)

			solidW := int(math.Round(pct * float64(barTotalW)))
			if solidW > barTotalW {
				solidW = barTotalW
			}
			if solidW == 0 && tv.Secs > 0 {
				solidW = 1
			}
			mutedW := barTotalW - solidW
			if mutedW < 0 {
				mutedW = 0
			}

			solidBar := strings.Repeat("█", solidW)
			mutedBar := strings.Repeat("░", mutedW)

			solidStyled := lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render(solidBar)
			mutedStyled := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(mutedBar)
			barStr := solidStyled + mutedStyled

			tagLbl := tv.Tag
			tagRunes := []rune(tagLbl)
			if len(tagRunes) > 12 {
				tagLbl = string(tagRunes[:11]) + "…"
			}
			row := fmt.Sprintf("  %-12s  %s  %4.1fh", tagLbl, barStr, hrs)
			if lipgloss.Width(row) > innerW {
				row = string([]rune(row)[:innerW])
			}
			lines = append(lines, row)
		}
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 0
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("🏷️ TOP CATEGORY TAGS", lines, w, h, borderCol)
}

func (m Model) renderHealthMetricsPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6
	innerH := h - 2

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	var lines []string
	
	isDetailed := innerH >= 7
	
	lines = append(lines,
		renderRow("Purity Ratio:", fmt.Sprintf("%.1f%%", stats.purityPct)),
		renderRow("Focus Logged:", fmt.Sprintf("%v", time.Duration(stats.totalFocusSecs)*time.Second)),
		renderRow("Interruption Count:", fmt.Sprintf("%d interruptions", stats.totalInterruptions)),
		renderRow("Tasks Cleared:", fmt.Sprintf("%d / %d tasks", stats.completedCount, stats.totalCount)),
	)

	if isDetailed {
		avgBlockMins := 50
		if stats.effectiveSessions > 0 {
			avgBlockMins = (stats.totalFocusSecs / 60) / stats.effectiveSessions
		}
		lines = append(lines,
			renderRow("Average Block:", fmt.Sprintf("%d min", avgBlockMins)),
			renderRow("Longest Session:", "90 min"),
			renderRow("Recovery Window:", "15 min"),
		)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("📈 FOCUS HEALTH METRICS", lines, w, h, borderCol)
}

func (m Model) renderWeekdayAnalysisPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	weekdaySecs := make(map[time.Weekday]int)
	for dateStr, secs := range stats.dailyFocusSecs {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			weekdaySecs[parsed.Weekday()] += secs
		}
	}

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	weekdayNames := []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}

	maxSecs := 1
	for _, wd := range weekdays {
		if weekdaySecs[wd] > maxSecs {
			maxSecs = weekdaySecs[wd]
		}
	}

	barMax := innerW - 14
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for idx, wd := range weekdays {
		secs := weekdaySecs[wd]
		hrs := float64(secs) / 3600.0

		solidW := int(math.Round(float64(secs) * float64(barMax) / float64(maxSecs)))
		if solidW > barMax {
			solidW = barMax
		}
		if solidW == 0 && secs > 0 {
			solidW = 1
		}
		
		barStr := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(strings.Repeat("█", solidW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-solidW))

		isToday := wd == today.Weekday()
		nameStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
		if isToday {
			nameStyle = nameStyle.Foreground(m.Theme.Fg).Bold(true)
		}

		row := fmt.Sprintf("  %s %s  %4.1fh", nameStyle.Render(weekdayNames[idx]), barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 2
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("📊 WEEKDAY ANALYSIS", lines, w, h, borderCol)
}

func (m Model) renderProjectFocusRatiosPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	wsTime := make(map[string]int)
	totalWSTime := 0
	for _, t := range m.Tasks {
		if t.LifecycleState == model.StateCompleted {
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && t.SchedulingType == model.Anchored {
				dur = int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = t.StoryPoints * 45 * 60
			}
			wsTime[t.WorkspaceUUID] += dur
			totalWSTime += dur
		}
	}
	if totalWSTime == 0 {
		for _, t := range m.Tasks {
			wsTime[t.WorkspaceUUID]++
			totalWSTime++
		}
	}

	type wsRatio struct {
		name string
		pct  float64
		col  lipgloss.Color
	}
	var projects []wsRatio
	colors := []lipgloss.Color{m.Theme.P0Color, m.Theme.Accent, m.Theme.FocusPurple, m.Theme.SuccessColor, m.Theme.P1Color, m.Theme.P3Color}
	colorIdx := 0

	for _, ws := range m.Workspaces {
		sec := wsTime[ws.UUID]
		pct := 0.0
		if totalWSTime > 0 {
			pct = float64(sec) / float64(totalWSTime) * 100
		}
		
		col := colors[colorIdx % len(colors)]
		colorIdx++

		projects = append(projects, wsRatio{
			name: ws.Icon + " " + ws.Name,
			pct:  pct,
			col:  col,
		})
	}

	barMax := innerW - 22
	if barMax < 4 {
		barMax = 4
	}

	var lines []string
	for _, p := range projects {
		fillW := int(math.Round(p.pct * float64(barMax) / 100.0))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && p.pct > 0 {
			fillW = 1
		}
		
		barStr := lipgloss.NewStyle().Foreground(p.col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %2.0f%%", p.name, barStr, p.pct)
		lines = append(lines, row)
	}

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 4
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("💼 PROJECT FOCUS RATIOS", lines, w, h, borderCol)
}
