package tui

import (
	"stream/internal/model"
)

func (m *Model) moveTaskSelection(dir int) {
	shelf := m.getTodoShelfTasks()
	if len(shelf) == 0 {
		return
	}

	idx := -1
	for i, t := range shelf {
		if t.UUID == m.SelectedTaskUUID {
			idx = i
			break
		}
	}

	if idx == -1 {
		m.SelectedTaskUUID = shelf[0].UUID
		return
	}

	idx += dir
	if idx < 0 {
		idx = len(shelf) - 1
	} else if idx >= len(shelf) {
		idx = 0
	}
	m.SelectedTaskUUID = shelf[idx].UUID
}

func (m *Model) selectDefaultTaskForSelectedDay() {
	if m.TodoShelfFocus {
		shelf := m.getTodoShelfTasks()
		if len(shelf) > 0 {
			m.SelectedTaskUUID = shelf[0].UUID
		} else {
			m.SelectedTaskUUID = ""
		}
		return
	}
	dayTasks := m.getDayTasks()
	if len(dayTasks) > 0 {
		m.SelectedTaskUUID = dayTasks[0].UUID
		m.TimelineHour = dayTasks[0].TimeWindow.Start.Hour()
	} else {
		m.SelectedTaskUUID = ""
	}
}

func (m *Model) selectFirstTaskInCurrentHour() {
	dayTasks := m.getDayTasks()
	for _, t := range dayTasks {
		if t.TimeWindow.Start.Hour() == m.TimelineHour {
			m.SelectedTaskUUID = t.UUID
			return
		}
	}
	for _, t := range dayTasks {
		if m.TimelineHour >= t.TimeWindow.Start.Hour() && m.TimelineHour < t.TimeWindow.End.Hour() {
			m.SelectedTaskUUID = t.UUID
			return
		}
	}
}

func (m *Model) autoScrollToSelectedTask() {
	if m.SelectedTaskUUID == "" {
		return
	}
	var selectedTask model.Task
	found := false
	for _, t := range m.Tasks {
		if t.UUID == m.SelectedTaskUUID && t.SchedulingType == model.Anchored {
			selectedTask = t
			found = true
			break
		}
	}
	if !found {
		return
	}

	startRow := timeToRow(selectedTask.TimeWindow.Start)
	durationMinutes := int(selectedTask.TimeWindow.End.Sub(selectedTask.TimeWindow.Start).Minutes())
	h := (durationMinutes * rowsPerHour + 59) / 60
	if startRow+h > totalRows {
		h = totalRows - startRow
	}
	if h < 1 {
		h = 1
	}

	appContentHeight := m.Height
	visibleH := appContentHeight - 4
	if visibleH < 8 {
		visibleH = 8
	}

	centerRow := m.TimelineHour * rowsPerHour
	startR := centerRow - visibleH/2

	if h >= visibleH {
		startR = startRow
	} else {
		if startRow < startR {
			startR = startRow
		}
		if startRow+h > startR+visibleH {
			startR = startRow + h - visibleH
		}
	}

	newCenterRow := startR + visibleH/2
	if newCenterRow < 0 {
		newCenterRow = 0
	}
	if newCenterRow >= totalRows {
		newCenterRow = totalRows - 1
	}

	m.TimelineHour = newCenterRow / rowsPerHour
}

func (m *Model) navigateVertical(dir int) {
	dayTasks := m.getDayTasks()
	if len(dayTasks) == 0 {
		return
	}

	rects := m.BuildDayTaskRects(dayTasks)
	if len(rects) == 0 {
		return
	}

	var current TaskRect
	found := false
	for _, r := range rects {
		if r.Task.UUID == m.SelectedTaskUUID {
			current = r
			found = true
			break
		}
	}
	if !found {
		m.SelectedTaskUUID = dayTasks[0].UUID
		return
	}

	bestScore := 1_000_000
	bestUUID := ""
	var bestRect TaskRect
	for _, r := range rects {
		if r.Task.UUID == current.Task.UUID {
			continue
		}
		if dir > 0 {
			if r.Top <= current.CenterY {
				continue
			}
			dy := absInt(r.Top - current.CenterY)
			dx := absInt(r.CenterX - current.CenterX)
			score := dy + dx*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Left < bestRect.Left || (r.Left == bestRect.Left && r.Top < bestRect.Top))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		} else {
			if r.Bottom >= current.CenterY {
				continue
			}
			dy := absInt(current.CenterY - r.Bottom)
			dx := absInt(r.CenterX - current.CenterX)
			score := dy + dx*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Left < bestRect.Left || (r.Left == bestRect.Left && r.Top < bestRect.Top))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		}
	}

	if bestUUID != "" {
		m.SelectedTaskUUID = bestUUID
	}
}

func (m *Model) navigateHorizontal(dir int) {
	dayTasks := m.getDayTasks()
	if len(dayTasks) <= 1 {
		return
	}

	rects := m.BuildDayTaskRects(dayTasks)
	if len(rects) == 0 {
		return
	}

	var current TaskRect
	found := false
	for _, r := range rects {
		if r.Task.UUID == m.SelectedTaskUUID {
			current = r
			found = true
			break
		}
	}
	if !found {
		m.SelectedTaskUUID = dayTasks[0].UUID
		return
	}

	bestScore := 1_000_000
	bestUUID := ""
	var bestRect TaskRect
	for _, r := range rects {
		if r.Task.UUID == current.Task.UUID {
			continue
		}
		if dir > 0 {
			if r.Left <= current.CenterX {
				continue
			}
			dx := absInt(r.Left - current.CenterX)
			dy := absInt(r.CenterY - current.CenterY)
			score := dx + dy*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Top < bestRect.Top || (r.Top == bestRect.Top && r.Left < bestRect.Left))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		} else {
			if r.Right >= current.CenterX {
				continue
			}
			dx := absInt(current.CenterX - r.Right)
			dy := absInt(r.CenterY - current.CenterY)
			score := dx + dy*2
			if score < bestScore || (score == bestScore && (bestUUID == "" || r.Top < bestRect.Top || (r.Top == bestRect.Top && r.Left < bestRect.Left))) {
				bestScore = score
				bestUUID = r.Task.UUID
				bestRect = r
			}
		}
	}

	if bestUUID != "" {
		m.SelectedTaskUUID = bestUUID
	}
}
