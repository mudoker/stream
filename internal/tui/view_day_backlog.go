package tui

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTodoShelf(shelfWidth int, height int) string {
	var shelfLines []string
	shelfLines = append(shelfLines, "TODO BACKLOG\n")

	shelfTasks := m.getTodoShelfTasks()

	priorities := []model.Priority{model.P0, model.P1, model.P2, model.P3}
	pTitles := []string{"P0 URGENT", "P1 HIGH", "P2 MEDIUM", "P3 LOW"}
	pColors := []lipgloss.Color{m.Theme.P0Color, m.Theme.P1Color, m.Theme.P2Color, m.Theme.P3Color}

	for idx, prio := range priorities {
		var list []model.Task
		for _, t := range shelfTasks {
			if t.Priority == prio {
				list = append(list, t)
			}
		}

		if len(list) == 0 && prio != model.P0 {
			continue
		}

		header := lipgloss.NewStyle().Foreground(pColors[idx]).Bold(true).Render(pTitles[idx])
		shelfLines = append(shelfLines, "\n"+header)

		if len(list) == 0 {
			shelfLines = append(shelfLines, "  ● no backlog items")
			continue
		}

		for _, t := range list {
			isSelected := m.TodoShelfFocus && t.UUID == m.SelectedTaskUUID
			bullet := lipgloss.NewStyle().Foreground(pColors[idx]).Render("●")

			title := sentenceCase(t.Title)
			if len(title) > shelfWidth-14 {
				title = title[:shelfWidth-16] + ".."
			}

			line := fmt.Sprintf("  %s %s", bullet, title)
			if isSelected {
				line = lipgloss.NewStyle().
					Background(m.Theme.SelectedBg).
					Foreground(m.Theme.FocusPurple).
					Bold(true).
					Padding(0, 1).
					Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(m.Theme.Fg).Render(line)
			}
			shelfLines = append(shelfLines, line)
		}
	}

	// Apply Shelf scroll offset
	shelfLinesRendered := strings.Join(shelfLines, "\n")
	shelfLinesList := strings.Split(shelfLinesRendered, "\n")

	if m.ShelfScrollOffset >= len(shelfLinesList) {
		m.ShelfScrollOffset = len(shelfLinesList) - 1
	}
	if m.ShelfScrollOffset < 0 {
		m.ShelfScrollOffset = 0
	}

	var visibleShelfList []string
	if m.ShelfScrollOffset > 0 {
		visibleShelfList = append(visibleShelfList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▲  (scroll up)"))
	}
	visibleShelfList = append(visibleShelfList, shelfLinesList[m.ShelfScrollOffset:]...)

	if len(visibleShelfList) > height-2 {
		visibleShelfList = visibleShelfList[:height-2]
		visibleShelfList = append(visibleShelfList, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  ▼  (scroll down)"))
	}

	return m.Theme.PanelStyle.
		Width(shelfWidth - 4).
		Height(height - 2).
		Render(strings.Join(visibleShelfList, "\n"))
}
