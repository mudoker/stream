package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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

type Cell struct {
	Text           string
	Style          string
	IsContinuation bool
}

func parseLineToCells(s string) []Cell {
	var cells []Cell
	var currentStyle strings.Builder
	var runes = []rune(s)
	var inEscape = false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			currentStyle.WriteRune(r)
			continue
		}
		if inEscape {
			currentStyle.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		w := runewidth.RuneWidth(r)
		if w <= 0 {
			if len(cells) > 0 {
				idx := len(cells) - 1
				for idx > 0 && cells[idx].IsContinuation {
					idx--
				}
				cells[idx].Text += string(r)
			}
			continue
		}

		styleStr := currentStyle.String()
		cells = append(cells, Cell{
			Text:           string(r),
			Style:          styleStr,
			IsContinuation: false,
		})
		for k := 1; k < w; k++ {
			cells = append(cells, Cell{
				Text:           "",
				Style:          styleStr,
				IsContinuation: true,
			})
		}
	}
	return cells
}

func cellsToLine(cells []Cell) string {
	var sb strings.Builder
	var lastStyle string

	for _, cell := range cells {
		if cell.IsContinuation {
			continue
		}
		if cell.Style != lastStyle {
			if cell.Style == "" {
				sb.WriteString("\x1b[0m")
			} else {
				sb.WriteString(cell.Style)
			}
			lastStyle = cell.Style
		}
		if cell.Text != "" {
			sb.WriteString(cell.Text)
		} else {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\x1b[0m")
	return sb.String()
}

func overlayString(base string, overlay string, x int, y int, baseWidth int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := 0
	for _, l := range overlayLines {
		cells := parseLineToCells(l)
		if len(cells) > overlayWidth {
			overlayWidth = len(cells)
		}
	}

	for i, oLine := range overlayLines {
		targetY := y + i
		if targetY >= len(baseLines) {
			break
		}
		bLine := baseLines[targetY]
		baseCells := parseLineToCells(bLine)
		overlayCells := parseLineToCells(oLine)

		// Pad base line to baseWidth
		for len(baseCells) < baseWidth {
			baseCells = append(baseCells, Cell{Text: " "})
		}
		if len(baseCells) > baseWidth {
			baseCells = baseCells[:baseWidth]
		}

		// Pad overlay line to overlayWidth
		for len(overlayCells) < overlayWidth {
			overlayCells = append(overlayCells, Cell{Text: " "})
		}
		if len(overlayCells) > overlayWidth {
			overlayCells = overlayCells[:overlayWidth]
		}

		// Overwrite base line cells starting at x
		for col := 0; col < overlayWidth; col++ {
			targetX := x + col
			if targetX >= 0 && targetX < baseWidth {
				baseCells[targetX] = overlayCells[col]
			}
		}

		baseLines[targetY] = cellsToLine(baseCells)
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
	titleRunes := []rune(title)
	if len(titleRunes) > 20 {
		title = string(titleRunes[:17]) + "..."
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
