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
				if m.ConfirmTask.SchedulingType == model.Habit {
					m.ConfirmTask.TimeWindow = model.TimeWindow{} // clear time window to deanchor
				} else {
					m.ConfirmTask.SchedulingType = model.Floating
				}
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

		if m.ConfirmActionType == "delete_recurring" {
			if keyStr == "j" || keyStr == "down" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex + 1) % 2
			} else if keyStr == "k" || keyStr == "up" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex - 1 + 2) % 2
			} else if keyStr == "enter" {
				if m.ConfirmSelectedIndex == 0 {
					m.AdjustSelectionBeforeDeletion(m.ConfirmTask.UUID)
					m.DB.DeleteTask(m.ConfirmTask.UUID)
					m.refreshTasks()
					m.triggerGCalPushIfAnchored(m.ConfirmTask)
					if m.DetailOpen && m.DetailTask.UUID == m.ConfirmTask.UUID {
						m.DetailOpen = false
					}
					if zt := m.ZenTimer; zt != nil && zt.Task.UUID == m.ConfirmTask.UUID {
						m.ZenTimer = nil
					}
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = fmt.Sprintf("Occurrence of '%s' deleted.", m.ConfirmTask.Title)
				} else {
					tasksToDelete := []string{}
					for _, t := range m.Tasks {
						if t.RecurringParentUUID == m.ConfirmTask.RecurringParentUUID {
							if t.UUID == m.ConfirmTask.UUID || !t.TimeWindow.Start.Before(m.ConfirmTask.TimeWindow.Start) {
								tasksToDelete = append(tasksToDelete, t.UUID)
							}
						}
					}
					m.AdjustSelectionBeforeDeletion(m.ConfirmTask.UUID)
					for _, uid := range tasksToDelete {
						m.DB.DeleteTask(uid)
					}
					m.refreshTasks()
					if m.DetailOpen && m.DetailTask.UUID == m.ConfirmTask.UUID {
						m.DetailOpen = false
					}
					if zt := m.ZenTimer; zt != nil && zt.Task.UUID == m.ConfirmTask.UUID {
						m.ZenTimer = nil
					}
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "This and all future occurrences deleted."
				}
			} else if keyStr == "esc" || keyStr == "q" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "Deletion canceled."
			}
			return true, nil
		}

		if m.ConfirmActionType == "edit_recurring" {
			if keyStr == "j" || keyStr == "down" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex + 1) % 2
			} else if keyStr == "k" || keyStr == "up" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex - 1 + 2) % 2
			} else if keyStr == "enter" {
				if m.ConfirmSelectedIndex == 0 {
					m.DB.UpdateTask(m.PendingEditTask)
					m.refreshTasks()
					m.triggerGCalPush(m.PendingEditTask)
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = fmt.Sprintf("Occurrence of '%s' updated.", m.PendingEditTask.Title)
				} else {
					originalStart := m.ConfirmTask.TimeWindow.Start
					timeShift := m.PendingEditTask.TimeWindow.Start.Sub(originalStart)
					durationShift := m.PendingEditTask.TimeWindow.End.Sub(m.PendingEditTask.TimeWindow.Start)

					for _, t := range m.Tasks {
						if t.RecurringParentUUID == m.ConfirmTask.RecurringParentUUID {
							isCurrent := t.UUID == m.ConfirmTask.UUID
							if isCurrent || !t.TimeWindow.Start.Before(originalStart) {
								t.Title = m.PendingEditTask.Title
								t.Description = m.PendingEditTask.Description
								t.Priority = m.PendingEditTask.Priority
								t.StoryPoints = m.PendingEditTask.StoryPoints
								t.Tags = m.PendingEditTask.Tags
								t.Location = m.PendingEditTask.Location
								t.CommuteBuffer = m.PendingEditTask.CommuteBuffer
								t.UpdatedAt = time.Now()

								if isCurrent {
									t.TimeWindow = m.PendingEditTask.TimeWindow
								} else {
									if t.SchedulingType == model.Anchored || t.SchedulingType == model.Event || t.SchedulingType == model.Habit {
										t.TimeWindow.Start = t.TimeWindow.Start.Add(timeShift)
										t.TimeWindow.End = t.TimeWindow.Start.Add(durationShift)
									}
								}

								m.DB.UpdateTask(t)
							}
						}
					}
					m.refreshTasks()
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "This and all future occurrences updated."
				}
			} else if keyStr == "esc" || keyStr == "q" {
				if m.DB != nil {
					m.refreshTasks()
				} else {
					for i, t := range m.Tasks {
						if t.UUID == m.ConfirmTask.UUID {
							m.Tasks[i] = m.ConfirmTask
							break
						}
					}
				}
				if !m.ConfirmTask.TimeWindow.Start.IsZero() {
					m.SelectedDay = m.ConfirmTask.TimeWindow.Start.Local()
				}
				m.AutoScrollToSelectedTask()
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "Edit canceled."
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
				m.AdjustSelectionBeforeDeletion(m.ConfirmTask.UUID)
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
