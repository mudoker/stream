package viewmodel

import (
	"fmt"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleHelpAndDetailKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.HelpOpen {
		switch msg.String() {
		case "esc", "q", "?":
			m.HelpOpen = false
			m.HelpScrollOffset = 0
			return true, nil
		case "j", "down":
			m.HelpScrollOffset++
			return true, nil
		case "k", "up":
			if m.HelpScrollOffset > 0 {
				m.HelpScrollOffset--
			}
			return true, nil
		case "ctrl+d":
			m.HelpScrollOffset += 5
			return true, nil
		case "ctrl+u":
			m.HelpScrollOffset -= 5
			if m.HelpScrollOffset < 0 {
				m.HelpScrollOffset = 0
			}
			return true, nil
		case "g":
			m.HelpScrollOffset = 0
			return true, nil
		case "G":
			m.HelpScrollOffset = 9999
			return true, nil
		}
		return true, nil
	}

	if m.DetailOpen {
		switch msg.String() {
		case "esc", "enter":
			m.DetailOpen = false
			return true, nil
		case "z":
			m.StartZenMode(m.DetailTask)
			m.DetailOpen = false
			return true, nil
		case "x":
			if m.DetailTask.SchedulingType == model.Reminder && m.DetailTask.LifecycleState != model.StateCompleted {
				m.ConfirmTask = m.DetailTask
				m.ConfirmOpen = true
				m.ConfirmActionType = "complete_reminder"
				return true, nil
			}
			if m.DetailTask.LifecycleState == model.StateCompleted {
				m.DetailTask.LifecycleState = model.StateBacklog
				m.StatusMsg = fmt.Sprintf("Task '%s' marked incomplete.", m.DetailTask.Title)
			} else {
				m.DetailTask.LifecycleState = model.StateCompleted
				m.StatusMsg = fmt.Sprintf("Task '%s' completed!", m.DetailTask.Title)
			}
			m.DB.UpdateTask(m.DetailTask)
			m.refreshTasks()
			m.DetailOpen = false
			return true, nil
		case "d":
			m.ConfirmTask = m.DetailTask
			m.ConfirmOpen = true
			m.ConfirmActionType = "delete"
			return true, nil
		case "e":
			m.startEditMode(m.DetailTask)
			m.DetailOpen = false
			return true, nil
		}
		return true, nil
	}

	return false, nil
}
