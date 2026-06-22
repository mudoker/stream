package viewmodel

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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
			} else if keyStr == "j" || keyStr == "down" || keyStr == "k" || keyStr == "up" {
				// Scoped to action buttons section, ignore vertical navigation across sections
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
					m.RecurringEditFromForm = false
					m.StatusMsg = fmt.Sprintf("Occurrence of '%s' updated.", m.PendingEditTask.Title)
				} else {
					originalStart := m.ConfirmTask.TimeWindow.Start
					durationShift := m.PendingEditTask.TimeWindow.End.Sub(m.PendingEditTask.TimeWindow.Start)

					if !m.RecurringEditFromForm {
						// Quick move or duration shift: keep existing pattern, just shift times of all future occurrences
						timeShift := m.PendingEditTask.TimeWindow.Start.Sub(originalStart)
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
						m.StatusMsg = "This and all future occurrences updated."
					} else {
						// Form edit: delete all future occurrences and regenerate them using the new pattern
						daysStr := strings.ToLower(m.Form.RecurringDaysInput.Value())
						endDateStr := strings.TrimSpace(m.Form.RecurringEndDateInput.Value())

						endDate, err := time.Parse("2006-01-02", endDateStr)
						if err != nil {
							endDate = originalStart.AddDate(0, 0, 7)
						}
						endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, originalStart.Location())

						maxEndDate := originalStart.AddDate(0, 1, 0)
						if endDate.After(maxEndDate) {
							endDate = time.Date(maxEndDate.Year(), maxEndDate.Month(), maxEndDate.Day(), 23, 59, 59, 0, originalStart.Location())
						}

						days := map[time.Weekday]bool{
							time.Sunday:    strings.Contains(daysStr, "sun") || strings.Contains(daysStr, "daily"),
							time.Monday:    strings.Contains(daysStr, "mon") || strings.Contains(daysStr, "daily"),
							time.Tuesday:   strings.Contains(daysStr, "tue") || strings.Contains(daysStr, "daily"),
							time.Wednesday: strings.Contains(daysStr, "wed") || strings.Contains(daysStr, "daily"),
							time.Thursday:  strings.Contains(daysStr, "thu") || strings.Contains(daysStr, "daily"),
							time.Friday:    strings.Contains(daysStr, "fri") || strings.Contains(daysStr, "daily"),
							time.Saturday:  strings.Contains(daysStr, "sat") || strings.Contains(daysStr, "daily"),
						}
						hasAny := false
						for _, v := range days {
							if v {
								hasAny = true
								break
							}
						}
						if !hasAny {
							for k := range days {
								days[k] = true
							}
						}

						// Delete all occurrences starting from originalStart
						for _, t := range m.Tasks {
							if t.RecurringParentUUID == m.ConfirmTask.RecurringParentUUID {
								if !t.TimeWindow.Start.Before(originalStart) {
									m.DB.DeleteTask(t.UUID)
								}
							}
						}

						// Regenerate occurrences starting from originalStart to endDate
						parentUUID := m.ConfirmTask.RecurringParentUUID
						current := originalStart
						for !current.After(endDate) {
							if days[current.Weekday()] {
								instance := m.PendingEditTask
								instance.UUID = uuid.New().String()
								instance.RecurringParentUUID = parentUUID
								instance.UpdatedAt = time.Now()

								if instance.SchedulingType == model.Habit && instance.TimeWindow.Start.IsZero() {
									instance.TimeWindow = model.TimeWindow{}
								} else {
									instance.TimeWindow.Start = time.Date(current.Year(), current.Month(), current.Day(), m.PendingEditTask.TimeWindow.Start.Hour(), m.PendingEditTask.TimeWindow.Start.Minute(), m.PendingEditTask.TimeWindow.Start.Second(), 0, m.PendingEditTask.TimeWindow.Start.Location())
									instance.TimeWindow.End = instance.TimeWindow.Start.Add(durationShift)
								}
								m.DB.AddTask(instance)
							}
							current = current.AddDate(0, 0, 1)
						}
						m.StatusMsg = "This and all future occurrences updated with the new recurrence pattern."
					}

					m.refreshTasks()
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.RecurringEditFromForm = false
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

