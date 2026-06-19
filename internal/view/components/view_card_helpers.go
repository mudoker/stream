package components

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

func cardBorderChars(task model.Task, hasRest bool) (string, string, string, string, string, string) {
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
		bottomLeftChar := "└"
		if hasRest {
			bottomLeftChar = "├"
		}
		bottomRightChar := "┘"
		if hasRest {
			bottomRightChar = "┤"
		}
		return "┌", "┐", bottomLeftChar, bottomRightChar, "╌", "┊"
	}

	bottomLeftChar := "╰"
	if hasRest {
		bottomLeftChar = "├"
	}
	bottomRightChar := "╯"
	if hasRest {
		bottomRightChar = "┤"
	}
	return "╭", "╮", bottomLeftChar, bottomRightChar, "─", "│"
}

func cardTitleStr(task model.Task, isCompleted bool) string {
	titleStr := theme.SentenceCase(task.Title)
	if isCompleted {
		titleStr = "✔ " + titleStr
	}
	if strings.HasSuffix(task.UUID, "_moving") {
		titleStr = "[Moving] " + titleStr
	} else if strings.HasSuffix(task.UUID, "_adjusting") {
		titleStr = "[Adjusting] " + titleStr
	}
	return titleStr
}

func cardTitleStyle(t theme.Theme, task model.Task, isZenFocus, isSelected, hasCollision, isCompleted bool) lipgloss.Style {
	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
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
	return titleStyle
}

func cardMetaStr(t theme.Theme, task model.Task, contentW int, priorityBadge, timeStr string) string {
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
		mutedStyle = mutedStyle.Faint(true)
	}
	spStrStyled := mutedStyle.Render(fmt.Sprintf("%d SP", task.StoryPoints))
	timeStrStyled := mutedStyle.Render(timeStr)
	bulletStyled := mutedStyle.Render("  •  ")

	var metaStr string
	if task.SchedulingType == model.Event {
		metaStr = fmt.Sprintf("%s%s%s", priorityBadge, bulletStyled, timeStrStyled)
	} else {
		metaStr = fmt.Sprintf("%s%s%s%s%s", priorityBadge, bulletStyled, spStrStyled, bulletStyled, timeStrStyled)
	}

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
	return metaStr
}
