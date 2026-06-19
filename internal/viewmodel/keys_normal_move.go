package viewmodel

import (
	"strings"
	"time"

	"stream/internal/model"
)

func (m *Model) MoveTaskSelection(dir int) {
	shelf := m.GetTodoShelfTasks()
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
		shelf := m.GetTodoShelfTasks()
		if len(shelf) > 0 {
			m.SelectedTaskUUID = shelf[0].UUID
		} else {
			m.SelectedTaskUUID = ""
		}
		return
	}
	dayTasks := m.GetDayTasks()
	if len(dayTasks) > 0 {
		m.SelectedTaskUUID = dayTasks[0].UUID
		m.TimelineHour = dayTasks[0].TimeWindow.Start.Hour()
		m.AutoScrollToSelectedTask()
	} else {
		m.SelectedTaskUUID = ""
		m.TimelineHour = time.Now().Hour()
	}
}

func (m *Model) selectFirstTaskInCurrentHour() {
	dayTasks := m.GetDayTasks()
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
	m.SelectedTaskUUID = ""
}

func (m *Model) AutoScrollToSelectedTask() {
	if m.SelectedTaskUUID == "" {
		return
	}
	var selectedTask model.Task
	found := false
	for _, t := range m.Tasks {
		if t.UUID == m.SelectedTaskUUID && model.IsTaskAnchored(t) {
			selectedTask = t
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Always sync selected day to the task's start day to keep it visible on day transition
	m.SelectedDay = selectedTask.TimeWindow.Start.Local()

	const scale = 5
	visualRowsPerHour := RowsPerHour / scale
	visualTotalRows := TotalRows / scale

	// Find the task's visual rect from BuildDayTaskRects to get exact start and end coordinates
	dayTasks := m.GetDayTasks()
	rects := m.BuildDayTaskRects(dayTasks)
	var rect TaskRect
	rectFound := false
	for _, r := range rects {
		if r.Task.UUID == m.SelectedTaskUUID {
			rect = r
			rectFound = true
			break
		}
	}
	if !rectFound {
		return
	}

	commuteRows := 0
	if selectedTask.SchedulingType == model.Event && strings.TrimSpace(selectedTask.Location) != "" && selectedTask.CommuteBuffer > 0 {
		commuteRows = (selectedTask.CommuteBuffer*visualRowsPerHour + 59) / 60
	}

	taskStart := rect.Top - commuteRows
	if taskStart < 0 {
		taskStart = 0
	}

	taskEnd := rect.Bottom
	if selectedTask.SchedulingType == model.Event && strings.TrimSpace(selectedTask.Location) != "" && selectedTask.CommuteBuffer > 0 {
		taskEnd += commuteRows
	}
	restDur := CalculateTaskRestTime(selectedTask)
	restMins := int(restDur.Minutes())
	if restMins > 0 {
		taskEnd += (restMins*visualRowsPerHour + 59) / 60
	}

	if taskEnd > visualTotalRows {
		taskEnd = visualTotalRows
	}

	appContentHeight := m.Height
	visibleH := appContentHeight - 3 // Match RenderDayTimeline exactly
	if visibleH < 8 {
		visibleH = 8
	}

	// Calculate exact clamped viewport boundaries matching RenderDayTimeline
	centerRow := m.TimelineHour * visualRowsPerHour
	startR := centerRow - visibleH/2
	if startR < 0 {
		startR = 0
	}
	if startR > visualTotalRows-visibleH {
		startR = visualTotalRows - visibleH
	}
	viewportStart := startR
	viewportEnd := startR + visibleH

	if taskEnd-taskStart >= visibleH {
		if m.CurrentMode == ModeTaskDurationAdjust {
			target := taskEnd - (visibleH - visibleH/2)
			m.TimelineHour = (target + visualRowsPerHour - 1) / visualRowsPerHour
		} else {
			m.TimelineHour = (taskStart + visibleH/2) / visualRowsPerHour
		}
	} else {
		if taskStart < viewportStart {
			m.TimelineHour = (taskStart + visibleH/2) / visualRowsPerHour
		} else if taskEnd > viewportEnd {
			target := taskEnd - (visibleH - visibleH/2)
			m.TimelineHour = (target + visualRowsPerHour - 1) / visualRowsPerHour
		}
	}

	if m.TimelineHour < 0 {
		m.TimelineHour = 0
	}
	if m.TimelineHour > 23 {
		m.TimelineHour = 23
	}
}

func (m *Model) NavigateVertical(dir int) {
	dayTasks := m.GetDayTasks()
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
		targetRow := m.TimelineHour * (RowsPerHour / 5)
		bestIdx := 0
		minDist := 1_000_000
		for i, r := range rects {
			dist := absInt(r.Top - targetRow)
			if dist < minDist {
				minDist = dist
				bestIdx = i
			}
		}
		m.SelectedTaskUUID = rects[bestIdx].Task.UUID
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

func (m *Model) NavigateHorizontal(dir int) {
	dayTasks := m.GetDayTasks()
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
		targetRow := m.TimelineHour * (RowsPerHour / 5)
		bestIdx := 0
		minDist := 1_000_000
		for i, r := range rects {
			dist := absInt(r.Top - targetRow)
			if dist < minDist {
				minDist = dist
				bestIdx = i
			}
		}
		m.SelectedTaskUUID = rects[bestIdx].Task.UUID
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
