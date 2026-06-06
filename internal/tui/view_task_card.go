package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) hasPriorityOverlapCollision(t model.Task) bool {
	if t.SchedulingType != model.Anchored {
		return false
	}
	for _, t2 := range m.Tasks {
		if t2.UUID == t.UUID || t2.SchedulingType != model.Anchored {
			continue
		}
		if !sameDay(t.TimeWindow.Start, t2.TimeWindow.Start) {
			continue
		}
		overlap := t.TimeWindow.Start.Before(t2.TimeWindow.End) && t2.TimeWindow.Start.Before(t.TimeWindow.End)
		if overlap {
			if t.Priority == model.P0 || t.Priority == model.P1 || t2.Priority == model.P0 || t2.Priority == model.P1 {
				return true
			}
		}
	}
	return false
}

// renderTaskCard renders a complete multi-line Lipgloss task card block.
// w and h are the card's outer dimensions (including borders).
func (m Model) renderTaskCard(task model.Task, w, h int, isActive bool, isSelected bool) string {
	pColor := m.priorityColor(task.Priority)
	now := time.Now()

	hasCollision := m.hasPriorityOverlapCollision(task)
	isZenFocus := m.ZenTimer != nil && m.ZenTimer.Running && m.ZenTimer.Task.UUID == task.UUID

	// Card border
	borderColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if isCompleted {
		borderColor = lipgloss.Color("#45475a") // Dim border for completed
	} else if isZenFocus {
		borderColor = m.Theme.SuccessColor
	} else if isSelected {
		borderColor = lipgloss.Color("#ff8700")
	} else if hasCollision {
		borderColor = lipgloss.Color("#ff0000")
	} else if isActive {
		borderColor = m.Theme.Accent
	}

	// Priority badge
	pName := string(task.Priority)
	priorityBadge := fmt.Sprintf("▲ %s", pName)
	if hasCollision {
		priorityBadge = fmt.Sprintf("⚠️ %s", pName)
	}

	// Time range string
	timeStr := fmt.Sprintf("⏱ %s → %s",
		task.TimeWindow.Start.Format("15:04"),
		task.TimeWindow.End.Format("15:04"),
	)
	if isActive {
		remaining := task.TimeWindow.End.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		h2 := int(remaining.Hours())
		mi := int(remaining.Minutes()) % 60
		s := int(remaining.Seconds()) % 60
		timeStr = fmt.Sprintf("⏱ %02d:%02d:%02d remaining", h2, mi, s)
	}

	// If card is very short (h < 3), use the left strip card
	if h < 3 {
		return m.renderShortCard(task, w, h, pColor, isActive, isSelected, hasCollision, isZenFocus, timeStr)
	}

	if w < 3 {
		w = 3
	}
	if h < 3 {
		h = 3
	}

	// Determine padding based on height
	paddingTopBottom := 0
	if h >= 7 {
		paddingTopBottom = 1
	}

	paddingLeftRight := 2
	if w < 10 {
		paddingLeftRight = 0
	} else if w < 14 {
		paddingLeftRight = 1
	}

	// Content width inside padding (2 border left/right, and 2 * paddingLeftRight)
	contentW := w - 2 - (2 * paddingLeftRight)
	if contentW < 1 {
		contentW = 1
	}

	contentH := h - 2 - (2 * paddingTopBottom)
	if contentH < 1 {
		contentH = 1
	}

	// Truncate title using safe rune slicing
	titleStr := sentenceCase(task.Title)
	titleRunes := []rune(titleStr)
	if len(titleRunes) > contentW-1 {
		if contentW > 2 {
			titleStr = string(titleRunes[:contentW-2]) + "…"
		} else {
			titleStr = string(titleRunes[:contentW-1])
		}
	}

	// Construct and scale metadata row to fit contentW-1
	spStr := fmt.Sprintf("%d SP", task.StoryPoints)
	metaStr := fmt.Sprintf("%s  •  %s  •  %s", priorityBadge, spStr, timeStr)
	if len([]rune(metaStr)) > contentW-1 {
		metaStr = fmt.Sprintf("%s  •  %s", priorityBadge, timeStr)
	}
	if len([]rune(metaStr)) > contentW-1 {
		metaStr = timeStr
	}
	if len([]rune(metaStr)) > contentW-1 {
		metaStr = priorityBadge
	}
	metaRunes := []rune(metaStr)
	if len(metaRunes) > contentW-1 {
		metaStr = string(metaRunes[:contentW-1])
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true)
	if isCompleted {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#a6e3a1")).Faint(true)
	} else if isZenFocus {
		titleStyle = titleStyle.Foreground(m.Theme.SuccessColor)
	} else if isSelected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#ff8700"))
	} else if hasCollision {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#ff0000"))
	}
	titleLine := titleStyle.Render(titleStr)
	metaLine := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(metaStr)

	innerWidth := contentW + (2 * paddingLeftRight)
	if innerWidth < 1 {
		innerWidth = 1
	}

	contentAreaW := contentW
	if contentAreaW < 1 {
		contentAreaW = 1
	}

	var contentLines []string
	if h == 3 {
		contentLines = []string{titleLine}
	} else if h == 4 {
		contentLines = []string{titleLine, metaLine}
	} else {
		sepLine := strings.Repeat("─", contentAreaW)
		contentLines = []string{titleLine, sepLine, metaLine}
	}

	heightContent := h - 2
	if heightContent < 1 {
		heightContent = 1
	}

	var bodyLines []string
	for i := 0; i < paddingTopBottom; i++ {
		bodyLines = append(bodyLines, strings.Repeat(" ", innerWidth))
	}

	for i, line := range contentLines {
		if len(bodyLines) >= heightContent-paddingTopBottom {
			break
		}
		visual := lipgloss.Width(line)
		if visual > contentAreaW {
			line = sliceAnsi(line, 0, contentAreaW)
		} else if visual < contentAreaW {
			line += strings.Repeat(" ", contentAreaW-visual)
		}
		if paddingLeftRight > 0 {
			line = strings.Repeat(" ", paddingLeftRight) + line + strings.Repeat(" ", paddingLeftRight)
		}
		bodyLines = append(bodyLines, line)
		if i == len(contentLines)-1 {
			break
		}
	}

	for len(bodyLines) < heightContent {
		bodyLines = append(bodyLines, strings.Repeat(" ", innerWidth))
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	topLine := borderStyle.Render("╭") + borderStyle.Render(strings.Repeat("─", innerWidth)) + borderStyle.Render("╮")
	bottomLine := borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", innerWidth)) + borderStyle.Render("╯")

	var cardLines []string
	cardLines = append(cardLines, topLine)
	for _, body := range bodyLines {
		cardLines = append(cardLines, borderStyle.Render("│")+body+borderStyle.Render("│"))
	}
	cardLines = append(cardLines, bottomLine)

	return strings.Join(cardLines, "\n")
}

// renderShortCard renders a compact card for h < 4 rows using a left strip bar.
func (m Model) renderShortCard(task model.Task, w, h int, pColor lipgloss.Color, isActive bool, isSelected bool, hasCollision bool, isZenFocus bool, timeStr string) string {
	stripColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if isCompleted {
		stripColor = lipgloss.Color("#45475a") // dim strip for completed
	} else if isZenFocus {
		stripColor = m.Theme.SuccessColor
	} else if isSelected {
		stripColor = lipgloss.Color("#ff8700")
	} else if hasCollision {
		stripColor = lipgloss.Color("#ff0000")
	} else if isActive {
		stripColor = m.Theme.Accent
	}

	strip := lipgloss.NewStyle().
		Foreground(stripColor).
		Render("▎")

	contentW := w - 1
	if contentW < 1 {
		contentW = 1
	}

	titleStr := sentenceCase(task.Title)
	if hasCollision {
		titleStr = "⚠️ " + titleStr
	}
	titleRunes := []rune(titleStr)
	if len(titleRunes) > contentW-1 {
		if contentW > 2 {
			titleStr = string(titleRunes[:contentW-2]) + "…"
		} else {
			titleStr = string(titleRunes[:contentW-1])
		}
	}

	var text string
	switch h {
	case 1:
		text = fmt.Sprintf(" %s", titleStr)
	case 2:
		text = fmt.Sprintf(" %s", titleStr)
	case 3:
		text = fmt.Sprintf(" %s", titleStr)
	}

	var textStyle lipgloss.Style
	if isCompleted {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Faint(true)
	} else if isZenFocus {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true)
	} else if isSelected {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8700")).Bold(true)
	} else if hasCollision {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
	} else if isActive {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
	} else {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.Fg)
	}

	row := strip + textStyle.
		Width(contentW).
		Render(text)
	rowW := lipgloss.Width(row)
	if rowW < w {
		row += strings.Repeat(" ", w-rowW)
	}

	if h <= 1 {
		return row
	}

	var rows []string
	rows = append(rows, row)
	if h >= 2 {
		pName := string(task.Priority)
		if hasCollision {
			pName = "⚠️ " + pName
		}
		meta := fmt.Sprintf(" %s  %s", pName, timeStr)
		metaRunes := []rune(meta)
		if len(metaRunes) > contentW {
			meta = string(metaRunes[:contentW])
		}
		metaLine := strip + lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Width(contentW).
			Render(meta)
		metaW := lipgloss.Width(metaLine)
		if metaW < w {
			metaLine += strings.Repeat(" ", w-metaW)
		}
		rows = append(rows, metaLine)
	}
	if h >= 3 {
		meta2 := fmt.Sprintf(" %d SP", task.StoryPoints)
		meta2Runes := []rune(meta2)
		if len(meta2Runes) > contentW {
			meta2 = string(meta2Runes[:contentW])
		}
		meta2Line := strip + lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Width(contentW).
			Render(meta2)
		meta2W := lipgloss.Width(meta2Line)
		if meta2W < w {
			meta2Line += strings.Repeat(" ", w-meta2W)
		}
		rows = append(rows, meta2Line)
	}
	return strings.Join(rows, "\n")
}

// sentenceCase capitalises only the first letter of a string.
func sentenceCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[:1]) + s[1:]
}
