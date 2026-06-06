package tui

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// renderTodoShelf renders the backlog todo shelf column.
// All item widths are explicitly bounded by m.Layout.TodoW to prevent line wrap.
func (m Model) renderTodoShelf(appContentHeight int) string {
	l := m.Layout
	innerW := l.TodoW - 2 // account for padding
	if innerW < 10 {
		innerW = 10
	}

	isTodoFocused := m.TodoShelfFocus && !m.SidebarFocus
	var titleStr string
	var sepColor lipgloss.Color
	if isTodoFocused {
		titleStr = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("● ") +
			lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("TODO SHELF")
		sepColor = m.Theme.Accent
	} else {
		titleStr = "  " + lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).Render("TODO SHELF")
		sepColor = lipgloss.Color("#2a2c37")
	}

	sep := lipgloss.NewStyle().Foreground(sepColor).
		Render(strings.Repeat("─", innerW))

	var rows []string
	rows = append(rows,
		titleStr,
		"",
		sep,
		"",
	)

	shelfTasks := m.getTodoShelfTasks()
	var reminders []model.Task
	var backlog []model.Task
	for _, t := range shelfTasks {
		if t.SchedulingType == model.Reminder {
			reminders = append(reminders, t)
		} else {
			backlog = append(backlog, t)
		}
	}

	// 1. Reminders Section
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("⏰ REMINDERS"))
	rows = append(rows, lipgloss.NewStyle().Foreground(sepColor).Render(strings.Repeat("─", innerW)))
	if len(reminders) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No reminders"), "")
	} else {
		for _, t := range reminders {
			rows = append(rows, m.renderShelfTaskRow(t, innerW)...)
		}
	}

	// 2. Backlog Section
	rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("☱ BACKLOG"))
	rows = append(rows, lipgloss.NewStyle().Foreground(sepColor).Render(strings.Repeat("─", innerW)))
	if len(backlog) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  No backlog tasks"), "")
	} else {
		for _, t := range backlog {
			rows = append(rows, m.renderShelfTaskRow(t, innerW)...)
		}
	}

	// Apply scroll offset
	all := strings.Join(rows, "\n")
	allLines := strings.Split(all, "\n")

	offset := m.ShelfScrollOffset
	if offset >= len(allLines) {
		offset = len(allLines) - 1
	}
	if offset < 0 {
		offset = 0
	}

	maxVisible := appContentHeight - 2
	var visible []string
	if offset > 0 {
		visible = append(visible,
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▲ scroll up"))
	}
	visible = append(visible, allLines[offset:]...)
	if len(visible) > maxVisible {
		visible = visible[:maxVisible-1]
		visible = append(visible,
			lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▼ scroll down"))
	}

	return lipgloss.NewStyle().
		Padding(1, 1).
		Render(strings.Join(visible, "\n"))
}

func (m Model) renderShelfTaskRow(t model.Task, innerW int) []string {
	isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID

	chk := "[ ]"
	if t.LifecycleState == model.StateCompleted {
		chk = "[✓]"
	}

	title := sentenceCase(t.Title)
	// Truncate title to fit innerW minus checkbox and indicator
	maxTitleW := innerW - 7
	if len([]rune(title)) > maxTitleW {
		if maxTitleW > 2 {
			title = string([]rune(title)[:maxTitleW-1]) + "…"
		} else {
			title = string([]rune(title)[:maxTitleW])
		}
	}

	prefix := "  "
	if isSelected {
		prefix = "▶ "
	}
	indicator := ""
	if t.SchedulingType == model.Reminder {
		indicator = " ⏰"
	}

	titleLine := fmt.Sprintf("%s%s %s%s", prefix, chk, title, indicator)

	var details []string
	details = append(details, fmt.Sprintf("%d SP", t.StoryPoints))
	if t.SchedulingType == model.Reminder {
		details = append(details, fmt.Sprintf("due %s", t.TimeWindow.Start.Format("15:04")))
	}
	if len(t.Tags) > 0 {
		details = append(details, strings.Join(t.Tags, ", "))
	}
	detailStr := strings.Join(details, " • ")
	maxDetailW := innerW - 5
	if len([]rune(detailStr)) > maxDetailW {
		if maxDetailW > 2 {
			detailStr = string([]rune(detailStr)[:maxDetailW-1]) + "…"
		} else {
			detailStr = string([]rune(detailStr)[:maxDetailW])
		}
	}
	detailLine := "     " + detailStr

	var titleStyle, detailStyle lipgloss.Style
	if isSelected {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8700")).Bold(true)
		detailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa44"))
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(m.getPriorityColor(t.Priority))
		detailStyle = lipgloss.NewStyle().Foreground(m.Theme.Muted)
	}

	var itemRows []string
	itemRows = append(itemRows, titleStyle.Render(titleLine))
	if detailStr != "" {
		itemRows = append(itemRows, detailStyle.Render(detailLine))
	}

	if isSelected && t.Description != "" {
		desc := t.Description
		maxDescW := innerW - 5
		if len([]rune(desc)) > maxDescW {
			if maxDescW > 2 {
				desc = string([]rune(desc)[:maxDescW-1]) + "…"
			} else {
				desc = string([]rune(desc)[:maxDescW])
			}
		}
		descLine := "     " + lipgloss.NewStyle().Italic(true).Render(desc)
		itemRows = append(itemRows, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(descLine))
	}

	itemRows = append(itemRows, "")
	return itemRows
}

func (m Model) getPriorityColor(p model.Priority) lipgloss.Color {
	switch p {
	case model.P0:
		return m.Theme.P0Color
	case model.P1:
		return m.Theme.P1Color
	case model.P2:
		return m.Theme.P2Color
	default:
		return m.Theme.P3Color
	}
}
