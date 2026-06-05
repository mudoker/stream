package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func sliceAnsi(s string, start, end int) string {
	var sb strings.Builder
	var runes = []rune(s)
	var inEscape = false
	var visualCount = 0

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			sb.WriteRune(r)
			continue
		}
		if inEscape {
			sb.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		if visualCount >= start && visualCount < end {
			sb.WriteRune(r)
		}
		visualCount++
	}
	return sb.String()
}

func overlayString(base string, overlay string, x int, y int, baseWidth int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := 0
	for _, l := range overlayLines {
		w := lipgloss.Width(l)
		if w > overlayWidth {
			overlayWidth = w
		}
	}

	for i, oLine := range overlayLines {
		targetY := y + i
		if targetY >= len(baseLines) {
			break
		}
		bLine := baseLines[targetY]

		leftPart := sliceAnsi(bLine, 0, x)
		rightPart := sliceAnsi(bLine, x+overlayWidth, baseWidth)

		leftVisualLen := lipgloss.Width(leftPart)
		if leftVisualLen < x {
			leftPart += strings.Repeat(" ", x-leftVisualLen)
		}

		baseLines[targetY] = leftPart + oLine + rightPart
	}
	return strings.Join(baseLines, "\n")
}

func wrapText(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	var words = strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var res []string
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > limit {
			res = append(res, currentLine)
			currentLine = word
		} else {
			if len(currentLine) > 0 {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if len(currentLine) > 0 {
		res = append(res, currentLine)
	}
	return strings.Join(res, "\n")
}

func indentText(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = indent + l
	}
	return strings.Join(lines, "\n")
}

func getNowBadge(width int, now time.Time) string {
	full := fmt.Sprintf("── NOW • %02d:%02d ──", now.Hour(), now.Minute())
	if len(full) <= width {
		return full
	}
	mid := fmt.Sprintf("NOW • %02d:%02d", now.Hour(), now.Minute())
	if len(mid) <= width {
		return mid
	}
	short := fmt.Sprintf("● %02d:%02d", now.Hour(), now.Minute())
	if len(short) <= width {
		return short
	}
	return "●"
}

func (m Model) overlayMiniZen(content string, workspaceWidth int) string {
	if m.ZenTimer == nil || !m.ZenTimer.Running {
		return content
	}

	zt := m.ZenTimer
	sess := zt.Sessions[zt.CurrentSessionIdx]
	hVal := int(zt.TimeRemaining.Hours())
	mVal := int(zt.TimeRemaining.Minutes()) % 60
	sVal := int(zt.TimeRemaining.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d Remaining", hVal, mVal, sVal)

	pct := 1.0 - (zt.TimeRemaining.Seconds() / sess.Duration.Seconds())
	bar := RenderProgressBar(18, pct)

	title := zt.Task.Title
	if len(title) > 20 {
		title = title[:17] + "..."
	}

	widgetWidth := 26
	widgetBg := m.Theme.SelectedBg
	if zt.IsPaused {
		widgetBg = m.Theme.PanelBg
	}

	sessionHeader := "● FOCUS RUNNING"
	if zt.IsPaused {
		sessionHeader = "● FOCUS PAUSED"
	}

	headerStyle := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true)
	if zt.IsPaused {
		headerStyle = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)
	}

	widgetStr := lipgloss.NewStyle().
		Background(widgetBg).
		Foreground(m.Theme.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Padding(0, 1).
		Width(widgetWidth).
		Render(fmt.Sprintf(
			"%s\n%s\n%s\n%s",
			headerStyle.Render(sessionHeader),
			sentenceCase(title),
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(timeStr),
			bar,
		))

	x := workspaceWidth - widgetWidth - 2
	if x < 0 {
		x = 0
	}
	return overlayString(content, widgetStr, x, 1, workspaceWidth)
}
