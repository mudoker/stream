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

	prefix := "  "
	if isSelected {
		prefix = "▶ "
	}
	idBadge := ""
	if task.ID != "" {
		idBadge = fmt.Sprintf("[%s] ", task.ID)
	}

	lead := prefix + chk + " " + idBadge
	leadW := lipgloss.Width(lead)
	title := theme.SentenceCase(task.Title)
	maxTitleW := innerW - leadW
	if maxTitleW < 3 {
		maxTitleW = 3
	}
	if lipgloss.Width(title) > maxTitleW {
		runes := []rune(title)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxTitleW {
			runes = runes[:len(runes)-1]
		}
		title = string(runes) + "…"
	}
	titleLine := lead + title

	// Line 2: Priority, Points/Duration, Schedule Status
	var metaParts []string
	metaParts = append(metaParts, string(task.Priority))
	if task.StoryPoints > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d SP", task.StoryPoints))
	} else if task.SchedulingType == model.Floating && task.EstimatedDurationMins > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%dm", task.EstimatedDurationMins))
	}

	if task.SchedulingType == model.Reminder {
		remDays := formatRemainingDays(task.TimeWindow.Start)
		if task.TimeWindow.Start.Second() == 1 {
			metaParts = append(metaParts, fmt.Sprintf("due (%s)", remDays))
		} else {
			metaParts = append(metaParts, fmt.Sprintf("due %s (%s)", task.TimeWindow.Start.Format("15:04"), remDays))
		}
	} else if task.SchedulingType == model.Floating {
		if task.AddedToToday {
			metaParts = append(metaParts, "Today")
		} else {
			metaParts = append(metaParts, "Unassigned")
		}
	}

	metaStr := strings.Join(metaParts, " • ")
	maxMetaW := innerW - 4
	if maxMetaW < 3 {
		maxMetaW = 3
	}
	if lipgloss.Width(metaStr) > maxMetaW {
		runes := []rune(metaStr)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxMetaW {
			runes = runes[:len(runes)-1]
		}
		metaStr = string(runes) + "…"
	}
	metaLine := "    " + metaStr

	// Line 3 (optional): Blocked status or Linked Feature
	var linkLine string
	if task.BlockedBy != "" {
		blockedStr := "⛔ Blocked by " + task.BlockedBy
		maxLinkW := innerW - 4
		if maxLinkW < 3 {
			maxLinkW = 3
		}
		if lipgloss.Width(blockedStr) > maxLinkW {
			runes := []rune(blockedStr)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxLinkW {
				runes = runes[:len(runes)-1]
			}
			blockedStr = string(runes) + "…"
		}
		linkLine = "    " + blockedStr
	} else if task.LinkedFeatureID != "" {
		linkedStr := "Link: " + task.LinkedFeatureID
		maxLinkW := innerW - 4
		if maxLinkW < 3 {
			maxLinkW = 3
		}
		if lipgloss.Width(linkedStr) > maxLinkW {
			runes := []rune(linkedStr)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxLinkW {
				runes = runes[:len(runes)-1]
			}
			linkedStr = string(runes) + "…"
		}
		linkLine = "    " + linkedStr
	}

	// Line 4 (optional): Tags
	var tagLine string
	if len(task.Tags) > 0 {
		tagStr := "# " + strings.Join(task.Tags, ", ")
		maxTagW := innerW - 4
		if maxTagW < 3 {
			maxTagW = 3
		}
		if lipgloss.Width(tagStr) > maxTagW {
			runes := []rune(tagStr)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxTagW {
				runes = runes[:len(runes)-1]
			}
			tagStr = string(runes) + "…"
		}
		tagLine = "    " + tagStr
	}

	// Line 5 (optional): Description (when selected)
	var descLine string
	if isSelected && strings.TrimSpace(task.Description) != "" {
		desc := strings.TrimSpace(task.Description)
		maxDescW := innerW - 4
		if maxDescW < 3 {
			maxDescW = 3
		}
		if lipgloss.Width(desc) > maxDescW {
			runes := []rune(desc)
			for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxDescW {
				runes = runes[:len(runes)-1]
			}
			desc = string(runes) + "…"
		}
		descLine = "    " + desc
	}

	var titleStyle, metaStyle lipgloss.Style
	if isSelected {
		titleStyle = lipgloss.NewStyle().
			Foreground(t.FocusPurple).
			Bold(true)
		metaStyle = lipgloss.NewStyle().
			Foreground(t.FocusPurple)
	} else if isDone {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#88b08b")).Bold(true)
		metaStyle = lipgloss.NewStyle().Foreground(t.Muted)
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(t.PriorityColor(task.Priority))
		metaStyle = lipgloss.NewStyle().Foreground(t.Muted)
	}

	var itemRows []string
	itemRows = append(itemRows, titleStyle.Render(titleLine))
	if metaStr != "" {
		itemRows = append(itemRows, metaStyle.Render(metaLine))
	}
	if linkLine != "" {
		var lStyle lipgloss.Style
		if isSelected {
			lStyle = lipgloss.NewStyle().Foreground(t.FocusPurple)
		} else if task.BlockedBy != "" {
			lStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		} else {
			lStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
		}
		itemRows = append(itemRows, lStyle.Render(linkLine))
	}
	if tagLine != "" {
		tagStyle := lipgloss.NewStyle().Foreground(t.Muted)
		if isSelected {
			tagStyle = lipgloss.NewStyle().Foreground(t.FocusPurple)
		}
		itemRows = append(itemRows, tagStyle.Render(tagLine))
	}
	if descLine != "" {
		descStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
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
