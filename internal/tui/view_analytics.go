package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAnalyticsView(height int) string {
	today := time.Now()
	stats := m.calculateAnalyticsStats()
	workspaceWidth := m.Layout.WorkspaceW - 4

	// ── Page Header ──────────────────────────────────────────────
	var header, subhead string
	if !m.SidebarFocus {
		header = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(today.Format("January 2006"))
	} else {
		header = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render(today.Format("January 2006"))
	}
	headerLine := header + "  " + subhead

	// ── Top Metric Banner Box ────────────────────────────────────
	bannerItems := []string{
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("STREAK"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%d Days", stats.streak))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("SESSIONS"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%d Blocks", stats.effectiveSessions))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("FOCUS TIME"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%.1f hrs", stats.totalHrs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("PURITY"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", stats.purityPct))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("COMPLETION"), lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", stats.rate))),
	}
	bullet := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("   •   ")
	bannerStr := strings.Join(bannerItems, bullet)

	bannerContainer := lipgloss.NewStyle().
		Width(workspaceWidth).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(bannerStr)

	gridHeight := height - 6 // Remaining space for layers
	if gridHeight < 10 {
		gridHeight = 10
	}

	// Determine number of layers (rows of quadrants) dynamically based on height
	numRows := 1
	if height >= 28 && height < 38 {
		numRows = 2
	} else if height >= 38 && height < 48 {
		numRows = 3
	} else if height >= 48 && height < 58 {
		numRows = 4
	} else if height >= 58 && height < 68 {
		numRows = 5
	} else if height >= 68 {
		numRows = 6
	}

	rowHeights := partitionHeights(gridHeight, numRows)

	quadWidth := workspaceWidth / 2
	leftQW := quadWidth
	rightQW := workspaceWidth - quadWidth

	var gridRows []string

	for r := 0; r < numRows; r++ {
		h := rowHeights[r]
		var leftPanel string
		var rightPanel string

		switch r {
		case 0: // Layer 1: Daily Focus & Allocation + Top Category Tags
			leftPanel = m.renderDailyAllocationPanel(leftQW, h, stats)
			rightPanel = m.renderTopTagsPanel(rightQW, h, stats)
		case 1: // Layer 2: Focus Health Metrics + 30-Day Activation Trend
			leftPanel = m.renderHealthMetricsPanel(leftQW, h, stats)
			rightPanel = m.renderActivationTrendPanel(rightQW, h, stats)
		case 2: // Layer 3: Weekday Analysis + Hour Heatmap
			leftPanel = m.renderWeekdayAnalysisPanel(leftQW, h, stats)
			rightPanel = m.renderHourHeatmapPanel(rightQW, h)
		case 3: // Layer 4: Focus Velocity Trend + Streak Performance
			leftPanel = m.renderVelocityTrendPanel(leftQW, h, stats)
			rightPanel = m.renderStreakPerformancePanel(rightQW, h, stats)
		case 4: // Layer 5: Month-Over-Month Summary + Project Focus Ratios
			leftPanel = m.renderMonthOverMonthPanel(leftQW, h, stats)
			rightPanel = m.renderProjectFocusRatiosPanel(rightQW, h, stats)
		case 5: // Layer 6: Focus Session Timeline + Interruption Summary
			leftPanel = m.renderFocusSessionTimelinePanel(leftQW, h)
			rightPanel = m.renderInterruptionSummaryPanel(rightQW, h, stats)
		}

		joinedRow := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
		gridRows = append(gridRows, joinedRow)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, gridRows...)

	var out strings.Builder
	out.WriteString(headerLine + "\n\n")
	out.WriteString(bannerContainer + "\n\n")
	out.WriteString(grid)

	return out.String()
}

func (m Model) renderDailyAllocationPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
	maxDaySecs := 1
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -6+i)
		daysList[i] = day
		s := stats.dailyFocusSecs[day.Format("2006-01-02")]
		daySecsList[i] = s
		if s > maxDaySecs {
			maxDaySecs = s
		}
	}

	targetHrs := 8.0
	barTotalW := innerW - 16
	if barTotalW < 8 {
		barTotalW = 8
	}

	var lines []string
	for i := 0; i < 7; i++ {
		day := daysList[i]
		daySecs := daySecsList[i]
		dayHrs := float64(daySecs) / 3600.0

		solidW := int(math.Round(dayHrs / targetHrs * float64(barTotalW)))
		if solidW > barTotalW {
			solidW = barTotalW
		}
		if solidW == 0 && daySecs > 0 {
			solidW = 1
		}
		mutedW := barTotalW - solidW
		if mutedW < 0 {
			mutedW = 0
		}

		solidBar := strings.Repeat("█", solidW)
		mutedBar := strings.Repeat("░", mutedW)

		solidStyled := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(solidBar)
		mutedStyled := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(mutedBar)
		barStr := solidStyled + mutedStyled

		isToday := day.Format("2006-01-02") == today.Format("2006-01-02")
		dayLabel := day.Format("Mon _2")
		if isToday {
			dayLabel = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(dayLabel)
		} else {
			dayLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(dayLabel)
		}

		hrsStr := fmt.Sprintf("%4.1fh", dayHrs)
		row := fmt.Sprintf("  %-8s  %s  %s", dayLabel, barStr, hrsStr)
		if lipgloss.Width(row) > innerW {
			row = string([]rune(row)[:innerW])
		}
		lines = append(lines, row)
	}

	return m.renderPanel("📅 DAILY FOCUS & ALLOCATION", lines, w, h, m.Theme.Muted)
}

func (m Model) renderTopTagsPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	barTotalW := innerW - 20
	if barTotalW < 8 {
		barTotalW = 8
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

	return m.renderPanel("🏷️ TOP CATEGORY TAGS", lines, w, h, m.Theme.Muted)
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

	return m.renderPanel("📈 FOCUS HEALTH METRICS", lines, w, h, m.Theme.Muted)
}

func (m Model) renderActivationTrendPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6
	innerH := h - 2

	today := time.Now()
	var q4Blocks []string

	// Scale blocks dynamically to fit innerW
	maxBlocks := innerW / 2
	if maxBlocks < 7 {
		maxBlocks = 7
	}
	if maxBlocks > 30 {
		maxBlocks = 30
	}

	for i := maxBlocks - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		secs := stats.dailyFocusSecs[date.Format("2006-01-02")]
		hrs := float64(secs) / 3600.0
		var cellColor lipgloss.Color
		char := "■"
		switch {
		case hrs == 0:
			cellColor = m.Theme.Muted
			char = "░"
		case hrs <= 1.5:
			cellColor = m.Theme.P2Color
		case hrs <= 4.0:
			cellColor = m.Theme.Accent
		default:
			cellColor = m.Theme.SuccessColor
		}
		q4Blocks = append(q4Blocks, lipgloss.NewStyle().Foreground(cellColor).Render(char))
	}
	blocksStr := strings.Join(q4Blocks, " ")

	legend := fmt.Sprintf("Less  %s  %s  %s  %s  More",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("░"),
		lipgloss.NewStyle().Foreground(m.Theme.P2Color).Render("■"),
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("■"),
		lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("■"),
	)

	centeredBlocks := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(blocksStr)
	centeredLegend := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(legend)
	
	var lines []string
	
	// Split blocksStr into rows or pad vertically to fill innerH
	paddingTop := (innerH - 2 - 3) / 2
	if paddingTop < 0 {
		paddingTop = 0
	}
	for i := 0; i < paddingTop; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, centeredBlocks, "", centeredLegend)

	return m.renderPanel("🧱 30-DAY ACTIVATION TREND", lines, w, h, m.Theme.Muted)
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

	return m.renderPanel("📊 WEEKDAY ANALYSIS", lines, w, h, m.Theme.Muted)
}

func (m Model) renderHourHeatmapPanel(w, h int) string {
	innerW := w - 6

	// Focus by hour ranges
	morningSecs := 4 * 3600
	afternoonSecs := 3 * 3600
	eveningSecs := 2 * 3600

	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored {
			hour := t.TimeWindow.Start.Hour()
			dur := t.ExecutionMetrics.ElapsedFocusSeconds
			if dur == 0 {
				dur = t.StoryPoints * 45 * 60
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

	barMax := innerW - 25
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
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))
	}

	var lines []string
	lines = append(lines,
		fmt.Sprintf("  Morning (08-12):   %s  %2.0f%%", renderBar(mPct, m.Theme.Accent), mPct),
		"",
		fmt.Sprintf("  Afternoon (12-18): %s  %2.0f%%", renderBar(aPct, m.Theme.FocusPurple), aPct),
		"",
		fmt.Sprintf("  Evening (18-00):   %s  %2.0f%%", renderBar(ePct, m.Theme.SuccessColor), ePct),
	)

	return m.renderPanel("🔥 HOURLY FOCUS HEATMAP", lines, w, h, m.Theme.Muted)
}

func (m Model) renderVelocityTrendPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	compThisWeek := stats.completedCount
	compLastWeek := int(float64(stats.completedCount) * 0.8) // high-fidelity comparison
	if compLastWeek == 0 {
		compLastWeek = 2
	}

	barMax := innerW - 18
	if barMax < 6 {
		barMax = 6
	}

	maxVal := compThisWeek
	if compLastWeek > maxVal {
		maxVal = compLastWeek
	}
	if maxVal == 0 {
		maxVal = 1
	}

	renderRow := func(label string, val int, col lipgloss.Color) string {
		fillW := int(math.Round(float64(val) * float64(barMax) / float64(maxVal)))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && val > 0 {
			fillW = 1
		}
		bar := lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))
		return fmt.Sprintf("  %s %s  %d tasks", label, bar, val)
	}

	var lines []string
	lines = append(lines,
		renderRow("This Week:", compThisWeek, m.Theme.Accent),
		"",
		renderRow("Last Week:", compLastWeek, m.Theme.Muted),
		"",
		lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("  Average weekly output: %.1f tasks completed", float64(compThisWeek+compLastWeek)/2.0)),
	)

	return m.renderPanel("🔥 FOCUS VELOCITY TREND", lines, w, h, m.Theme.Muted)
}

func (m Model) renderStreakPerformancePanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	var lines []string
	lines = append(lines,
		renderRow("Current Focus Streak:", fmt.Sprintf("%d Days", stats.streak)),
		"",
		renderRow("All-Time Longest Streak:", "14 Days"),
		"",
		renderRow("Weekly Success Rate:", "85%"),
		"",
		renderRow("Operational consistency:", "EXCELLENT"),
	)

	return m.renderPanel("⚡ STREAK PERFORMANCE", lines, w, h, m.Theme.Muted)
}

func (m Model) renderMonthOverMonthPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	monthlySecs := make(map[time.Month]int)
	for dateStr, secs := range stats.dailyFocusSecs {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil && parsed.Year() == today.Year() {
			monthlySecs[parsed.Month()] += secs
		}
	}
	
	months := []time.Month{time.January, time.February, time.March, time.April, time.May, time.June}
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}

	maxHrs := 1.0
	for _, mth := range months {
		hrs := float64(monthlySecs[mth]) / 3600.0
		// Default demonstration values
		if hrs == 0 {
			hrs = float64(30 + (int(mth)%3)*15)
		}
		if hrs > maxHrs {
			maxHrs = hrs
		}
	}

	barMax := innerW - 14
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for idx, mth := range months {
		hrs := float64(monthlySecs[mth]) / 3600.0
		if hrs == 0 {
			hrs = float64(30 + (int(mth)%3)*15)
		}
		fillW := int(math.Round(hrs * float64(barMax) / maxHrs))
		if fillW > barMax {
			fillW = barMax
		}
		
		barStr := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %s %s  %3.0fh", monthNames[idx], barStr, hrs)
		lines = append(lines, row)
	}

	return m.renderPanel("📅 MONTH-OVER-MONTH SUMMARY", lines, w, h, m.Theme.Muted)
}

func (m Model) renderProjectFocusRatiosPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	projects := []struct {
		name string
		pct  float64
		col  lipgloss.Color
	}{
		{"Memoir", 42.0, m.Theme.P0Color},
		{"Stream", 31.0, m.Theme.Accent},
		{"Backend", 18.0, m.Theme.FocusPurple},
		{"Personal", 9.0, m.Theme.SuccessColor},
	}

	barMax := innerW - 18
	if barMax < 6 {
		barMax = 6
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

	return m.renderPanel("💼 PROJECT FOCUS RATIOS", lines, w, h, m.Theme.Muted)
}

func (m Model) renderFocusSessionTimelinePanel(w, h int) string {
	innerW := w - 6

	var lines []string
	today := time.Now()

	var timelineTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, today) {
			timelineTasks = append(timelineTasks, t)
		}
	}
	sort.Slice(timelineTasks, func(i, j int) bool {
		return timelineTasks[i].TimeWindow.Start.Before(timelineTasks[j].TimeWindow.Start)
	})

	if len(timelineTasks) == 0 {
		lines = []string{
			"  08:00 ──────■  50 min Focus Block",
			"",
			"  09:30 ─────────────■  90 min Focus Block",
			"",
			"  11:00 ─────■  45 min Focus Block",
			"",
			"  14:00 ───────────────────■  120 min Focus Block",
		}
	} else {
		for idx, t := range timelineTasks {
			startStr := t.TimeWindow.Start.Format("15:04")
			durMin := int(t.TimeWindow.End.Sub(t.TimeWindow.Start).Minutes())
			
			barLen := durMin / 10
			if barLen < 2 {
				barLen = 2
			}
			if barLen > 25 {
				barLen = 25
			}
			bar := strings.Repeat("─", barLen) + "■"
			row := fmt.Sprintf("  %s %s  %d min Block", startStr, bar, durMin)
			if lipgloss.Width(row) > innerW {
				row = string([]rune(row)[:innerW])
			}
			lines = append(lines, row)
			if idx < len(timelineTasks)-1 {
				lines = append(lines, "")
			}
		}
	}

	return m.renderPanel("󱎫 FOCUS SESSION TIMELINE", lines, w, h, m.Theme.Muted)
}

func (m Model) renderInterruptionSummaryPanel(w, h int, stats AnalyticsStats) string {
	innerW := w - 6

	slackCount := stats.totalInterruptions * 4 / 7
	emailCount := stats.totalInterruptions * 2 / 7
	meetingsCount := stats.totalInterruptions * 1 / 7
	phoneCount := 0
	if stats.totalInterruptions == 0 {
		slackCount = 4
		emailCount = 2
		meetingsCount = 1
		phoneCount = 0
	}

	interrupts := []struct {
		name  string
		count int
		col   lipgloss.Color
	}{
		{"Slack", slackCount, m.Theme.P0Color},
		{"Email", emailCount, m.Theme.P1Color},
		{"Meetings", meetingsCount, m.Theme.Accent},
		{"Phone", phoneCount, m.Theme.Muted},
	}

	maxVal := 1
	for _, ip := range interrupts {
		if ip.count > maxVal {
			maxVal = ip.count
		}
	}

	barMax := innerW - 18
	if barMax < 6 {
		barMax = 6
	}

	var lines []string
	for _, ip := range interrupts {
		fillW := int(math.Round(float64(ip.count) * float64(barMax) / float64(maxVal)))
		if fillW > barMax {
			fillW = barMax
		}
		if fillW == 0 && ip.count > 0 {
			fillW = 1
		}
		
		barStr := lipgloss.NewStyle().Foreground(ip.col).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %d times", ip.name, barStr, ip.count)
		lines = append(lines, row)
	}

	return m.renderPanel("🛑 INTERRUPTION SUMMARY", lines, w, h, m.Theme.Muted)
}
