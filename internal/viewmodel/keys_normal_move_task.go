package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/viewmodel/common/constants"
	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) EnterTaskMoveMode() {
	m.enterTaskMoveModeInternal(false)
}

func (m *Model) EnterTaskCloneMoveMode() {
	m.enterTaskMoveModeInternal(true)
}

func (m *Model) enterTaskMoveModeInternal(isClone bool) {
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to move."
		return
	}
	if !model.IsTaskAnchored(task) {
		if isClone {
			m.StatusMsg = "Only anchored tasks, events, and habits can be cloned with Y."
		} else {
			m.StatusMsg = "Only anchored tasks, events, and habits can be moved with y."
		}
		return
	}

	m.CurrentMode = ModeTaskMove
	m.TaskMoveIsClone = isClone
	m.TaskMovePrefix = ""
	m.TaskMoveOriginalTimeWindow = task.TimeWindow

	// Create a cloned placeholder version of the task
	clone := task
	clone.UUID = task.UUID + "_moving"

	// Append the clone to m.Tasks so it renders
	m.Tasks = append(m.Tasks, clone)

	// Focus selection on the clone
	m.SelectedTaskUUID = clone.UUID

	if isClone {
		m.StatusMsg = fmt.Sprintf("Cloning '%s'. Use j/k or count+j/k to position in %dm steps. Enter to confirm, Esc to cancel.", task.Title, constants.TaskMoveStepMinutes)
	} else {
		m.StatusMsg = fmt.Sprintf("Locked '%s'. Use j/k or count+j/k to move in %dm steps. Enter to confirm, Esc to cancel.", task.Title, constants.TaskMoveStepMinutes)
	}
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
		if m.TaskMoveIsClone {
			m.StatusMsg = "Only anchored tasks, events, and habits can be cloned with Y."
		} else {
			m.StatusMsg = "Only anchored tasks, events, and habits can be moved with y."
		}
		return
	}

	delta := time.Duration(steps*constants.TaskMoveStepMinutes) * time.Minute
	task.TimeWindow.Start = task.TimeWindow.Start.Add(delta)
	task.TimeWindow.End = task.TimeWindow.End.Add(delta)
	m.updateTaskInMemory(task)
	m.AutoScrollToSelectedTask()
	m.TaskMovePrefix = ""
	moveDir := "down"
	if direction < 0 {
		moveDir = "up"
	}
	actionStr := "Moved"
	if m.TaskMoveIsClone {
		actionStr = "Positioned clone of"
	}
	m.StatusMsg = fmt.Sprintf("%s '%s' %d minutes %s. Enter to confirm, Esc to cancel.", actionStr, task.Title, absInt(steps*constants.TaskMoveStepMinutes), moveDir)
}
