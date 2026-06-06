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
	if isCompleted {
		stripColor = lipgloss.Color("#45475a") // dim border for completed
	} else if isZenFocus {
		stripColor = m.Theme.SuccessColor
	} else if isSelected {
		stripColor = lipgloss.Color("#ff8700")
	} else if hasCollision {
		stripColor = lipgloss.Color("#ff0000")
	} else if isActive {
		stripColor = m.Theme.Accent
	}

	leftBorder := lipgloss.NewStyle().Foreground(stripColor).Render("│")
	rightBorder := lipgloss.NewStyle().Foreground(stripColor).Render("│")

	contentW := w - 2
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

	text := " " + titleStr

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

	row := leftBorder + textStyle.Width(contentW).Render(text) + rightBorder

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
		meta := " " + pName + "  " + timeStr
		metaRunes := []rune(meta)
		if len(metaRunes) > contentW {
			meta = string(metaRunes[:contentW])
		}
		metaLine := leftBorder + lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Width(contentW).
			Render(meta) + rightBorder
		rows = append(rows, metaLine)
	}
	if h >= 3 {
		meta2 := fmt.Sprintf(" %d SP", task.StoryPoints)
		meta2Runes := []rune(meta2)
		if len(meta2Runes) > contentW {
			meta2 = string(meta2Runes[:contentW])
		}
		meta2Line := leftBorder + lipgloss.NewStyle().
			Foreground(m.Theme.Muted).
			Width(contentW).
			Render(meta2) + rightBorder
		rows = append(rows, meta2Line)
	}
	return strings.Join(rows, "\n")
}
