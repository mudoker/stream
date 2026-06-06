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
	workspaceWidth := m.Layout.WorkspaceW - 4

	var header, subhead string
	if !m.SidebarFocus {
		header = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(today.Format("January 2006"))
	} else {
		header = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("▲ Analytics")
		subhead = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render(today.Format("January 2006"))
	}
	headerLine := header + "  " + subhead

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

	gridHeight := height - 8
	if gridHeight < 10 {
		gridHeight = 10
	}

	totalLayers := 6
	defaultRowH := 13
	defaultTotalH := totalLayers * defaultRowH

	var rowHeights []int
	if gridHeight > defaultTotalH {
		rowHeights = partitionHeights(gridHeight, totalLayers)
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
			leftPanel = m.renderDailyAllocationPanel(leftQW, h, stats)
			rightPanel = m.renderTopTagsPanel(rightQW, h, stats)
		case 1:
			leftPanel = m.renderHealthMetricsPanel(leftQW, h, stats)
			rightPanel = m.renderActivationTrendPanel(rightQW, h, stats)
		case 2:
			leftPanel = m.renderWeekdayAnalysisPanel(leftQW, h, stats)
			rightPanel = m.renderHourHeatmapPanel(rightQW, h)
		case 3:
			leftPanel = m.renderVelocityTrendPanel(leftQW, h, stats)
			rightPanel = m.renderStreakPerformancePanel(rightQW, h, stats)
		case 4:
			leftPanel = m.renderMonthOverMonthPanel(leftQW, h, stats)
			rightPanel = m.renderProjectFocusRatiosPanel(rightQW, h, stats)
		case 5:
			leftPanel = m.renderFocusSessionTimelinePanel(leftQW, h)
			rightPanel = m.renderInterruptionSummaryPanel(rightQW, h, stats)
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

	borderCol := m.Theme.Muted
	isFocused := !m.SidebarFocus && m.AnalyticsFocusCol == 0 && m.AnalyticsFocusRow == 0
	if isFocused {
		borderCol = m.Theme.Accent
	}
	return m.renderPanel("📅 DAILY FOCUS & ALLOCATION", lines, w, h, borderCol)
}
