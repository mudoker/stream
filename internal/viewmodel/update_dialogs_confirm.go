package viewmodel

import (
	"fmt"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/common"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleConfirmDialogKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	defer func() {
		if !m.ConfirmOpen {
			m.ConfirmFocusArea = 0
		}
	}()

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
		numOpts := 2
		if m.ConfirmActionType == "exit_focus" {
			numOpts = 3
		}

		// Handle direct shortcuts for Yes/No
		if keyStr == "y" || keyStr == "Y" {
			m.ConfirmSelectedIndex = 0
			keyStr = "enter"
		} else if keyStr == "n" || keyStr == "N" {
			m.ConfirmSelectedIndex = 1
			keyStr = "enter"
		}

		// Handle Tab / Shift+Tab to switch focus between options list and buttons
		if keyStr == "tab" {
			m.ConfirmFocusArea = (m.ConfirmFocusArea + 1) % 3
			return true, nil
		} else if keyStr == "shift+tab" {
			m.ConfirmFocusArea = (m.ConfirmFocusArea - 1 + 3) % 3
			return true, nil
		}

		// Handle key navigation based on focus area
		if m.ConfirmFocusArea == 0 {
			if keyStr == "j" || keyStr == "down" || keyStr == "l" || keyStr == "right" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex + 1) % numOpts
				return true, nil
			} else if keyStr == "k" || keyStr == "up" || keyStr == "h" || keyStr == "left" {
				m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex - 1 + numOpts) % numOpts
				return true, nil
			}
		} else {
			// Area 1 (Confirm) or Area 2 (Cancel)
			if keyStr == "h" || keyStr == "left" {
				m.ConfirmFocusArea = 1
				return true, nil
			} else if keyStr == "l" || keyStr == "right" {
				m.ConfirmFocusArea = 2
				return true, nil
			} else if keyStr == "k" || keyStr == "up" {
				m.ConfirmFocusArea = 0
				return true, nil
			}
		}

		if keyStr == "enter" {
			if m.ConfirmFocusArea == 2 {
				// Cancel button selected: treat like esc/cancel
				keyStr = "esc"
			}
		}

		if keyStr == "enter" {
			switch m.ConfirmActionType {
			case "exit_focus":
				common.HandleExitFocusOption(m, m.ConfirmSelectedIndex)
			case "deanchor":
				if m.ConfirmSelectedIndex == 0 {
					common.ConfirmDeanchor(m, m.ConfirmTask)
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "De-anchoring canceled."
				}
			case "delete_recurring":
				if m.ConfirmSelectedIndex == 0 {
					common.DeleteTaskOccurrence(m, m.ConfirmTask)
				} else {
					common.DeleteAllOccurrences(m, m.ConfirmTask, m.Tasks)
				}
			case "edit_recurring":
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
									if t.SchedulingType == model.Event || t.SchedulingType == model.Habit || t.SchedulingType == model.Anchored {
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
			case "complete_reminder":
				if m.ConfirmSelectedIndex == 0 {
					common.CompleteReminder(m, m.ConfirmTask)
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "Completion canceled."
				}
			case "log_session_confirm":
				if m.ConfirmSelectedIndex == 0 {
					common.InitiateLogSession(m, m.ConfirmTask)
				} else {
					common.CancelLogSession(m, m.ConfirmTask)
				}
			case "start_late_confirm":
				if m.ConfirmSelectedIndex == 0 {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StartZenMode(m.ConfirmTask)
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StartZenModeWithTrim(m.ConfirmTask)
				}
			default: // delete
				if m.ConfirmSelectedIndex == 0 {
					common.DeleteTaskOccurrence(m, m.ConfirmTask)
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.StatusMsg = "Deletion canceled."
				}
			}
			return true, nil
		}

		if keyStr == "esc" || keyStr == "q" {
			if m.ConfirmActionType == "exit_focus" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				if m.ZenTimer != nil {
					m.ZenTimer.IsPaused = false
				}
				m.StatusMsg = "Focus session resumed."
			} else if m.ConfirmActionType == "edit_recurring" {
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
			} else {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.StatusMsg = "Action canceled."
			}
			return true, nil
		}
		return true, nil
	}

	return false, nil
}

func (m *Model) HandleExitFocusOption(index int) {
	common.HandleExitFocusOption(m, index)
}

