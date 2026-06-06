package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSidebarFooter(innerW int, appContentHeight int, occupied int) []string {
	var rows []string
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))
	muted := lipgloss.NewStyle().Foreground(m.Theme.Muted)

	footOccupied := 5
	if m.ZenTimer != nil && m.ZenTimer.Running {
		footOccupied++
	}
	remaining := appContentHeight - (occupied + footOccupied + 9)
	if remaining > 0 {
		rows = append(rows, strings.Repeat("\n", remaining))
	}

	memPct := 45 + int(time.Now().Unix()%20)
	barW := innerW - 15
	if barW < 4 {
		barW = 4
	}
	solidCount := memPct * barW / 100
	if solidCount < 0 {
		solidCount = 0
	}
	if solidCount > barW {
		solidCount = barW
	}
	emptyCount := barW - solidCount
	barStr := strings.Repeat("█", solidCount) + strings.Repeat("░", emptyCount)

	leftText := fmt.Sprintf("  RAM  [%s]", barStr)
	rightText := fmt.Sprintf("%d%%", memPct)
	leftW := lipgloss.Width(leftText)
	rightW := lipgloss.Width(rightText)
	spaceCount := innerW - leftW - rightW - 1
	if spaceCount < 0 {
		spaceCount = 0
	}
	memRow := leftText + strings.Repeat(" ", spaceCount) + rightText + " "
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(memRow))

	rows = append(rows, sep)

	syncColor := m.Theme.Muted
	if m.Sync.IsOnline() {
		syncColor = m.Theme.SuccessColor
	}
	gcal := lipgloss.NewStyle().Foreground(syncColor).Render("● gcal")

	modeColor := m.Theme.Muted
	switch m.CurrentMode {
	case ModeZen:
		modeColor = m.Theme.FocusPurple
	case ModeCommand:
		modeColor = m.Theme.P1Color
	case ModeForm:
		modeColor = m.Theme.Accent
	case ModeWorkspaceForm:
		modeColor = m.Theme.Accent
	case ModeWorkspacePicker:
		modeColor = m.Theme.Accent
	}
	modeBadge := lipgloss.NewStyle().Foreground(modeColor).Bold(true).
		Render(strings.ToLower(string(m.CurrentMode)))
	clock := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).
		Render(time.Now().Format("15:04"))

	if m.ZenTimer != nil && m.ZenTimer.Running {
		zt := m.ZenTimer
		hVal := int(zt.TimeRemaining.Hours())
		mVal := int(zt.TimeRemaining.Minutes()) % 60
		sVal := int(zt.TimeRemaining.Seconds()) % 60
		focusText := fmt.Sprintf("󱎫 FOCUS (%02d:%02d:%02d)", hVal, mVal, sVal)
		if zt.IsPaused {
			focusText = fmt.Sprintf("󱎫 PAUSED (%02d:%02d:%02d)", hVal, mVal, sVal)
		}
		focusStyle := lipgloss.NewStyle().
			Foreground(m.Theme.FocusPurple).
			Bold(true)
		rows = append(rows, "  "+focusStyle.Render(focusText))
	}

	rows = append(rows, modeBadge+"  "+gcal)
	rows = append(rows, clock)
	rows = append(rows, muted.Render("? help"))
	return rows
}
