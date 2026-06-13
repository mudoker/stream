package pages

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// renderTimelineBufferBlock abstracts rendering a timeline block with dashed borders
// and centered text inside. It supports both top (┌) and bottom (└) corner borders.
func renderTimelineBufferBlock(w, h int, text string, isTop bool, color lipgloss.Color) string {
	if w < 3 {
		w = 3
	}
	if h < 1 {
		h = 1
	}

	borderStyle := lipgloss.NewStyle().Foreground(color)
	textStyle := lipgloss.NewStyle().Foreground(color).Italic(true)

	var leftCorner, rightCorner string
	if isTop {
		leftCorner = "┌"
		rightCorner = "┐"
	} else {
		leftCorner = "└"
		rightCorner = "┘"
	}

	horizChar := "╌"
	vertChar := "┊"

	var lines []string

	if h == 1 {
		line := embedTextInLine(leftCorner, rightCorner, horizChar, text, w, borderStyle, textStyle)
		lines = append(lines, line)
	} else {
		borderLine := borderStyle.Render(leftCorner + strings.Repeat(horizChar, w-2) + rightCorner)
		centerRow := (h - 1) / 2

		if isTop {
			lines = append(lines, borderLine)
		}

		for i := 0; i < h-1; i++ {
			var line string
			if i == centerRow {
				line = embedTextInLine(vertChar, vertChar, " ", text, w, borderStyle, textStyle)
			} else {
				line = borderStyle.Render(vertChar) + strings.Repeat(" ", w-2) + borderStyle.Render(vertChar)
			}
			lines = append(lines, line)
		}

		if !isTop {
			lines = append(lines, borderLine)
		}
	}

	return strings.Join(lines, "\n")
}

// RenderTopCommuteBlock renders the top commute buffer block above the event card.
func RenderTopCommuteBlock(t theme.Theme, w, h int, commuteMins int, startTime time.Time, isFocused bool) string {
	color := lipgloss.Color("#f9e2af")
	if isFocused {
		color = t.FocusPurple
	}
	text := fmt.Sprintf("🚗 Commute %dm (%s)", commuteMins, startTime.Format("15:04"))
	return renderTimelineBufferBlock(w, h, text, true, color)
}

// RenderBottomCommuteBlock renders the bottom commute buffer block below the event card.
func RenderBottomCommuteBlock(t theme.Theme, w, h int, commuteMins int, endTime time.Time, isFocused bool) string {
	color := lipgloss.Color("#f9e2af")
	if isFocused {
		color = t.FocusPurple
	}
	text := fmt.Sprintf("🚗 Commute %dm (%s)", commuteMins, endTime.Format("15:04"))
	return renderTimelineBufferBlock(w, h, text, false, color)
}
