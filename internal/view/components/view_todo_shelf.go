package components

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderTodoShelf renders the backlog todo shelf column.
func RenderTodoShelf(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	innerW := l.TodoW - 2 // account for padding
	if innerW < 10 {
		innerW = 10
	}

	isTodoFocused := m.TodoShelfFocus && !m.SidebarFocus
	var titleStr string
	var sepColor lipgloss.Color
	if isTodoFocused {
		titleStr = lipgloss.NewStyle().Foreground(t.Accent).Render("● ") +
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("TODO SHELF")
		sepColor = t.Accent
	} else {
		titleStr = "  " + lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render("TODO SHELF")
		sepColor = lipgloss.Color("#2a2c37")
	}

	sep := lipgloss.NewStyle().Foreground(sepColor).
		Render(strings.Repeat("─", innerW))

	// Track line markers so we can dynamically snap scroll to our active selection
	selectedLineStart := -1
	selectedLineEnd := -1

	var rows []string
	rows = append(rows,
		titleStr,
		"",
		sep,
		"",
	)

	shelfTasks := m.GetTodoShelfTasks()
	var reminders []model.Task
	var habits []model.Task
	var backlog []model.Task
	var completed []model.Task
	for _, task := range shelfTasks {
		isDone := false
		if task.SchedulingType == model.Habit {
			dateStr := m.SelectedDay.Format("2006-01-02")
			for _, d := range task.CompletedDates {
				if d == dateStr {
					isDone = true
					break
				}
			}
		} else {
			isDone = task.LifecycleState == model.StateCompleted
		}

		if isDone {
			completed = append(completed, task)
		} else if task.SchedulingType == model.Reminder {
			reminders = append(reminders, task)
		} else if task.SchedulingType == model.Habit {
			habits = append(habits, task)
		} else {
			backlog = append(backlog, task)
		}
	}

	subtleSep := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", innerW))

	// ── 1. Reminders Section ─────────────────────────────────────────
	remindersHeader := lipgloss.NewStyle().
		Background(t.SelectedBg).
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("⏰ REMINDERS (%d)", len(reminders)))
	rows = append(rows, remindersHeader, subtleSep)
	if len(reminders) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  No reminders"), "")
	} else {
		for _, task := range reminders {
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineStart = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineEnd = len(rows)
			}
		}
	}

	// ── 2. Habits Section ────────────────────────────────────────────
	habitsHeader := lipgloss.NewStyle().
		Background(t.SelectedBg).
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("🔁 HABITS (%d)", len(habits)))
	rows = append(rows, habitsHeader, subtleSep)
	if len(habits) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  No habits"), "")
	} else {
		for _, task := range habits {
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineStart = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineEnd = len(rows)
			}
		}
	}

	// ── 3. Backlog Section ───────────────────────────────────────────
	backlogHeader := lipgloss.NewStyle().
		Background(t.SelectedBg).
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("☱ BACKLOG (%d)", len(backlog)))
	rows = append(rows, backlogHeader, subtleSep)
	if len(backlog) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  No backlog tasks"), "")
	} else {
		for _, task := range backlog {
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineStart = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineEnd = len(rows)
			}
		}
	}

	// ── 4. Completed Section ─────────────────────────────────────────
	completedHeader := lipgloss.NewStyle().
		Background(t.SelectedBg).
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("✓ COMPLETED (%d)", len(completed)))
	rows = append(rows, completedHeader, subtleSep)
	if len(completed) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render("  No completed tasks"), "")
	} else {
		for _, task := range completed {
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineStart = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
			if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
				selectedLineEnd = len(rows)
			}
		}
	}

	// Flatten rows array down to raw line tokens
	allLines := strings.Split(strings.Join(rows, "\n"), "\n")
	maxVisible := appContentHeight - 4
	if maxVisible < 4 {
		maxVisible = 4
	}

	// ── Smart Auto-Scrolling Viewport Window Clamping ───────────────
	offset := m.ShelfScrollOffset

	// If a task row is actively selected, force viewport boundaries to wrap it cleanly
	if selectedLineStart != -1 {
		isFirstTask := len(shelfTasks) > 0 && m.SelectedTaskUUID == shelfTasks[0].UUID
		if isFirstTask {
			offset = 0
		} else {
			paddingTop := 1
			paddingBottom := 3 // Be generous to render fully

			// If selection is positioned above the current screen layout view
			if selectedLineStart < offset+paddingTop {
				offset = selectedLineStart - paddingTop
				// If we are close to the top, scroll all the way to the top to show headers
				if offset <= 8 {
					offset = 0
				}
			}
			// If selection dips below the bottom visible edge line bounds
			if selectedLineEnd+paddingBottom >= offset+maxVisible {
				offset = selectedLineEnd + paddingBottom - maxVisible
			}
			// Make sure we never scroll past the start of the task
			if offset > selectedLineStart {
				offset = selectedLineStart
			}
		}
		m.ShelfScrollOffset = offset
	}

	// Safeguard scroll limits safely against structural text length
	if offset > len(allLines)-maxVisible {
		offset = len(allLines) - maxVisible
	}
	if offset < 0 {
		offset = 0
	}

	// Slice visible items matching our clamped row calculations
	var visible []string
	if offset > 0 {
		visible = append(visible, lipgloss.NewStyle().Foreground(t.Muted).Render("  ▲ scroll up"))
	} else {
		visible = append(visible, "") // Top edge aesthetic gap buffer
	}

	endSlice := offset + maxVisible
	if endSlice > len(allLines) {
		endSlice = len(allLines)
	}

	visible = append(visible, allLines[offset:endSlice]...)

	if endSlice < len(allLines) {
		visible = append(visible, lipgloss.NewStyle().Foreground(t.Muted).Render("  ▼ scroll down"))
	} else {
		visible = append(visible, "") // Bottom edge aesthetic gap buffer
	}

	return strings.Join(visible, "\n")
}
