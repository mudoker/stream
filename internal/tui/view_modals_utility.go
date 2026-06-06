package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderCommandPalette() string {
	var sb strings.Builder

	divColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2c37"))
	mutedStyle := lipgloss.NewStyle().Foreground(m.Theme.Muted)
	innerW := m.Width - 8

	sb.WriteString(lipgloss.NewStyle().Padding(1, 2).Render(m.CommandInput.View()) + "\n")
	sb.WriteString(divColor.Render(strings.Repeat("─", innerW)) + "\n\n")

	val := strings.ToLower(m.CommandInput.Value())
	allCommands := m.getCommandList()

	var genericEntries []CommandEntry
	var wsEntries []CommandEntry
	for _, c := range allCommands {
		if strings.HasPrefix(c.Name, "ws-switch ") {
			wsEntries = append(wsEntries, c)
		} else {
			genericEntries = append(genericEntries, c)
		}
	}

	filterGroup := func(src []CommandEntry) []CommandEntry {
		var out []CommandEntry
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
	totalEntries := len(filteredGeneric) + len(filteredWS)

	selIdx := m.CommandSelectedIndex
	if selIdx < 0 {
		selIdx = 0
	}
	if totalEntries > 0 && selIdx >= totalEntries {
		selIdx = totalEntries - 1
	}

	nameW := 26
	renderRow := func(globalIdx int, c CommandEntry) string {
		isSelected := globalIdx == selIdx
		if isSelected {
			indicator := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render("┃")
			keyword := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).
				Render(fmt.Sprintf("%-*s", nameW, c.Name))
			desc := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render(c.Desc)
			return lipgloss.NewStyle().Width(innerW).
				Render(fmt.Sprintf("%s  %s  %s", indicator, keyword, desc))
		}
		keyword := lipgloss.NewStyle().Foreground(m.Theme.Fg).
			Render(fmt.Sprintf("%-*s", nameW, c.Name))
		desc := mutedStyle.Render(c.Desc)
		return lipgloss.NewStyle().Width(innerW).
			Render(fmt.Sprintf("   %s  %s", keyword, desc))
	}

	if len(filteredGeneric) > 0 {
		sb.WriteString("  " + mutedStyle.Render("COMMANDS") + "\n\n")
		for i, e := range filteredGeneric {
			sb.WriteString(renderRow(i, e) + "\n")
		}
		sb.WriteString("\n")
	}

	if len(filteredWS) > 0 {
		sb.WriteString("  " + mutedStyle.Render("SWITCH WORKSPACE") + "\n\n")
		base := len(filteredGeneric)
		for i, e := range filteredWS {
			sb.WriteString(renderRow(base+i, e) + "\n")
		}
		sb.WriteString("\n")
	}

	if totalEntries == 0 {
		sb.WriteString("  " + mutedStyle.Render("No matching commands") + "\n\n")
	}

	sb.WriteString(divColor.Render(strings.Repeat("─", innerW)) + "\n")
	sb.WriteString(mutedStyle.Render("  ↑↓ navigate  ↵ execute  esc close  w/W quick-switch") + "\n")

	return lipgloss.NewStyle().
		Foreground(m.Theme.Fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Accent).
		Width(m.Width-4).
		Padding(0, 2).
		Render(sb.String())
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

	termH := m.Height
	if termH < 20 {
		termH = 20
	}
	visibleRows := termH - 14
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
	body = append(body, formatKeyVal("a", "Quick anchor / de-anchor task"))
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
		lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(progressBar)

	var out []string

	brand := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).Render("▲ s t r e a m")
	midDot := mutedStyle.Render("   •   ")
	cmdRef := mutedStyle.Render("c o m m a n d   r e f e r e n c e")
	out = append(out, brand+midDot+cmdRef)
	out = append(out, sepLine)
	out = append(out, visible...)
	out = append(out, sepLine)
	out = append(out, progressStr)
	navHint := mutedStyle.Render("  j/k scroll  ctrl+d/u page  g/G top/btm  esc close")
	out = append(out, navHint)

	return m.Theme.ModalStyle.Render(m.prepareModalContent(strings.Join(out, "\n"), innerW))
}
