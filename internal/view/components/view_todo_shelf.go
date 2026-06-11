package components

import (
	"fmt"
	"strings"
	"time"

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
	selectedLineIdx := -1

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
	for _, task := range shelfTasks {
		if task.SchedulingType == model.Reminder {
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
				selectedLineIdx = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
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
				selectedLineIdx = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
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
				selectedLineIdx = len(rows)
			}
			rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
		}
	}

	// Flatten rows array down to raw line tokens
	allLines := strings.Split(strings.Join(rows, "\n"), "\n")
	maxVisible := appContentHeight - 2
	if maxVisible < 4 {
		maxVisible = 4
	}

	// ── Smart Auto-Scrolling Viewport Window Clamping ───────────────
	offset := m.ShelfScrollOffset

	// If a task row is actively selected, force viewport boundaries to wrap it cleanly
	if selectedLineIdx != -1 && m.SelectedTaskUUID != m.PrevSelectedTaskUUID {
		// If selection is positioned above the current screen layout view
		if selectedLineIdx < offset {
			offset = selectedLineIdx
			// If we are close to the top, scroll all the way to the top to show headers
			if offset <= 8 {
				offset = 0
			}
		}
		// If selection dips below the bottom visible edge line bounds
		if selectedLineIdx >= offset+(maxVisible-2) {
			offset = selectedLineIdx - (maxVisible - 3)
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
	}

	return lipgloss.NewStyle().
		Padding(1, 1).
		Render(strings.Join(visible, "\n"))
}

func renderShelfTaskRow(m *viewmodel.Model, t theme.Theme, task model.Task, innerW int) []string {
	isSelected := m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID

	chk := "☐"
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
		chk = "☑"
	}

	title := theme.SentenceCase(task.Title)
	maxTitleW := innerW - 6
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
	titleLine := fmt.Sprintf("%s%s %s", prefix, chk, title)

	var details []string
	details = append(details, string(task.Priority))
	details = append(details, fmt.Sprintf("%d SP", task.StoryPoints))
	if task.SchedulingType == model.Reminder {
		remDays := formatRemainingDays(task.TimeWindow.Start)
		details = append(details, fmt.Sprintf("due %s (%s)", task.TimeWindow.Start.Format("15:04"), remDays))
	}
	if len(task.Tags) > 0 {
		details = append(details, strings.Join(task.Tags, ", "))
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

	if isSelected {
		titleLineLen := lipgloss.Width(titleLine)
		if titleLineLen < innerW {
			titleLine += strings.Repeat(" ", innerW-titleLineLen)
		}
		if detailStr != "" {
			detailLineLen := lipgloss.Width(detailLine)
			if detailLineLen < innerW {
				detailLine += strings.Repeat(" ", innerW-detailLineLen)
			}
		}
	}

	var titleStyle, detailStyle lipgloss.Style
	if isSelected {
		titleStyle = lipgloss.NewStyle().
			Foreground(t.FocusPurple).
			Bold(true).
			Background(t.SelectedBg)
		detailStyle = lipgloss.NewStyle().
			Foreground(t.FocusPurple).
			Background(t.SelectedBg)
	} else if isDone {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true)
		detailStyle = lipgloss.NewStyle().Foreground(t.Muted)
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(t.PriorityColor(task.Priority))
		detailStyle = lipgloss.NewStyle().Foreground(t.Muted)
	}

	var itemRows []string
	itemRows = append(itemRows, titleStyle.Render(titleLine))
	if detailStr != "" {
		itemRows = append(itemRows, detailStyle.Render(detailLine))
	}

	if isSelected && task.Description != "" {
		desc := task.Description
		maxDescW := innerW - 5
		if len([]rune(desc)) > maxDescW {
			if maxDescW > 2 {
				desc = string([]rune(desc)[:maxDescW-1]) + "…"
			} else {
				desc = string([]rune(desc)[:maxDescW])
			}
		}
		descLine := "     " + desc
		descLineLen := lipgloss.Width(descLine)
		if descLineLen < innerW {
			descLine += strings.Repeat(" ", innerW-descLineLen)
		}
		descStyle := lipgloss.NewStyle().
			Foreground(t.Muted).
			Italic(true).
			Background(t.SelectedBg)
		itemRows = append(itemRows, descStyle.Render(descLine))
	}

	itemRows = append(itemRows, "")
	return itemRows
}

func formatRemainingDays(due time.Time) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
	
	dueLocal := dueDay.In(today.Location())
	duration := dueLocal.Sub(today)
	var days int
	if duration >= 0 {
		days = int((duration.Hours() + 12) / 24)
	} else {
		days = int((duration.Hours() - 12) / 24)
	}

	if days == 0 {
		return "due today"
	} else if days == 1 {
		return "1 day remaining"
	} else if days > 1 {
		return fmt.Sprintf("%d days remaining", days)
	} else if days == -1 {
		return "overdue by 1 day"
	} else {
		return fmt.Sprintf("overdue by %d days", -days)
	}
}
