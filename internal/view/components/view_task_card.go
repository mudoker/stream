package components

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderTaskCard renders a complete multi-line Lipgloss task card block.
// w and h are the card's outer dimensions (including borders).
func RenderTaskCard(m *viewmodel.Model, t theme.Theme, task model.Task, w, h int, isActive bool, isSelected bool) string {
	pColor := t.PriorityColor(task.Priority)
	if strings.HasSuffix(task.UUID, "_moving") {
		pColor = t.Muted
	}
	now := time.Now()

	hasCollision := m.HasPriorityOverlapCollision(task)
	isZenFocus := m.ZenTimer != nil && m.ZenTimer.Running && m.ZenTimer.Task.UUID == task.UUID

	// Card border
	borderColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if strings.HasSuffix(task.UUID, "_moving") {
		borderColor = t.Muted
	} else if isZenFocus {
		borderColor = t.SuccessColor
	} else if isSelected {
		borderColor = t.FocusPurple
	} else if hasCollision {
		borderColor = lipgloss.Color("#ff0000")
	} else if isActive {
		borderColor = t.Accent
	} else if isCompleted {
		borderColor = lipgloss.Color("#4c644f") // Soft dark forest green border for completed
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
		return RenderShortCard(m, t, task, w, h, pColor, isActive, isSelected, hasCollision, isZenFocus, timeStr)
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
	titleStr := theme.SentenceCase(task.Title)
	if isCompleted {
		titleStr = "✔ " + titleStr
	}
	if strings.HasSuffix(task.UUID, "_moving") {
		titleStr = "[Moving] " + titleStr
	}
	titleRunes := []rune(titleStr)
	if len(titleRunes) > contentW-1 {
		if contentW > 2 {
			titleStr = string(titleRunes[:contentW-2]) + "…"
		} else {
			titleStr = string(titleRunes[:contentW-1])
		}
	}

	// Construct and scale metadata row to fit contentW-1
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	if strings.HasSuffix(task.UUID, "_moving") {
		mutedStyle = mutedStyle.Faint(true)
	}
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
		metaStr = theme.SliceAnsi(metaStr, 0, contentW-1)
	}

	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	if strings.HasSuffix(task.UUID, "_moving") {
		titleStyle = lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Faint(true)
	} else if isZenFocus {
		titleStyle = titleStyle.Foreground(t.SuccessColor)
	} else if isSelected {
		titleStyle = titleStyle.Foreground(t.FocusPurple)
	} else if hasCollision {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#ff0000"))
	} else if isCompleted {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#88b08b")).Bold(true) // Softer sage green for completed title
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
			line = theme.SliceAnsi(line, 0, contentAreaW)
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

	restDur := viewmodel.CalculateTaskRestTime(task)
	hasRest := restDur > 0 && task.SchedulingType == model.Anchored

	var topLeftChar, topRightChar, bottomLeftChar, bottomRightChar, horizChar, vertChar string
	if strings.HasSuffix(task.UUID, "_moving") {
		topLeftChar = "┌"
		topRightChar = "┐"
		bottomLeftChar = "└"
		bottomRightChar = "┘"
		horizChar = "╌"
		vertChar = "┊"
		if hasRest {
			bottomLeftChar = "├"
			bottomRightChar = "┤"
		}
	} else {
		topLeftChar = "╭"
		topRightChar = "╮"
		bottomLeftChar = "╰"
		bottomRightChar = "╯"
		horizChar = "─"
		vertChar = "│"
		if hasRest {
			bottomLeftChar = "├"
			bottomRightChar = "┤"
		}
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	topLine := borderStyle.Render(topLeftChar) + borderStyle.Render(strings.Repeat(horizChar, innerWidth)) + borderStyle.Render(topRightChar)
	
	bottomHorizChar := horizChar
	if hasRest {
		bottomHorizChar = "╌"
	}
	bottomLine := borderStyle.Render(bottomLeftChar) + borderStyle.Render(strings.Repeat(bottomHorizChar, innerWidth)) + borderStyle.Render(bottomRightChar)

	var cardLines []string
	cardLines = append(cardLines, topLine)
	for _, body := range bodyLines {
		cardLines = append(cardLines, borderStyle.Render(vertChar)+body+borderStyle.Render(vertChar))
	}
	cardLines = append(cardLines, bottomLine)

	return strings.Join(cardLines, "\n")
}

// RenderShortCard renders a compact card for h < 3 rows using left and right borders.
func RenderShortCard(m *viewmodel.Model, t theme.Theme, task model.Task, w, h int, pColor lipgloss.Color, isActive bool, isSelected bool, hasCollision bool, isZenFocus bool, timeStr string) string {
	stripColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if strings.HasSuffix(task.UUID, "_moving") {
		stripColor = t.Muted
	} else if isZenFocus {
		stripColor = t.SuccessColor
	} else if isSelected {
		stripColor = t.FocusPurple
	} else if hasCollision {
		stripColor = lipgloss.Color("#ff0000")
	} else if isActive {
		stripColor = t.Accent
	} else if isCompleted {
		stripColor = lipgloss.Color("#4c644f") // Soft dark forest green strip for completed
	}

	var vertChar string
	if strings.HasSuffix(task.UUID, "_moving") {
		vertChar = "┊"
	} else {
		vertChar = "│"
	}
	leftBorder := lipgloss.NewStyle().Foreground(stripColor).Render(vertChar)
	rightBorder := lipgloss.NewStyle().Foreground(stripColor).Render(vertChar)

	contentW := w - 2
	if contentW < 1 {
		contentW = 1
	}

	titleStr := theme.SentenceCase(task.Title)
	if isCompleted {
		titleStr = "✔ " + titleStr
	}
	if strings.HasSuffix(task.UUID, "_moving") {
		titleStr = "[Moving] " + titleStr
	}
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

	text := " " + titleStr

	var textStyle lipgloss.Style
	if strings.HasSuffix(task.UUID, "_moving") {
		textStyle = lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Faint(true)
	} else if isZenFocus {
		textStyle = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true)
	} else if isSelected {
		textStyle = lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true)
	} else if hasCollision {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
	} else if isActive {
		textStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	} else if isCompleted {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true) // Softer sage green for completed title
	} else {
		textStyle = lipgloss.NewStyle().Foreground(t.Fg)
	}

	row := leftBorder + textStyle.Width(contentW).Render(text) + rightBorder

	if h <= 1 {
		return row
	}

	var rows []string
	rows = append(rows, row)
	if h >= 2 {
		pName := string(task.Priority)
		var pBadge string
		pColorToUse := pColor
		if strings.HasSuffix(task.UUID, "_moving") {
			pColorToUse = t.Muted
		}

		if hasCollision {
			pBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true).Render("⚠️ " + pName)
		} else {
			pBadge = lipgloss.NewStyle().Foreground(pColorToUse).Bold(true).Render("▲ " + pName)
		}

		timeStyle := lipgloss.NewStyle().Foreground(t.Muted)
		if strings.HasSuffix(task.UUID, "_moving") {
			timeStyle = timeStyle.Faint(true)
		}
		timeStyled := timeStyle.Render(timeStr)
		meta := " " + pBadge + "  " + timeStyled

		if lipgloss.Width(meta) > contentW {
			meta = " " + pBadge
		}
		if lipgloss.Width(meta) > contentW {
			meta = theme.SliceAnsi(meta, 0, contentW)
		}

		visualW := lipgloss.Width(meta)
		if visualW < contentW {
			meta += strings.Repeat(" ", contentW-visualW)
		}

		metaLine := leftBorder + meta + rightBorder
		rows = append(rows, metaLine)
	}
	if h >= 3 {
		meta2 := fmt.Sprintf(" %d SP", task.StoryPoints)
		meta2Runes := []rune(meta2)
		if len(meta2Runes) > contentW {
			meta2 = string(meta2Runes[:contentW])
		}
		meta2Style := lipgloss.NewStyle().Foreground(t.Muted)
		if strings.HasSuffix(task.UUID, "_moving") {
			meta2Style = meta2Style.Faint(true)
		}
		meta2Line := leftBorder + meta2Style.
			Width(contentW).
			Render(meta2) + rightBorder
		rows = append(rows, meta2Line)
	}
	return strings.Join(rows, "\n")
}
