package viewmodel

import (
	"fmt"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleHelpAndDetailKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.HelpOpen {
		switch msg.String() {
		case "esc", "q", "?", "enter":
			m.HelpOpen = false
			m.HelpScrollOffset = 0
			return true, nil
		case "j", "down", "J":
			m.HelpScrollOffset++
			return true, nil
		case "k", "up", "K":
			if m.HelpScrollOffset > 0 {
				m.HelpScrollOffset--
			}
			return true, nil
		case "ctrl+d", "d", "pgdown", "space":
			m.HelpScrollOffset += 6
			return true, nil
		case "ctrl+u", "u", "pgup":
			m.HelpScrollOffset -= 6
			if m.HelpScrollOffset < 0 {
				m.HelpScrollOffset = 0
			}
			return true, nil
		case "g", "home":
			m.HelpScrollOffset = 0
			return true, nil
		case "G", "end":
			m.HelpScrollOffset = 100
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
			m.CheckAndStartZenMode(m.DetailTask)
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
