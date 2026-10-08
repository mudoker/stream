package viewmodel

import (
	"fmt"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/tasks"
)

func (m *Model) handleDashboardOrAnalyticsNav(key string) {
	if m.CurrentView == DashboardView {
		switch key {
		case "h", "left":
			m.DashboardFocusCol = 0
		case "l", "right":
			m.DashboardFocusCol = 1
		case "k", "up":
			m.DashboardFocusRow--
			if m.DashboardFocusRow < 0 {
				m.DashboardFocusRow = 2
			}
		case "j", "down":
			m.DashboardFocusRow = (m.DashboardFocusRow + 1) % 3
		}

		availH := m.Height - 11
		if availH < 10 {
			availH = 10
		}

		var rowHeights []int
		if availH > 45 {
			contentH := availH - 6
			if contentH < 3 {
				contentH = 3
			}
			rowHeights = PartitionHeights(contentH, 3)
			for i := range rowHeights {
				rowHeights[i] += 2
			}
		} else {
			rowHeights = []int{15, 15, 15}
		}

		var yStart, yEnd int
		for i := 0; i < m.DashboardFocusRow; i++ {
			yStart += rowHeights[i]
		}
		yEnd = yStart + rowHeights[m.DashboardFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+availH {
			m.ScrollOffset = yEnd - availH
		}
	} else if m.CurrentView == AnalyticsView {
		totalLayers := 6
		switch key {
		case "h", "left":
			m.AnalyticsFocusCol = 0
		case "l", "right":
			m.AnalyticsFocusCol = 1
		case "k", "up":
			m.AnalyticsFocusRow--
			if m.AnalyticsFocusRow < 0 {
				m.AnalyticsFocusRow = totalLayers - 1
			}
		case "j", "down":
			m.AnalyticsFocusRow = (m.AnalyticsFocusRow + 1) % totalLayers
		}

		yStart := m.AnalyticsFocusRow * 13
		yEnd := (m.AnalyticsFocusRow + 1) * 13

		gridHeight := m.Height - 11
		if gridHeight < 10 {
			gridHeight = 10
		}

		defaultTotalH := 78
		rowHeights := make([]int, totalLayers)
		if gridHeight > defaultTotalH {
			contentH := gridHeight - 12
			if contentH < 6 {
				contentH = 6
			}
			rowHeights = PartitionHeights(contentH, totalLayers)
			for i := range rowHeights {
				rowHeights[i] += 2
			}
		} else {
			for i := range rowHeights {
				rowHeights[i] = 13
			}
		}

		var yStartAcc int
		for i := 0; i < m.AnalyticsFocusRow; i++ {
			yStartAcc += rowHeights[i]
		}
		yStart = yStartAcc
		yEnd = yStartAcc + rowHeights[m.AnalyticsFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+gridHeight {
			m.ScrollOffset = yEnd - gridHeight
		}
	}
}

func (m *Model) handleMonthNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "j":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	case "k":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, -1, 0)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 1, 0)
	case "enter":
		m.CurrentView = DayView
	}
}

func (m *Model) handleWeekNav(key string) {
	switch key {
	case "h", "left":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
		m.selectDefaultTaskForSelectedWeekDay()
		m.AutoScrollToSelectedWeekTask()
	case "l", "right":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
		m.selectDefaultTaskForSelectedWeekDay()
		m.AutoScrollToSelectedWeekTask()
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
		m.selectDefaultTaskForSelectedWeekDay()
		m.AutoScrollToSelectedWeekTask()
	case "L", "K":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
		m.selectDefaultTaskForSelectedWeekDay()
		m.AutoScrollToSelectedWeekTask()
	case "j", "down":
		m.ScrollOffset++
		dayTasks := m.getWeekDayTasks(m.SelectedDay)
		if len(dayTasks) > 0 {
			curIdx := -1
			for idx, t := range dayTasks {
				if t.UUID == m.SelectedTaskUUID {
					curIdx = idx
					break
				}
			}
			if curIdx == -1 {
				m.SelectedTaskUUID = dayTasks[0].UUID
			} else {
				nextIdx := (curIdx + 1) % len(dayTasks)
				m.SelectedTaskUUID = dayTasks[nextIdx].UUID
			}
			m.AutoScrollToSelectedWeekTask()
		}
	case "k", "up":
		m.ScrollOffset--
		if m.ScrollOffset < 0 {
			m.ScrollOffset = 0
		}
		dayTasks := m.getWeekDayTasks(m.SelectedDay)
		if len(dayTasks) > 0 {
			curIdx := -1
			for idx, t := range dayTasks {
				if t.UUID == m.SelectedTaskUUID {
					curIdx = idx
					break
				}
			}
			if curIdx == -1 {
				m.SelectedTaskUUID = dayTasks[len(dayTasks)-1].UUID
			} else {
				prevIdx := (curIdx - 1 + len(dayTasks)) % len(dayTasks)
				m.SelectedTaskUUID = dayTasks[prevIdx].UUID
			}
			m.AutoScrollToSelectedWeekTask()
		}
	}
}

func (m *Model) getWeekDayTasks(day time.Time) []model.Task {
	var dayTasks []model.Task
	for _, task := range m.Tasks {
		if model.IsTaskAnchored(task) && SameDay(task.TimeWindow.Start, day) {
			dayTasks = append(dayTasks, task)
		}
	}
	var sorted []model.Task
	resolved := ResolveOverlaps(dayTasks)
	for _, rc := range resolved {
		sorted = append(sorted, rc.Task)
	}
	return sorted
}

func (m *Model) selectDefaultTaskForSelectedWeekDay() {
	dayTasks := m.getWeekDayTasks(m.SelectedDay)
	if len(dayTasks) > 0 {
		m.SelectedTaskUUID = dayTasks[0].UUID
	} else {
		m.SelectedTaskUUID = ""
	}
}

func (m *Model) getWeekAvailLaneH() int {
	appContentHeight := m.Height - 1
	if appContentHeight < 10 {
		appContentHeight = 10
	}

	height := appContentHeight - 2
	laneHeight := height - 4
	if laneHeight < 10 {
		laneHeight = 10
	}

	availLaneH := laneHeight - 2
	if availLaneH < 1 {
		availLaneH = 1
	}
	return availLaneH
}

func (m *Model) AutoScrollToSelectedWeekTask() {
	if m.SelectedTaskUUID == "" {
		return
	}

	dayTasks := m.getWeekDayTasks(m.SelectedDay)
	if len(dayTasks) == 0 {
		return
	}

	taskStart := -1
	taskEnd := -1
	lineIdx := 0

	for idx, task := range dayTasks {
		dur := task.TimeWindow.End.Sub(task.TimeWindow.Start)
		durMins := int(dur.Minutes())
		outerH := (durMins * 6) / 60
		if outerH < 4 {
			outerH = 4
		}

		if idx > 0 {
			prevTask := dayTasks[idx-1]
			if task.TimeWindow.Start.Equal(prevTask.TimeWindow.End) {
				// Contiguous - do not add empty line spacing
			} else {
				lineIdx += 1 // empty line spacing
			}
		}

		if task.UUID == m.SelectedTaskUUID {
			taskStart = lineIdx
			taskEnd = lineIdx + outerH
			break
		}

		lineIdx += outerH
	}

	if taskStart == -1 {
		return
	}

	availLaneH := m.getWeekAvailLaneH()

	if taskStart < m.ScrollOffset {
		m.ScrollOffset = taskStart
	} else if taskEnd > m.ScrollOffset+availLaneH {
		m.ScrollOffset = taskEnd - availLaneH
	}
}

func (m *Model) handleDayNav(key string) {
	switch key {
	case "h":
		if !m.TodoShelfFocus {
			m.NavigateHorizontal(-1)
			m.AutoScrollToSelectedTask()
		}
	case "l":
		if !m.TodoShelfFocus {
			m.NavigateHorizontal(1)
			m.AutoScrollToSelectedTask()
		}
	case "j":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(1)
		} else {
			m.NavigateVertical(1)
			m.AutoScrollToSelectedTask()
		}
	case "k":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(-1)
		} else {
			m.NavigateVertical(-1)
			m.AutoScrollToSelectedTask()
		}
	case "J":
		if m.TodoShelfFocus {
			m.MoveShelfSection(1)
		} else {
			m.TimelineHour = (m.TimelineHour + 1) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "K":
		if m.TodoShelfFocus {
			m.MoveShelfSection(-1)
		} else {
			m.TimelineHour = (m.TimelineHour - 1 + 24) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
		m.selectDefaultTaskForSelectedDay()
		m.AutoScrollToSelectedTask()
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
		m.selectDefaultTaskForSelectedDay()
		m.AutoScrollToSelectedTask()
	}
}

func (m *Model) handleSprintNav(key string) {
	activeSprint, ok := m.GetActiveSprint()
	if !ok {
		return
	}

	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	lanes := [][]model.Task{defined, inProgress, review, testing, completed}

	switch key {
	case "h", "left":
		if !m.TodoShelfFocus && !m.SidebarFocus {
			curLane := lanes[m.SprintSwimlaneIdx]
			curIdx := 0
			for i, t := range curLane {
				if t.UUID == m.SelectedTaskUUID {
					curIdx = i
					break
				}
			}

			targetLaneIdx := -1
			for idx := m.SprintSwimlaneIdx - 1; idx >= 0; idx-- {
				if len(lanes[idx]) > 0 {
					targetLaneIdx = idx
					break
				}
			}

			if targetLaneIdx != -1 {
				m.SprintSwimlaneIdx = targetLaneIdx
				targetLane := lanes[targetLaneIdx]
				newIdx := curIdx
				if newIdx >= len(targetLane) {
					newIdx = len(targetLane) - 1
				}
				m.SelectedTaskUUID = targetLane[newIdx].UUID
				m.AutoScrollSprintLane()
				m.StatusMsg = fmt.Sprintf("Switched to %s swimlane.", sprintLaneName(targetLaneIdx))
			}
		}
	case "l", "right":
		if !m.TodoShelfFocus && !m.SidebarFocus {
			curLane := lanes[m.SprintSwimlaneIdx]
			curIdx := 0
			for i, t := range curLane {
				if t.UUID == m.SelectedTaskUUID {
					curIdx = i
					break
				}
			}

			targetLaneIdx := -1
			for idx := m.SprintSwimlaneIdx + 1; idx < len(lanes); idx++ {
				if len(lanes[idx]) > 0 {
					targetLaneIdx = idx
					break
				}
			}

			if targetLaneIdx != -1 {
				m.SprintSwimlaneIdx = targetLaneIdx
				targetLane := lanes[targetLaneIdx]
				newIdx := curIdx
				if newIdx >= len(targetLane) {
					newIdx = len(targetLane) - 1
				}
				m.SelectedTaskUUID = targetLane[newIdx].UUID
				m.AutoScrollSprintLane()
				m.StatusMsg = fmt.Sprintf("Switched to %s swimlane.", sprintLaneName(targetLaneIdx))
			}
		}
	case "j", "down":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(1)
		} else {
			curLane := lanes[m.SprintSwimlaneIdx]
			if len(curLane) > 0 {
				curIdx := -1
				for i, t := range curLane {
					if t.UUID == m.SelectedTaskUUID {
						curIdx = i
						break
					}
				}
				if curIdx == -1 {
					m.SelectedTaskUUID = curLane[0].UUID
				} else {
					nextIdx := (curIdx + 1) % len(curLane)
					m.SelectedTaskUUID = curLane[nextIdx].UUID
				}
			}
		}
	case "k", "up":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(-1)
		} else {
			curLane := lanes[m.SprintSwimlaneIdx]
			if len(curLane) > 0 {
				curIdx := -1
				for i, t := range curLane {
					if t.UUID == m.SelectedTaskUUID {
						curIdx = i
						break
					}
				}
				if curIdx == -1 {
					m.SelectedTaskUUID = curLane[len(curLane)-1].UUID
				} else {
					prevIdx := (curIdx - 1 + len(curLane)) % len(curLane)
					m.SelectedTaskUUID = curLane[prevIdx].UUID
				}
			}
		}
	case "J":
		if m.TodoShelfFocus {
			m.MoveShelfSection(1)
		}
	case "K":
		if m.TodoShelfFocus {
			m.MoveShelfSection(-1)
		}
	case "H":
		if !m.TodoShelfFocus {
			// Switch active swimlane to the left
			newLaneIdx := m.SprintSwimlaneIdx - 1
			if newLaneIdx < 0 {
				newLaneIdx = 4
			}
			m.SprintSwimlaneIdx = newLaneIdx
			m.AutoScrollSprintLane()
			curLane := lanes[m.SprintSwimlaneIdx]
			if len(curLane) > 0 {
				m.SelectedTaskUUID = curLane[0].UUID
			} else {
				m.SelectedTaskUUID = ""
			}
			m.StatusMsg = fmt.Sprintf("Switched to %s swimlane.", sprintLaneName(newLaneIdx))
		}
	case "L":
		if !m.TodoShelfFocus {
			// Switch active swimlane to the right
			newLaneIdx := m.SprintSwimlaneIdx + 1
			if newLaneIdx > 4 {
				newLaneIdx = 0
			}
			m.SprintSwimlaneIdx = newLaneIdx
			m.AutoScrollSprintLane()
			curLane := lanes[m.SprintSwimlaneIdx]
			if len(curLane) > 0 {
				m.SelectedTaskUUID = curLane[0].UUID
			} else {
				m.SelectedTaskUUID = ""
			}
			m.StatusMsg = fmt.Sprintf("Switched to %s swimlane.", sprintLaneName(newLaneIdx))
		}
	case "s", "]":
		if !m.TodoShelfFocus && !m.SidebarFocus {
			if len(m.Sprints) > 1 {
				idx := -1
				for i, s := range m.Sprints {
					if s.UUID == m.ActiveSprintUUID {
						idx = i
						break
					}
				}
				if idx != -1 {
					nextIdx := (idx + 1) % len(m.Sprints)
					m.ActiveSprintUUID = m.Sprints[nextIdx].UUID
					m.StatusMsg = fmt.Sprintf("Switched to sprint '%s'.", m.Sprints[nextIdx].Name)
				}
			}
		}
	case "S", "[":
		if !m.TodoShelfFocus && !m.SidebarFocus {
			if len(m.Sprints) > 1 {
				idx := -1
				for i, s := range m.Sprints {
					if s.UUID == m.ActiveSprintUUID {
						idx = i
						break
					}
				}
				if idx != -1 {
					prevIdx := (idx - 1 + len(m.Sprints)) % len(m.Sprints)
					m.ActiveSprintUUID = m.Sprints[prevIdx].UUID
					m.StatusMsg = fmt.Sprintf("Switched to sprint '%s'.", m.Sprints[prevIdx].Name)
				}
			}
		}
	}
}

func sprintLaneName(idx int) string {
	switch idx {
	case 0:
		return "Defined"
	case 1:
		return "In Progress"
	case 2:
		return "Review"
	case 3:
		return "Testing"
	case 4:
		return "Completed"
	default:
		return "Defined"
	}
}

func (m *Model) AutoScrollSprintLane() {
	numCols := 5
	availW := m.Layout.TimelineW - 2
	minColW := 28
	if availW < minColW {
		availW = minColW
	}

	visibleCols := (availW + 1) / (minColW + 1)
	if visibleCols < 1 {
		visibleCols = 1
	}
	if visibleCols > numCols {
		visibleCols = numCols
	}

	maxOffset := numCols - visibleCols
	if maxOffset < 0 {
		maxOffset = 0
	}

	if m.SprintSwimlaneIdx < m.SprintScrollColOffset {
		m.SprintScrollColOffset = m.SprintSwimlaneIdx
	} else if m.SprintSwimlaneIdx >= m.SprintScrollColOffset+visibleCols {
		m.SprintScrollColOffset = m.SprintSwimlaneIdx - visibleCols + 1
	}

	if m.SprintScrollColOffset > maxOffset {
		m.SprintScrollColOffset = maxOffset
	}
	if m.SprintScrollColOffset < 0 {
		m.SprintScrollColOffset = 0
	}
}

