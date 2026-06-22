package modals

import (
	"fmt"
	"strconv"
	"strings"

	"stream/internal/viewmodel"
	"stream/internal/view/components"
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

	priorityValStr := viewmodel.PriorityOptions[f.PriorityIdx]
	spValStr := fmt.Sprintf("%d", viewmodel.SPOptions[f.SPIdx])
	typeValStr := viewmodel.TaskTypeOptions[f.TaskTypeIdx]

	fieldNum := 1
	nextFieldNum := func() string {
		num := strconv.Itoa(fieldNum)
		fieldNum++
		return num
	}

	fields = append(fields, components.RenderFormField(nextFieldNum(), "Title", f.TitleInput.View(), f.ActiveField == 0, t))
	fields = append(fields, components.RenderFormField(nextFieldNum(), "Description", f.DescInput.View(), f.ActiveField == 1, t))
	fields = append(fields, components.RenderFormDropdown(nextFieldNum(), "Priority", priorityValStr, f.ActiveField == 2, t))
	if f.TaskTypeIdx == 0 {
		fields = append(fields, components.RenderFormDropdown(nextFieldNum(), "Story Points", spValStr, f.ActiveField == 3, t))
	}
	fields = append(fields, components.RenderFormDropdown(nextFieldNum(), "Type", typeValStr, f.ActiveField == 4, t))
	if f.TaskTypeIdx == 0 {
		ancOptStr := "No"
		if f.IsAnchoredIdx == 1 {
			ancOptStr = "Yes"
		}
		fields = append(fields, components.RenderFormDropdown(nextFieldNum(), "Is Anchored", ancOptStr, f.ActiveField == 16, t))
	}

	if f.TaskTypeIdx == 0 {
		if f.IsAnchoredIdx == 1 {
			fields = append(fields, components.RenderFormField(nextFieldNum(), "Start Time", f.StartTimeInput.View(), f.ActiveField == 5, t))
			fields = append(fields, components.RenderFormField(nextFieldNum(), "Duration (min)", f.DurationInput.View(), f.ActiveField == 6, t))
		} else {
			fields = append(fields, components.RenderFormField(nextFieldNum(), "Est. Duration (min)", f.DurationInput.View(), f.ActiveField == 6, t))
		}
	} else if f.TaskTypeIdx == 1 {
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Due Date", f.DueDateInput.View(), f.ActiveField == 5, t))
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Due Time", f.StartTimeInput.View(), f.ActiveField == 6, t))
	} else if f.TaskTypeIdx == 2 {
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Start Time", f.StartTimeInput.View(), f.ActiveField == 5, t))
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Duration (min)", f.DurationInput.View(), f.ActiveField == 6, t))
	} else if f.TaskTypeIdx == 3 {
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Start Date", f.StartDateInput.View(), f.ActiveField == 14, t))
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Start Time", f.StartTimeInput.View(), f.ActiveField == 5, t))
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Duration (min)", f.DurationInput.View(), f.ActiveField == 6, t))
		fields = append(fields, components.RenderFormField(nextFieldNum(), "Location", f.LocationInput.View(), f.ActiveField == 7, t))
		if strings.TrimSpace(f.LocationInput.Value()) != "" {
			fields = append(fields, components.RenderFormField(nextFieldNum(), "Commute buffer (m)", f.CommuteInput.View(), f.ActiveField == 8, t))
		}
	}

	if f.TaskTypeIdx == 2 {
		// Habit is always recurring
		fields = append(fields, components.RenderFormField(nextFieldNum(), "End Date", f.RecurringEndDateInput.View(), f.ActiveField == 12, t))
		fields = append(fields, components.RenderDaysSelect(nextFieldNum(), "Recurring Days", f.RecurringDaysSelected[:], f.RecurringDaysSubIdx, f.ActiveField == 13, t))
	} else if f.TaskTypeIdx == 0 || f.TaskTypeIdx == 1 || f.TaskTypeIdx == 3 {
		recOptStr := "No"
		if f.IsRecurringIdx == 1 {
			recOptStr = "Yes"
		}
		fields = append(fields, components.RenderFormDropdown(nextFieldNum(), "Is Recurring", recOptStr, f.ActiveField == 11, t))
		if f.IsRecurringIdx == 1 {
			fields = append(fields, components.RenderFormField(nextFieldNum(), "End Date", f.RecurringEndDateInput.View(), f.ActiveField == 12, t))
			fields = append(fields, components.RenderDaysSelect(nextFieldNum(), "Recurring Days", f.RecurringDaysSelected[:], f.RecurringDaysSubIdx, f.ActiveField == 13, t))
		}
	}

	tagsView := f.TagsInput.View()
	if f.ActiveField == 9 {
		if sug := m.GetTagsAutocompleteSuggestion(); sug != "" {
			tagsView += lipgloss.NewStyle().Foreground(t.Muted).Render(sug)
		}
	}
	fields = append(fields, components.RenderFormField(nextFieldNum(), "Tags (csv)", tagsView, f.ActiveField == 9, t))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	fields = append(fields, "  "+components.RenderFormSubmitButton("Submit", f.ActiveField == 10, t))

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

	fields = append(fields, components.RenderFormField("1", "Name", f.NameInput.View(), f.ActiveField == 0, t))
	fields = append(fields, components.RenderFormField("2", "Icon", f.IconInput.View(), f.ActiveField == 1, t))
	fields = append(fields, components.RenderFormField("3", "Badge", f.BadgeInput.View(), f.ActiveField == 2, t))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	fields = append(fields, "  "+components.RenderFormSubmitButton("Submit", f.ActiveField == 3, t))

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

	fields = append(fields, components.RenderFormFieldWide("1", "Display Username", f.UsernameInput.View(), 20, f.ActiveField == 0, t))
	fields = append(fields, components.RenderFormFieldWide("2", "Password/Key", f.PasswordInput.View(), 20, f.ActiveField == 1, t))
	fields = append(fields, components.RenderFormFieldWide("3", "Lock Timeout (Mins)", f.LockTimeoutInput.View(), 20, f.ActiveField == 2, t))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	fields = append(fields, "  "+components.RenderFormSubmitButton("Submit", f.ActiveField == 3, t))

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

	modeVal := viewmodel.SyncModeOptions[f.ModeIdx]
	fields = append(fields, components.RenderFormDropdownWide("1", "Sync Mode", modeVal, 22, f.ActiveField == 0, t))
	fields = append(fields, components.RenderFormFieldWide("2", "Auto-Sync Interval", f.IntervalInput.View()+" sec", 22, f.ActiveField == 1, t))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(t.Muted).Render("  Offline-first: sync failures never block the UI."))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	fields = append(fields, "  "+components.RenderFormSubmitButton("Submit", f.ActiveField == 2, t))

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}
