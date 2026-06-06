package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderZenMode() string {
	if m.ZenTimer == nil {
		return "No active focus session timer."
	}

	t := m.ZenTimer.Task
	sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]

	// 1. Ribbon Header
	// Active milestone name in standard title-case capitalization with bold text formatting.
	titleStr := fmt.Sprintf(" %s ", sentenceCase(t.Title))
	var topDivider string
	if len(titleStr) < m.Width-6 {
		topDivider = "───" + lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(titleStr) + strings.Repeat("─", m.Width-3-lipgloss.Width(titleStr))
	} else {
		topDivider = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(titleStr)
	}

	// Metadata group: Priority, Weight, Date
	priorityStr := lipgloss.NewStyle().Foreground(m.priorityColor(t.Priority)).Bold(true).Render(string(t.Priority))
	weightStr := fmt.Sprintf("%d SP", t.StoryPoints)
	dateStr := time.Now().Format("Monday, January 2")
	metaText := fmt.Sprintf("  Priority: %s  •  Weight: %s  •  Date: %s", priorityStr, weightStr, dateStr)
	bottomDivider := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", m.Width))

	// 2. Large Clock (5-row solid block characters clock)
	clockStr := RenderLargeTime(m.ZenTimer.TimeRemaining)
	clockBox := centerMultiLine(clockStr, m.Width)

	// 3. Padded Focus State Badge
	badgeText := fmt.Sprintf("Session %d / %d : %s",
		m.ZenTimer.CurrentSessionIdx+1,
		len(m.ZenTimer.Sessions),
		strings.ToUpper(string(sess.Type)),
	)
	styledBadgeText := lipgloss.NewStyle().
		Foreground(m.Theme.FocusPurple).
		Bold(true).
		Render(badgeText)
	badgeBoxRaw := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.Theme.Accent).
		Padding(0, 2).
		Render(styledBadgeText)
	badgeBox := centerMultiLine(badgeBoxRaw, m.Width)

	// 4. Progress bar and percentages
	pct := 1.0 - (m.ZenTimer.TimeRemaining.Seconds() / sess.Duration.Seconds())
	pctText := fmt.Sprintf(" %d%% Complete", int(pct*100))

	// Left margin: 4, Right margin: 4
	// Bracket prefix: "[ " (2 chars)
	// Bracket suffix: " ]" (2 chars)
	decorW := 12 + len(pctText)
	barW := m.Width - decorW
	if barW < 10 {
		barW = 10
	}
	progBar := RenderProgressBar(barW, pct)
	progressLine := fmt.Sprintf("    [ %s ]%s", progBar, pctText)

	// 5. Dual-column contextual data deck
	elapsed := m.ZenTimer.TotalDuration - m.ZenTimer.TimeRemaining
	if elapsed < 0 {
		elapsed = 0
	}
	elMin := int(elapsed.Minutes())
	elSec := int(elapsed.Seconds()) % 60
	elStr := fmt.Sprintf("⏱  %dm %02ds Elapsed", elMin, elSec)

	remMin := int(m.ZenTimer.TimeRemaining.Minutes())
	remSec := int(m.ZenTimer.TimeRemaining.Seconds()) % 60
	remStr := fmt.Sprintf("⏳ %dm %02ds Remaining", remMin, remSec)

	padCount := barW - lipgloss.Width(elStr) - lipgloss.Width(remStr)
	if padCount < 1 {
		padCount = 1
	}
	dataRow := fmt.Sprintf("      %s%s%s", elStr, strings.Repeat(" ", padCount), remStr)

	// 6. Three-Row Keybinding Guide Footer
	triggerStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)

	col1W := 44
	// Row 1
	r1Part1 := triggerStyle.Render("[ space ]") + descStyle.Render("  Pause / Resume Focus Session")
	r1Part2 := triggerStyle.Render("[  +  ]") + descStyle.Render("  Add 5 Minutes to Clock")
	r1Pad := col1W - lipgloss.Width(r1Part1)
	if r1Pad < 1 {
		r1Pad = 1
	}
	row1 := r1Part1 + strings.Repeat(" ", r1Pad) + r1Part2

	// Row 2
	r2Part1 := triggerStyle.Render("[   b   ]") + descStyle.Render("  Skip Active Interval Block")
	r2Part2 := triggerStyle.Render("[   r   ]") + descStyle.Render("  Restart Current Session")
	r2Pad := col1W - lipgloss.Width(r2Part1)
	if r2Pad < 1 {
		r2Pad = 1
	}
	row2 := r2Part1 + strings.Repeat(" ", r2Pad) + r2Part2

	// Row 3
	r3Part1 := triggerStyle.Render("[   q   ]") + descStyle.Render("  Stop / Abort Focus Session")
	r3Part2 := triggerStyle.Render("[ esc ]") + descStyle.Render("  Exit Focus Mode Workspace")
	r3Pad := col1W - lipgloss.Width(r3Part1)
	if r3Pad < 1 {
		r3Pad = 1
	}
	row3 := r3Part1 + strings.Repeat(" ", r3Pad) + r3Part2

	centeredRow1 := centerMultiLine(row1, m.Width)
	centeredRow2 := centerMultiLine(row2, m.Width)
	centeredRow3 := centerMultiLine(row3, m.Width)

	// 7. Dynamic Height Padding
	// Components height:
	// Ribbon: 3 rows (topDivider, metaText, bottomDivider)
	// Clock: 5 rows (using the 5-row solid font)
	// Session Badge Box: 3 rows (border top/bottom + text)
	// Progress Bar: 1 row
	// Data Deck: 1 row
	// Footer: 3 rows (space, b, q)
	// Fixed spacing: 2 blank rows below clock block to badge
	// Total component height = 3 + 5 + 3 + 1 + 1 + 3 = 16 rows.
	// Plus 2 fixed spacing rows = 18 rows.
	remHeight := m.Height - 18
	if remHeight < 0 {
		remHeight = 0
	}
	pad1 := remHeight / 3
	pad3 := remHeight / 3
	pad4 := remHeight - pad1 - pad3

	// Pad with safety checks
	if pad1 < 1 {
		pad1 = 1
	}
	if pad3 < 1 {
		pad3 = 1
	}
	if pad4 < 1 {
		pad4 = 1
	}

	var sb []string
	// Ribbon Header
	sb = append(sb, topDivider)
	sb = append(sb, metaText)
	sb = append(sb, bottomDivider)

	// Spacer 1
	for i := 0; i < pad1; i++ {
		sb = append(sb, "")
	}

	// Large Clock
	sb = append(sb, clockBox)

	// Fixed Spacer 2: exactly 2 blank rows
	sb = append(sb, "", "")

	// Session Badge
	sb = append(sb, badgeBox)

	// Spacer 3
	for i := 0; i < pad3; i++ {
		sb = append(sb, "")
	}

	// Progress Track
	sb = append(sb, progressLine)
	sb = append(sb, dataRow)

	// Spacer 4
	for i := 0; i < pad4; i++ {
		sb = append(sb, "")
	}

	// Footer
	sb = append(sb, centeredRow1)
	sb = append(sb, centeredRow2)
	sb = append(sb, centeredRow3)

	return strings.Join(sb, "\n")
}

func centerMultiLine(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			pad := (width - w) / 2
			lines[i] = strings.Repeat(" ", pad) + line
		}
	}
	return strings.Join(lines, "\n")
}
