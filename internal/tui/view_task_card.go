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
	priorityBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render("▲ " + pName)
	if hasCollision {
		priorityBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true).Render("⚠️ " + pName)
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
	mutedStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
	spStrStyled := mutedStyle.Render(fmt.Sprintf("%d SP", task.StoryPoints))
	timeStrStyled := mutedStyle.Render(timeStr)
	bulletStyled := mutedStyle.Render("  •  ")

	metaStr := fmt.Sprintf("%s%s%s%s%s", priorityBadge, bulletStyled, spStrStyled, bulletStyled, timeStrStyled)
	if lipgloss.Width(metaStr) > contentW-1 {
		metaStr = fmt.Sprintf("%s%s%s", priorityBadge, bulletStyled, timeStrStyled)
	}
	if lipgloss.Width(metaStr) > contentW-1 {
		metaStr = timeStrStyled
	}
	if lipgloss.Width(metaStr) > contentW-1 {
		metaStr = priorityBadge
	}
	if lipgloss.Width(metaStr) > contentW-1 {
		metaStr = sliceAnsi(metaStr, 0, contentW-1)
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
	metaLine := metaStr

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


// sentenceCase capitalises only the first letter of a string.
func sentenceCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[:1]) + s[1:]
}
