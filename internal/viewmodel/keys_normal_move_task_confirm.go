package viewmodel

import (
	"fmt"
	"strings"

	"stream/internal/model"

	"github.com/google/uuid"
)

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
		m.TaskMoveIsClone = false
		return
	}

	if m.TaskMoveIsClone {
		var originalTask model.Task
		originalFound := false
		for _, t := range m.Tasks {
			if t.UUID == originalUUID {
				originalTask = t
				originalFound = true
				break
			}
		}

		if !originalFound {
			m.CurrentMode = ModeNormal
			m.TaskMoveIsClone = false
			return
		}

		newTask := originalTask
		newTask.UUID = uuid.New().String()
		newTask.TimeWindow = finalTimeWindow
		if newTask.LifecycleState == model.StateOverdue {
			newTask.LifecycleState = model.StateScheduled
		}
		newTask.GCalMetadata = model.GCalMetadata{}

		if m.DB != nil {
			if err := m.DB.AddTask(newTask); err != nil {
				m.StatusMsg = fmt.Sprintf("Failed to clone task: %v", err)
			} else {
				m.refreshTasks()
				m.triggerGCalPush(newTask)
			}
		} else {
			m.Tasks = append(m.Tasks, newTask)
		}

		m.SelectedTaskUUID = newTask.UUID
		m.AutoScrollToSelectedTask()

		m.CurrentMode = ModeNormal
		m.TaskMovePrefix = ""
		m.TaskMoveIsClone = false
		m.StatusMsg = fmt.Sprintf("Cloned '%s' to %s.", newTask.Title, newTask.TimeWindow.Start.Format("15:04"))
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
		if originalTask.RecurringParentUUID != "" {
			confirmTask := originalTask
			confirmTask.TimeWindow = m.TaskMoveOriginalTimeWindow

			m.ConfirmTask = confirmTask
			m.PendingEditTask = originalTask
			m.ConfirmOpen = true
			m.ConfirmActionType = "edit_recurring"
			m.ConfirmSelectedIndex = 0
			m.RecurringEditFromForm = false
			m.CurrentMode = ModeNormal
			m.TaskMovePrefix = ""
			m.TaskMoveIsClone = false
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
	m.TaskMoveIsClone = false
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
	m.AutoScrollToSelectedTask()

	m.CurrentMode = ModeNormal
	m.TaskMovePrefix = ""
	m.TaskMoveIsClone = false
	m.StatusMsg = "Task move canceled."
}
