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

	addSection(&body, "NAVIGATION")
	body = append(body, formatKeyVal("1 - 5", "Switch views"))
	body = append(body, formatKeyVal("j / k", "Navigate tasks vertically"))
	body = append(body, formatKeyVal("h / l", "Navigate overlapping tasks"))
	body = append(body, formatKeyVal("J / K", "Timeline hours / shelf sections"))
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

	addSection(&body, "SPRINTS & KANBAN")
	body = append(body, formatKeyVal("3", "Switch to Sprint Kanban view"))
	body = append(body, formatKeyVal("I", "Create new sprint"))
	body = append(body, formatKeyVal("E", "Edit active sprint"))
	body = append(body, formatKeyVal("D", "Delete active sprint (or clear section)"))
	body = append(body, formatKeyVal("y", "Move mode (j/k reorder, h/l swimlane, ↵ save, esc cancel)"))
	body = append(body, formatKeyVal("H / L", "Switch swimlane column left / right"))
	body = append(body, formatKeyVal("j / k", "Navigate features within current swimlane"))
	body = append(body, formatKeyVal("p", "Create linked task on Today Shelf"))
	body = append(body, formatKeyVal("a", "Anchor / de-anchor item to sprint"))
	body = append(body, formatKeyVal("t", "Toggle item on Today Shelf"))

	addSection(&body, "TASK ACTIONS")
	body = append(body, formatKeyVal("i", "Open task/feature creation form"))
	body = append(body, formatKeyVal("e", "Edit selected task/feature"))
	body = append(body, formatKeyVal("a", "Quick anchor / de-anchor task"))
	body = append(body, formatKeyVal("y", "Move task start time"))
	body = append(body, formatKeyVal("Y", "Clone task to new start time"))
	body = append(body, formatKeyVal("x", "Complete selected task"))
	body = append(body, formatKeyVal("d", "Delete selected task"))
	body = append(body, formatKeyVal("z", "Start / resume Zen session"))
	body = append(body, formatKeyVal("Enter", "Inspect task details"))

	addSection(&body, "ZEN FOCUS MODE")
	body = append(body, formatKeyVal("Space", "Pause / resume timer"))
	body = append(body, formatKeyVal("[x]+ / [x]-", "Adjust timer by +/- 30s (x multiplier)"))
	body = append(body, formatKeyVal("b", "Skip current block"))
	body = append(body, formatKeyVal("r", "Restart block timer"))
	body = append(body, formatKeyVal("Esc", "Exit to background (timer runs)"))

	addSection(&body, "SYSTEM & COMMANDS")
	body = append(body, formatKeyVal(":", "Open command palette"))
	body = append(body, formatKeyVal("?", "Toggle this help modal"))
	body = append(body, formatKeyVal(":feature <title>", "Create feature in active sprint"))
	body = append(body, formatKeyVal(":defect <title>", "Create defect in active sprint"))
	body = append(body, formatKeyVal(":improvement <title>", "Create improvement in active sprint"))
	body = append(body, formatKeyVal(":task <title>", "Create backlog floating task"))
	body = append(body, formatKeyVal(":create <title>", "Create anchored task (9:00 AM)"))
	body = append(body, formatKeyVal(":habit <title>", "Create daily repeatable habit"))
	body = append(body, formatKeyVal(":sprint-create", "Create new sprint"))
	body = append(body, formatKeyVal(":sprint-edit", "Edit active sprint settings"))
	body = append(body, formatKeyVal(":sprint-delete", "Delete active sprint"))
	body = append(body, formatKeyVal(":sprint-generate", "Generate recurring sprints"))
	body = append(body, formatKeyVal(":pull / :push", "Pull from / Push to Google Calendar"))
	body = append(body, formatKeyVal(":sync-settings", "Configure GCal sync mode & interval"))
	body = append(body, formatKeyVal(":auth", "Authenticate Google Calendar"))
	body = append(body, formatKeyVal(":review", "Open shutdown review"))
	body = append(body, formatKeyVal(":tags", "Manage system tags"))
	body = append(body, formatKeyVal(":music", "Open jazz lounge player"))
	body = append(body, formatKeyVal(":stop", "Abort Zen focus session"))
	body = append(body, formatKeyVal(":quit / :q", "Exit stream"))

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
