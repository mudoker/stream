package components

import (
	"fmt"
	"strings"

	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderTodoShelf renders the backlog todo shelf column using the headless shelf UI model.
func RenderTodoShelf(m *viewmodel.Model, t theme.Theme, appContentHeight int) string {
	l := m.Layout
	innerW := l.TodoW - 2 // account for padding
	if innerW < 10 {
		innerW = 10
	}

	isTodoFocused := m.TodoShelfFocus && !m.SidebarFocus
	shelfData := m.GetShelfData()

	var titleStr string
	var sepColor lipgloss.Color
	if isTodoFocused {
		titleStr = lipgloss.NewStyle().Foreground(t.Accent).Render("● ") +
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(shelfData.Title)
		sepColor = t.Accent
	} else {
		titleStr = "  " + lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render(shelfData.Title)
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
		sep,
	)

	var subtleSep string
	if isTodoFocused {
		subtleSep = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", innerW))
	} else {
		subtleSep = lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", innerW))
	}

	// Render each section defined in the headless shelf model
	for _, sec := range shelfData.Sections {
		secHeaderColor := t.Muted
		if isTodoFocused {
			secHeaderColor = t.Accent
		}
		header := lipgloss.NewStyle().
			Foreground(secHeaderColor).
			Bold(isTodoFocused).
			Padding(0, 1).
			Render(sec.Title)
		rows = append(rows, header, subtleSep)

		if len(sec.Tasks) == 0 {
			emptyMsg := fmt.Sprintf("  No %s tasks", strings.ToLower(string(sec.Type)))
			if sec.Type == viewmodel.SectionToday {
				emptyMsg = "  No today tasks"
			} else if sec.Type == viewmodel.SectionReminders {
				emptyMsg = "  No reminders"
			} else if sec.Type == viewmodel.SectionHabits {
				emptyMsg = "  No habits"
			} else if sec.Type == viewmodel.SectionCompleted {
				emptyMsg = "  No completed tasks"
			} else if sec.Type == viewmodel.SectionBacklog {
				emptyMsg = "  No backlog tasks"
			}
			rows = append(rows, lipgloss.NewStyle().Foreground(t.Muted).Render(emptyMsg), "")
		} else {
			for _, task := range sec.Tasks {
				if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
					selectedLineStart = len(rows)
				}
				rows = append(rows, renderShelfTaskRow(m, t, task, innerW)...)
				if m.TodoShelfFocus && task.UUID == m.SelectedTaskUUID {
					selectedLineEnd = len(rows)
				}
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
		isFirstTask := len(shelfData.Tasks) > 0 && m.SelectedTaskUUID == shelfData.Tasks[0].UUID
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
