package modals

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/view/components"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

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
	if m.PromptTask.SchedulingType != model.Event {
		restTime := viewmodel.CalculateTaskRestTime(m.PromptTask)
		if restTime > 0 {
			restInfo := fmt.Sprintf("+%d mins Rest", int(restTime.Minutes()))
			lines = append(lines, fmt.Sprintf("  Rest:      %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render(restInfo)))
		}
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
	var bodyLines []string

	bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("  Auth link copied to clipboard!"))
	bodyLines = append(bodyLines, "")

	wrappedMsg := theme.WrapText(m.AuthNoticeMsg, innerW-4)
	for _, line := range strings.Split(wrappedMsg, "\n") {
		bodyLines = append(bodyLines, "  "+line)
	}
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Fg).Render("  Complete sign-in in your browser."))
	bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Fg).Render("  The dialog closes automatically on success."))

	closeBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[Esc/Enter/Q] Close and return to normal mode")

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "🔗 GOOGLE CALENDAR AUTH",
		BodyLines:  bodyLines,
		FooterText: closeBtn,
		InnerWidth: innerW,
		Theme:      t,
	})
}

func RenderWarningModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var bodyLines []string

	wrappedMsg := theme.WrapText(m.WarningMsg, innerW-4)
	for _, line := range strings.Split(wrappedMsg, "\n") {
		bodyLines = append(bodyLines, "  "+line)
	}

	okBtn := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("[Enter/Esc/Space] Close")

	titleStyle := lipgloss.NewStyle().Foreground(t.P0Color).Bold(true)

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "⚠️  VALIDATION ERROR",
		TitleStyle: &titleStyle,
		BodyLines:  bodyLines,
		FooterText: okBtn,
		InnerWidth: innerW,
		Theme:      t,
	})
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
	switch m.ConfirmActionType {
	case "complete_reminder":
		return components.RenderBaseConfirmModal(
			"Complete Reminder",
			[]string{
				"Are you sure you want to complete",
				"and remove this reminder task",
				fmt.Sprintf("\"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)),
			},
			[]string{"Yes, Complete", "No, Cancel"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "deanchor":
		return components.RenderBaseConfirmModal(
			"De-anchor Task",
			[]string{
				"Are you sure you want to de-anchor",
				"and return this task to the backlog?",
				fmt.Sprintf("\"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)),
			},
			[]string{"Yes, De-anchor", "No, Cancel"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "delete_recurring":
		return components.RenderBaseConfirmModal(
			"♻️  DELETE RECURRING TASK",
			[]string{
				"This is a recurring task/habit:",
				"  " + theme.SentenceCase(m.ConfirmTask.Title),
				"Choose deletion option:",
			},
			[]string{"Only this occurrence", "This and all remaining occurrences"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "edit_recurring":
		return components.RenderBaseConfirmModal(
			"♻️  EDIT RECURRING TASK",
			[]string{
				"This is a recurring task/habit:",
				"  " + theme.SentenceCase(m.ConfirmTask.Title),
				"Choose update option:",
			},
			[]string{"Only this occurrence", "This and all remaining occurrences"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "exit_focus":
		return components.RenderBaseConfirmModal(
			"Focus Session Active",
			[]string{
				"An active focus session is running for:",
				"  " + theme.SentenceCase(m.ConfirmTask.Title),
				"Choose an option:",
			},
			[]string{"1. Mark as complete", "2. Complete and resume", "3. Discard session changes"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "log_session_confirm":
		return components.RenderBaseConfirmModal(
			"Log Focus Session?",
			[]string{
				"Would you like to log the focus time spent",
				"on this completed task?",
			},
			[]string{"Yes, Log Time", "No, Just Complete"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	case "start_late_confirm":
		lateDur := time.Now().Sub(m.ConfirmTask.TimeWindow.Start)
		return components.RenderBaseConfirmModal(
			"Start Late - Adjust Timer?",
			[]string{
				fmt.Sprintf("You are starting late by %s.", formatDuration(lateDur)),
				"Would you like to trim the timer to match the",
				"remaining scheduled time, or start with the",
				"full planned duration?",
			},
			[]string{"Start with Full Duration", "Trim to Current Time"},
			m.ConfirmSelectedIndex,
			-1,
			m.ConfirmFocusArea,
			t,
		)
	default: // delete
		return components.RenderBaseConfirmModal(
			"Confirm Delete",
			[]string{
				"Are you sure you want to delete task",
				fmt.Sprintf("\"%s\"?", theme.SentenceCase(m.ConfirmTask.Title)),
			},
			[]string{"Yes, Delete", "No, Cancel"},
			m.ConfirmSelectedIndex,
			0, // Option 0 is destructive (Delete)
			m.ConfirmFocusArea,
			t,
		)
	}
}

func RenderAnchorPromptModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var bodyLines []string

	bodyLines = append(bodyLines, fmt.Sprintf("  Task:  %s", lipgloss.NewStyle().Bold(true).Render(theme.SentenceCase(m.AnchorPromptTask.Title))))
	bodyLines = append(bodyLines, fmt.Sprintf("  Est:   %d SP (%d mins)", m.AnchorPromptTask.StoryPoints, m.AnchorPromptTask.StoryPoints*45))
	bodyLines = append(bodyLines, "")

	renderField := func(label string, view string, isActive bool) string {
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if isActive {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-16s %s", lblStyle.Render(label), view)
	}

	bodyLines = append(bodyLines, renderField("Start Time", m.AnchorTimeInput.View(), m.AnchorActiveField == 0))
	bodyLines = append(bodyLines, renderField("Duration (min)", m.AnchorDurationInput.View(), m.AnchorActiveField == 1))

	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("Tab switch  Enter confirm  Esc cancel")

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "Anchor Task to Timeline",
		BodyLines:  bodyLines,
		FooterText: hint,
		InnerWidth: innerW,
		Theme:      t,
	})
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
	var bodyLines []string

	bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Fg).Render("Your session is about to expire."))
	bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Fg).Render("Would you like to extend your session?"))

	yesBtn := lipgloss.NewStyle().Foreground(t.SuccessColor).Bold(true).Render("[Y] Yes, Reset Timer")
	noBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("[N] No, Allow Lock")

	titleStyle := lipgloss.NewStyle().Foreground(t.FocusPurple).Bold(true)

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "⚠️  SESSION EXPIRES IN 1 MINUTE",
		TitleStyle: &titleStyle,
		BodyLines:  bodyLines,
		Buttons:    []string{yesBtn, noBtn},
		InnerWidth: innerW,
		Theme:      t,
	})
}

func RenderUpdatePromptModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 52
	var bodyLines []string

	msg := "A new version of the application is available. Pulling the latest changes is highly recommended to avoid missing features or database out-of-sync bugs."
	wrappedMsg := lipgloss.NewStyle().Width(innerW - 4).Foreground(t.Fg).Render(msg)
	bodyLines = append(bodyLines, wrappedMsg)
	bodyLines = append(bodyLines, "")

	if len(m.UpdateCommits) > 0 {
		bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("CHANGELOG:"))
		maxCommits := 6
		for i, commit := range m.UpdateCommits {
			if i >= maxCommits {
				remaining := len(m.UpdateCommits) - maxCommits
				bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  • ... and %d more commits", remaining)))
				break
			}
			commitStyle := lipgloss.NewStyle().Foreground(t.Fg)
			bodyLines = append(bodyLines, fmt.Sprintf("  • %s", commitStyle.Render(commit)))
		}
		bodyLines = append(bodyLines, "")
	}

	var updateBtn, snoozeBtn string
	if m.UpdatePromptSelectedIdx == 0 {
		updateBtn = lipgloss.NewStyle().
			Background(t.SuccessColor).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).
			Render(" 🚀 Update & Restart ")
	} else {
		updateBtn = lipgloss.NewStyle().
			Foreground(t.SuccessColor).
			Render("  [ Update & Restart ] ")
	}

	if m.UpdatePromptSelectedIdx == 1 {
		snoozeBtn = lipgloss.NewStyle().
			Background(t.Accent).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).
			Render(" ⏳ Snooze 1 Hour ")
	} else {
		snoozeBtn = lipgloss.NewStyle().
			Foreground(t.Muted).
			Render("  [ Snooze 1 Hour ]  ")
	}

	titleStyle := lipgloss.NewStyle().Foreground(t.P1Color).Bold(true)

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "🚀  UPDATE AVAILABLE!",
		TitleStyle: &titleStyle,
		BodyLines:  bodyLines,
		Buttons:    []string{updateBtn, snoozeBtn},
		InnerWidth: innerW,
		Theme:      t,
	})
}

func RenderLogSessionPromptModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 46
	var bodyLines []string

	bodyLines = append(bodyLines, fmt.Sprintf("  Task:  %s", lipgloss.NewStyle().Bold(true).Render(theme.SentenceCase(m.LogSessionPromptTask.Title))))
	bodyLines = append(bodyLines, "")

	renderField := func(label string, view string, isActive bool) string {
		lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if isActive {
			lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-20s %s", lblStyle.Render(label), view)
	}

	bodyLines = append(bodyLines, renderField("Focus duration (min)", m.LogSessionFocusInput.View(), m.LogSessionActiveField == 0))
	bodyLines = append(bodyLines, renderField("Break duration (min)", m.LogSessionBreakInput.View(), m.LogSessionActiveField == 1))

	hint := lipgloss.NewStyle().Foreground(t.Muted).Render("Tab switch  Enter save  Esc cancel")

	return components.RenderBaseModal(components.BaseModalConfig{
		Title:      "Log Focus Session",
		BodyLines:  bodyLines,
		FooterText: hint,
		InnerWidth: innerW,
		Theme:      t,
	})
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if secs == 0 {
		return fmt.Sprintf("%d minutes", mins)
	}
	return fmt.Sprintf("%dm %ds", mins, secs)
}
