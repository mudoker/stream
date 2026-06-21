package components

import (
	"fmt"
	"strings"

	"stream/internal/view/theme"
	"stream/internal/viewmodel/common/constants"

	"github.com/charmbracelet/lipgloss"
)

// PrepareModalContent pads lines in a modal to match the target inner width.
func PrepareModalContent(content string, innerW int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		cells := theme.ParseLineToCells(line)
		w := len(cells)
		if w < innerW {
			for len(cells) < innerW {
				cells = append(cells, theme.Cell{Text: " "})
			}
			lines[i] = theme.CellsToLine(cells)
		}
	}
	return strings.Join(lines, "\n")
}

// ModalSep draws a horizontal divider line.
func ModalSep(w int) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(constants.ColorSeparator)).Render(strings.Repeat("─", w))
}

// RenderBaseConfirmModal draws a standardized confirmation dialog with key navigation.
func RenderBaseConfirmModal(title string, descLines []string, options []string, selectedIdx int, destructiveIdx int, focusArea int, t theme.Theme) string {
	const innerW = 50
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(title))
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	for _, d := range descLines {
		lines = append(lines, "  "+d)
	}
	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	// Render selectable options
	for idx, opt := range options {
		var optStr string
		if idx == selectedIdx {
			color := t.Accent
			if idx == destructiveIdx {
				color = t.P0Color
			}
			if focusArea == 0 {
				optStr = lipgloss.NewStyle().Foreground(color).Bold(true).Render("  ▶ " + opt + " ◀")
			} else {
				optStr = lipgloss.NewStyle().Foreground(color).Render("  ◦ " + opt)
			}
		} else {
			optStr = lipgloss.NewStyle().Foreground(t.Muted).Render("    " + opt)
		}
		lines = append(lines, optStr)
	}

	lines = append(lines, "")
	lines = append(lines, ModalSep(innerW))
	lines = append(lines, "")

	var confirmBtn, cancelBtn string
	var hintText string
	if focusArea == 0 {
		confirmBtn = lipgloss.NewStyle().Foreground(t.Accent).Render("  Confirm  ")
		cancelBtn = lipgloss.NewStyle().Foreground(t.Muted).Render("  Cancel  ")
		hintText = lipgloss.NewStyle().Foreground(t.Muted).Render("j/k select, Tab navigate")
	} else if focusArea == 1 {
		confirmBtn = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▶ Confirm ◀")
		cancelBtn = lipgloss.NewStyle().Foreground(t.Muted).Render("  Cancel  ")
		hintText = lipgloss.NewStyle().Foreground(t.Muted).Render("Enter choose, Tab navigate")
	} else {
		confirmBtn = lipgloss.NewStyle().Foreground(t.Accent).Render("  Confirm  ")
		cancelBtn = lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("▶ Cancel ◀")
		hintText = lipgloss.NewStyle().Foreground(t.Muted).Render("Enter choose, Tab navigate")
	}
	lines = append(lines, fmt.Sprintf("  %s   %s   %s", confirmBtn, cancelBtn, hintText))

	return t.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}
