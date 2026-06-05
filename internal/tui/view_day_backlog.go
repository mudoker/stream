package tui

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// renderTodoShelf renders the backlog todo shelf column.
// All item widths are explicitly bounded by m.Layout.TodoW to prevent line wrap.
func (m Model) renderTodoShelf() string {
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

	priorities := []model.Priority{model.P0, model.P1, model.P2, model.P3}
	pTitles := []string{"Urgent", "High", "Medium", "Low"}
	pColors := []lipgloss.Color{m.Theme.P0Color, m.Theme.P1Color, m.Theme.P2Color, m.Theme.P3Color}

	hasAny := false
	for idx, prio := range priorities {
		var list []model.Task
		for _, t := range shelfTasks {
			if t.Priority == prio {
				list = append(list, t)
			}
		}

		if len(list) == 0 {
			continue
		}
		hasAny = true

		pHeader := lipgloss.NewStyle().Foreground(pColors[idx]).Bold(true).
			Render(fmt.Sprintf("▲ %s", pTitles[idx]))
		rows = append(rows, pHeader)

		for _, t := range list {
			isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID

			title := sentenceCase(t.Title)
			// truncate to fit innerW minus bullet and padding
			maxTitleW := innerW - 4
			if len([]rune(title)) > maxTitleW {
				if maxTitleW > 2 {
					title = string([]rune(title)[:maxTitleW-1]) + "…"
				} else {
					title = string([]rune(title)[:maxTitleW])
				}
			}

			bullet := lipgloss.NewStyle().Foreground(pColors[idx]).Render("●")

			if isSelected {
				// Selected: full-width highlight — Width() ensures no jagged edge
				line := fmt.Sprintf(" %s %s", bullet, title)
				rows = append(rows,
					lipgloss.NewStyle().
						Background(m.Theme.SelectedBg).
						Foreground(m.Theme.FocusPurple).
						Bold(true).
						Width(innerW).
						Render(line),
				)
			} else {
				line := fmt.Sprintf(" %s %s", bullet, title)
				rows = append(rows,
					lipgloss.NewStyle().
						Foreground(m.Theme.Fg).
						Width(innerW).
						Render(line),
				)
			}
		}
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

	maxVisible := l.Height - 2
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


