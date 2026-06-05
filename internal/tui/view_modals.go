package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

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

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("ℹ  TASK INSPECTOR\n"))
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", 44)) + "\n\n")

	sb.WriteString(fmt.Sprintf("  Title:        %s\n", lipgloss.NewStyle().Bold(true).Render(sentenceCase(t.Title))))
	sb.WriteString(fmt.Sprintf("  Priority:     %s      •  Story Points:  %d SP\n", t.Priority, t.StoryPoints))
	sb.WriteString(fmt.Sprintf("  State:        %s  •  Schedule:      %s\n\n", t.LifecycleState, t.SchedulingType))

	if t.SchedulingType == model.Anchored {
		sb.WriteString(fmt.Sprintf("  Start Time:   %s\n", t.TimeWindow.Start.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("  End Time:     %s\n\n", t.TimeWindow.End.Format("15:04")))
	}

	sb.WriteString("  DESCRIPTION:\n")
	desc := t.Description
	if desc == "" {
		desc = "(No description provided)"
	}
	wrappedDesc := wrapText(desc, 40)
	sb.WriteString(lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(indentText(wrappedDesc, "    ")) + "\n\n")

	sb.WriteString("  EXECUTION METRICS:\n")
	sb.WriteString(fmt.Sprintf("   ● Focus Logged:    %v\n", time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds)*time.Second))
	sb.WriteString(fmt.Sprintf("   ● Pomodoros:       %d/%d\n", t.ExecutionMetrics.TotalCompletedPomodoros, t.ExecutionMetrics.TargetPomodoros))
	sb.WriteString(fmt.Sprintf("   ● Interruptions:   %d\n", t.ExecutionMetrics.InterruptionCount))

	sb.WriteString("\n  [z] Start Focus   [x] Complete   [d] Delete   [Esc/Enter] Close")

	return m.Theme.ModalStyle.
		Width(48).
		Render(sb.String())
}

func (m Model) renderFormModal() string {
	f := m.Form

	var fields []string
	fields = append(fields, "  CREATE WORK ITEM TASK\n  ─────────────────────\n")

	renderField := func(label string, input string, index int) string {
		style := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if f.ActiveField == index {
			style = style.Foreground(m.Theme.Accent).Bold(true)
		}
		return fmt.Sprintf("  %-15s %s", style.Render(label), input)
	}

	fields = append(fields, renderField("1. Title:", f.TitleInput.View(), 0))
	fields = append(fields, renderField("2. Description:", f.DescInput.View(), 1))
	fields = append(fields, renderField("3. Priority:", f.PriorityInput.View(), 2))
	fields = append(fields, renderField("4. Story Points:", f.SPInput.View(), 3))
	fields = append(fields, renderField("5. Anchored (Y/N):", f.AnchorInput.View(), 4))
	fields = append(fields, renderField("6. Start Time:", f.StartTimeInput.View(), 5))
	fields = append(fields, renderField("7. Duration (m):", f.DurationInput.View(), 6))

	submitText := " [SUBMIT] "
	if f.ActiveField == 7 {
		submitText = lipgloss.NewStyle().Background(m.Theme.SuccessColor).Foreground(m.Theme.CanvasBg).Bold(true).Render(submitText)
	} else {
		submitText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Render(submitText)
	}
	fields = append(fields, "\n  "+submitText)

	return m.Theme.ModalStyle.
		Width(48).
		Render(strings.Join(fields, "\n"))
}

func (m Model) renderCommandPalette() string {
	var sb strings.Builder
	sb.WriteString(m.CommandInput.View() + "\n")
	sb.WriteString("  ────────────────────────────────────────────\n")

	val := strings.ToLower(m.CommandInput.Value())
	cmds := []string{"create", "todo", "complete", "delete", "sync", "auth", "dashboard", "month", "week", "day", "analytics", "quit"}

	count := 0
	for _, c := range cmds {
		if strings.Contains(c, val) {
			bullet := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("❯")
			sb.WriteString(fmt.Sprintf("  %s %-12s  %s\n", bullet, c, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("command action")))
			count++
			if count >= 4 {
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
		Padding(1, 2).
		Render(sb.String())
}

func (m Model) renderPromptModal() string {
	var lines []string
	lines = append(lines, "  ⚡ TASK READY FOR FOCUS\n  ──────────────────────\n")
	lines = append(lines, fmt.Sprintf("  Title:    %s", lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(m.PromptTask.Title))))
	lines = append(lines, fmt.Sprintf("  Priority: %s  •  Story Points: %d", m.PromptTask.Priority, m.PromptTask.StoryPoints))
	lines = append(lines, fmt.Sprintf("  Time:     %s - %s", m.PromptTask.TimeWindow.Start.Format("15:04"), m.PromptTask.TimeWindow.End.Format("15:04")))
	lines = append(lines, "\n  [Enter] Start Focus   [s] Snooze 5m   [d/Esc] Dismiss")

	return m.Theme.ModalStyle.
		Width(48).
		BorderForeground(m.Theme.Accent).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderReviewModal() string {
	var lines []string
	lines = append(lines, "  📊 DAILY SHUTDOWN REVIEW\n  ────────────────────────\n")
	lines = append(lines, fmt.Sprintf("  Completed Tasks:   %d", m.ReviewTasksCompleted))
	lines = append(lines, fmt.Sprintf("  Deferred Tasks:    %d", m.ReviewTasksDeferred))
	lines = append(lines, fmt.Sprintf("  Total Focus Logged: %v", time.Duration(m.ReviewFocusSeconds)*time.Second))
	lines = append(lines, "\n  Move unfinished scheduled tasks to tomorrow?")
	lines = append(lines, "  [y] Yes, defer them   [n/Esc] No, leave as overdue")

	return m.Theme.ModalStyle.
		Width(48).
		BorderForeground(m.Theme.SuccessColor).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderHelpModal() string {
	var lines []string
	lines = append(lines, "  ▲ S T R E A M   C O M M A N D   R E F E R E N C E\n  ────────────────────────────────────────────────\n")
	lines = append(lines, "  KEYBOARD SHORTCUTS")
	lines = append(lines, "    1 - 5       Switch views (Dashboard, Month, Week, Day, Stats)")
	lines = append(lines, "    Tab / h / l Toggle Focus between Panels (Timeline / Shelf)")
	lines = append(lines, "    j / k       Navigate items or timeline hours")
	lines = append(lines, "    H / L       Navigate days backward / forward")
	lines = append(lines, "    ctrl+d / u  Scroll active pane down / up")
	lines = append(lines, "    i           Open task creation wizard form")
	lines = append(lines, "    x           Complete selected task")
	lines = append(lines, "    d           Delete selected task")
	lines = append(lines, "    z           Start Zen Mode focus session for task")
	lines = append(lines, "    :           Enter Command Palette mode")
	lines = append(lines, "    ?           Toggle this help documentation modal")
	lines = append(lines, "")
	lines = append(lines, "  COMMAND PALETTE (:command)")
	lines = append(lines, "    :create <t> Anchor a new task for today at 9:00 AM")
	lines = append(lines, "    :todo <t>   Add a floating task to the Backlog Shelf")
	lines = append(lines, "    :complete   Complete active task")
	lines = append(lines, "    :delete     Delete active task")
	lines = append(lines, "    :sync       Force Google Calendar sync")
	lines = append(lines, "    :review     Open Daily Shutdown Review checklist")
	lines = append(lines, "    :quit       Exit the stream application")
	lines = append(lines, "\n  Press [Esc / Enter / ?] to dismiss this help window")

	return m.Theme.ModalStyle.
		Width(54).
		BorderForeground(m.Theme.Accent).
		Render(strings.Join(lines, "\n"))
}
