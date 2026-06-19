package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/view/theme"
	"stream/internal/viewmodel"

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
