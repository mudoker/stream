package tui

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) enterTaskMoveMode() {
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if task.SchedulingType != model.Anchored {
		m.StatusMsg = "Only anchored tasks can be moved with v."
		return
	}

	m.CurrentMode = ModeTaskMove
	m.TaskMovePrefix = ""
	m.TaskMoveOriginalTimeWindow = task.TimeWindow
	m.StatusMsg = fmt.Sprintf("Locked '%s'. Use j/k or count+j/k to move in 15m steps. Enter to confirm, Esc to cancel.", task.Title)
}

func (m *Model) handleTaskMoveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		m.TaskMovePrefix += key
		m.StatusMsg = fmt.Sprintf("Move count set to %s. Press j/k to apply.", m.TaskMovePrefix)
		return m, nil
	}

	switch key {
	case "j", "down":
		m.applyTaskMove(1)
	case "k", "up":
		m.applyTaskMove(-1)
	case "enter":
		m.confirmTaskMove()
	case "esc":
		m.cancelTaskMove()
	}

	return m, nil
}

func (m *Model) parseTaskMoveCount() int {
	count := 1
	if m.TaskMovePrefix != "" {
		if parsed, err := strconv.Atoi(m.TaskMovePrefix); err == nil && parsed > 0 {
			count = parsed
		}
	}
	return count
}

func (m *Model) applyTaskMove(direction int) {
	count := m.parseTaskMoveCount()
	steps := count * direction
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if task.SchedulingType != model.Anchored {
		m.StatusMsg = "Only anchored tasks can be moved with v."
		return
	}

	delta := time.Duration(steps*15) * time.Minute
	task.TimeWindow.Start = task.TimeWindow.Start.Add(delta)
	task.TimeWindow.End = task.TimeWindow.End.Add(delta)
	m.updateTaskInMemory(task)
	m.TaskMovePrefix = ""
	moveDir := "down"
	if direction < 0 {
		moveDir = "up"
	}
	m.StatusMsg = fmt.Sprintf("Moved '%s' %d minutes %s. Enter to confirm, Esc to cancel.", task.Title, absInt(steps*15), moveDir)
}

func (m *Model) confirmTaskMove() {
	task, exists := m.getActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		m.CurrentMode = ModeNormal
		return
	}
	if m.DB != nil {
		m.DB.UpdateTask(task)
		m.refreshTasks()
	}
	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = fmt.Sprintf("Task '%s' moved to %s.", task.Title, task.TimeWindow.Start.Format("15:04"))
}

func (m *Model) cancelTaskMove() {
	if m.SelectedTaskUUID != "" {
		for i, t := range m.Tasks {
			if t.UUID == m.SelectedTaskUUID {
				t.TimeWindow = m.TaskMoveOriginalTimeWindow
				m.Tasks[i] = t
				break
			}
		}
	}
	if m.DB != nil {
		m.refreshTasks()
	}
	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = "Task move canceled."
}
