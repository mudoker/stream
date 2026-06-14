package modals

import (
	"fmt"
	"strconv"
	"strings"

	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

func RenderFormModal(m *viewmodel.Model, t theme.Theme) string {
	f := &m.Form
	f.SyncDaysSelectedFromInput()
	const innerW = 52

	var fields []string
	headerText := "Create Task"
	if m.IsEditing {
		headerText = "Edit Task"
	}
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(headerText)
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	renderField := func(num, label string, input string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), input)
	}

	renderDropdown := func(num, label string, value string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
			valStr := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
			return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
		}
		valStr := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
	}

	renderDaysSelect := func(num, label string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		isActiveField := f.ActiveField == index
		if isActiveField {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}

		dayNames := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
		var dayStrs []string
		for i, name := range dayNames {
			sel := f.RecurringDaysSelected[i]
			isCursor := isActiveField && f.RecurringDaysSubIdx == i

			var dStr string
			if sel && isCursor {
				// Selected + cursor: bright accent with filled checkbox
				dStr = lipgloss.NewStyle().
					Foreground(t.CanvasBg).
					Background(t.Accent).
					Bold(true).
					Render(" ✓" + name + " ")
			} else if sel && !isCursor {
				// Selected, no cursor: success color with check
				dStr = lipgloss.NewStyle().
					Foreground(t.SuccessColor).
					Bold(true).
					Render(" ✓" + name + " ")
			} else if !sel && isCursor {
				// Unselected + cursor: accent outline style
				dStr = lipgloss.NewStyle().
					Foreground(t.CanvasBg).
					Background(t.Muted).
					Render(" ·" + strings.ToLower(name) + " ")
			} else {
				// Unselected, no cursor: dim
				dStr = lipgloss.NewStyle().
					Foreground(t.Muted).
					Render("  " + strings.ToLower(name) + " ")
			}
			dayStrs = append(dayStrs, dStr)
		}

		daysRow := strings.Join(dayStrs, "")
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), daysRow)
	}

	priorityValStr := viewmodel.PriorityOptions[f.PriorityIdx]
	spValStr := fmt.Sprintf("%d", viewmodel.SPOptions[f.SPIdx])
	typeValStr := viewmodel.TaskTypeOptions[f.TaskTypeIdx]

	fieldNum := 1
	nextFieldNum := func() string {
		num := strconv.Itoa(fieldNum)
		fieldNum++
		return num
	}

	fields = append(fields, renderField(nextFieldNum(), "Title", f.TitleInput.View(), 0))
	fields = append(fields, renderField(nextFieldNum(), "Description", f.DescInput.View(), 1))
	fields = append(fields, renderDropdown(nextFieldNum(), "Priority", priorityValStr, 2))
	if f.TaskTypeIdx != 2 && f.TaskTypeIdx != 3 && f.TaskTypeIdx != 4 {
		fields = append(fields, renderDropdown(nextFieldNum(), "Story Points", spValStr, 3))
	}
	fields = append(fields, renderDropdown(nextFieldNum(), "Type", typeValStr, 4))
	if f.TaskTypeIdx == 0 {
		fields = append(fields, renderField(nextFieldNum(), "Start Time", f.StartTimeInput.View(), 5))
		fields = append(fields, renderField(nextFieldNum(), "Duration (min)", f.DurationInput.View(), 6))
	} else if f.TaskTypeIdx == 2 {
		fields = append(fields, renderField(nextFieldNum(), "Due Date", f.DueDateInput.View(), 5))
		fields = append(fields, renderField(nextFieldNum(), "Due Time", f.StartTimeInput.View(), 6))
	} else if f.TaskTypeIdx == 4 {
		fields = append(fields, renderField(nextFieldNum(), "Start Time", f.StartTimeInput.View(), 5))
		fields = append(fields, renderField(nextFieldNum(), "Duration (min)", f.DurationInput.View(), 6))
		fields = append(fields, renderField(nextFieldNum(), "Location", f.LocationInput.View(), 7))
		if strings.TrimSpace(f.LocationInput.Value()) != "" {
			fields = append(fields, renderField(nextFieldNum(), "Commute buffer (m)", f.CommuteInput.View(), 8))
		}
	}

	if !f.IsEditing {
		if f.TaskTypeIdx == 3 {
			// Habit is always recurring
			fields = append(fields, renderField(nextFieldNum(), "End Date", f.RecurringEndDateInput.View(), 12))
			fields = append(fields, renderDaysSelect(nextFieldNum(), "Recurring Days", 13))
		} else if f.TaskTypeIdx == 0 || f.TaskTypeIdx == 1 {
			recOptStr := "No"
			if f.IsRecurringIdx == 1 {
				recOptStr = "Yes"
			}
			fields = append(fields, renderDropdown(nextFieldNum(), "Is Recurring", recOptStr, 11))
			if f.IsRecurringIdx == 1 {
				fields = append(fields, renderField(nextFieldNum(), "End Date", f.RecurringEndDateInput.View(), 12))
				fields = append(fields, renderDaysSelect(nextFieldNum(), "Recurring Days", 13))
			}
		}
	}

	fields = append(fields, renderField(nextFieldNum(), "Tags (csv)", f.TagsInput.View(), 9))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	submitFg := t.Muted
	submitText := "  Submit  "
	if f.ActiveField == 10 {
		submitFg = t.SuccessColor
		submitText = "[ Submit ]"
	}
	fields = append(fields, "  "+lipgloss.NewStyle().Foreground(submitFg).Bold(true).Render(submitText))

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}

func RenderWorkspaceFormModal(m *viewmodel.Model, t theme.Theme) string {
	f := m.WorkspaceForm
	const innerW = 52

	var fields []string
	headerText := "Create Workspace"
	if m.IsEditingWorkspace {
		headerText = "Edit Workspace"
	}
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(headerText)
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	renderField := func(num, label string, input string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), input)
	}

	fields = append(fields, renderField("1", "Name", f.NameInput.View(), 0))
	fields = append(fields, renderField("2", "Icon", f.IconInput.View(), 1))
	fields = append(fields, renderField("3", "Badge", f.BadgeInput.View(), 2))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	submitFg := t.Muted
	submitText := "  Submit  "
	if f.ActiveField == 3 {
		submitFg = t.SuccessColor
		submitText = "[ Submit ]"
	}
	submitBtn := lipgloss.NewStyle().
		Foreground(submitFg).
		Bold(true).
		Render(submitText)
	fields = append(fields, "  "+submitBtn)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}

func RenderWorkspacePickerModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var lines []string

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Switch Workspace")
	lines = append(lines, title)
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	for i, ws := range m.Workspaces {
		isSelected := i == m.WorkspacePickerIdx
		isActive := ws.UUID == m.ActiveWorkspaceUUID

		pointer := "  "
		if isSelected {
			pointer = lipgloss.NewStyle().Foreground(t.Accent).Render("› ")
		}

		var badgeStr string
		if ws.Badge != "" {
			badgeStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" [%s]", ws.Badge))
		}

		wsText := fmt.Sprintf("%s %s%s", ws.Icon, ws.Name, badgeStr)

		activeMarker := ""
		if isActive {
			activeMarker = lipgloss.NewStyle().Foreground(t.SuccessColor).Render(" ✓")
		}

		var row string
		if isSelected {
			row = pointer + lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(wsText) + activeMarker
		} else {
			row = pointer + lipgloss.NewStyle().Foreground(t.Fg).Render(wsText) + activeMarker
		}

		lines = append(lines, "  "+row)
	}

	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("↑↓ navigate  ↵ switch  esc cancel")
	lines = append(lines, "  "+hint)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderProfileFormModal(m *viewmodel.Model, t theme.Theme) string {
	f := m.ProfileForm
	const innerW = 54

	var fields []string
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Profile & Security Settings")
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	renderField := func(num, label string, input string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %s  %-20s %s", numStyle, lblStyle.Render(label), input)
	}

	fields = append(fields, renderField("1", "Display Username", f.UsernameInput.View(), 0))
	fields = append(fields, renderField("2", "Password/Key", f.PasswordInput.View(), 1))
	fields = append(fields, renderField("3", "Lock Timeout (Mins)", f.LockTimeoutInput.View(), 2))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	submitFg := t.Muted
	submitText := "  Submit  "
	if f.ActiveField == 3 {
		submitFg = t.SuccessColor
		submitText = "[ Submit ]"
	}
	submitBtn := lipgloss.NewStyle().
		Foreground(submitFg).
		Bold(true).
		Render(submitText)
	fields = append(fields, "  "+submitBtn)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}

func RenderSyncFormModal(m *viewmodel.Model, t theme.Theme) string {
	f := m.SyncForm
	const innerW = 58

	var fields []string
	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Google Calendar Sync Settings")
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	renderDropdown := func(num, label string, value string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
			valStr := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
			return fmt.Sprintf("  %s  %-22s %s", numStyle, lblStyle.Render(label), valStr)
		}
		valStr := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
		return fmt.Sprintf("  %s  %-22s %s", numStyle, lblStyle.Render(label), valStr)
	}

	renderField := func(num, label string, input string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %s  %-22s %s", numStyle, lblStyle.Render(label), input)
	}

	modeVal := viewmodel.SyncModeOptions[f.ModeIdx]
	fields = append(fields, renderDropdown("1", "Sync Mode", modeVal, 0))
	fields = append(fields, renderField("2", "Auto-Sync Interval", f.IntervalInput.View()+" sec", 1))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(t.Muted).Render("  Offline-first: sync failures never block the UI."))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	submitFg := t.Muted
	submitText := "  Submit  "
	if f.ActiveField == 2 {
		submitFg = t.SuccessColor
		submitText = "[ Submit ]"
	}
	submitBtn := lipgloss.NewStyle().
		Foreground(submitFg).
		Bold(true).
		Render(submitText)
	fields = append(fields, "  "+submitBtn)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}
