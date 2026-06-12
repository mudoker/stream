package viewmodel

import (
	"fmt"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleConfirmDialogKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.WarningOpen {
		switch msg.String() {
		case "enter", "q", "space":
			m.WarningOpen = false
			m.WarningMsg = ""
			return true, nil
		}
		return true, nil
	}

	if m.AuthNoticeOpen {
		switch msg.String() {
		case "esc", "enter", "q":
			m.AuthNoticeOpen = false
			m.AuthNoticeMsg = ""
			m.StatusMsg = "Returned to normal mode."
			return true, nil
		}
		return true, nil
	}

	if m.SessionExpiryPromptOpen {
		switch msg.String() {
		case "y", "Y", "enter":
			m.SessionTimeRemainingSeconds = m.DB.GetUserSettings().LockTimeoutMinutes * 60
			m.SessionExpiryPromptOpen = false
			m.StatusMsg = "Session timer reset."
			return true, nil
		case "n", "N", "esc":
			m.SessionExpiryPromptOpen = false
			m.StatusMsg = "Session will lock in 1 minute."
			return true, nil
		}
		return true, nil
	}

	if m.ConfirmOpen {
		keyStr := msg.String()
		if m.ConfirmActionType == "deanchor" {
			if keyStr == "enter" {
				m.ConfirmTask.SchedulingType = model.Floating
				m.ConfirmTask.LifecycleState = model.StateReady
				m.ConfirmTask.UpdatedAt = time.Now()
				if m.DB != nil {
					m.DB.UpdateTask(m.ConfirmTask)
					m.refreshTasks()
				} else {
					m.updateTaskInMemory(m.ConfirmTask)
				}
				m.triggerGCalPushIfAnchored(m.ConfirmTask)
				m.StatusMsg = fmt.Sprintf("Task '%s' de-anchored to backlog.", m.ConfirmTask.Title)
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
			} else {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "De-anchoring canceled."
			}
			return true, nil
		}

		switch keyStr {
		case "y", "Y", "enter":
			if m.ConfirmActionType == "complete_reminder" {
				if keyStr == "enter" {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "Completion canceled."
					return true, nil
				}
				m.ConfirmTask.LifecycleState = model.StateCompleted
				m.ConfirmTask.UpdatedAt = time.Now()
				m.DB.UpdateTask(m.ConfirmTask)
				m.refreshTasks()
				if m.DetailOpen && m.DetailTask.UUID == m.ConfirmTask.UUID {
					m.DetailOpen = false
				}
				m.StatusMsg = fmt.Sprintf("Reminder '%s' completed!", m.ConfirmTask.Title)
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				return true, nil
			} else {
				m.DB.DeleteTask(m.ConfirmTask.UUID)
				m.refreshTasks()
				m.triggerGCalPushIfAnchored(m.ConfirmTask)
				if m.DetailOpen && m.DetailTask.UUID == m.ConfirmTask.UUID {
					m.DetailOpen = false
				}
				if zt := m.ZenTimer; zt != nil && zt.Task.UUID == m.ConfirmTask.UUID {
					m.ZenTimer = nil
				}
				m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", m.ConfirmTask.Title)
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				return true, nil
			}
		case "n", "N", "esc":
			if m.ConfirmActionType == "complete_reminder" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "Completion canceled."
			} else {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "Deletion canceled."
			}
			return true, nil
		}
		return true, nil
	}

	return false, nil
}
