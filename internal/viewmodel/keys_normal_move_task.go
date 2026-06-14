package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/constant"
	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) EnterTaskMoveMode() {
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if !model.IsTaskAnchored(task) {
		m.StatusMsg = "Only anchored tasks, events, and habits can be moved with v."
		return
	}

	m.CurrentMode = ModeTaskMove
	m.TaskMovePrefix = ""
	m.TaskMoveOriginalTimeWindow = task.TimeWindow

	// Create a cloned placeholder version of the task
	clone := task
	clone.UUID = task.UUID + "_moving"

	// Append the clone to m.Tasks so it renders
	m.Tasks = append(m.Tasks, clone)

	// Focus selection on the clone
	m.SelectedTaskUUID = clone.UUID

	m.StatusMsg = fmt.Sprintf("Locked '%s'. Use j/k or count+j/k to move in %dm steps. Enter to confirm, Esc to cancel.", task.Title, constant.TaskMoveStepMinutes)
}

func (m *Model) HandleTaskMoveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if !model.IsTaskAnchored(task) {
		m.StatusMsg = "Only anchored tasks, events, and habits can be moved with v."
		return
	}

	delta := time.Duration(steps*constant.TaskMoveStepMinutes) * time.Minute
	task.TimeWindow.Start = task.TimeWindow.Start.Add(delta)
	task.TimeWindow.End = task.TimeWindow.End.Add(delta)
	m.updateTaskInMemory(task)
	m.AutoScrollToSelectedTask()
	m.TaskMovePrefix = ""
	moveDir := "down"
	if direction < 0 {
		moveDir = "up"
	}
	m.StatusMsg = fmt.Sprintf("Moved '%s' %d minutes %s. Enter to confirm, Esc to cancel.", task.Title, absInt(steps*constant.TaskMoveStepMinutes), moveDir)
}
