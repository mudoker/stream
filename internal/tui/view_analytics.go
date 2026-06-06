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

	// ── Page Header ──────────────────────────────────────────────
	header := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲ Analytics")
	subhead := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(today.Format("January 2006"))
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

	// ── Balanced Four-Quadrant Data Grid ─────────────────────────
	quadWidth := (workspaceWidth - 4) / 2
	quadHeight := (height - 12) / 2
	if quadHeight < 10 {
		quadHeight = 10
	}

	makeQuad := func(title string, content string) string {
		return lipgloss.NewStyle().
			Width(quadWidth - 2).
			Height(quadHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.Theme.Muted).
			Padding(1, 2).
			Render(
				lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(title) + "\n\n" +
				content,
			)
	}

	// Quadrant 1: Daily Focus & Allocation (Top Left)
	var q1Lines []string
	maxDaySecs := 1
	daySecsList := make([]int, 7)
	daysList := make([]time.Time, 7)
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
	barTotalW := 20
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
		q1Lines = append(q1Lines, fmt.Sprintf("  %-8s  %s  %s", dayLabel, barStr, hrsStr))
	}
	q1Content := strings.Join(q1Lines, "\n")

	// Quadrant 2: Top Tags Breakdown (Top Right)
	var q2Lines []string
	if len(stats.tags) == 0 {
		q2Lines = append(q2Lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No tagged sessions."))
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
			q2Lines = append(q2Lines, fmt.Sprintf("  %-12s  %s  %4.1fh", tagLbl, barStr, hrs))
		}
	}
	q2Content := strings.Join(q2Lines, "\n")

	// Quadrant 3: Focus Health Summary (Bottom Left)
	tableW := quadWidth - 10
	if tableW < 20 {
		tableW = 20
	}
	renderTableRow := func(title, val string) string {
		padSize := tableW - len([]rune(title)) - len([]rune(val))
		if padSize < 1 {
			padSize = 1
		}
		return title + strings.Repeat(" ", padSize) + val
	}

	var q3Lines []string
	q3Lines = append(q3Lines, renderTableRow("Purity Ratio:", fmt.Sprintf("%.1f%%", stats.purityPct)))
	q3Lines = append(q3Lines, renderTableRow("Focus Logged:", fmt.Sprintf("%v", time.Duration(stats.totalFocusSecs)*time.Second)))
	q3Lines = append(q3Lines, renderTableRow("Interruption Count:", fmt.Sprintf("%d", stats.totalInterruptions)))
	q3Lines = append(q3Lines, renderTableRow("Tasks Cleared:", fmt.Sprintf("%d / %d", stats.completedCount, stats.totalCount)))
	q3Content := strings.Join(q3Lines, "\n\n")

	// Quadrant 4: 30-Day Contribution Grid (Bottom Right)
	var q4Blocks []string
	for i := 29; i >= 0; i-- {
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

	centeredBlocks := lipgloss.NewStyle().Width(quadWidth - 6).Align(lipgloss.Center).Render(blocksStr)
	centeredLegend := lipgloss.NewStyle().Width(quadWidth - 6).Align(lipgloss.Center).Render(legend)
	q4Content := centeredBlocks + "\n\n\n" + centeredLegend

	// Assemble Grid
	q1 := makeQuad("📅 DAILY FOCUS & ALLOCATION", q1Content)
	q2 := makeQuad("🏷️ TOP CATEGORY TAGS", q2Content)
	q3 := makeQuad("📈 FOCUS HEALTH METRICS", q3Content)
	q4 := makeQuad("🧱 30-DAY ACTIVATION TREND", q4Content)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, q1, q2)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, q3, q4)
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	var out strings.Builder
	out.WriteString(headerLine + "\n\n")
	out.WriteString(bannerContainer + "\n\n")
	out.WriteString(grid)

	return out.String()
}
