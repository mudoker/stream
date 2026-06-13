package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderTopCommuteBlock renders the top commute buffer block above the event card.
func RenderTopCommuteBlock(t theme.Theme, w, h int, commuteMins int, startTime time.Time, isFocused bool) string {
	if w < 3 {
		w = 3
	}
	if h < 1 {
		h = 1
	}

	var borderColor lipgloss.Color
	if isFocused {
		borderColor = t.FocusPurple
	} else {
		borderColor = lipgloss.Color("#f9e2af") // Warm peach/yellow color for commute buffers
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	var textStyle lipgloss.Style
	if isFocused {
		textStyle = lipgloss.NewStyle().Foreground(t.FocusPurple).Italic(true)
	} else {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Italic(true)
	}

	topLeft := "┌"
	topRight := "┐"
	horizChar := "╌"
	vertChar := "┊"

	commuteText := fmt.Sprintf("🚗 Commute %dm (%s)", commuteMins, startTime.Format("15:04"))

	var lines []string

	if h == 1 {
		// For h = 1, show text in one line with top border
		line := embedTextInLine(topLeft, topRight, horizChar, commuteText, w, borderStyle, textStyle)
		lines = append(lines, line)
	} else {
		// For h >= 2, render top border and side borders
		topLine := borderStyle.Render(topLeft + strings.Repeat(horizChar, w-2) + topRight)
		lines = append(lines, topLine)

		centerRow := (h - 1) / 2
		for i := 0; i < h-1; i++ {
			var line string
			if i == centerRow {
				line = embedTextInLine(vertChar, vertChar, " ", commuteText, w, borderStyle, textStyle)
			} else {
				line = borderStyle.Render(vertChar) + strings.Repeat(" ", w-2) + borderStyle.Render(vertChar)
			}
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// RenderBottomCommuteBlock renders the bottom commute buffer block below the event card.
func RenderBottomCommuteBlock(t theme.Theme, w, h int, commuteMins int, endTime time.Time, isFocused bool) string {
	if w < 3 {
		w = 3
	}
	if h < 1 {
		h = 1
	}

	var borderColor lipgloss.Color
	if isFocused {
		borderColor = t.FocusPurple
	} else {
		borderColor = lipgloss.Color("#f9e2af")
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	var textStyle lipgloss.Style
	if isFocused {
		textStyle = lipgloss.NewStyle().Foreground(t.FocusPurple).Italic(true)
	} else {
		textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Italic(true)
	}

	bottomLeft := "└"
	bottomRight := "┘"
	horizChar := "╌"
	vertChar := "┊"

	commuteText := fmt.Sprintf("🚗 Commute %dm (%s)", commuteMins, endTime.Format("15:04"))

	var lines []string

	if h == 1 {
		// For h = 1, show text in one line with bottom border
		line := embedTextInLine(bottomLeft, bottomRight, horizChar, commuteText, w, borderStyle, textStyle)
		lines = append(lines, line)
	} else {
		// For h >= 2, render side borders and bottom border
		bottomLine := borderStyle.Render(bottomLeft + strings.Repeat(horizChar, w-2) + bottomRight)

		centerRow := (h - 1) / 2
		for i := 0; i < h-1; i++ {
			var line string
			if i == centerRow {
				line = embedTextInLine(vertChar, vertChar, " ", commuteText, w, borderStyle, textStyle)
			} else {
				line = borderStyle.Render(vertChar) + strings.Repeat(" ", w-2) + borderStyle.Render(vertChar)
			}
			lines = append(lines, line)
		}
		lines = append(lines, bottomLine)
	}

	return strings.Join(lines, "\n")
}
