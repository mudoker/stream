package viewmodel

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/viewmodel/common/constants"
	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) EnterTaskDurationAdjustMode() {
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to adjust duration."
		return
	}
	if !model.IsTaskAnchored(task) {
		m.StatusMsg = "Only anchored tasks, events, and habits can have their duration adjusted with V."
		return
	}

	m.CurrentMode = ModeTaskDurationAdjust
	m.TaskMovePrefix = ""
	m.TaskMoveOriginalTimeWindow = task.TimeWindow

	// Create a cloned placeholder version of the task
	clone := task
	clone.UUID = task.UUID + "_adjusting"

	// Append the clone to m.Tasks so it renders
	m.Tasks = append(m.Tasks, clone)

	// Focus selection on the clone
	m.SelectedTaskUUID = clone.UUID

	m.AutoScrollToSelectedTask()

	m.StatusMsg = fmt.Sprintf("Adjusting duration for '%s'. Use j/k or count+j/k to increase/decrease by 15m. Enter to confirm, Esc to cancel.", task.Title)
}

func (m *Model) HandleTaskDurationAdjustKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		m.TaskMovePrefix += key
		m.StatusMsg = fmt.Sprintf("Duration change count set to %s. Press j/k to apply.", m.TaskMovePrefix)
		return m, nil
	}

	switch key {
	case "j", "down":
		m.applyTaskDurationAdjust(1)
	case "k", "up":
		m.applyTaskDurationAdjust(-1)
	case "enter":
		m.confirmTaskDurationAdjust()
	case "esc":
		m.cancelTaskDurationAdjust()
	}

	return m, nil
}

func (m *Model) applyTaskDurationAdjust(direction int) {
	count := m.parseTaskMoveCount()
	steps := count * direction
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No task selected to adjust duration."
		return
	}
	if !model.IsTaskAnchored(task) {
		m.StatusMsg = "Only anchored tasks, events, and habits can have their duration adjusted with V."
		return
	}

	delta := time.Duration(steps*constants.TaskDurationStepMinutes) * time.Minute
	newEnd := task.TimeWindow.End.Add(delta)

	// Ensure duration is at least 15 minutes
	if newEnd.Sub(task.TimeWindow.Start) < constants.MinTaskDurationMinutes*time.Minute {
		newEnd = task.TimeWindow.Start.Add(constants.MinTaskDurationMinutes * time.Minute)
	}

	task.TimeWindow.End = newEnd
	m.updateTaskInMemory(task)
	m.AutoScrollToSelectedTask()
	m.TaskMovePrefix = ""

	newDurMinutes := int(task.TimeWindow.End.Sub(task.TimeWindow.Start).Minutes())
	m.StatusMsg = fmt.Sprintf("Adjusted duration to %d minutes. Enter to confirm, Esc to cancel.", newDurMinutes)
}

func (m *Model) confirmTaskDurationAdjust() {
	var originalUUID string
	if strings.HasSuffix(m.SelectedTaskUUID, "_adjusting") {
		originalUUID = strings.TrimSuffix(m.SelectedTaskUUID, "_adjusting")
	} else {
		originalUUID = m.SelectedTaskUUID
	}

	var finalTimeWindow model.TimeWindow
	cloneFound := false
	for _, t := range m.Tasks {
		if t.UUID == originalUUID+"_adjusting" {
			finalTimeWindow = t.TimeWindow
			cloneFound = true
			break
		}
	}

	// Unconditionally remove all adjusting placeholders from memory
	var cleanTasks []model.Task
	for _, t := range m.Tasks {
		if !strings.HasSuffix(t.UUID, "_adjusting") {
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
			originalTask = m.Tasks[i]
			originalFound = true
			break
		}
	}

	m.SelectedTaskUUID = originalUUID

	if originalFound && m.DB != nil {
		if originalTask.RecurringParentUUID != "" {
			confirmTask := originalTask
			confirmTask.TimeWindow = m.TaskMoveOriginalTimeWindow

			m.ConfirmTask = confirmTask
			m.PendingEditTask = originalTask
			m.ConfirmOpen = true
			m.ConfirmActionType = "edit_recurring"
			m.ConfirmSelectedIndex = 0
			m.CurrentMode = ModeNormal
			m.TaskMovePrefix = ""
			m.StatusMsg = "Choose recurring update option."
			return
		}

		m.DB.UpdateTask(originalTask)
		m.refreshTasks()
		m.triggerGCalPush(originalTask)
	}

	// Auto-scroll to the confirmed task to ensure it is visible
	m.AutoScrollToSelectedTask()

	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	newDurMinutes := int(originalTask.TimeWindow.End.Sub(originalTask.TimeWindow.Start).Minutes())
	m.StatusMsg = fmt.Sprintf("Task '%s' duration adjusted to %d minutes.", originalTask.Title, newDurMinutes)
}

func (m *Model) cancelTaskDurationAdjust() {
	var originalUUID string
	if strings.HasSuffix(m.SelectedTaskUUID, "_adjusting") {
		originalUUID = strings.TrimSuffix(m.SelectedTaskUUID, "_adjusting")
	} else {
		originalUUID = m.SelectedTaskUUID
	}
	m.SelectedTaskUUID = originalUUID

	// Unconditionally remove all adjusting placeholders from memory
	var cleanTasks []model.Task
	for _, t := range m.Tasks {
		if !strings.HasSuffix(t.UUID, "_adjusting") {
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
	m.AutoScrollToSelectedTask()

	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.StatusMsg = "Duration adjustment canceled."
}
