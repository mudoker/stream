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

func renderHourHeatmapPanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6

	morningSecs := 0
	afternoonSecs := 0
	eveningSecs := 0

	for _, task := range m.Tasks {
		if task.SchedulingType == model.Anchored {
			hour := task.TimeWindow.Start.Hour()
			dur := task.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 {
				dur = task.StoryPoints * 45 * 60
			}
			if hour >= 8 && hour < 12 {
				morningSecs += dur
			} else if hour >= 12 && hour < 18 {
				afternoonSecs += dur
			} else {
				eveningSecs += dur
			}
		}
	}

	total := morningSecs + afternoonSecs + eveningSecs
	if total == 0 {
		total = 1
	}

	mPct := float64(morningSecs) / float64(total) * 100
	aPct := float64(afternoonSecs) / float64(total) * 100
	ePct := float64(eveningSecs) / float64(total) * 100

	barMax := innerW - 21
	if barMax < 6 {
		barMax = 6
	}

	renderBar := func(pct float64, col lipgloss.Color) string {
		fillW := int(math.Round(pct * float64(barMax) / 100.0))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && pct > 0 {
			fillW = 1
		}
		return lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))
	}

	mLabel := lipgloss.NewStyle().Foreground(t.Accent).Render("Morning (08-12):  ")
	aLabel := lipgloss.NewStyle().Foreground(t.FocusPurple).Render("Afternoon (12-18):")
	eLabel := lipgloss.NewStyle().Foreground(t.P1Color).Render("Evening (18-00):  ")

	var lines []string
	lines = append(lines,
		fmt.Sprintf("  %s %s", mLabel, renderBar(mPct, t.Accent)),
		"",
		fmt.Sprintf("  %s %s", aLabel, renderBar(aPct, t.FocusPurple)),
		"",
		fmt.Sprintf("  %s %s", eLabel, renderBar(ePct, t.P1Color)),
	)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 2
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🔥 HOURLY FOCUS HEATMAP", lines, w, h, borderCol)
}

func renderTopTagsPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	barTotalW := innerW - 24
	if barTotalW < 4 {
		barTotalW = 4
	}

	var lines []string
	if len(stats.Tags) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("  No tagged focus blocks found."))
	} else {
		maxSecs := stats.Tags[0].Secs
		if maxSecs == 0 {
			maxSecs = 1
		}
		for idx, tv := range stats.Tags {
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

			solidStyled := lipgloss.NewStyle().Foreground(t.FocusPurple).Render(solidBar)
			mutedStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(mutedBar)
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

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 0
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🏷️ TOP CATEGORY TAGS", lines, w, h, borderCol)
}

func renderProjectFocusRatiosPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	wsTime := make(map[string]int)
	totalWSTime := 0
	for _, task := range m.Tasks {
		if task.LifecycleState == model.StateCompleted {
			dur := task.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 && task.SchedulingType == model.Anchored {
				dur = int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Seconds())
			} else if dur == 0 {
				dur = task.StoryPoints * 45 * 60
			}
			wsTime[task.WorkspaceUUID] += dur
			totalWSTime += dur
		}
	}
	if totalWSTime == 0 {
		for _, task := range m.Tasks {
			wsTime[task.WorkspaceUUID]++
			totalWSTime++
		}
	}

	type wsRatio struct {
		name string
		pct  float64
		col  lipgloss.Color
	}
	var projects []wsRatio
	colors := []lipgloss.Color{t.P0Color, t.Accent, t.FocusPurple, t.SuccessColor, t.P1Color, t.P3Color}
	colorIdx := 0

	for _, ws := range m.Workspaces {
		sec := wsTime[ws.UUID]
		pct := 0.0
		if totalWSTime > 0 {
			pct = float64(sec) / float64(totalWSTime) * 100
		}

		col := colors[colorIdx%len(colors)]
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
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %2.0f%%", p.name, barStr, p.pct)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 4
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "💼 PROJECT FOCUS RATIOS", lines, w, h, borderCol)
}
