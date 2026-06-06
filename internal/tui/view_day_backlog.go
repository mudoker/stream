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

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).
		Render(strings.Repeat("─", innerW))
	mutedB := lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true)

	var rows []string
	rows = append(rows,
		mutedB.Render("BACKLOG"),
		"",
		sep,
		"",
	)

	shelfTasks := m.getTodoShelfTasks()

	groups := []struct {
		name  string
		color lipgloss.Color
		tasks []model.Task
	}{
		{"URGENT", m.Theme.P0Color, nil},
		{"TODAY", m.Theme.P1Color, nil},
		{"BACKLOG", m.Theme.P2Color, nil},
	}

	for _, t := range shelfTasks {
		switch t.Priority {
		case model.P0:
			groups[0].tasks = append(groups[0].tasks, t)
		case model.P1:
			groups[1].tasks = append(groups[1].tasks, t)
		default:
			groups[2].tasks = append(groups[2].tasks, t)
		}
	}

	hasAny := false
	for _, g := range groups {
		if len(g.tasks) == 0 {
			continue
		}
		hasAny = true

		pHeader := lipgloss.NewStyle().Foreground(g.color).Bold(true).Render("▲ " + g.name)
		rows = append(rows, pHeader)

		for _, t := range g.tasks {
			isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID

			chk := "[ ]"
			if t.LifecycleState == model.StateCompleted {
				chk = "[✓]"
			}

			title := sentenceCase(t.Title)
			// Truncate to fit innerW minus checkbox and padding
			maxTitleW := innerW - 5
			if len([]rune(title)) > maxTitleW {
				if maxTitleW > 2 {
					title = string([]rune(title)[:maxTitleW-1]) + "…"
				} else {
					title = string([]rune(title)[:maxTitleW])
				}
			}

			line := fmt.Sprintf(" %s %s", chk, title)
			var itemStyle lipgloss.Style
			if isSelected {
				itemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8700")).Bold(true)
			} else {
				itemStyle = lipgloss.NewStyle().Foreground(g.color)
			}
			rows = append(rows, itemStyle.Width(innerW).Render(line))
		}
		// Add a blank row only after the section to separate them
		rows = append(rows, "")
	}

	if !hasAny {
		rows = append(rows,
			lipgloss.NewStyle().Foreground(m.Theme.Muted).
				Width(innerW).Render("  No backlog tasks"),
		)
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
