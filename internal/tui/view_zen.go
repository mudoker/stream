package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderZenMode() string {
	if m.ZenTimer == nil {
		return "No focus session timer."
	}

	t := m.ZenTimer.Task
	sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]

	var sb []string
	sb = append(sb, "")
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(strings.ToUpper(t.Title)))

	pBadge := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("▲ " + string(t.Priority))
	spBadge := fmt.Sprintf("• %d SP", t.StoryPoints)
	sb = append(sb, lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(fmt.Sprintf("%s %s", pBadge, spBadge)))

	sb = append(sb, "")

	clockStr := RenderLargeTime(m.ZenTimer.TimeRemaining)
	clockBox := lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(clockStr)
	sb = append(sb, clockBox)

	sb = append(sb, "")
	sessInfo := fmt.Sprintf("[ Session %d / %d: %s ]", m.ZenTimer.CurrentSessionIdx+1, len(m.ZenTimer.Sessions), strings.ToUpper(string(sess.Type)))
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.SuccessColor).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(sessInfo))

	sb = append(sb, "")

	pct := 1.0 - (m.ZenTimer.TimeRemaining.Seconds() / sess.Duration.Seconds())
	barWidth := int(float64(m.Width) * 0.70)
	if barWidth < 20 {
		barWidth = 20
	}
	progBar := RenderProgressBar(barWidth, pct)
	sb = append(sb, lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.Width).
		Render(fmt.Sprintf("%s %d%%", progBar, int(pct*100))))

	sb = append(sb, "\n")
	instructions := "space pause/resume   + add 5m   b skip block   esc exit focus"
	sb = append(sb, lipgloss.NewStyle().
		Foreground(m.Theme.Muted).
		Align(lipgloss.Center).
		Width(m.Width).
		Render(instructions))

	return strings.Join(sb, "\n")
}
