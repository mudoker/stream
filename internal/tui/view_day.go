package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"stream/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDayView(height int) string {
	sidebarWidth := int(float64(m.Width) * 0.13)
	if sidebarWidth < 18 {
		sidebarWidth = 18
	} else if sidebarWidth > 26 {
		sidebarWidth = 26
	}
	workspaceWidth := m.Width - sidebarWidth - 3
	if workspaceWidth < 30 {
		workspaceWidth = 30
	}

	// 75% Timeline, 25% Todo Shelf
	timelineWidth := int(float64(workspaceWidth) * 0.75)
	shelfWidth := workspaceWidth - timelineWidth - 4
	if timelineWidth < 30 {
		timelineWidth = 30
	}
	if shelfWidth < 20 {
		shelfWidth = 20
	}

	var anchoredTasks []model.Task
	for _, t := range m.Tasks {
		if t.SchedulingType == model.Anchored &&
			t.TimeWindow.Start.Year() == m.SelectedDay.Year() &&
			t.TimeWindow.Start.Month() == m.SelectedDay.Month() &&
			t.TimeWindow.Start.Day() == m.SelectedDay.Day() {
			anchoredTasks = append(anchoredTasks, t)
		}
	}

	cols := ResolveOverlaps(anchoredTasks)

	now := time.Now()
	isToday := m.SelectedDay.Year() == now.Year() && m.SelectedDay.Month() == now.Month() && m.SelectedDay.Day() == now.Day()

	// Content area width for the timeline grid (excluding 6 char timestamp)
	W := timelineWidth - 6
	if W < 10 {
		W = 10
	}

	// Calculate 24 hours * 4 rows/hour = 96 rows total
	var timelineLines []string

	// Add title header
	headerText := fmt.Sprintf("DAILY TIMELINE  /  %s", strings.ToUpper(m.SelectedDay.Format("Monday, Jan _2")))
	timelineLines = append(timelineLines, lipgloss.NewStyle().Foreground(m.Theme.Accent).Bold(true).Render(headerText)+"\n")

	// Calculate "NOW" indicator row index
	nowRow := -1
	if isToday {
		nowRow = (now.Hour() * 4) + (now.Minute() / 15)
	}

	// Render the 96 rows (00:00 to 23:45)
	for r := 0; r < 96; r++ {
		// Find all active tasks overlapping row r
		type ActiveTaskCol struct {
			ColIndex int
			TotalCol int
			Task     model.Task
		}
		var activeTasks []ActiveTaskCol
		for _, rc := range cols {
			startRow := (rc.Task.TimeWindow.Start.Hour() * 4) + (rc.Task.TimeWindow.Start.Minute() / 15)
			endRow := (rc.Task.TimeWindow.End.Hour() * 4) + (rc.Task.TimeWindow.End.Minute() / 15)
			if r >= startRow && r < endRow {
				activeTasks = append(activeTasks, ActiveTaskCol{
					ColIndex: rc.ColIndex,
					TotalCol: rc.TotalCol,
					Task:     rc.Task,
				})
			}
		}

		// Partition W into segments
		type RowSegment struct {
			Start int
			End   int
			Text  string
		}
		var segments []RowSegment

		for _, at := range activeTasks {
			startRow := (at.Task.TimeWindow.Start.Hour() * 4) + (at.Task.TimeWindow.Start.Minute() / 15)
			endRow := (at.Task.TimeWindow.End.Hour() * 4) + (at.Task.TimeWindow.End.Minute() / 15)
			h := endRow - startRow

			colStart := (at.ColIndex * W) / at.TotalCol
			colEnd := ((at.ColIndex + 1) * W) / at.TotalCol
			w := colEnd - colStart

			isActiveBlock := isToday && now.After(at.Task.TimeWindow.Start) && now.Before(at.Task.TimeWindow.End)

			// Render the segment of the task card at line relative to startRow
			isNowRow := (r == nowRow)
			isLeftmost := (colStart == 0)
			lineText := m.renderTaskCardLine(at.Task, w, h, r-startRow, isActiveBlock, isNowRow, isLeftmost)
			segments = append(segments, RowSegment{Start: colStart, End: colEnd, Text: lineText})
		}

		// Sort segments by Start position
		sort.Slice(segments, func(i, j int) bool {
			return segments[i].Start < segments[j].Start
		})

		// Fill in the gaps with guides/empty space
		var rowParts []string
		curr := 0
		for _, seg := range segments {
			if seg.Start > curr {
				gapW := seg.Start - curr
				var gapText string
				if r == nowRow {
					if curr == 0 {
						badge := getNowBadge(gapW, now)
						gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(badge + strings.Repeat("─", gapW-len(badge)))
					} else {
						gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", gapW))
					}
				} else if r%4 == 0 {
					gapText = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", gapW))
				} else {
					gapText = strings.Repeat(" ", gapW)
				}
				rowParts = append(rowParts, gapText)
			}
			rowParts = append(rowParts, seg.Text)
			curr = seg.End
		}
		if curr < W {
			gapW := W - curr
			var gapText string
			if r == nowRow {
				if curr == 0 {
					badge := getNowBadge(gapW, now)
					gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(badge + strings.Repeat("─", gapW-len(badge)))
				} else {
					gapText = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(strings.Repeat("─", gapW))
				}
			} else if r%4 == 0 {
				gapText = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(strings.Repeat("─", gapW))
			} else {
				gapText = strings.Repeat(" ", gapW)
			}
			rowParts = append(rowParts, gapText)
		}

		// Clean left-aligned timestamp (removing vertical │)
		var hourLabel string
		isSelectedHour := !m.TodoShelfFocus && m.TimelineHour == (r/4) && (r%4 == 0)
		if r == nowRow {
			timeLabel := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
			hourLabel = lipgloss.NewStyle().Foreground(m.Theme.SuccessColor).Bold(true).Render(timeLabel) + " "
		} else if r%4 == 0 {
			timeLabel := fmt.Sprintf("%02d:00", r/4)
			if isSelectedHour {
				hourLabel = lipgloss.NewStyle().Background(m.Theme.Accent).Foreground(m.Theme.CanvasBg).Bold(true).Render(timeLabel) + " "
			} else {
				hourLabel = lipgloss.NewStyle().Foreground(m.Theme.Muted).Render(timeLabel) + " "
			}
		} else {
			hourLabel = "      "
		}

		timelineLines = append(timelineLines, hourLabel+strings.Join(rowParts, ""))
	}

	// Scroll timeline dynamically centering around m.TimelineHour
	timelineStartRow := 8 * 4 // Default start at 8:00 AM
	maxRowsVisible := height - 4
	if maxRowsVisible < 10 {
		maxRowsVisible = 10
	}

	// Center around m.TimelineHour
	targetCenterRow := m.TimelineHour * 4
	timelineStartRow = targetCenterRow - maxRowsVisible/2
	if timelineStartRow < 0 {
		timelineStartRow = 0
	}
	timelineEndRow := timelineStartRow + maxRowsVisible - 1
	if timelineEndRow > 95 {
		timelineEndRow = 95
		timelineStartRow = timelineEndRow - maxRowsVisible + 1
		if timelineStartRow < 0 {
			timelineStartRow = 0
		}
	}

	var visibleTimelineLines []string
	visibleTimelineLines = append(visibleTimelineLines, timelineLines[0]) // Header
	if timelineStartRow > 0 {
		visibleTimelineLines = append(visibleTimelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("      ▲  (scroll up)"))
	}

	for r := timelineStartRow; r <= timelineEndRow; r++ {
		visibleTimelineLines = append(visibleTimelineLines, timelineLines[r+1])
	}

	if timelineEndRow < 95 {
		visibleTimelineLines = append(visibleTimelineLines, lipgloss.NewStyle().Foreground(m.Theme.Muted).Render("      ▼  (scroll down)"))
	}

	leftBox := m.Theme.PanelStyle.
		Width(timelineWidth - 4).
		Height(height - 2).
		Render(strings.Join(visibleTimelineLines, "\n"))

	rightBox := m.renderTodoShelf(shelfWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "    ", rightBox)
}
