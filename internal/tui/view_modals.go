package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// modalSep returns a styled horizontal rule for use inside modals.
func (m Model) modalSep(w int) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", w))
}

func (m Model) renderDetailPanel(height int) string {
	t := m.DetailTask

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(strings.ToUpper(t.Title)) + "\n")
	sb.WriteString(strings.Repeat("─", 32) + "\n\n")

	sb.WriteString(fmt.Sprintf("Priority      %s\n", t.Priority))
	sb.WriteString(fmt.Sprintf("Story Points  %d\n", t.StoryPoints))
	sb.WriteString(fmt.Sprintf("Lifecycle     %s\n", t.LifecycleState))
	sb.WriteString(fmt.Sprintf("Schedule      %s\n\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("Start Time    %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("End Time      %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("DESCRIPTION\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(desc) + "\n\n")

	sb.WriteString("EXECUTION METRICS\n")
	sb.WriteString(fmt.Sprintf(" • Focus Logged:    %s\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf(" • Pomodoros:       %d/%d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf(" • Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))

	return lipgloss.NewStyle().
		Background(m.Theme.SelectedBg).
		Foreground(m.Theme.Fg).
		Padding(1, 2).
		Height(height - 2).
		Render(sb.String())
}

func (m Model) renderDetailModal() string {
	t := m.DetailTask
	const innerW = 46

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Task Inspector") + "\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	titleStr := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(sentenceCase(t.Title))
	sb.WriteString(fmt.Sprintf("  %s\n\n", titleStr))

	pColor := m.priorityColor(t.Priority)
	pBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render(fmt.Sprintf("▲ %s", t.Priority))
	sb.WriteString(fmt.Sprintf("  %s  •  %d SP  •  %s\n", pBadge, t.StoryPoints, t.LifecycleState))
	sb.WriteString(fmt.Sprintf("  Schedule: %s\n", t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s  →  %s\n",
			t.TimeWindow.Start.Format("Mon Jan 2  15:04"),
			t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	desc := t.Description
	if desc == "" {
		desc = "(no description)"
	}
	wrapped := wrapText(desc, innerW-2)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(indentText(wrapped, "  ")) + "\n\n")
	sb.WriteString(m.modalSep(innerW) + "\n\n")

	sb.WriteString(fmt.Sprintf("  Focus logged   %v\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf("  Pomodoros      %d / %d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf("  Interruptions  %d\n", t.ExecutionMetrics.InterruptionCount))

	sb.WriteString("\n")
	sb.WriteString(m.modalSep(innerW) + "\n")
	hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("z focus  x complete  d delete  Esc close")
	sb.WriteString("  " + hint)

	return m.Theme.ModalStyle.Width(innerW + 4).Render(sb.String())
}

func (m Model) renderFormModal() string {
	f := m.Form
	const innerW = 52

	var fields []string
	title := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("Create Task")
	fields = append(fields, title)
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	renderField := func(num, label string, input string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(m.Theme.Accent).Bold(true)
		}
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), input)
	}

	fields = append(fields, renderField("1", "Title", f.TitleInput.View(), 0))
	fields = append(fields, renderField("2", "Description", f.DescInput.View(), 1))
	fields = append(fields, renderField("3", "Priority", f.PriorityInput.View(), 2))
	fields = append(fields, renderField("4", "Story Points", f.SPInput.View(), 3))
	fields = append(fields, renderField("5", "Anchored (Y/N)", f.AnchorInput.View(), 4))
	fields = append(fields, renderField("6", "Start Time", f.StartTimeInput.View(), 5))
	fields = append(fields, renderField("7", "Duration (min)", f.DurationInput.View(), 6))
	fields = append(fields, "")
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	submitBg := m.Theme.PanelBg
	submitFg := m.Theme.SuccessColor
	if f.ActiveField == 7 {
		submitBg = m.Theme.SuccessColor
		submitFg = m.Theme.CanvasBg
	}
	submitBtn := lipgloss.NewStyle().
		Background(submitBg).
		Foreground(submitFg).
		Bold(true).
		Padding(0, 2).
		Render("Submit")
	fields = append(fields, "  "+submitBtn)

	return m.Theme.ModalStyle.Width(innerW + 4).Render(strings.Join(fields, "\n"))
}

func (m Model) renderCommandPalette() string {
	var sb strings.Builder
	sb.WriteString(m.CommandInput.View() + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", m.Width-8)) + "\n")

	val := strings.ToLower(m.CommandInput.Value())
	type cmdEntry struct {
		name string
		desc string
	}
	cmds := []cmdEntry{
		{"create", "anchor a new task for today at 9:00 AM"},
		{"todo", "add a floating task to the backlog shelf"},
		{"complete", "complete the selected task"},
		{"delete", "delete the selected task"},
		{"sync", "force Google Calendar sync"},
		{"auth", "authenticate with Google Calendar"},
		{"review", "open daily shutdown review"},
		{"dashboard", "switch to dashboard view"},
		{"month", "switch to month grid view"},
		{"week", "switch to week lanes view"},
		{"day", "switch to day timeline view"},
		{"analytics", "switch to analytics view"},
		{"quit", "exit stream"},
	}

	count := 0
	for _, c := range cmds {
		if strings.Contains(c.name, val) {
			bullet := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("❯")
			nameStr := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(c.name)
			descStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(c.desc)
			sb.WriteString(fmt.Sprintf("  %s %-12s  %s\n", bullet, nameStr, descStr))
			count++
			if count >= 5 {
				break
			}
		}
	}

	return lipgloss.NewStyle().
		Background(m.Theme.ModalBg).
		Foreground(m.Theme.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Width(m.Width - 4).
		Padding(0, 2).
		Render(sb.String())
}

func (m Model) renderPromptModal() string {
	const innerW = 46
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.P1Color).Bold(true).Render("Task Ready for Focus"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s", lipgloss.NewStyle().Bold(true).Render(sentenceCase(m.PromptTask.Title))))
	lines = append(lines, fmt.Sprintf("  %s  •  %d SP", m.PromptTask.Priority, m.PromptTask.StoryPoints))
	lines = append(lines, fmt.Sprintf("  %s → %s",
		m.PromptTask.TimeWindow.Start.Format("15:04"),
		m.PromptTask.TimeWindow.End.Format("15:04")))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("Enter start  s snooze 5m  d dismiss")
	lines = append(lines, "  "+hint)

	return m.Theme.ModalStyle.Width(innerW + 4).Render(strings.Join(lines, "\n"))
}

func (m Model) renderReviewModal() string {
	const innerW = 46
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("Daily Shutdown Review"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Completed tasks   %d", m.ReviewTasksCompleted))
	lines = append(lines, fmt.Sprintf("  Deferred tasks    %d", m.ReviewTasksDeferred))
	lines = append(lines, fmt.Sprintf("  Focus logged      %v", time.Duration(m.ReviewFocusSeconds)*time.Second))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, "  Move unfinished anchored tasks to tomorrow?")
	lines = append(lines, "")
	yesStr := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("y")
	noStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("n / Esc")
	lines = append(lines, fmt.Sprintf("  [%s] defer them   [%s] leave as overdue", yesStr, noStr))

	return m.Theme.ModalStyle.Width(innerW + 4).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHelpModal() string {
	const innerW = 56
	accent := lipgloss.NewStyle().Foreground(m.Theme.Accent)
	muted := lipgloss.NewStyle().Foreground(m.Theme.Muted)
	bold := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true)

	section := func(s string) string {
		return "\n" + muted.Bold(true).Render(s)
	}
	key := func(k, desc string) string {
		return fmt.Sprintf("  %s  %s",
			accent.Render(fmt.Sprintf("%-16s", k)),
			muted.Render(desc))
	}

	var lines []string
	lines = append(lines, bold.Render("▲ stream")+"  "+muted.Render("command reference"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, section("NAVIGATION"))
	lines = append(lines, key("1 – 5", "Switch views"))
	lines = append(lines, key("j / k", "Navigate items / timeline hours"))
	lines = append(lines, key("H / L", "Day backward / forward"))
	lines = append(lines, key("Tab", "Toggle timeline ↔ backlog shelf"))
	lines = append(lines, key("ctrl+d / ctrl+u", "Scroll pane down / up"))
	lines = append(lines, section("TASK ACTIONS"))
	lines = append(lines, key("i", "Create new task"))
	lines = append(lines, key("x", "Complete selected task"))
	lines = append(lines, key("d", "Delete selected task"))
	lines = append(lines, key("z", "Start Zen focus session"))
	lines = append(lines, key("Enter", "Inspect task details"))
	lines = append(lines, section("WORKSPACE"))
	lines = append(lines, key(":", "Open command palette"))
	lines = append(lines, key("?", "Toggle this help modal"))
	lines = append(lines, key(":sync", "Force Google Calendar sync"))
	lines = append(lines, key(":review", "Open shutdown review"))
	lines = append(lines, key(":quit", "Exit stream"))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, muted.Render("  Esc / Enter / ? to close"))

	return m.Theme.ModalStyle.Width(innerW + 4).Render(strings.Join(lines, "\n"))
}
