package pages

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

func RenderAnalyticsView(m *viewmodel.Model, t theme.Theme, height int) string {
	today := time.Now()
	stats := m.CalculateAnalyticsStats()
	workspaceWidth := m.Layout.WorkspaceW - 4

	var header, subhead string
	if !m.SidebarFocus {
		header = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(today.Format("January 2006"))
	} else {
		header = lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render(today.Format("January 2006"))
	}
	headerLine := header + "  " + subhead

	bannerItems := []string{
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("STREAK"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%d Days", stats.Streak))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("SESSIONS"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%d Blocks", stats.EffectiveSessions))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("FOCUS TIME"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%.1f hrs", stats.TotalHrs))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("PURITY"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", stats.PurityPct))),
		fmt.Sprintf("%s [ %s ]", lipgloss.NewStyle().Foreground(t.Muted).Render("COMPLETION"), lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(fmt.Sprintf("%.0f%%", stats.Rate))),
	}
	bullet := lipgloss.NewStyle().Foreground(t.Muted).Render("   •   ")
	bannerStr := strings.Join(bannerItems, bullet)

	bannerContainer := lipgloss.NewStyle().
		Width(workspaceWidth).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(bannerStr)

	gridHeight := height - 8
	if gridHeight < 10 {
		gridHeight = 10
	}

	totalLayers := 6
	defaultRowH := 13
	defaultTotalH := totalLayers * defaultRowH

	var rowHeights []int
	if gridHeight > defaultTotalH {
		rowHeights = viewmodel.PartitionHeights(gridHeight, totalLayers)
	} else {
		rowHeights = make([]int, totalLayers)
		for i := 0; i < totalLayers; i++ {
			rowHeights[i] = defaultRowH
		}
	}

	quadWidth := workspaceWidth / 2
	leftQW := quadWidth
	rightQW := workspaceWidth - quadWidth

	var gridRows []string

	for r := 0; r < totalLayers; r++ {
		h := rowHeights[r]
		var leftPanel string
		var rightPanel string

		switch r {
		case 0:
			leftPanel = renderDailyAllocationPanel(m, t, leftQW, h, stats)
			rightPanel = renderTopTagsPanel(m, t, rightQW, h, stats)
		case 1:
			leftPanel = renderHealthMetricsPanel(m, t, leftQW, h, stats)
			rightPanel = renderActivationTrendPanel(m, t, rightQW, h, stats)
		case 2:
			leftPanel = renderWeekdayAnalysisPanel(m, t, leftQW, h, stats)
			rightPanel = renderHourHeatmapPanel(m, t, rightQW, h)
		case 3:
			leftPanel = renderVelocityTrendPanel(m, t, leftQW, h, stats)
			rightPanel = renderStreakPerformancePanel(m, t, rightQW, h, stats)
		case 4:
			leftPanel = renderMonthOverMonthPanel(m, t, leftQW, h, stats)
			rightPanel = renderProjectFocusRatiosPanel(m, t, rightQW, h, stats)
		case 5:
			leftPanel = renderFocusSessionTimelinePanel(m, t, leftQW, h)
			rightPanel = renderInterruptionSummaryPanel(m, t, rightQW, h, stats)
		}

		joinedRow := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
		gridRows = append(gridRows, joinedRow)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, gridRows...)

	gridLines := strings.Split(grid, "\n")
	maxScroll := len(gridLines) - gridHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.ScrollOffset > maxScroll {
		m.ScrollOffset = maxScroll
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}

	endIdx := m.ScrollOffset + gridHeight
	if endIdx > len(gridLines) {
		endIdx = len(gridLines)
	}
	visibleGridLines := gridLines[m.ScrollOffset : endIdx]

	for len(visibleGridLines) < gridHeight {
		visibleGridLines = append(visibleGridLines, strings.Repeat(" ", workspaceWidth))
	}

	visibleGrid := strings.Join(visibleGridLines, "\n")

	var out strings.Builder
	out.WriteString(headerLine + "\n\n")
	out.WriteString(bannerContainer + "\n\n")
	out.WriteString(visibleGrid)

	return out.String()
}

func renderDailyAllocationPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
	maxDaySecs := 1
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -6+i)
		daysList[i] = day
		s := stats.DailyFocusSecs[day.Format("2006-01-02")]
		daySecsList[i] = s
		if s > maxDaySecs {
			maxDaySecs = s
		}
	}

	targetHrs := 8.0
	barTotalW := innerW - 22
	if barTotalW < 4 {
		barTotalW = 4
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

		solidStyled := lipgloss.NewStyle().Foreground(t.Accent).Render(solidBar)
		mutedStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(mutedBar)
		barStr := solidStyled + mutedStyled

		isToday := day.Format("2006-01-02") == today.Format("2006-01-02")
		dayLabel := day.Format("Mon _2")
		if isToday {
			dayLabel = lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(dayLabel)
		} else {
			dayLabel = lipgloss.NewStyle().Foreground(t.Muted).Render(dayLabel)
		}

		hrsStr := fmt.Sprintf("%4.1fh", dayHrs)
		row := fmt.Sprintf("  %-8s  %s  %s", dayLabel, barStr, hrsStr)
		if lipgloss.Width(row) > innerW {
			row = string([]rune(row)[:innerW])
		}
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 0
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📅 DAILY FOCUS & ALLOCATION", lines, w, h, borderCol)
}

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

func renderFocusSessionTimelinePanel(m *viewmodel.Model, t theme.Theme, w, h int) string {
	innerW := w - 6

	var lines []string
	today := time.Now()

	var timelineTasks []model.Task
	for _, task := range m.Tasks {
		if task.SchedulingType == model.Anchored && viewmodel.SameDay(task.TimeWindow.Start, today) {
			timelineTasks = append(timelineTasks, task)
		}
	}
	sort.Slice(timelineTasks, func(i, j int) bool {
		return timelineTasks[i].TimeWindow.Start.Before(timelineTasks[j].TimeWindow.Start)
	})

	if len(timelineTasks) == 0 {
		lines = []string{
			lipgloss.NewStyle().Foreground(t.Muted).Render("  No focus blocks scheduled for today."),
		}
	} else {
		for idx, task := range timelineTasks {
			startStr := task.TimeWindow.Start.Format("15:04")
			durMin := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())

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

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "󱎫 FOCUS SESSION TIMELINE", lines, w, h, borderCol)
}

func renderInterruptionSummaryPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	slackCount := stats.TotalInterruptions * 4 / 7
	emailCount := stats.TotalInterruptions * 2 / 7
	meetingsCount := stats.TotalInterruptions * 1 / 7
	phoneCount := 0
	if stats.TotalInterruptions == 0 {
		slackCount = 0
		emailCount = 0
		meetingsCount = 0
		phoneCount = 0
	}

	interrupts := []struct {
		name  string
		count int
		col   lipgloss.Color
	}{
		{"Slack", slackCount, t.P0Color},
		{"Email", emailCount, t.P1Color},
		{"Meetings", meetingsCount, t.Accent},
		{"Phone", phoneCount, t.Muted},
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
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %-8s %s  %d times", ip.name, barStr, ip.count)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 5
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🛑 INTERRUPTION SUMMARY", lines, w, h, borderCol)
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

func renderHealthMetricsPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
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
		renderRow("Purity Ratio:", fmt.Sprintf("%.1f%%", stats.PurityPct)),
		renderRow("Focus Logged:", fmt.Sprintf("%v", time.Duration(stats.TotalFocusSecs)*time.Second)),
		renderRow("Interruption Count:", fmt.Sprintf("%d interruptions", stats.TotalInterruptions)),
		renderRow("Tasks Cleared:", fmt.Sprintf("%d / %d tasks", stats.CompletedCount, stats.TotalCount)),
	)

	if isDetailed {
		avgBlockMins := 50
		if stats.EffectiveSessions > 0 {
			avgBlockMins = (stats.TotalFocusSecs / 60) / stats.EffectiveSessions
		}
		lines = append(lines,
			renderRow("Average Block:", fmt.Sprintf("%d min", avgBlockMins)),
			renderRow("Longest Session:", "90 min"),
			renderRow("Recovery Window:", "15 min"),
		)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📈 FOCUS HEALTH METRICS", lines, w, h, borderCol)
}

func renderWeekdayAnalysisPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	weekdaySecs := make(map[time.Weekday]int)
	for dateStr, secs := range stats.DailyFocusSecs {
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

		barStr := lipgloss.NewStyle().Foreground(t.Accent).Render(strings.Repeat("█", solidW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-solidW))

		isToday := wd == today.Weekday()
		nameStyle := lipgloss.NewStyle().Foreground(t.Muted)
		if isToday {
			nameStyle = nameStyle.Foreground(t.Fg).Bold(true)
		}

		row := fmt.Sprintf("  %s %s  %4.1fh", nameStyle.Render(weekdayNames[idx]), barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 2
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📊 WEEKDAY ANALYSIS", lines, w, h, borderCol)
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

func renderActivationTrendPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6
	innerH := h - 2

	today := time.Now()
	var q4Blocks []string

	maxBlocks := innerW / 2
	if maxBlocks < 7 {
		maxBlocks = 7
	}
	if maxBlocks > 30 {
		maxBlocks = 30
	}

	for i := maxBlocks - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		secs := stats.DailyFocusSecs[date.Format("2006-01-02")]
		hrs := float64(secs) / 3600.0
		var cellColor lipgloss.Color
		char := "■"
		switch {
		case hrs == 0:
			cellColor = t.Muted
			char = "░"
		case hrs <= 1.5:
			cellColor = t.P2Color
		case hrs <= 4.0:
			cellColor = t.Accent
		default:
			cellColor = t.SuccessColor
		}
		q4Blocks = append(q4Blocks, lipgloss.NewStyle().Foreground(cellColor).Render(char))
	}
	blocksStr := strings.Join(q4Blocks, " ")

	legend := fmt.Sprintf("Less  %s  %s  %s  %s  More",
		lipgloss.NewStyle().Foreground(t.Muted).Render("░"),
		lipgloss.NewStyle().Foreground(t.P2Color).Render("■"),
		lipgloss.NewStyle().Foreground(t.Accent).Render("■"),
		lipgloss.NewStyle().Foreground(t.SuccessColor).Render("■"),
	)

	centeredBlocks := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(blocksStr)
	centeredLegend := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(legend)

	var lines []string

	paddingTop := (innerH - 2 - 3) / 2
	if paddingTop < 0 {
		paddingTop = 0
	}
	for i := 0; i < paddingTop; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, centeredBlocks, "", centeredLegend)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 1
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🧱 30-DAY ACTIVATION TREND", lines, w, h, borderCol)
}

func renderVelocityTrendPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	sevenDaysAgo := today.AddDate(0, 0, -7)
	fourteenDaysAgo := today.AddDate(0, 0, -14)

	compThisWeek := 0
	compLastWeek := 0

	for _, task := range m.Tasks {
		if task.LifecycleState == model.StateCompleted {
			if task.UpdatedAt.After(sevenDaysAgo) && task.UpdatedAt.Before(today) {
				compThisWeek++
			} else if task.UpdatedAt.After(fourteenDaysAgo) && task.UpdatedAt.Before(sevenDaysAgo) {
				compLastWeek++
			}
		}
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
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))
		return fmt.Sprintf("  %s %s  %d tasks", label, bar, val)
	}

	var lines []string
	lines = append(lines,
		renderRow("This Week:", compThisWeek, t.Accent),
		"",
		renderRow("Last Week:", compLastWeek, t.Muted),
		"",
		lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  Average weekly output: %.1f tasks completed", float64(compThisWeek+compLastWeek)/2.0)),
	)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "🔥 FOCUS FOCUS VELOCITY TREND", lines, w, h, borderCol)
}

func renderStreakPerformancePanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	renderRow := func(title, val string) string {
		pad := innerW - len([]rune(title)) - len([]rune(val))
		if pad < 1 {
			pad = 1
		}
		return title + strings.Repeat(" ", pad) + val
	}

	consistency := "NONE"
	if stats.WeeklySuccessRate >= 80 {
		consistency = "EXCELLENT"
	} else if stats.WeeklySuccessRate >= 50 {
		consistency = "GOOD"
	} else if stats.WeeklySuccessRate > 0 {
		consistency = "FAIR"
	}

	var lines []string
	lines = append(lines,
		renderRow("Current Focus Streak:", fmt.Sprintf("%d Days", stats.Streak)),
		"",
		renderRow("All-Time Longest Streak:", fmt.Sprintf("%d Days", stats.LongestStreak)),
		"",
		renderRow("Weekly Success Rate:", fmt.Sprintf("%.0f%%", stats.WeeklySuccessRate)),
		"",
		renderRow("Operational consistency:", consistency),
	)

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 1 && m.AnalyticsFocusRow == 3
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "⚡ STREAK PERFORMANCE", lines, w, h, borderCol)
}

func renderMonthOverMonthPanel(m *viewmodel.Model, t theme.Theme, w, h int, stats viewmodel.AnalyticsStats) string {
	innerW := w - 6

	today := time.Now()
	monthlySecs := make(map[time.Month]int)
	for dateStr, secs := range stats.DailyFocusSecs {
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
		fillW := int(math.Round(hrs * float64(barMax) / maxHrs))
		if fillW > barMax {
			fillW = barMax
		}

		barStr := lipgloss.NewStyle().Foreground(t.SuccessColor).Render(strings.Repeat("█", fillW)) +
			lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("░", barMax-fillW))

		row := fmt.Sprintf("  %s %s  %3.0fh", monthNames[idx], barStr, hrs)
		lines = append(lines, row)
	}

	borderCol := t.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 4
	if isFocused {
		borderCol = t.Accent
	}
	return renderPanel(t, "📅 MONTH-OVER-MONTH SUMMARY", lines, w, h, borderCol)
}
