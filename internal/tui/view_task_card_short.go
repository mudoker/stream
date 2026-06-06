package tui

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

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
