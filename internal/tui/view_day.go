package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

const (
	rowsPerHour = 8   // 7.5-minute slots per hour (8 rows/hour)
	totalRows   = 192 // 24h * 8 rows
	gutterWidth = 11  // " HH:MM ───┼" timestamp gutter
)

// renderDayTimeline renders the 24-hour timeline grid for the day view.
func (m Model) renderDayTimeline(appContentHeight int) string {
	l := m.Layout
	now := time.Now()
	isToday := sameDay(m.SelectedDay, now)

	// Gutter / timestamp lane is exactly 7 characters wide: " HH:MM "
	const timestampLaneW = 7
	leftSpacerW := 4
	rightSpacerW := 2

	// Grid content width (excluding timestamp gutter)
	gridW := l.TimelineW - timestampLaneW
	if gridW < 10 {
		gridW = 10
	}

	// ── Header ──────────────────────────────────────────────────────
	isTimelineFocused := !m.SidebarFocus && !m.TodoShelfFocus

	var dayName, dayDate string
	var prefix string
	var sepColor lipgloss.Color
	if isTimelineFocused {
		prefix = lipgloss.NewStyle().Foreground(m.Theme.Accent).Render("● ")
		dayName = lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).
			Render(m.SelectedDay.Format("Monday"))
		dayDate = lipgloss.NewStyle().Foreground(m.Theme.Fg).Bold(true).
			Render(m.SelectedDay.Format("January 2, 2006"))
		sepColor = m.Theme.Accent
	} else {
		prefix = "  "
		dayName = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).
			Render(m.SelectedDay.Format("Monday"))
		dayDate = lipgloss.NewStyle().Foreground(m.Theme.Muted).Bold(true).
			Render(m.SelectedDay.Format("January 2, 2006"))
		sepColor = lipgloss.Color("#2a2c37")
	}

	sep := lipgloss.NewStyle().
		Foreground(sepColor).
		Render(strings.Repeat("─", l.TimelineW-2))

	navHint := lipgloss.NewStyle().Foreground(m.Theme.Muted).
		Render("H ◂ · ▸ L")

	// Right-align navHint in the header line
	usedW := lipgloss.Width(prefix) + lipgloss.Width(dayName) + 2 + lipgloss.Width(dayDate)
	padW := (l.TimelineW - 2) - usedW - lipgloss.Width(navHint)
	if padW < 1 {
		padW = 1
	}
	headerLine := prefix + dayName + "  " + dayDate + strings.Repeat(" ", padW) + navHint

	// ── Resolve overlapping tasks and overlay cards ──────────────────
	var anchoredTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored && sameDay(t.TimeWindow.Start, m.SelectedDay) {
			anchoredTasks = append(anchoredTasks, t)
		}
	}
	cols := ResolveOverlaps(anchoredTasks)

	// Find number of overlapping columns
	numCols := 1
	for _, rc := range cols {
		if rc.TotalCol > numCols {
			numCols = rc.TotalCol
		}
	}

	// Columns area width (gridW - left spacer - right spacer)
	colsAreaW := gridW - leftSpacerW - rightSpacerW
	colW := colsAreaW / numCols
	if colW < 8 {
		colW = 8
	}

	// Rounding remainder is added to the right spacer
	actualRightSpacerW := rightSpacerW + (colsAreaW - (numCols * colW))

	nowRow := -1
	if isToday {
		nowRow = timeToRow(now)
	}

	// ── Initialize Columns ───────────────────────────────────────────
	gutterRows := make([]string, totalRows)
	leftSpacerRows := make([]string, totalRows)
	rightSpacerRows := make([]string, totalRows)
	taskRows := make([][]string, numCols)
	for c := 0; c < numCols; c++ {
		taskRows[c] = make([]string, totalRows)
	}

	for r := 0; r < totalRows; r++ {
		isHourRow := r%rowsPerHour == 0
		hour := r / rowsPerHour

		// 1. Gutter / Timestamp Lane (7 chars: " HH:MM ")
		var label string
		if r == nowRow {
			label = fmt.Sprintf(" %02d:%02d ", now.Hour(), now.Minute())
			gutterRows[r] = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(label)
		} else if isHourRow {
			label = fmt.Sprintf(" %02d:00 ", hour)
			isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == hour
			if isSelectedHour {
				gutterRows[r] = lipgloss.NewStyle().
					Foreground(m.Theme.Accent).
					Bold(true).
					Render(label)
			} else {
				gutterRows[r] = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(label)
			}
		} else {
			gutterRows[r] = "       "
		}

		// 2. Left Spacer Column (leftSpacerW chars)
		if r == nowRow {
			leftSpacerRows[r] = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", leftSpacerW))
		} else if isHourRow {
			leftSpacerRows[r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", leftSpacerW))
		} else {
			leftSpacerRows[r] = strings.Repeat(" ", leftSpacerW)
		}

		// 3. Task Columns (colW chars each)
		for c := 0; c < numCols; c++ {
			if r == nowRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", colW))
			} else if isHourRow {
				taskRows[c][r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", colW))
			} else {
				taskRows[c][r] = strings.Repeat(" ", colW)
			}
		}

		// 4. Right Spacer Column (actualRightSpacerW chars)
		if r == nowRow {
			rightSpacerRows[r] = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", actualRightSpacerW))
		} else if isHourRow {
			rightSpacerRows[r] = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", actualRightSpacerW))
		} else {
			rightSpacerRows[r] = strings.Repeat(" ", actualRightSpacerW)
		}
	}

	// Dynamic layout variables tracking position targets for auto-clamping visibility
	selectedStartRow := -1
	selectedEndRow := -1

	// Overlay tasks onto the columns
	for _, rc := range cols {
		startRow := timeToRow(rc.Task.TimeWindow.Start)
		endRow := timeToRow(rc.Task.TimeWindow.End)
		
		// Map structural height accurately across row milestones
		h := endRow - startRow + 1
		
		// Compensate for top and bottom border frames generated by Lipgloss
		if h < 1 {
			h = 1
		}

		if startRow+h > totalRows {
			h = totalRows - startRow
		}

		colIndex := rc.ColIndex
		if colIndex >= numCols {
			colIndex = numCols - 1
		}

		isActive := isToday && now.After(rc.Task.TimeWindow.Start) && now.Before(rc.Task.TimeWindow.End)
		isSelected := !m.TodoShelfFocus && !m.SidebarFocus && rc.Task.UUID == m.SelectedTaskUUID
		
		// Capture exact coordinates of focused task card
		if isSelected {
			selectedStartRow = startRow
			selectedEndRow = startRow + h
		}

		cardStr := m.renderTaskCard(rc.Task, colW, h, isActive, isSelected)
		cardLines := strings.Split(cardStr, "\n")

		for i, line := range cardLines {
			r := startRow + i
			if r >= totalRows {
				break
			}
			taskRows[colIndex][r] = line
		}
	}

	// ── Assemble all rows ────────────────────────────────────────────
	allRows := make([]string, totalRows)
	for r := 0; r < totalRows; r++ {
		var sb strings.Builder
		sb.WriteString(gutterRows[r])
		sb.WriteString(leftSpacerRows[r])
		for c := 0; c < numCols; c++ {
			sb.WriteString(taskRows[c][r])
		}
		sb.WriteString(rightSpacerRows[r])
		allRows[r] = sb.String()
	}

	// ── Seamless Viewport Looping Slicing ────────────────────────────
	visibleH := appContentHeight
	if visibleH < 8 {
		visibleH = 8
	}

	centerRow := m.TimelineHour * rowsPerHour
	startR := centerRow - visibleH/2

	// ── Smart Focus Tracking Adjustment Mechanism ───────────────────
	if selectedStartRow != -1 {
		endR := startR + visibleH

		if selectedStartRow < startR {
			startR = selectedStartRow
		}	
		if selectedEndRow > endR {
			startR = selectedEndRow - visibleH
		}
	}

	var visible []string
	visible = append(visible, headerLine, sep, "")

	for i := 0; i < visibleH; i++ {
		rowIndex := startR + i
		r := (rowIndex%totalRows + totalRows) % totalRows
		visible = append(visible, allRows[r])
	}

	return strings.Join(visible, "\n")
}

// buildNowLine renders the NOW indicator as a full-width accent line.
func (m Model) buildNowLine(width int, now time.Time) string {
	badge := fmt.Sprintf("── NOW • %02d:%02d ──", now.Hour(), now.Minute())
	if len(badge) > width {
		badge = "NOW "
	}
	rest := strings.Repeat("─", width-len([]rune(badge)))
	return lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).
		Render(badge + rest)
}

// timeToRow converts a time.Time to its local day row index (0 to totalRows-1).
func timeToRow(t time.Time) int {
	local := t.Local()
	return (local.Hour() * rowsPerHour) + (local.Minute() * rowsPerHour / 60)
}

// sameDay returns true if a and b are on the same calendar day in local time.
func sameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}
