package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderPromptModal() string {
	const innerW = 46
	var lines []string

	title := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Prompt Modal")
	lines = append(lines, title)
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s", lipgloss.NewStyle().Bold(true).Render(sentenceCase(m.PromptTask.Title))))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}

func (m Model) renderReviewModal() string {
	const innerW = 46
	var lines []string

	title := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Shutdown Review")
	lines = append(lines, title)
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}

func (m Model) renderConfirmModal() string {
	const innerW = 46
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("Confirm Delete"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, "  Are you sure you want to delete task")
	lines = append(lines, fmt.Sprintf("  \"%s\"?", sentenceCase(m.ConfirmTask.Title)))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")

	yesBtn := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("[Y] Yes, Delete")
	noBtn := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("[N] No, Cancel")
	lines = append(lines, fmt.Sprintf("  %s      %s", yesBtn, noBtn))

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}

func (m Model) renderAnchorPromptModal() string {
	const innerW = 46
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Anchor Task to Timeline"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Task:  %s", lipgloss.NewStyle().Bold(true).Render(sentenceCase(m.AnchorPromptTask.Title))))
	lines = append(lines, fmt.Sprintf("  Est:   %d SP (%d mins)", m.AnchorPromptTask.StoryPoints, m.AnchorPromptTask.StoryPoints*45))
	lines = append(lines, "")

	renderField := func(label string, view string, isActive bool) string {
		lblStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if isActive {
			lblStyle = lblStyle.Foreground(m.Theme.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-16s %s", lblStyle.Render(label), view)
	}

	lines = append(lines, renderField("Start Time", m.AnchorTimeInput.View(), m.AnchorActiveField == 0))
	lines = append(lines, renderField("Duration (min)", m.AnchorDurationInput.View(), m.AnchorActiveField == 1))

	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Tab switch  Enter confirm  Esc cancel")
	lines = append(lines, "  "+hint)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}

func (m Model) renderLockScreen() string {
	const innerW = 44
	var fields []string

	title := lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("🔒 STREAM SESSION LOCKED")
	fields = append(fields, title)
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(m.Theme.Fg).Render("This terminal session is protected by a password."))
	fields = append(fields, "")

	inputStr := m.LockPasswordInput.View()
	fields = append(fields, "  "+inputStr)
	fields = append(fields, "")
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	statusStr := "Press Enter to unlock"
	statusColor := m.Theme.Muted
	if strings.Contains(m.StatusMsg, "Incorrect") || strings.Contains(m.StatusMsg, "❌") {
		statusStr = m.StatusMsg
		statusColor = m.Theme.P0Color
	}
	fields = append(fields, lipgloss.NewStyle().Foreground(statusColor).Render(statusStr))

	modalBox := m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(fields, "\n"), innerW))

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

func (m Model) renderSessionExpiryModal() string {
	const innerW = 46
	var fields []string

	title := lipgloss.NewStyle().Foreground(m.Theme.FocusPurple).Bold(true).Render("⚠️  SESSION EXPIRES IN 1 MINUTE")
	fields = append(fields, title)
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")
	fields = append(fields, lipgloss.NewStyle().Foreground(m.Theme.Fg).Render("Your session is about to expire."))
	fields = append(fields, lipgloss.NewStyle().Foreground(m.Theme.Fg).Render("Would you like to extend your session?"))
	fields = append(fields, "")
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	yesBtn := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("[Y] Yes, Reset Timer")
	noBtn := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("[N] No, Allow Lock")
	fields = append(fields, fmt.Sprintf("  %s      %s", yesBtn, noBtn))

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(fields, "\n"), innerW))
}
