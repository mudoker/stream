package modals

import (
	"fmt"
	"strings"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

func RenderPromptModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 50
	var lines []string

	var title string
	if m.PromptTask.SchedulingType == model.Reminder {
		title = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("🔔 TASK REMINDER")
	} else {
		title = lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true).Render("⏰ TIME TO START FOCUS SESSION")
	}

	lines = append(lines, title)
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	// Task Title
	lines = append(lines, "  "+lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(theme.SentenceCase(m.PromptTask.Title)))
	lines = append(lines, "")

	// Task Metadata: Priority and Story Points
	pColor := t.PriorityColor(m.PromptTask.Priority)
	pBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render(fmt.Sprintf("▲ %s", m.PromptTask.Priority))
	spInfo := fmt.Sprintf("%d SP (%d mins)", m.PromptTask.StoryPoints, m.PromptTask.StoryPoints*45)
	lines = append(lines, fmt.Sprintf("  Priority: %s   •   Est: %s", pBadge, spInfo))

	// Scheduled time or due time
	if m.PromptTask.SchedulingType == model.Anchored || m.PromptTask.SchedulingType == model.Event {
		timeInfo := fmt.Sprintf("%s - %s", m.PromptTask.TimeWindow.Start.Format("15:04"), m.PromptTask.TimeWindow.End.Format("15:04"))
		lines = append(lines, fmt.Sprintf("  Scheduled: %s", lipgloss.NewStyle().Foreground(t.Accent).Render(timeInfo)))
		if m.PromptTask.SchedulingType == model.Event {
			if m.PromptTask.Location != "" {
				lines = append(lines, fmt.Sprintf("  Location:  %s", lipgloss.NewStyle().Foreground(t.Fg).Render(m.PromptTask.Location)))
			}
			if m.PromptTask.CommuteBuffer > 0 {
				lines = append(lines, fmt.Sprintf("  Commute:   %d mins", m.PromptTask.CommuteBuffer))
			}
		}
	} else if m.PromptTask.SchedulingType == model.Reminder {
		dueInfo := m.PromptTask.TimeWindow.Start.Format("15:04")
		lines = append(lines, fmt.Sprintf("  Due Time:  %s", lipgloss.NewStyle().Foreground(t.Accent).Render(dueInfo)))
	}

	// Calculate and render rest time
	restTime := viewmodel.CalculateTaskRestTime(m.PromptTask)
	if restTime > 0 {
		restInfo := fmt.Sprintf("+%d mins Rest", int(restTime.Minutes()))
		lines = append(lines, fmt.Sprintf("  Rest:      %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render(restInfo)))
	}

	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	// Action buttons
	enterActionText := "[Enter] Start Focus"
	if m.PromptTask.SchedulingType == model.Reminder {
		enterActionText = "[Enter] Dismiss"
	}

	var enterAction, snoozeAction, dismissAction string

	if m.PromptSelectedIdx == 0 {
		enterAction = lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("▶ " + enterActionText + " ◀")
	} else {
		enterAction = lipgloss.NewStyle().Foreground(t.Fg).Render("  " + enterActionText + "  ")
	}

	if m.PromptSelectedIdx == 1 {
		snoozeAction = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▶ [S] Snooze 5m ◀")
	} else {
		snoozeAction = lipgloss.NewStyle().Foreground(t.Muted).Render("  [S] Snooze 5m  ")
	}

	if m.PromptSelectedIdx == 2 {
		dismissAction = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▶ [D/Esc] Dismiss ◀")
	} else {
		dismissAction = lipgloss.NewStyle().Foreground(t.Muted).Render("  [D/Esc] Dismiss  ")
	}

	lines = append(lines, fmt.Sprintf("  %s   %s   %s",
		enterAction,
		snoozeAction,
		dismissAction,
	))

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderAuthNoticeModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 54
	var lines []string

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("🔗 GOOGLE CALENDAR AUTH")
	lines = append(lines, title)
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("  Auth link copied to clipboard!"))
	lines = append(lines, "")

	wrappedMsg := theme.WrapText(m.AuthNoticeMsg, innerW-4)
	for _, line := range strings.Split(wrappedMsg, "\n") {
		lines = append(lines, "  "+line)
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(t.Fg).Render("  Complete sign-in in your browser."))
	lines = append(lines, lipgloss.NewStyle().Foreground(t.Fg).Render("  The dialog closes automatically on success."))
	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	closeBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[Esc/Enter/Q] Close and return to normal mode")
	lines = append(lines, "  "+closeBtn)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderWarningModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var lines []string

	title := lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("⚠️  VALIDATION ERROR")
	lines = append(lines, title)
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	wrappedMsg := theme.WrapText(m.WarningMsg, innerW-4)
	for _, line := range strings.Split(wrappedMsg, "\n") {
		lines = append(lines, "  "+line)
	}
	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	okBtn := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[Enter/Esc/Space] Close")
	lines = append(lines, "  "+okBtn)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderReviewModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var lines []string

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Shutdown Review")
	lines = append(lines, title)
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderConfirmModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var lines []string

	if m.ConfirmActionType == "complete_reminder" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Complete Reminder"))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
		lines = append(lines, "  Are you sure you want to complete")
		lines = append(lines, "  and remove this reminder task")
		lines = append(lines, fmt.Sprintf("  \"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)))
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")

		yesBtn := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[Y] Yes, Complete")
		noBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[N] No, Cancel")
		lines = append(lines, fmt.Sprintf("  %s      %s", yesBtn, noBtn))
	} else if m.ConfirmActionType == "deanchor" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("De-anchor Task"))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
		lines = append(lines, "  Are you sure you want to de-anchor")
		lines = append(lines, "  and return this task to the backlog?")
		lines = append(lines, fmt.Sprintf("  \"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)))
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")

		yesBtn := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[Enter] Confirm")
		noBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[Any Key] Cancel")
		lines = append(lines, fmt.Sprintf("  %s      %s", yesBtn, noBtn))
	} else if m.ConfirmActionType == "delete_recurring" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("Delete Recurring Task"))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
		lines = append(lines, "  This is a recurring task/habit.")
		lines = append(lines, "  Do you want to delete:")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Render("  [1] Only this occurrence"))
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Render("  [2] This and all remaining occurrences"))
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("  [Esc] Cancel"))
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
	} else if m.ConfirmActionType == "edit_recurring" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Edit Recurring Task"))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
		lines = append(lines, "  This is a recurring task/habit.")
		lines = append(lines, "  Do you want to apply changes to:")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Render("  [1] Only this occurrence"))
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Render("  [2] This and all remaining occurrences"))
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("  [Esc] Cancel"))
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("Confirm Delete"))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
		lines = append(lines, "  Are you sure you want to delete task")
		lines = append(lines, fmt.Sprintf("  \"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)))
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")

		yesBtn := lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("[Y/Enter] Yes, Delete")
		noBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[N/Esc] No, Cancel")
		lines = append(lines, fmt.Sprintf("  %s      %s", yesBtn, noBtn))
	}

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderAnchorPromptModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Anchor Task to Timeline"))
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Task:  %s", lipgloss.NewStyle().Bold(true).Render(theme.SentenceCase(m.AnchorPromptTask.Title))))
	lines = append(lines, fmt.Sprintf("  Est:   %d SP (%d mins)", m.AnchorPromptTask.StoryPoints, m.AnchorPromptTask.StoryPoints*45))
	lines = append(lines, "")

	renderField := func(label string, view string, isActive bool) string {
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if isActive {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-16s %s", lblStyle.Render(label), view)
	}

	lines = append(lines, renderField("Start Time", m.AnchorTimeInput.View(), m.AnchorActiveField == 0))
	lines = append(lines, renderField("Duration (min)", m.AnchorDurationInput.View(), m.AnchorActiveField == 1))

	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("Tab switch  Enter confirm  Esc cancel")
	lines = append(lines, "  "+hint)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

func RenderLockScreen(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 44
	var fields []string

	title := lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("🔒 STREAM SESSION LOCKED")
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(t.Fg).Render("This terminal session is protected by a password."))
	fields = append(fields, "")

	inputStr := m.LockPasswordInput.View()
	fields = append(fields, "  "+inputStr)
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	statusStr := "Press Enter to unlock"
	statusColor := t.Muted
	if strings.Contains(m.StatusMsg, "Incorrect") || strings.Contains(m.StatusMsg, "❌") {
		statusStr = m.StatusMsg
		statusColor = t.P0Color
	}
	fields = append(fields, lipgloss.NewStyle().Foreground(statusColor).Render(statusStr))

	modalBox := t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))

	modalW := lipgloss.Width(modalBox)
	modalH := lipgloss.Height(modalBox)
	topPad := (m.Height - modalH) / 2
	leftPad := (m.Width - modalW) / 2
	if topPad < 0 {
		topPad = 0
	}
	if leftPad < 0 {
		leftPad = 0
	}

	var lines []string
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}
	modalLines := strings.Split(modalBox, "\n")
	leftSpace := strings.Repeat(" ", leftPad)
	for _, ml := range modalLines {
		lines = append(lines, leftSpace+ml)
	}
	for len(lines) < m.Height {
		lines = append(lines, "")
	}

	if len(lines) > m.Height {
		lines = lines[:m.Height]
	}

	return strings.Join(lines, "\n")
}

func RenderSessionExpiryModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var fields []string

	title := lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true).Render("⚠️  SESSION EXPIRES IN 1 MINUTE")
	fields = append(fields, title)
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(t.Fg).Render("Your session is about to expire."))
	fields = append(fields, lipgloss.NewStyle().Foreground(t.Fg).Render("Would you like to extend your session?"))
	fields = append(fields, "")
	fields = append(fields, ModalSep(innerW))
	fields = append(fields, "")

	yesBtn := lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("[Y] Yes, Reset Timer")
	noBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[N] No, Allow Lock")
	fields = append(fields, fmt.Sprintf("  %s      %s", yesBtn, noBtn))

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(fields, "\n"), innerW))
}
