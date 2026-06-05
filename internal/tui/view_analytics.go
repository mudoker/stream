package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAnalyticsView(height int) string {
	today := time.Now()
	stats := m.calculateAnalyticsStats()

	// 1. Top KPI Row Cards
	cardStyle := lipgloss.NewStyle().
		Background(m.Theme.PanelBg).
		Padding(0, 2).
		Height(3)

	card1 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("STREAK"),
		lipgloss.NewStyle().Foreground(m.Theme.P1Color).Bold(true).Render(fmt.Sprintf("🔥 %d DAYS", stats.streak)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Consecutive"),
	))

	card2 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("SESSIONS"),
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(fmt.Sprintf("🎯 %d BLOCKS", stats.effectiveSessions)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Pomodoros"),
	))

	card3 := cardStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("FOCUS TIME"),
		lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(fmt.Sprintf("⏱️ %.1f HRS", stats.totalHrs)),
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Work + Pers"),
	))

	sidebarWidth := int(float64(m.Width) * 0.13)
	if sidebarWidth < 18 {
		sidebarWidth = 18
	} else if sidebarWidth > 26 {
		sidebarWidth = 26
	}
	workspaceWidth := m.Width - sidebarWidth - 3
	if workspaceWidth < 40 {
		workspaceWidth = 40
	}

	var kpiRow string
	if workspaceWidth >= 60 {
		kpiRow = lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3)
	} else {
		kpiRow = lipgloss.JoinVertical(lipgloss.Left, card1, "\n", card2, "\n", card3)
	}

	// 2. Left Column: Daily timeline & focus purity
	var timelineLines []string
	timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("DAILY PRODUCTIVITY TIMELINE"))

	// Compute last 7 days daily focus
	maxDaySecs := 0
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -6+i)
		daysList[i] = day
		daySecs := stats.dailyFocusSecs[day.Format("2006-01-02")]
		daySecsList[i] = daySecs
		if daySecs > maxDaySecs {
			maxDaySecs = daySecs
		}
	}
	if maxDaySecs == 0 {
		maxDaySecs = 1
	}

	timelineBarWidth := 15
	for i := 0; i < 7; i++ {
		day := daysList[i]
		daySecs := daySecsList[i]
		dayHrs := float64(daySecs) / 3600.0

		pct := float64(daySecs) / float64(maxDaySecs)
		barLen := int(math.Round(pct * float64(timelineBarWidth)))
		barStr := strings.Repeat("█", barLen)
		if barStr == "" && dayHrs > 0 {
			barStr = "▏"
		}

		var barColor lipgloss.Color
		if dayHrs == 0 {
			barColor = m.Theme.Muted
		} else if dayHrs <= 2.0 {
			barColor = m.Theme.P2Color
		} else if dayHrs <= 5.0 {
			barColor = m.Theme.Accent
		} else {
			barColor = m.Theme.SuccessColor
		}

		coloredBar := lipgloss.NewStyle().Foreground(barColor).Render(barStr)
		timelineLines = append(timelineLines, fmt.Sprintf("  %s │ %s %.1fh", day.Format("Jan _2"), coloredBar, dayHrs))
	}

	var statsLines []string
	statsLines = append(statsLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("FOCUS HEALTH & DISTRACTION"))
	statsLines = append(statsLines, fmt.Sprintf("  Purity (No Interrupt): %.1f%%", stats.purityPct))
	statsLines = append(statsLines, fmt.Sprintf("  Total Focus Logged:    %s", time.Duration(stats.totalFocusSecs)*time.Second))
	statsLines = append(statsLines, fmt.Sprintf("  Total Interruptions:   %d times", stats.totalInterruptions))
	statsLines = append(statsLines, fmt.Sprintf("  Completed Tasks Rate:  %d/%d (%.1f%%)", stats.completedCount, stats.totalCount, stats.rate))

	// 3. Right Column: Time distribution ratios & Top tags & 30-Day Trend
	var ratioLines []string
	ratioLines = append(ratioLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TIME ALLOCATION RATIOS"))

	ratioBarWidth := 20
	var coloredRatioBar string
	if stats.totalHrs == 0 {
		coloredRatioBar = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("·", ratioBarWidth))
	} else {
		workBarLen := int(math.Round(stats.workPct * float64(ratioBarWidth)))
		persBarLen := ratioBarWidth - workBarLen
		if workBarLen == 0 && stats.workHrs > 0 {
			workBarLen = 1
		}
		if persBarLen == 0 && stats.personalHrs > 0 {
			persBarLen = 1
		}

		if workBarLen+persBarLen > ratioBarWidth {
			if workBarLen > persBarLen {
				workBarLen = ratioBarWidth - persBarLen
			} else {
				persBarLen = ratioBarWidth - workBarLen
			}
		}

		workBarStr := strings.Repeat("█", workBarLen)
		persBarStr := strings.Repeat("█", persBarLen)
		coloredRatioBar = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(workBarStr) +
			lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(persBarStr)
	}

	ratioLines = append(ratioLines, fmt.Sprintf("  %s", coloredRatioBar))
	ratioLines = append(ratioLines, fmt.Sprintf("  Work Focus     %.0f%% (%.1fh)", stats.workPct*100, stats.workHrs))
	ratioLines = append(ratioLines, fmt.Sprintf("  Personal Focus %.0f%% (%.1fh)", stats.personalPct*100, stats.personalHrs))

	var tagLines []string
	tagLines = append(tagLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TOP TAGS BY FOCUS TIME"))
	if len(stats.tags) == 0 {
		tagLines = append(tagLines, "  No tagged sessions logged.")
	} else {
		maxSecs := stats.tags[0].Secs
		if maxSecs == 0 {
			maxSecs = 1
		}
		for idx, tv := range stats.tags {
			if idx >= 3 {
				break
			}
			hrs := float64(tv.Secs) / 3600.0
			pct := float64(tv.Secs) / float64(maxSecs)
			barLen := int(math.Round(pct * 12))
			barStr := strings.Repeat("█", barLen)
			if barStr == "" && tv.Secs > 0 {
				barStr = "▏"
			}
			coloredBar := lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render(barStr)

			tagName := tv.Tag
			if len(tagName) > 8 {
				tagName = tagName[:7] + "…"
			}
			tagLines = append(tagLines, fmt.Sprintf("  %-8s %s %.1fh", tagName, coloredBar, hrs))
		}
	}

	// 30-Day activity trend sparkline
	var heatmapLines []string
	heatmapLines = append(heatmapLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("30-DAY FOCUS TREND"))

	var trendSB strings.Builder
	trendSB.WriteString("  ")
	for i := 29; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		secs := stats.dailyFocusSecs[dateStr]
		hrs := float64(secs) / 3600.0

		var cellColor lipgloss.Color
		if hrs == 0 {
			cellColor = m.Theme.Muted
		} else if hrs <= 1.5 {
			cellColor = m.Theme.P2Color
		} else if hrs <= 4.0 {
			cellColor = m.Theme.Accent
		} else {
			cellColor = m.Theme.SuccessColor
		}

		char := "■"
		if hrs == 0 {
			char = "·"
		}
		trendSB.WriteString(lipgloss.NewStyle().Foreground(cellColor).Render(char) + " ")
	}
	heatmapLines = append(heatmapLines, trendSB.String())

	legendSB := strings.Builder{}
	legendSB.WriteString("    Less ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("·") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.P2Color).Render("■") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("■") + " ")
	legendSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("■") + " ")
	legendSB.WriteString(" More")
	heatmapLines = append(heatmapLines, legendSB.String())

	var contentSB strings.Builder
	contentSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲  E X E C U T I O N   A N A L Y T I C S") + "\n")
	contentSB.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", workspaceWidth-4)) + "\n\n")
	contentSB.WriteString(kpiRow + "\n\n")

	if workspaceWidth >= 75 {
		leftColContent := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(timelineLines, "\n"),
			"\n",
			strings.Join(statsLines, "\n"),
		)

		rightColContent := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(ratioLines, "\n"),
			"\n",
			strings.Join(tagLines, "\n"),
			"\n",
			strings.Join(heatmapLines, "\n"),
		)

		leftStyled := lipgloss.NewStyle().Width(36).Render(leftColContent)
		rightStyled := lipgloss.NewStyle().Width(workspaceWidth - 40).Render(rightColContent)

		columns := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, "    ", rightStyled)
		contentSB.WriteString(columns)
	} else {
		stacked := lipgloss.JoinVertical(lipgloss.Left,
			strings.Join(timelineLines, "\n"),
			"\n",
			strings.Join(statsLines, "\n"),
			"\n",
			strings.Join(ratioLines, "\n"),
			"\n",
			strings.Join(tagLines, "\n"),
			"\n",
			strings.Join(heatmapLines, "\n"),
		)
		contentSB.WriteString(stacked)
	}

	return m.Theme.PanelStyle.
		Width(m.Width - 28).
		Height(height).
		Render(contentSB.String())
}
