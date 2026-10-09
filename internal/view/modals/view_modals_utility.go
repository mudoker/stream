package modals

import (
	"fmt"
	"strings"

	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func RenderCommandPalette(m *viewmodel.Model, t theme.Theme) string {
	modalW := 80
	if modalW > m.Width-4 {
		modalW = m.Width - 4
	}
	if modalW < 40 {
		modalW = 40
	}
	innerW := modalW - 6

	termH := m.Height
	if termH < 15 {
		termH = 15
	}
	maxVisible := termH - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if maxVisible > 12 {
		maxVisible = 12
	}

	var sb strings.Builder

	divColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37"))
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)

	sb.WriteString(lipgloss.NewStyle().Padding(0, 1).Render(m.CommandInput.View()) + "\n")
	sb.WriteString(divColor.Render(strings.Repeat("─", innerW)) + "\n")

	val := strings.ToLower(m.CommandInput.Value())
	allCommands := m.GetCommandList()

	var genericEntries []viewmodel.CommandEntry
	var wsEntries []viewmodel.CommandEntry
	for _, c := range allCommands {
		if strings.HasPrefix(c.Name, "ws-switch ") {
			wsEntries = append(wsEntries, c)
		} else {
			genericEntries = append(genericEntries, c)
		}
	}

	filterGroup := func(src []viewmodel.CommandEntry) []viewmodel.CommandEntry {
		var out []viewmodel.CommandEntry
		for _, c := range src {
			if strings.Contains(strings.ToLower(c.Name), val) ||
				strings.Contains(strings.ToLower(c.Desc), val) {
				out = append(out, c)
			}
		}
		return out
	}
	filteredGeneric := filterGroup(genericEntries)
	filteredWS := filterGroup(wsEntries)

	var allFiltered []viewmodel.CommandEntry
	allFiltered = append(allFiltered, filteredGeneric...)
	allFiltered = append(allFiltered, filteredWS...)
	totalEntries := len(allFiltered)

	selIdx := m.CommandSelectedIndex
	if totalEntries > 0 && selIdx >= totalEntries {
		selIdx = totalEntries - 1
	}

	nameW := 24
	renderRow := func(globalIdx int, c viewmodel.CommandEntry) string {
		isSelected := selIdx >= 0 && globalIdx == selIdx
		if isSelected {
			indicator := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("┃")
			keyword := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
				Render(fmt.Sprintf("%-*s", nameW, c.Name))
			desc := lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(c.Desc)
			return lipgloss.NewStyle().Width(innerW).
				Render(fmt.Sprintf("%s %s %s", indicator, keyword, desc))
		}
		keyword := lipgloss.NewStyle().Foreground(t.Fg).
			Render(fmt.Sprintf("%-*s", nameW, c.Name))
		desc := mutedStyle.Render(c.Desc)
		return lipgloss.NewStyle().Width(innerW).
			Render(fmt.Sprintf("  %s %s", keyword, desc))
	}

	if totalEntries == 0 {
		sb.WriteString("  " + mutedStyle.Render("No matching commands") + "\n")
	} else {
		startIdx := 0
		if selIdx >= maxVisible {
			startIdx = selIdx - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > totalEntries {
			endIdx = totalEntries
		}

		for i := startIdx; i < endIdx; i++ {
			sb.WriteString(renderRow(i, allFiltered[i]) + "\n")
		}
	}

	sb.WriteString(divColor.Render(strings.Repeat("─", innerW)) + "\n")
	footerStr := "  ↑↓ navigate  ↵ execute  esc close  w/W quick-switch"
	if totalEntries > maxVisible {
		footerStr = fmt.Sprintf("  ↑↓ (%d/%d)  ↵ execute  esc close", selIdx+1, totalEntries)
	}
	sb.WriteString(mutedStyle.Render(footerStr) + "\n")

	return lipgloss.NewStyle().
		Foreground(t.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Width(innerW + 2).
		Padding(0, 1).
		Render(sb.String())
}

func RenderHelpModal(m *viewmodel.Model, t theme.Theme) string {
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

	termH := m.Height
	if termH < 20 {
		termH = 20
	}
	visibleRows := termH - 14
	if visibleRows < 5 {
		visibleRows = 5
	}
	if visibleRows > 18 {
		visibleRows = 18
	}

	accent := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	sectStyle := lipgloss.NewStyle().Foreground(t.Muted)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)

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

	var body []string

	addSection(&body, "VIEWS & GLOBAL NAVIGATION")
	body = append(body, formatKeyVal("1 - 6", "1:Dash, 2:Month, 3:Sprint, 4:Week, 5:Day, 6:Analytics"))
	body = append(body, formatKeyVal("j / k", "Navigate tasks / items vertically (down / up)"))
	body = append(body, formatKeyVal("h / l", "Navigate overlapping tasks horizontally"))
	body = append(body, formatKeyVal("J / K", "Scroll timeline hours (Day) / cycle shelf sections"))
	body = append(body, formatKeyVal("H / L", "Day backward / forward (-1 / +1 day)"))
	body = append(body, formatKeyVal("t", "Jump directly to today's date"))
	body = append(body, formatKeyVal("Tab", "Switch focus between Day Timeline ↔ Backlog Shelf"))
	body = append(body, formatKeyVal("ctrl+d / ctrl+u", "Half-page scroll down / up"))
	body = append(body, formatKeyVal("ctrl+f / ctrl+b", "Full-page scroll down / up"))
	body = append(body, formatKeyVal("g / G", "Jump to top / bottom of timeline or list"))
	body = append(body, formatKeyVal("w / W", "Switch to next / previous workspace"))

	addSection(&body, "TASK & WORK ITEM ACTIONS")
	body = append(body, formatKeyVal("i", "Open creation wizard (Task, Feature, Defect, Habit...)"))
	body = append(body, formatKeyVal("e", "Edit selected task / work item in form wizard"))
	body = append(body, formatKeyVal("x", "Toggle completion checkmark on selected task"))
	body = append(body, formatKeyVal("d", "Delete selected task (occurrence or recurring series)"))
	body = append(body, formatKeyVal("D", "Clear current shelf section (Today/Backlog/Habits...)"))
	body = append(body, formatKeyVal("a", "Quick anchor / de-anchor task (or anchor to today)"))
	body = append(body, formatKeyVal("y", "Enter Move mode (shift start time with j/k; ↵ save)"))
	body = append(body, formatKeyVal("Y", "Enter Clone mode (duplicate task to new time with j/k)"))
	body = append(body, formatKeyVal("[ / ]", "Adjust task duration (-15m / +15m)"))
	body = append(body, formatKeyVal("z", "Start or resume Zen Pomodoro focus session"))
	body = append(body, formatKeyVal("Enter", "Inspect full task details & blocker dependencies"))

	addSection(&body, "SPRINT KANBAN BOARD (View 3)")
	body = append(body, formatKeyVal("3", "Switch to Sprint Kanban Board"))
	body = append(body, formatKeyVal("H / L", "Switch active swimlane column (Backlog → Done)"))
	body = append(body, formatKeyVal("j / k", "Navigate features within current swimlane"))
	body = append(body, formatKeyVal("y", "Sprint Move mode (j/k reorder, h/l column, ↵ save)"))
	body = append(body, formatKeyVal("p", "Create child task on Today Shelf linked to feature"))
	body = append(body, formatKeyVal("a", "Anchor task to today timeline (with confirmation)"))
	body = append(body, formatKeyVal("t", "Toggle feature into / out of Today Shelf"))
	body = append(body, formatKeyVal("I", "Open Create New Sprint dialog"))
	body = append(body, formatKeyVal("E", "Open Edit Active Sprint settings dialog"))
	body = append(body, formatKeyVal("D", "Open Delete Active Sprint dialog"))

	addSection(&body, "MOVE & DURATION MODES")
	body = append(body, formatKeyVal("j / k (or ↓ / ↑)", "Move start time down / up (15m step) or reorder"))
	body = append(body, formatKeyVal("H / L", "Jump time offset by 1 hour (or 1 day)"))
	body = append(body, formatKeyVal("Enter", "Confirm move / duration change and save"))
	body = append(body, formatKeyVal("Esc", "Cancel move and restore previous position"))

	addSection(&body, "ZEN FOCUS & POMODORO (z)")
	body = append(body, formatKeyVal("Space", "Pause / resume countdown timer"))
	body = append(body, formatKeyVal("+ / -", "Adjust timer +/- 30s (supports prefix e.g. 5+)"))
	body = append(body, formatKeyVal("b", "Skip current block (Focus ↔ Break)"))
	body = append(body, formatKeyVal("r", "Restart timer for current block"))
	body = append(body, formatKeyVal("x", "Complete task directly from Zen session"))
	body = append(body, formatKeyVal("Esc", "Minimize to background (timer continues ticking)"))
	body = append(body, formatKeyVal(":stop", "Abort focus session and log completed metrics"))

	addSection(&body, "COMMAND PALETTE (:)")
	body = append(body, formatKeyVal(":", "Open command palette input overlay"))
	body = append(body, formatKeyVal("?", "Toggle this Command Reference Help modal"))
	body = append(body, formatKeyVal("↑ / ↓ (or Tab)", "Navigate command suggestions in palette"))
	body = append(body, formatKeyVal(":feature <title>", "Quick create Feature in active sprint"))
	body = append(body, formatKeyVal(":defect <title>", "Quick create Defect in active sprint"))
	body = append(body, formatKeyVal(":improvement <title>", "Quick create Improvement in active sprint"))
	body = append(body, formatKeyVal(":task <title>", "Quick create Backlog floating task"))
	body = append(body, formatKeyVal(":todo <title>", "Quick create Backlog floating task"))
	body = append(body, formatKeyVal(":create <title>", "Quick create Anchored task (9:00 AM)"))
	body = append(body, formatKeyVal(":habit <title>", "Quick create daily repeatable Habit"))
	body = append(body, formatKeyVal(":sprint-create", "Open Create Sprint wizard dialog"))
	body = append(body, formatKeyVal(":sprint-edit", "Open Edit Active Sprint settings"))
	body = append(body, formatKeyVal(":sprint-delete", "Open Delete Active Sprint confirmation"))
	body = append(body, formatKeyVal(":sprint-generate", "Auto-generate recurring sprints schedule"))
	body = append(body, formatKeyVal(":ws-create", "Open Create Workspace dialog"))
	body = append(body, formatKeyVal(":ws-edit", "Open Edit Active Workspace settings"))
	body = append(body, formatKeyVal(":ws-delete [name]", "Delete specified or active workspace"))
	body = append(body, formatKeyVal(":ws-switch [name]", "Switch workspace directly or open picker"))
	body = append(body, formatKeyVal(":pull / :push", "Pull from / Push to Google Calendar"))
	body = append(body, formatKeyVal(":sync-settings", "Configure Google Calendar sync mode & interval"))
	body = append(body, formatKeyVal(":auth", "Start Google Calendar OAuth2 authorization"))
	body = append(body, formatKeyVal(":tags", "Open System Tags manager (CRUD)"))
	body = append(body, formatKeyVal(":music", "Open Jazz Lounge lofi audio player"))
	body = append(body, formatKeyVal(":profile", "Edit profile username and password/lock"))
	body = append(body, formatKeyVal(":review", "Open daily shutdown evaluation & productivity review"))
	body = append(body, formatKeyVal(":complete", "Mark selected task completed"))
	body = append(body, formatKeyVal(":delete", "Permanently delete selected task"))
	body = append(body, formatKeyVal(":factory-reset", "Wipe all local database data and reset to defaults"))
	body = append(body, formatKeyVal(":quit / :q / :exit", "Exit Stream application"))

	addSection(&body, "SYSTEM TAGS MANAGER (:tags)")
	body = append(body, formatKeyVal("j / k", "Navigate tag list"))
	body = append(body, formatKeyVal("a", "Add new tag"))
	body = append(body, formatKeyVal("e", "Edit selected tag name"))
	body = append(body, formatKeyVal("d", "Delete selected tag"))
	body = append(body, formatKeyVal("Esc", "Close tags manager"))

	addSection(&body, "FORMS & WIZARDS")
	body = append(body, formatKeyVal("Tab / Shift+Tab", "Navigate between form input fields"))
	body = append(body, formatKeyVal("h / l (or ← / →)", "Toggle / change dropdown options"))
	body = append(body, formatKeyVal("Enter", "Submit form and save task / sprint / settings"))
	body = append(body, formatKeyVal("Esc", "Cancel form without saving changes"))

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
	m.HelpScrollOffset = offset

	end := offset + visibleRows
	if end > len(body) {
		end = len(body)
	}
	visible := body[offset:end]

	for len(visible) < visibleRows {
		visible = append(visible, "")
	}

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
		lipgloss.NewStyle().Foreground(t.Accent).Render(progressBar)

	var out []string

	brand := lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render("▲ s t r e a m")
	midDot := mutedStyle.Render("   •   ")
	cmdRef := mutedStyle.Render("c o m m a n d   r e f e r e n c e")
	out = append(out, brand+midDot+cmdRef)
	out = append(out, sepLine)
	out = append(out, visible...)
	out = append(out, sepLine)
	out = append(out, progressStr)
	navHint := mutedStyle.Render("  j/k scroll  ctrl+d/u page  g/G top/btm  esc close")
	out = append(out, navHint)

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(out, "\n"), innerW))
}
