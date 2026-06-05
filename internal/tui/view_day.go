package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

const (
	rowsPerHour  = 4  // 15-minute slots per hour
	totalRows    = 96 // 24h * 4 rows
	gutterWidth  = 7  // "HH:MM  " timestamp gutter
)

// renderDayTimeline renders the 24-hour timeline grid for the day view.
// It uses a canvas-compositing approach:
//  1. Build 96 empty grid rows
//  2. For each task: render a full Lipgloss block → overlay onto canvas
//  3. Insert the NOW indicator line at the exact row
//  4. Slice to visible window and return
func (m Model) renderDayTimeline() string {
	l := m.Layout
	now := time.Now()
	isToday := sameDay(m.SelectedDay, now)

	// Grid content width (timeline minus the timestamp gutter)
	gridW := l.TimelineW - gutterWidth
	if gridW < 10 {
		gridW = 10
	}

	// ── Header ──────────────────────────────────────────────────────
	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2a2c37")).
		Render(strings.Repeat("─", l.TimelineW-2))

	dayName := lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).
		Render(m.SelectedDay.Format("Monday"))
	dayDate := lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).
		Render(m.SelectedDay.Format("January 2, 2006"))
	navHint := lipgloss.NewStyle().Foreground(m.Theme.Muted).
		Render("H ◂ · ▸ L")

	// Right-align navHint in the header line
	usedW := lipgloss.Width(dayName) + 2 + lipgloss.Width(dayDate)
	padW := (l.TimelineW - 2) - usedW - lipgloss.Width(navHint)
	if padW < 1 {
		padW = 1
	}
	headerLine := dayName + "  " + dayDate + strings.Repeat(" ", padW) + navHint

	// ── Build 96-row empty canvas ────────────────────────────────────
	// Each canvas[r] holds the raw grid content (gridW chars wide) for row r.
	// We start with empty/hour-separator rows, then overlay tasks.
	canvas := make([]string, totalRows)

	hourRowSep := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2a2c37")).
		Render(strings.Repeat("─", gridW))
	emptyRow := strings.Repeat(" ", gridW)

	for r := 0; r < totalRows; r++ {
		if r%rowsPerHour == 0 {
			canvas[r] = hourRowSep
		} else {
			canvas[r] = emptyRow
		}
	}

	// ── Resolve overlapping tasks and overlay cards ──────────────────
	var anchoredTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, m.SelectedDay) {
			anchoredTasks = append(anchoredTasks, t)
		}
	}
	cols := ResolveOverlaps(anchoredTasks)

	for _, rc := range cols {
		startRow := timeToRow(rc.Task.TimeWindow.Start)
		endRow := timeToRow(rc.Task.TimeWindow.End)
		if endRow > totalRows {
			endRow = totalRows
		}
		h := endRow - startRow
		if h < 1 {
			h = 1
		}

		// Column partition
		colW := gridW / rc.TotalCol
		colX := rc.ColIndex * colW
		if rc.ColIndex == rc.TotalCol-1 {
			colW = gridW - colX // last column takes remainder
		}
		if colW < 3 {
			colW = 3
		}

		isActive := isToday && now.After(rc.Task.TimeWindow.Start) && now.Before(rc.Task.TimeWindow.End)
		cardStr := m.renderTaskCard(rc.Task, colW, h, isActive)
		cardLines := strings.Split(cardStr, "\n")

		// Overlay card lines onto canvas rows
		for i, line := range cardLines {
			r := startRow + i
			if r >= totalRows {
				break
			}
			lineW := lipgloss.Width(line)
			// Pad or trim to colW
			if lineW < colW {
				line += strings.Repeat(" ", colW-lineW)
			}
			// Place in the correct column position within the row
			rowRunes := []rune(canvas[r])
			// Ensure the row is wide enough
			for len(rowRunes) < gridW {
				rowRunes = append(rowRunes, ' ')
			}
			// Replace the slice [colX..colX+colW] with the card line
			lineRunes := []rune(line)
			for j := 0; j < colW && colX+j < gridW; j++ {
				if j < len(lineRunes) {
					rowRunes[colX+j] = lineRunes[j]
				}
			}
			canvas[r] = string(rowRunes)
		}
	}

	// ── NOW indicator row ─────────────────────────────────────────────
	nowRow := -1
	if isToday {
		nowRow = timeToRow(now)
	}

	// ── Assemble final rows: gutter + canvas ────────────────────────
	allRows := make([]string, totalRows)
	for r := 0; r < totalRows; r++ {
		var gutter string
		if r == nowRow {
			label := fmt.Sprintf("%02d:%02d ", now.Hour(), now.Minute())
			gutter = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(label)
		} else if r%rowsPerHour == 0 {
			hour := r / rowsPerHour
			label := fmt.Sprintf("%02d:00 ", hour)
			isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == hour
			if isSelectedHour {
				gutter = lipgloss.NewStyle().
					Background(m.Theme.Accent).
					Foreground(m.Theme.CanvasBg).
					Bold(true).
					Render(label)
			} else {
				gutter = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(label)
			}
		} else {
			gutter = strings.Repeat(" ", gutterWidth)
		}

		// Replace now row canvas content with the NOW line
		rowContent := canvas[r]
		if r == nowRow {
			nowContent := buildNowLine(gridW, now)
			rowContent = nowContent
		}

		allRows[r] = gutter + rowContent
	}

	// ── Viewport slicing (scroll) ─────────────────────────────────────
	visibleH := l.Height - 4 // reserve 4 rows for header+sep+spacer+scroll hints
	if visibleH < 8 {
		visibleH = 8
	}

	centerRow := m.TimelineHour * rowsPerHour
	startR := centerRow - visibleH/2
	if startR < 0 {
		startR = 0
	}
	endR := startR + visibleH - 1
	if endR >= totalRows {
		endR = totalRows - 1
		startR = endR - visibleH + 1
		if startR < 0 {
			startR = 0
		}
	}

	var visible []string
	visible = append(visible, headerLine, sep, "")

	if startR > 0 {
		hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(
			strings.Repeat(" ", gutterWidth) + "▲ scroll up")
		visible = append(visible, hint)
	}

	for r := startR; r <= endR; r++ {
		visible = append(visible, allRows[r])
	}

	if endR < totalRows-1 {
		hint := lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(
			strings.Repeat(" ", gutterWidth) + "▼ scroll down")
		visible = append(visible, hint)
	}

	return strings.Join(visible, "\n")
}

// buildNowLine renders the NOW indicator as a full-width accent line.
func buildNowLine(width int, now time.Time) string {
	badge := fmt.Sprintf("── NOW %02d:%02d ", now.Hour(), now.Minute())
	if len(badge) > width {
		badge = "NOW "
	}
	rest := strings.Repeat("─", width-len(badge))
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Bold(true).
		Render(badge + rest)
}

// timeToRow converts a time.Time to its 15-minute row index (0–95).
func timeToRow(t time.Time) int {
	return (t.Hour() * rowsPerHour) + (t.Minute() / 15)
}

// sameDay returns true if a and b are on the same calendar day.
func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
