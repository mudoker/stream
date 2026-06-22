package viewmodel

import (
	"fmt"
	"strings"

	"stream/internal/model"
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
	m.StatusMsg = "Task move canceled."
}
