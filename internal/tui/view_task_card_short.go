package tui

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// renderShortCard renders a compact card for h < 3 rows using left and right borders.
func (m Model) renderShortCard(task model.Task, w, h int, pColor lipgloss.Color, isActive bool, isSelected bool, hasCollision bool, isZenFocus bool, timeStr string) string {
	stripColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if strings.HasSuffix(task.UUID, "_moving") {
		stripColor = m.Theme.Muted
	} else if isZenFocus {
		stripColor = m.Theme.SuccessColor
	} else if isSelected {
		stripColor = lipgloss.Color("#ff8700")
	} else if hasCollision {
		stripColor = lipgloss.Color("#ff0000")
	} else if isActive {
		stripColor = m.Theme.Accent
	} else if isCompleted {
		stripColor = lipgloss.Color("#45475a") // dim border for completed
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

	titleStr := sentenceCase(task.Title)
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
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.Muted).Italic(true).Faint(true)
	} else if isZenFocus {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true)
	} else if isSelected {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8700")).Bold(true)
	} else if hasCollision {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
	} else if isActive {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
	} else if isCompleted {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Faint(true)
	} else {
		textStyle = lipgloss.NewStyle().Foreground(m.Theme.Fg)
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
			pColorToUse = m.Theme.Muted
		}

		if hasCollision {
			pBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true).Render("⚠️ " + pName)
		} else {
			pBadge = lipgloss.NewStyle().Foreground(pColorToUse).Bold(true).Render("▲ " + pName)
		}
		
		timeStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
		if strings.HasSuffix(task.UUID, "_moving") {
			timeStyle = timeStyle.Faint(true)
		}
		timeStyled := timeStyle.Render(timeStr)
		meta := " " + pBadge + "  " + timeStyled
		
		if lipgloss.Width(meta) > contentW {
			meta = " " + pBadge
		}
		if lipgloss.Width(meta) > contentW {
			meta = sliceAnsi(meta, 0, contentW)
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
		meta2Style := lipgloss.NewStyle().Foreground(m.Theme.Muted)
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
