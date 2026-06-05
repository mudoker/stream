package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTaskCardLine(task model.Task, w int, h int, lineIdx int, isActive bool, isNowRow bool, isLeftmost bool) string {
	var pBarColor lipgloss.Color = m.Theme.P2Color
	if task.Priority == model.P0 {
		pBarColor = m.Theme.P0Color
	} else if task.Priority == model.P1 {
		pBarColor = m.Theme.P1Color
	} else if task.Priority == model.P3 {
		pBarColor = m.Theme.P3Color
	}

	bgStyle := lipgloss.NewStyle().Background(m.Theme.PanelBg).Foreground(m.Theme.Fg)
	if isActive {
		bgStyle = bgStyle.Background(m.Theme.SelectedBg)
	}

	timeStr := fmt.Sprintf("%s–%s", task.TimeWindow.Start.Format("15:04"), task.TimeWindow.End.Format("15:04"))
	now := time.Now()
	if isActive {
		remaining := task.TimeWindow.End.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		hVal := int(remaining.Hours())
		mVal := int(remaining.Minutes()) % 60
		sVal := int(remaining.Seconds()) % 60
		timeStr = fmt.Sprintf("%02d:%02d:%02d Remaining", hVal, mVal, sVal)
	}

	// Handle short task blocks (h < 4) using the borderless side-strip block style
	if h < 4 {
		var lineText string
		if isNowRow {
			contentW := w - 1
			if contentW < 1 {
				contentW = 1
			}
			if isLeftmost {
				badge := getNowBadge(contentW, now)
				lineText = badge + strings.Repeat("─", contentW-len(badge))
			} else {
				lineText = strings.Repeat("─", contentW)
			}
			leftStrip := lipgloss.NewStyle().Foreground(pBarColor).Background(bgStyle.GetBackground()).Render("┃")
			return leftStrip + lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Background(bgStyle.GetBackground()).Bold(true).Render(lineText)
		}

		if h == 1 {
			// Single row representation: sentence-case Title (Priority)
			lineText = fmt.Sprintf(" %s (%s)", sentenceCase(task.Title), task.Priority)
		} else if h == 2 {
			if lineIdx == 0 {
				lineText = " " + sentenceCase(task.Title)
			} else {
				lineText = fmt.Sprintf(" ▲ %s • %d SP", task.Priority, task.StoryPoints)
			}
		} else { // h == 3
			if lineIdx == 0 {
				lineText = " " + sentenceCase(task.Title)
			} else if lineIdx == 1 {
				lineText = fmt.Sprintf(" ▲ %s • %d SP", task.Priority, task.StoryPoints)
			} else {
				lineText = " " + timeStr
			}
		}

		contentW := w - 1
		if contentW < 1 {
			contentW = 1
		}
		if len(lineText) > contentW {
			if contentW > 3 {
				lineText = lineText[:contentW-3] + "..."
			} else {
				lineText = lineText[:contentW]
			}
		} else {
			lineText = lineText + strings.Repeat(" ", contentW-len(lineText))
		}

		leftStrip := lipgloss.NewStyle().Foreground(pBarColor).Background(bgStyle.GetBackground()).Render("┃")
		return leftStrip + bgStyle.Render(lineText)
	}

	// Standardize task blocks as solid layout cards with custom borders (h >= 4)
	customBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "┃",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	cardStyle := lipgloss.NewStyle().
		Border(customBorder).
		BorderForeground(pBarColor).
		Background(bgStyle.GetBackground()).
		Width(w).
		Height(h)

	// Build card content lines
	var content []string
	content = append(content, sentenceCase(task.Title))

	// Active block real-time progress countdown pulse
	if isActive {
		timerText := "● ACTIVE"
		content = append(content, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(timerText))
	} else {
		content = append(content, "")
	}

	// Metadata grouping inline at the bottom
	pColorName := "P2"
	if task.Priority == model.P0 {
		pColorName = "P0"
	} else if task.Priority == model.P1 {
		pColorName = "P1"
	} else if task.Priority == model.P3 {
		pColorName = "P3"
	}
	metaRow := fmt.Sprintf("▲ %s • %d SP • %s",
		pColorName,
		task.StoryPoints,
		timeStr,
	)
	content = append(content, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(metaRow))

	// Join content and pad/border render
	var visibleContent []string
	maxLines := h - 2
	for i := 0; i < maxLines; i++ {
		if isNowRow && i == lineIdx-1 {
			innerWidth := w - 2
			if innerWidth < 1 {
				innerWidth = 1
			}
			var lineText string
			if isLeftmost {
				badge := getNowBadge(innerWidth, now)
				lineText = badge + strings.Repeat("─", innerWidth-len(badge))
			} else {
				lineText = strings.Repeat("─", innerWidth)
			}
			visibleContent = append(visibleContent, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(lineText))
		} else {
			val := ""
			if i < len(content) {
				val = content[i]
			}
			visibleContent = append(visibleContent, val)
		}
	}

	rendered := cardStyle.Render(strings.Join(visibleContent, "\n"))
	renderedLines := strings.Split(rendered, "\n")
	if lineIdx < len(renderedLines) {
		return renderedLines[lineIdx]
	}
	return strings.Repeat(" ", w)
}

func sentenceCase(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[:1]) + s[1:]
}
