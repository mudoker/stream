package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// renderTaskCard renders a complete multi-line Lipgloss task card block.
// w and h are the card's outer dimensions (including borders).
// Returns the full rendered string — the caller overlays it onto the grid canvas.
func (m Model) renderTaskCard(task model.Task, w, h int, isActive bool) string {
	pColor := m.priorityColor(task.Priority)
	now := time.Now()

	// Card background
	cardBg := m.Theme.PanelBg
	borderColor := pColor
	if isActive {
		cardBg = m.Theme.SelectedBg
		borderColor = m.Theme.Accent
	}

	// Time range / countdown string
	timeStr := fmt.Sprintf("%s–%s",
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
		timeStr = fmt.Sprintf("%02d:%02d:%02d remaining", h2, mi, s)
	}

	// Priority badge
	pName := string(task.Priority)
	priorityBadge := lipgloss.NewStyle().Foreground(pColor).Render(
		fmt.Sprintf("▲ %s • %d SP", pName, task.StoryPoints),
	)

	innerW := w - 2 // subtract border width
	innerH := h - 2 // subtract border height
	if innerW < 2 {
		innerW = 2
	}
	if innerH < 1 {
		innerH = 1
	}

	// Truncate title to fit inner width
	title := sentenceCase(task.Title)
	if len(title) > innerW-1 {
		if innerW > 4 {
			title = title[:innerW-4] + "…"
		} else {
			title = title[:innerW]
		}
	}

	// Build content lines
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(title))

	if isActive {
		badge := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("● ACTIVE")
		lines = append(lines, badge)
		timerLine := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(timeStr)
		lines = append(lines, timerLine)
	} else if innerH >= 3 {
		lines = append(lines, "") // empty spacer
	}

	// Metadata row (only if there's room)
	if innerH >= 2 {
		meta := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(
			fmt.Sprintf("%s  %s", priorityBadge, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeStr)),
		)
		lines = append(lines, meta)
	}

	// Very short blocks (h < 4): use a left-strip style without full borders
	if h < 4 {
		return m.renderShortCard(task, w, h, pColor, cardBg, isActive, timeStr)
	}

	content := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		Background(cardBg).
		Foreground(m.Theme.Fg).
		Render(strings.Join(lines, "\n"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(cardBg).
		Width(innerW).
		Render(content)
}

// renderShortCard renders a compact card for h < 4 rows using a left strip bar.
func (m Model) renderShortCard(task model.Task, w, h int, pColor, cardBg lipgloss.Color, isActive bool, timeStr string) string {
	strip := lipgloss.NewStyle().
		Foreground(pColor).
		Background(cardBg).
		Render("▎")

	contentW := w - 1
	if contentW < 1 {
		contentW = 1
	}

	var text string
	switch h {
	case 1:
		text = fmt.Sprintf(" %s", sentenceCase(task.Title))
	case 2:
		if !isActive {
			text = fmt.Sprintf(" %s  %s", sentenceCase(task.Title),
				lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(string(task.Priority)))
		} else {
			text = fmt.Sprintf(" %s  ●", sentenceCase(task.Title))
		}
	case 3:
		text = fmt.Sprintf(" %s", sentenceCase(task.Title))
	}

	if len(text) > contentW {
		if contentW > 3 {
			text = text[:contentW-1] + "…"
		} else {
			text = text[:contentW]
		}
	}
	padding := contentW - len([]rune(text))
	if padding < 0 {
		padding = 0
	}
	text += strings.Repeat(" ", padding)

	row := strip + lipgloss.NewStyle().
		Background(cardBg).
		Foreground(m.Theme.Fg).
		Render(text)

	if h <= 1 {
		return row
	}

	var rows []string
	rows = append(rows, row)
	if h >= 2 {
		meta := fmt.Sprintf(" %s  %s", task.Priority, timeStr)
		if len(meta) > contentW {
			meta = meta[:contentW]
		}
		meta += strings.Repeat(" ", contentW-len([]rune(meta)))
		rows = append(rows, strip+lipgloss.NewStyle().Background(cardBg).Foreground(m.Theme.Muted).Render(meta))
	}
	if h >= 3 {
		meta2 := fmt.Sprintf(" %d SP", task.StoryPoints)
		meta2 += strings.Repeat(" ", contentW-len([]rune(meta2)))
		rows = append(rows, strip+lipgloss.NewStyle().Background(cardBg).Foreground(m.Theme.Muted).Render(meta2))
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
