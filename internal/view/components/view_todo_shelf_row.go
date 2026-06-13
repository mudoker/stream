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
	if task.SchedulingType != model.Reminder {
		details = append(details, fmt.Sprintf("%d SP", task.StoryPoints))
	}
	if task.SchedulingType == model.Reminder {
		remDays := formatRemainingDays(task.TimeWindow.Start)
		if task.TimeWindow.Start.Second() == 1 {
			details = append(details, fmt.Sprintf("due (%s)", remDays))
		} else {
			details = append(details, fmt.Sprintf("due %s (%s)", task.TimeWindow.Start.Format("15:04"), remDays))
		}
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
