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

	titleStr := sentenceCase(t.Title)
	titleRunes := []rune(titleStr)
	maxTitleW := innerW - 4
	if len(titleRunes) > maxTitleW {
		titleStr = string(titleRunes[:maxTitleW-3]) + "..."
	}
	titleRendered := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(titleStr)
	sb.WriteString(fmt.Sprintf("  %s\n\n", titleRendered))

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
	hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("z focus  x complete  e edit  d delete  Esc close")
	sb.WriteString("  " + hint)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(sb.String(), innerW))
}

func (m Model) renderFormModal() string {
	f := m.Form
	const innerW = 52

	var fields []string
	headerText := "Create Task"
	if m.IsEditing {
		headerText = "Edit Task"
	}
	title := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)
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

	renderDropdown := func(num, label string, value string, index int) string {
		numStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(num)
		lblStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
		if f.ActiveField == index {
			lblStyle = lblStyle.Foreground(m.Theme.Accent).Bold(true)
			valStr := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
			return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
		}
		valStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(fmt.Sprintf("  %s  ", value))
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
	}

	priorityValStr := PriorityOptions[f.PriorityIdx]
	spValStr := fmt.Sprintf("%d", SPOptions[f.SPIdx])
	anchoredValStr := "No"
	if f.IsAnchored {
		anchoredValStr = "Yes"
	}

	fields = append(fields, renderField("1", "Title", f.TitleInput.View(), 0))
	fields = append(fields, renderField("2", "Description", f.DescInput.View(), 1))
	fields = append(fields, renderDropdown("3", "Priority", priorityValStr, 2))
	fields = append(fields, renderDropdown("4", "Story Points", spValStr, 3))
	fields = append(fields, renderDropdown("5", "Anchored", anchoredValStr, 4))
	fields = append(fields, renderField("6", "Start Time", f.StartTimeInput.View(), 5))
	fields = append(fields, renderField("7", "Duration (min)", f.DurationInput.View(), 6))
	fields = append(fields, "")
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	submitFg := m.Theme.Muted
	submitText := "  Submit  "
	if f.ActiveField == 7 {
		submitFg = m.Theme.SuccessColor
		submitText = "[ Submit ]"
	}
	submitBtn := lipgloss.NewStyle().
		Foreground(submitFg).
		Bold(true).
		Render(submitText)
	fields = append(fields, "  "+submitBtn)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(fields, "\n"), innerW))
}

func (m Model) renderWorkspaceFormModal() string {
	f := m.WorkspaceForm
	const innerW = 52

	var fields []string
	headerText := "Create Workspace"
	if m.IsEditingWorkspace {
		headerText = "Edit Workspace"
	}
	title := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)
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

	fields = append(fields, renderField("1", "Name", f.NameInput.View(), 0))
	fields = append(fields, renderField("2", "Icon", f.IconInput.View(), 1))
	fields = append(fields, renderField("3", "Badge", f.BadgeInput.View(), 2))
	fields = append(fields, "")
	fields = append(fields, m.modalSep(innerW))
	fields = append(fields, "")

	submitFg := m.Theme.Muted
	submitText := "  Submit  "
	if f.ActiveField == 3 {
		submitFg = m.Theme.SuccessColor
		submitText = "[ Submit ]"
	}
	submitBtn := lipgloss.NewStyle().
		Foreground(submitFg).
		Bold(true).
		Render(submitText)
	fields = append(fields, "  "+submitBtn)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(fields, "\n"), innerW))
}

func (m Model) renderCommandPalette() string {
	var sb strings.Builder

	// Top/bottom inner padding on the query input area
	inputStyle := lipgloss.NewStyle().Padding(1, 2)
	sb.WriteString(inputStyle.Render(m.CommandInput.View()) + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", m.Width-8)) + "\n\n")

	// Section Tag (40% Opacity, Unbolded)
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("COMMANDS") + "\n\n")

	val := strings.ToLower(m.CommandInput.Value())
	var filtered []CommandEntry
	for _, c := range DefaultCommands {
		if strings.Contains(c.Name, val) {
			filtered = append(filtered, c)
		}
	}

	// Clamp selected index
	selIdx := m.CommandSelectedIndex
	if selIdx < 0 {
		selIdx = 0
	}
	if len(filtered) > 0 && selIdx >= len(filtered) {
		selIdx = len(filtered) - 1
	}

	// Calculate container inner width
	innerW := m.Width - 8

	for idx, c := range filtered {
		isSelected := idx == selIdx

		var line string
		if isSelected {
			indicator := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("┃")
			keywordStyle := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
			descStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true)
			keyword := keywordStyle.Render(fmt.Sprintf("%-10s", c.Name))
			desc := descStyle.Render(c.Desc)

			rowText := fmt.Sprintf("%s  %s  %s", indicator, keyword, desc)
			line = lipgloss.NewStyle().
				Width(innerW).
				Render(rowText)
		} else {
			keywordStyle := lipgloss.NewStyle().Foreground(m.Theme.Fg)
			descStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
			keyword := keywordStyle.Render(fmt.Sprintf("%-10s", c.Name))
			desc := descStyle.Render(c.Desc)

			rowText := fmt.Sprintf("   %s  %s", keyword, desc)
			line = lipgloss.NewStyle().
				Width(innerW).
				Render(rowText)
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37")).Render(strings.Repeat("─", m.Width-8)) + "\n")
	footerTip := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("  💡 Use ↑↓ to navigate • ↵ to execute • esc to close")
	sb.WriteString(footerTip + "\n")

	return lipgloss.NewStyle().
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

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
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

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}

func (m Model) renderConfirmModal() string {
	const innerW = 46
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(m.Theme.P0Color).Bold(true).Render("Confirm Delete"))
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Are you sure you want to delete task:"))
	lines = append(lines, fmt.Sprintf("  \"%s\"?", sentenceCase(m.ConfirmTask.Title)))
	lines = append(lines, "")
	lines = append(lines, m.modalSep(innerW))
	lines = append(lines, "")
	yesStr := lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render("y")
	noStr := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("n / Esc")
	lines = append(lines, fmt.Sprintf("  [%s] Yes, Delete   [%s] Cancel", yesStr, noStr))

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(lines, "\n"), innerW))
}



func (m Model) renderHelpModal() string {
	modalW := 82
	if modalW > m.Width-4 {
		modalW = m.Width - 4
	}
	if modalW < 42 {
		modalW = 42
	}
	const paddingL = 2
	const paddingR = 2
	const borderW = 2
	innerW := modalW - paddingL - paddingR - borderW

	// Visible content rows (excludes pinned header + footer)
	// Reserve: 2 header lines + 1 blank + 3 footer lines = 6
	termH := m.Height
	if termH < 20 {
		termH = 20
	}
	visibleRows := termH - 14 // how many body lines fit
	if visibleRows < 5 {
		visibleRows = 5
	}

	accent := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	sectStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
	mutedStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)

	sepLine := mutedStyle.Render(strings.Repeat("─", innerW))

	formatKeyVal := func(key, desc string) string {
		keyStr := accent.Render(fmt.Sprintf("%-18s", key))
		gutter := "  "
		maxDescLen := innerW - 18 - 2
		descRunes := []rune(desc)
		if len(descRunes) > maxDescLen {
			desc = string(descRunes[:maxDescLen-3]) + "..."
		}
		return keyStr + gutter + descStyle.Render(desc)
	}

	addSection := func(lines *[]string, name string) {
		*lines = append(*lines,
			"",
			"  "+sectStyle.Bold(true).Render(strings.ToUpper(name)),
			"",
		)
	}

	// ── Build full body lines ──────────────────────────────────────────────
	var body []string

	addSection(&body, "NAVIGATION")
	body = append(body, formatKeyVal("1 - 5", "Switch views"))
	body = append(body, formatKeyVal("j / k", "Navigate tasks vertically"))
	body = append(body, formatKeyVal("h / l", "Navigate overlapping tasks"))
	body = append(body, formatKeyVal("J / K", "Scroll timeline hours"))
	body = append(body, formatKeyVal("H / L", "Day backward / forward"))
	body = append(body, formatKeyVal("t", "Jump to today"))
	body = append(body, formatKeyVal("Tab", "Toggle timeline ↔ backlog shelf"))
	body = append(body, formatKeyVal("ctrl+d / ctrl+u", "Scroll pane down / up"))

	addSection(&body, "WORKSPACES")
	body = append(body, formatKeyVal("w", "Next workspace →"))
	body = append(body, formatKeyVal("W", "Previous workspace ←"))
	body = append(body, formatKeyVal(":ws-create", "Create new workspace"))
	body = append(body, formatKeyVal(":ws-edit", "Edit active workspace"))
	body = append(body, formatKeyVal(":ws-delete [name]", "Delete workspace"))
	body = append(body, formatKeyVal(":ws-switch <name>", "Switch to named workspace"))

	addSection(&body, "TASK ACTIONS")
	body = append(body, formatKeyVal("i", "Open task creation form"))
	body = append(body, formatKeyVal("e", "Edit selected task"))
	body = append(body, formatKeyVal("x", "Complete selected task"))
	body = append(body, formatKeyVal("d", "Delete selected task"))
	body = append(body, formatKeyVal("z", "Start / resume Zen session"))
	body = append(body, formatKeyVal("Enter", "Inspect task details"))

	addSection(&body, "ZEN FOCUS MODE")
	body = append(body, formatKeyVal("Space", "Pause / resume timer"))
	body = append(body, formatKeyVal("+", "Inject +5 min to session"))
	body = append(body, formatKeyVal("b", "Skip current block"))
	body = append(body, formatKeyVal("r", "Restart block timer"))
	body = append(body, formatKeyVal("Esc", "Exit to background (timer runs)"))

	addSection(&body, "SYSTEM")
	body = append(body, formatKeyVal(":", "Open command palette"))
	body = append(body, formatKeyVal("?", "Toggle this help modal"))
	body = append(body, formatKeyVal(":create <title>", "Create anchored task (9:00 AM)"))
	body = append(body, formatKeyVal(":todo <title>", "Create floating backlog task"))
	body = append(body, formatKeyVal(":sync", "Force Google Calendar sync"))
	body = append(body, formatKeyVal(":auth", "Authenticate Google Calendar"))
	body = append(body, formatKeyVal(":review", "Open shutdown review"))
	body = append(body, formatKeyVal(":stop", "Abort Zen focus session"))
	body = append(body, formatKeyVal(":quit / :q", "Exit stream"))

	// ── Clamp scroll offset ────────────────────────────────────────────────
	maxScroll := len(body) - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	offset := m.HelpScrollOffset
	if offset > maxScroll {
		offset = maxScroll
	}
	if offset < 0 {
		offset = 0
	}

	// ── Slice the visible window ───────────────────────────────────────────
	end := offset + visibleRows
	if end > len(body) {
		end = len(body)
	}
	visible := body[offset:end]

	// Pad short pages so the modal height is stable
	for len(visible) < visibleRows {
		visible = append(visible, "")
	}

	// ── Scroll progress bar ────────────────────────────────────────────────
	scrollPct := 0
	if maxScroll > 0 {
		scrollPct = (offset * 100) / maxScroll
	}
	barWidth := innerW - 14
	if barWidth < 4 {
		barWidth = 4
	}
	filled := (scrollPct * barWidth) / 100
	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	progressStr := mutedStyle.Render(fmt.Sprintf(" %3d%%  ", scrollPct)) +
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(progressBar)

	// ── Assemble final output ──────────────────────────────────────────────
	var out []string

	// Pinned header (always visible)
	brand := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render("▲ s t r e a m")
	midDot := mutedStyle.Render("   •   ")
	cmdRef := mutedStyle.Render("c o m m a n d   r e f e r e n c e")
	out = append(out, brand+midDot+cmdRef)
	out = append(out, sepLine)

	// Scrollable body
	out = append(out, visible...)

	// Pinned footer
	out = append(out, sepLine)
	out = append(out, progressStr)
	navHint := mutedStyle.Render("  j/k scroll  ctrl+d/u page  g/G top/btm  esc close")
	out = append(out, navHint)

	return m.Theme.ModalStyle.
		Width(innerW + 4).
		Render(m.prepareModalContent(strings.Join(out, "\n"), innerW))
}

func (m Model) prepareModalContent(content string, innerW int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if strings.Contains(line, "▲ s t r e a m") {
			w += 2
		}
		if strings.Contains(line, "💡") {
			w += 1
		}
		if w < innerW {
			lines[i] = line + strings.Repeat(" ", innerW-w)
		}
	}
	return strings.Join(lines, "\n")
}
