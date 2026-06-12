package components

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderShortCard renders a compact card for h < 3 rows using left and right borders.
func RenderShortCard(m *viewmodel.Model, t theme.Theme, task model.Task, w, h int, pColor lipgloss.Color, isActive bool, isSelected bool, hasCollision bool, isZenFocus bool, timeStr string) string {
	stripColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
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
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
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
	} else if strings.HasSuffix(task.UUID, "_adjusting") {
		titleStr = "[Adjusting] " + titleStr
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
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
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
		if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
			pColorToUse = t.Muted
		}

		if hasCollision {
			pBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true).Render("⚠️ " + pName)
		} else {
			pBadge = lipgloss.NewStyle().Foreground(pColorToUse).Bold(true).Render("▲ " + pName)
		}

		timeStyle := lipgloss.NewStyle().Foreground(t.Muted)
		if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
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
		if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
			meta2Style = meta2Style.Faint(true)
		}
		meta2Line := leftBorder + meta2Style.
			Width(contentW).
			Render(meta2) + rightBorder
		rows = append(rows, meta2Line)
	}
	return strings.Join(rows, "\n")
}
