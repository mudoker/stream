package components

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel"
	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderCard renders a complete multi-line Lipgloss card block.
// w and h are the card's outer dimensions (including borders).
func RenderCard(m *viewmodel.Model, t theme.Theme, task model.Task, w, h int, isActive bool, isSelected bool) string {
	pColor := t.PriorityColor(task.Priority)
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
		pColor = t.Muted
	}
	now := time.Now()

	hasCollision := m.HasPriorityOverlapCollision(task)
	isZenFocus := m.ZenTimer != nil && m.ZenTimer.Running && m.ZenTimer.Task.UUID == task.UUID

	// Card border
	borderColor := pColor
	isCompleted := task.LifecycleState == model.StateCompleted
	if strings.HasSuffix(task.UUID, "_moving") || strings.HasSuffix(task.UUID, "_adjusting") {
		borderColor = t.Muted
	} else if isZenFocus {
		borderColor = t.SuccessColor
	} else if isSelected {
		borderColor = t.FocusPurple
	} else if hasCollision {
		borderColor = lipgloss.Color("#ff0000")
	} else if isActive {
		borderColor = t.Accent
	} else if isCompleted {
		borderColor = lipgloss.Color("#4c644f") // Soft dark forest green border for completed
	}

	// Priority badge
	pName := string(task.Priority)
	priorityBadge := lipgloss.NewStyle().Foreground(pColor).Bold(true).Render("▲ " + pName)
	if hasCollision {
		priorityBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true).Render("⚠️ " + pName)
	}

	// Time range string
	timeStr := fmt.Sprintf("⏱ %s → %s",
		task.TimeWindow.Start.Format("15:04"),
		task.TimeWindow.End.Format("15:04"),
	)
	if isActive {
		remaining := task.TimeWindow.End.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		h2 := int(remaining.Hours())
		mi := int(remaining.Minutes()) % 60
		s := int(remaining.Seconds()) % 60
		timeStr = fmt.Sprintf("⏱ %02d:%02d:%02d remaining", h2, mi, s)
	}

	// If card is very short (h < 3), use the left strip card
	if h < 3 {
		return RenderShortCard(m, t, task, w, h, pColor, isActive, isSelected, hasCollision, isZenFocus, timeStr)
	}

	if w < 3 {
		w = 3
	}
	if h < 3 {
		h = 3
	}

	// Determine padding based on height
	paddingTopBottom := 0
	if h >= 7 {
		paddingTopBottom = 1
	}

	paddingLeftRight := 2
	if w < 10 {
		paddingLeftRight = 0
	} else if w < 14 {
		paddingLeftRight = 1
	}

	// Content width inside padding (2 border left/right, and 2 * paddingLeftRight)
	contentW := w - 2 - (2 * paddingLeftRight)
	if contentW < 1 {
		contentW = 1
	}

	contentH := h - 2 - (2 * paddingTopBottom)
	if contentH < 1 {
		contentH = 1
	}

	// Truncate title using safe rune slicing
	titleStr := cardTitleStr(task, isCompleted)
	titleRunes := []rune(titleStr)
	if len(titleRunes) > contentW-1 {
		if contentW > 2 {
			titleStr = string(titleRunes[:contentW-2]) + "…"
		} else {
			titleStr = string(titleRunes[:contentW-1])
		}
	}

	// Construct and scale metadata row to fit contentW-1
	metaStr := cardMetaStr(t, task, contentW, priorityBadge, timeStr)

	titleStyle := cardTitleStyle(t, task, isZenFocus, isSelected, hasCollision, isCompleted)
	titleLine := titleStyle.Render(titleStr)
	metaLine := metaStr

	innerWidth := contentW + (2 * paddingLeftRight)
	if innerWidth < 1 {
		innerWidth = 1
	}

	contentAreaW := contentW
	if contentAreaW < 1 {
		contentAreaW = 1
	}

	var contentLines []string
	if task.SchedulingType == model.Event && task.Location != "" {
		locStr := "📍 " + task.Location
		locRunes := []rune(locStr)
		if len(locRunes) > contentAreaW {
			if contentAreaW > 2 {
				locStr = string(locRunes[:contentAreaW-1]) + "…"
			} else {
				locStr = string(locRunes[:contentAreaW])
			}
		}
		locLine := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render(locStr)
		if h == 3 {
			contentLines = []string{titleLine}
		} else if h == 4 {
			contentLines = []string{titleLine, locLine}
		} else if h == 5 {
			contentLines = []string{titleLine, locLine, metaLine}
		} else {
			sepLine := strings.Repeat("─", contentAreaW)
			contentLines = []string{titleLine, sepLine, locLine, metaLine}
		}
	} else {
		if h == 3 {
			contentLines = []string{titleLine}
		} else if h == 4 {
			contentLines = []string{titleLine, metaLine}
		} else {
			sepLine := strings.Repeat("─", contentAreaW)
			contentLines = []string{titleLine, sepLine, metaLine}
		}
	}

	heightContent := h - 2
	if heightContent < 1 {
		heightContent = 1
	}

	var bodyLines []string
	for i := 0; i < paddingTopBottom; i++ {
		bodyLines = append(bodyLines, strings.Repeat(" ", innerWidth))
	}

	for i, line := range contentLines {
		if len(bodyLines) >= heightContent-paddingTopBottom {
			break
		}
		visual := lipgloss.Width(line)
		if visual > contentAreaW {
			line = theme.SliceAnsi(line, 0, contentAreaW)
		} else if visual < contentAreaW {
			line += strings.Repeat(" ", contentAreaW-visual)
		}
		if paddingLeftRight > 0 {
			line = strings.Repeat(" ", paddingLeftRight) + line + strings.Repeat(" ", paddingLeftRight)
		}
		bodyLines = append(bodyLines, line)
		if i == len(contentLines)-1 {
			break
		}
	}

	for len(bodyLines) < heightContent {
		bodyLines = append(bodyLines, strings.Repeat(" ", innerWidth))
	}

	restDur := viewmodel.CalculateTaskRestTime(task)
	hasRest := restDur > 0 && task.SchedulingType == model.Anchored

	topLeftChar, topRightChar, bottomLeftChar, bottomRightChar, horizChar, vertChar := cardBorderChars(task, hasRest)

	// Check if this task has consecutive predecessors in overlapping columns
	hasLeftConsecutive := false
	hasRightConsecutive := false
	if !strings.HasSuffix(task.UUID, "_moving") && !strings.HasSuffix(task.UUID, "_adjusting") {
		clones := make(map[string]bool)
		for _, tVal := range m.Tasks {
			if strings.HasSuffix(tVal.UUID, "_moving") {
				clones[strings.TrimSuffix(tVal.UUID, "_moving")] = true
			} else if strings.HasSuffix(tVal.UUID, "_adjusting") {
				clones[strings.TrimSuffix(tVal.UUID, "_adjusting")] = true
			}
		}

		var dayTasks []model.Task
		for _, tVal := range m.Tasks {
			if clones[tVal.UUID] {
				continue
			}
			if model.IsTaskAnchored(tVal) && viewmodel.SameDay(tVal.TimeWindow.Start, task.TimeWindow.Start) {
				dayTasks = append(dayTasks, tVal)
			}
		}
		cols := viewmodel.ResolveOverlaps(dayTasks)

		numCols := 1
		for _, rc := range cols {
			if rc.TotalCol > numCols {
				numCols = rc.TotalCol
			}
		}

		var currRc *viewmodel.ScheduledColumn
		for i := range cols {
			if cols[i].Task.UUID == task.UUID {
				currRc = &cols[i]
				break
			}
		}

		if currRc != nil {
			for _, other := range cols {
				if other.Task.UUID != task.UUID {
					if strings.HasSuffix(other.Task.UUID, "_moving") || strings.HasSuffix(other.Task.UUID, "_adjusting") {
						continue
					}
					predEnd := other.Task.TimeWindow.End
					if other.Task.SchedulingType == model.Event && strings.TrimSpace(other.Task.Location) != "" && other.Task.CommuteBuffer > 0 {
						predEnd = predEnd.Add(time.Duration(other.Task.CommuteBuffer) * time.Minute)
					}
					restDur := viewmodel.CalculateTaskRestTime(other.Task)
					if restDur > 0 {
						predEnd = predEnd.Add(restDur)
					}

					currStart := task.TimeWindow.Start
					if task.SchedulingType == model.Event && strings.TrimSpace(task.Location) != "" && task.CommuteBuffer > 0 {
						currStart = currStart.Add(-time.Duration(task.CommuteBuffer) * time.Minute)
					}

					if predEnd.Equal(currStart) {
						currColStart := currRc.ColIndex
						otherColStart := other.ColIndex
						if currRc.TotalCol == 1 {
							currColStart = 0
						}
						if other.TotalCol == 1 {
							otherColStart = 0
						}
						if currColStart * other.TotalCol == otherColStart * currRc.TotalCol {
							hasLeftConsecutive = true
						}

						currColEnd := currRc.ColIndex + 1
						otherColEnd := other.ColIndex + 1
						currTotalCol := currRc.TotalCol
						otherTotalCol := other.TotalCol
						if currRc.TotalCol == 1 {
							currColEnd = 1
							currTotalCol = 1
						}
						if other.TotalCol == 1 {
							otherColEnd = 1
							otherTotalCol = 1
						}
						if currColEnd * otherTotalCol == otherColEnd * currTotalCol {
							hasRightConsecutive = true
						}
					}
				}
			}
		}
	}

	if hasLeftConsecutive {
		topLeftChar = "├"
	}
	if hasRightConsecutive {
		topRightChar = "┤"
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	topLine := borderStyle.Render(topLeftChar) + borderStyle.Render(strings.Repeat(horizChar, innerWidth)) + borderStyle.Render(topRightChar)

	bottomHorizChar := horizChar
	if hasRest {
		bottomHorizChar = "╌"
	}
	bottomLine := borderStyle.Render(bottomLeftChar) + borderStyle.Render(strings.Repeat(bottomHorizChar, innerWidth)) + borderStyle.Render(bottomRightChar)

	var cardLines []string
	cardLines = append(cardLines, topLine)
	for _, body := range bodyLines {
		cardLines = append(cardLines, borderStyle.Render(vertChar)+body+borderStyle.Render(vertChar))
	}
	cardLines = append(cardLines, bottomLine)

	return strings.Join(cardLines, "\n")
}
