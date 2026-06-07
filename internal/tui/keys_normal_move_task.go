package tui

import (
	"fmt"
	"strconv"
	"strings"
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

	// Create a cloned placeholder version of the task
	clone := task
	clone.UUID = task.UUID + "_moving"

	// Append the clone to m.Tasks so it renders
	m.Tasks = append(m.Tasks, clone)

	// Focus selection on the clone
	m.SelectedTaskUUID = clone.UUID

	m.StatusMsg = fmt.Sprintf("Locked '%s'. Use j/k or count+j/k to move in 5m steps. Enter to confirm, Esc to cancel.", task.Title)
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

	delta := time.Duration(steps*5) * time.Minute
	task.TimeWindow.Start = task.TimeWindow.Start.Add(delta)
	task.TimeWindow.End = task.TimeWindow.End.Add(delta)
	m.updateTaskInMemory(task)
	m.autoScrollToSelectedTask()
	m.TaskMovePrefix = ""
	moveDir := "down"
	if direction < 0 {
		moveDir = "up"
	}
	m.StatusMsg = fmt.Sprintf("Moved '%s' %d minutes %s. Enter to confirm, Esc to cancel.", task.Title, absInt(steps*5), moveDir)
}

func (m *Model) confirmTaskMove() {
	var originalUUID string
	if strings.HasSuffix(m.SelectedTaskUUID, "_moving") {
		originalUUID = strings.TrimSuffix(m.SelectedTaskUUID, "_moving")
	} else {
		originalUUID = m.SelectedTaskUUID
	}

	var finalTimeWindow model.TimeWindow
	cloneFound := false
	for _, t := range m.Tasks {
		if t.UUID == originalUUID+"_moving" {
			finalTimeWindow = t.TimeWindow
			cloneFound = true
			break
		}
	}

	// Unconditionally remove all moving placeholders from memory
	var cleanTasks []model.Task
	for _, t := range m.Tasks {
		if !strings.HasSuffix(t.UUID, "_moving") {
			cleanTasks = append(cleanTasks, t)
		}
	}
	m.Tasks = cleanTasks

	if !cloneFound {
		m.CurrentMode = ModeNormal
		return
	}

	// Update original task's TimeWindow in memory
	var originalTask model.Task
	originalFound := false
	for i, t := range m.Tasks {
		if t.UUID == originalUUID {
			m.Tasks[i].TimeWindow = finalTimeWindow
			if m.Tasks[i].LifecycleState == model.StateOverdue {
				m.Tasks[i].LifecycleState = model.StateScheduled
			}
			originalTask = m.Tasks[i]
			originalFound = true
			break
		}
	}

	m.SelectedTaskUUID = originalUUID

	if originalFound && m.DB != nil {
		m.DB.UpdateTask(originalTask)
		m.refreshTasks()
	}

	// Auto-scroll to the confirmed task to ensure it is visible
	m.autoScrollToSelectedTask()

	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = fmt.Sprintf("Task '%s' moved to %s.", originalTask.Title, originalTask.TimeWindow.Start.Format("15:04"))
}

func (m *Model) cancelTaskMove() {
	var originalUUID string
	if strings.HasSuffix(m.SelectedTaskUUID, "_moving") {
		originalUUID = strings.TrimSuffix(m.SelectedTaskUUID, "_moving")
	} else {
		originalUUID = m.SelectedTaskUUID
	}
	m.SelectedTaskUUID = originalUUID

	// Unconditionally remove all moving placeholders from memory
	var cleanTasks []model.Task
	for _, t := range m.Tasks {
		if !strings.HasSuffix(t.UUID, "_moving") {
			cleanTasks = append(cleanTasks, t)
		}
	}
	m.Tasks = cleanTasks

	// Restore original task's time window in memory
	if originalUUID != "" {
		for i, t := range m.Tasks {
			if t.UUID == originalUUID {
				m.Tasks[i].TimeWindow = m.TaskMoveOriginalTimeWindow
				break
			}
		}
	}

	// Restore selected day to the original start day
	m.SelectedDay = m.TaskMoveOriginalTimeWindow.Start.Local()

	if m.DB != nil {
		m.refreshTasks()
	}

	// Auto-scroll back to the original task
	m.autoScrollToSelectedTask()

	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = "Task move canceled."
}
