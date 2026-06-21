package components

import (
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

// BaseModalConfig defines the configuration structure for reusable modals.
type BaseModalConfig struct {
	Title      string
	TitleColor lipgloss.Color // optional title foreground override
	BodyLines  []string
	Buttons    []string // optional list of buttons to be centered
	FooterText string   // optional centered footer text
	InnerWidth int
	Theme      theme.Theme
}

// RenderBaseModal renders a standardized modal with centered header, body, action buttons, and footer.
func RenderBaseModal(cfg BaseModalConfig) string {
	innerW := cfg.InnerWidth
	if innerW <= 0 {
		innerW = 50
	}
	var lines []string

	if cfg.Title != "" {
		titleColor := cfg.Theme.Accent
		if string(cfg.TitleColor) != "" {
			titleColor = cfg.TitleColor
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render(cfg.Title))
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")
	}

	for _, line := range cfg.BodyLines {
		lines = append(lines, line)
	}

	if len(cfg.Buttons) > 0 {
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")

		buttonsLine := strings.Join(cfg.Buttons, "      ")
		visibleW := lipgloss.Width(buttonsLine)
		leftPadding := (innerW - visibleW) / 2
		if leftPadding > 0 {
			buttonsLine = strings.Repeat(" ", leftPadding) + buttonsLine
		}
		lines = append(lines, buttonsLine)
	}

	if cfg.FooterText != "" {
		lines = append(lines, "")
		lines = append(lines, ModalSep(innerW))
		lines = append(lines, "")

		visibleW := lipgloss.Width(cfg.FooterText)
		leftPadding := (innerW - visibleW) / 2
		footerLine := cfg.FooterText
		if leftPadding > 0 {
			footerLine = strings.Repeat(" ", leftPadding) + footerLine
		}
		lines = append(lines, footerLine)
	}

	return cfg.Theme.ModalStyle.Render(PrepareModalContent(strings.Join(lines, "\n"), innerW))
}

// RenderBaseConfirmModal draws a standardized confirmation dialog with key navigation.
func RenderBaseConfirmModal(title string, descLines []string, options []string, selectedIdx int, destructiveIdx int, focusArea int, t theme.Theme) string {
	const innerW = 50
	var bodyLines []string

	for _, d := range descLines {
		bodyLines = append(bodyLines, "  "+d)
	}
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, ModalSep(innerW))
	bodyLines = append(bodyLines, "")

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
		bodyLines = append(bodyLines, optStr)
	}

	var buttons []string
	if focusArea == 0 {
		buttons = []string{
			lipgloss.NewStyle().Foreground(t.Accent).Render("  Confirm  "),
			lipgloss.NewStyle().Foreground(t.Muted).Render("  Cancel  "),
		}
	} else if focusArea == 1 {
		buttons = []string{
			lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▶ Confirm ◀"),
			lipgloss.NewStyle().Foreground(t.Muted).Render("  Cancel  "),
		}
	} else {
		buttons = []string{
			lipgloss.NewStyle().Foreground(t.Accent).Render("  Confirm  "),
			lipgloss.NewStyle().Foreground(t.P0Color).Bold(true).Render("▶ Cancel ◀"),
		}
	}

	return RenderBaseModal(BaseModalConfig{
		Title:      title,
		BodyLines:  bodyLines,
		Buttons:    buttons,
		InnerWidth: innerW,
		Theme:      t,
	})
}
