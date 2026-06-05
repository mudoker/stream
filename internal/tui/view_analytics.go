package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAnalyticsView() string {
	today := time.Now()
	stats := m.calculateAnalyticsStats()

	workspaceWidth := m.Layout.WorkspaceW - 4

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", workspaceWidth))

	// ── Page Header ──────────────────────────────────────────────
	header := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("▲ Analytics")
	subhead := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(today.Format("January 2006"))

	// ── KPI Cards Row ────────────────────────────────────────────
	kpiRow := m.renderAnalyticsKPICards(workspaceWidth, stats)

	leftColW := 36
	rightColW := workspaceWidth - leftColW - 6
	if rightColW < 30 {
		rightColW = 30
	}
	leftCol := m.renderAnalyticsLeftCol(stats, leftColW, today)
	rightCol := m.renderAnalyticsRightCol(stats, rightColW, workspaceWidth, today)

	columns := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftColW).Render(leftCol),
		"    ",
		lipgloss.NewStyle().Width(rightColW).Render(rightCol),
	)

	var out strings.Builder
	out.WriteString(header + "  " + subhead + "\n")
	out.WriteString(sep + "\n\n")
	out.WriteString(kpiRow + "\n\n")
	out.WriteString(sep + "\n\n")
	out.WriteString(columns)

	return out.String()
}

// renderAnalyticsKPICards renders the top KPI metric strip.
func (m Model) renderAnalyticsKPICards(w int, stats AnalyticsStats) string {
	type kpiEntry struct {
		label  string
		value  string
		sub    string
		color  lipgloss.Color
	}
	kpis := []kpiEntry{
		{
			label: "STREAK",
			value: fmt.Sprintf("%d days", stats.streak),
			sub:   "consecutive",
			color: m.Theme.P1Color,
		},
		{
			label: "SESSIONS",
			value: fmt.Sprintf("%d blocks", stats.effectiveSessions),
			sub:   "pomodoros",
			color: m.Theme.Accent,
		},
		{
			label: "FOCUS TIME",
			value: fmt.Sprintf("%.1f hrs", stats.totalHrs),
			sub:   "work + personal",
			color: m.Theme.SuccessColor,
		},
		{
			label: "PURITY",
			value: fmt.Sprintf("%.0f%%", stats.purityPct),
			sub:   "no interruptions",
			color: m.Theme.FocusPurple,
		},
		{
			label: "COMPLETION",
			value: fmt.Sprintf("%.0f%%", stats.rate),
			sub:   fmt.Sprintf("%d/%d tasks", stats.completedCount, stats.totalCount),
			color: m.Theme.SuccessColor,
		},
	}

	cardW := (w - 4) / len(kpis)
	if cardW < 14 {
		cardW = 14
	}

	var cards []string
	for _, k := range kpis {
		lbl := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render(k.label)
		val := lipgloss.NewStyle().Foreground(k.color).Bold(true).Render(k.value)
		sub := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(k.sub)
		card := lipgloss.NewStyle().
			Background(m.Theme.PanelBg).
			Padding(0, 2).
			Width(cardW - 1).
			Render(lbl + "\n" + val + "\n" + sub)
		cards = append(cards, card)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
}

// renderAnalyticsLeftCol renders daily timeline + focus health.
func (m Model) renderAnalyticsLeftCol(stats AnalyticsStats, w int, today time.Time) string {
	mutedB := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)

	// Daily productivity timeline (last 7 days)
	var lines []string
	lines = append(lines, mutedB.Render("DAILY FOCUS  (7 DAYS)"))
	lines = append(lines, "")

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

	barMaxW := w - 18
	if barMaxW < 6 {
		barMaxW = 6
	}
	for i := 0; i < 7; i++ {
		day := daysList[i]
		daySecs := daySecsList[i]
		dayHrs := float64(daySecs) / 3600.0
		pct := float64(daySecs) / float64(maxDaySecs)
		barLen := int(math.Round(pct * float64(barMaxW)))
		barStr := strings.Repeat("█", barLen)
		if barStr == "" && dayHrs > 0 {
			barStr = "▏"
		}
		var barColor lipgloss.Color
		switch {
		case dayHrs == 0:
			barColor = m.Theme.Muted
		case dayHrs <= 2.0:
			barColor = m.Theme.P2Color
		case dayHrs <= 5.0:
			barColor = m.Theme.Accent
		default:
			barColor = m.Theme.SuccessColor
		}
		isToday := day.Format("2006-01-02") == today.Format("2006-01-02")
		dayLabel := day.Format("Mon _2")
		if isToday {
			dayLabel = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(dayLabel)
		} else {
			dayLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(dayLabel)
		}
		colored := lipgloss.NewStyle().Foreground(barColor).Render(barStr)
		hrsStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("%.1fh", dayHrs))
		lines = append(lines, fmt.Sprintf("  %s │ %s %s", dayLabel, colored, hrsStr))
	}

	lines = append(lines, "")
	lines = append(lines, mutedB.Render("FOCUS HEALTH"))
	lines = append(lines, "")

	statsData := []struct {
		label string
		value string
	}{
		{"Purity", fmt.Sprintf("%.1f%%", stats.purityPct)},
		{"Focus Logged", fmt.Sprintf("%v", time.Duration(stats.totalFocusSecs)*time.Second)},
		{"Interruptions", fmt.Sprintf("%d", stats.totalInterruptions)},
		{"Tasks done", fmt.Sprintf("%d / %d", stats.completedCount, stats.totalCount)},
	}
	for _, s := range statsData {
		lbl := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("  %-16s", s.label))
		val := lipgloss.NewStyle().Foreground(m.Theme.Fg).Render(s.value)
		lines = append(lines, lbl+val)
	}

	return strings.Join(lines, "\n")
}

// renderAnalyticsRightCol renders time ratios, top tags, and 30-day trend.
func (m Model) renderAnalyticsRightCol(stats AnalyticsStats, w int, workW int, today time.Time) string {
	mutedB := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)
	var lines []string

	// Time allocation ratio
	lines = append(lines, mutedB.Render("TIME ALLOCATION"))
	lines = append(lines, "")

	ratioBarW := w - 2
	if ratioBarW > 28 {
		ratioBarW = 28
	}
	var ratioBar string
	if stats.totalHrs == 0 {
		ratioBar = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("·", ratioBarW))
	} else {
		wLen := int(math.Round(stats.workPct * float64(ratioBarW)))
		pLen := ratioBarW - wLen
		ratioBar = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(strings.Repeat("█", wLen)) +
			lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(strings.Repeat("█", pLen))
	}

	lines = append(lines, "  "+ratioBar)
	lines = append(lines, fmt.Sprintf("  %s Work  %.0f%% (%.1fh)",
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("█"),
		stats.workPct*100, stats.workHrs))
	lines = append(lines, fmt.Sprintf("  %s Personal  %.0f%% (%.1fh)",
		lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("█"),
		stats.personalPct*100, stats.personalHrs))

	lines = append(lines, "")
	lines = append(lines, mutedB.Render("TOP TAGS"))
	lines = append(lines, "")

	if len(stats.tags) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No tagged sessions."))
	} else {
		maxSecs := stats.tags[0].Secs
		if maxSecs == 0 {
			maxSecs = 1
		}
		for idx, tv := range stats.tags {
			if idx >= 5 {
				break
			}
			hrs := float64(tv.Secs) / 3600.0
			pct := float64(tv.Secs) / float64(maxSecs)
			bLen := int(math.Round(pct * 12))
			bar := strings.Repeat("█", bLen)
			if bar == "" && tv.Secs > 0 {
				bar = "▏"
			}
			colored := lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Render(bar)
			tagName := tv.Tag
			if len(tagName) > 10 {
				tagName = tagName[:9] + "…"
			}
			lines = append(lines, fmt.Sprintf("  %-10s %s %.1fh", tagName, colored, hrs))
		}
	}

	lines = append(lines, "")
	lines = append(lines, mutedB.Render("30-DAY TREND"))
	lines = append(lines, "")

	var trendSB strings.Builder
	trendSB.WriteString("  ")
	for i := 29; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		secs := stats.dailyFocusSecs[date.Format("2006-01-02")]
		hrs := float64(secs) / 3600.0
		var cellColor lipgloss.Color
		char := "■"
		switch {
		case hrs == 0:
			cellColor = m.Theme.Muted
			char = "·"
		case hrs <= 1.5:
			cellColor = m.Theme.P2Color
		case hrs <= 4.0:
			cellColor = m.Theme.Accent
		default:
			cellColor = m.Theme.SuccessColor
		}
		trendSB.WriteString(lipgloss.NewStyle().Foreground(cellColor).Render(char) + " ")
	}
	lines = append(lines, trendSB.String())

	var legend strings.Builder
	legend.WriteString("  less ")
	legend.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("·") + " ")
	legend.WriteString(lipgloss.NewStyle().Foreground(m.Theme.P2Color).Render("■") + " ")
	legend.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("■") + " ")
	legend.WriteString(lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render("■") + " ")
	legend.WriteString("more")
	lines = append(lines, legend.String())

	return strings.Join(lines, "\n")
}
