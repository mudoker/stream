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
			m.LogRemainingOnConfirm = false
			m.ShrinkRemainingMins = 0
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

		// Handle key navigation between options
		if keyStr == "j" || keyStr == "down" || keyStr == "l" || keyStr == "right" || keyStr == "tab" {
			m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex + 1) % numOpts
			return true, nil
		} else if keyStr == "k" || keyStr == "up" || keyStr == "h" || keyStr == "left" || keyStr == "shift+tab" {
			m.ConfirmSelectedIndex = (m.ConfirmSelectedIndex - 1 + numOpts) % numOpts
			return true, nil
		}

		if keyStr == "enter" {
			switch m.ConfirmActionType {
			case "factory_reset":
				if m.ConfirmSelectedIndex == 0 {
					if m.FactoryResetCountdown > 0 {
						m.StatusMsg = fmt.Sprintf("⚠️ Please wait %d seconds before confirming factory reset.", m.FactoryResetCountdown)
						return true, nil
					}
					if m.DB != nil {
						_ = m.DB.FactoryReset()
					}
					m.refreshWorkspaces()
					m.refreshSprints()
					m.refreshTasks()
					m.CurrentView = DashboardView
					m.SelectedTaskUUID = ""
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.FactoryResetCountdown = 0
					m.StatusMsg = "💥 Factory reset complete. All data has been wiped."
					return true, nil
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.FactoryResetCountdown = 0
					m.StatusMsg = "Factory reset cancelled."
					return true, nil
				}
			case "save_tag_confirm":
				if m.ConfirmSelectedIndex == 0 {
					tags := m.DB.GetTags()
					for _, tagName := range m.PendingNewTags {
						tags = append(tags, model.TagInfo{Name: tagName, Frequency: 1})
					}
					m.DB.SaveTags(tags)
				}
				m.FinalizeSubmitTask(m.PendingTaskToSubmit)
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.PendingNewTags = nil
				m.CurrentMode = ModeNormal
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
			case "delete_sprint":
				if m.ConfirmSelectedIndex == 0 {
					common.ConfirmDeleteSprint(m, m.ConfirmSprint)
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmSprint = model.Sprint{}
					m.StatusMsg = "Sprint deletion canceled."
				}
			case "create_task_from_feature":
				if m.ConfirmSelectedIndex == 0 {
					feat := m.ConfirmTask
					taskID := GenerateWorkItemID(model.WorkItemTask, m.Tasks)
					newTask := model.Task{
						UUID:            uuid.New().String(),
						ID:              taskID,
						WorkspaceUUID:   feat.WorkspaceUUID,
						Title:           feat.Title,
						Description:     feat.Description,
						Priority:        feat.Priority,
						StoryPoints:     feat.StoryPoints,
						SchedulingType:  model.Floating,
						LifecycleState:  model.StateReady,
						AddedToToday:    true,
						LinkedFeatureID: feat.ID,
						Tags:            feat.Tags,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}
					if newTask.WorkspaceUUID == "" {
						newTask.WorkspaceUUID = m.ActiveWorkspaceUUID
					}
					m.DB.AddTask(newTask)
					m.refreshTasks()
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmTask = model.Task{}
					m.SelectedTaskUUID = newTask.UUID
					m.StatusMsg = fmt.Sprintf("Created task '%s' linked to %s and added to Today Shelf.", newTask.Title, feat.ID)
					return true, nil
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmTask = model.Task{}
					m.StatusMsg = "Task creation canceled."
					return true, nil
				}
			case "anchor_task_to_today":
				if m.ConfirmSelectedIndex == 0 {
					t := m.ConfirmTask
					now := time.Now()
					durMins := t.EstimatedDurationMins
					if durMins <= 0 {
						durMins = t.StoryPoints * 45
						if durMins <= 0 {
							durMins = 60
						}
					}
					min := (now.Minute() / 15) * 15
					startTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), min, 0, 0, now.Location())
					dur := time.Duration(durMins) * time.Minute

					t.SchedulingType = model.Anchored
					t.LifecycleState = model.StateScheduled
					t.SprintUUID = ""
					t.AddedToToday = false
					t.TimeWindow = model.TimeWindow{
						Start: startTime,
						End:   startTime.Add(dur),
					}
					t.UpdatedAt = time.Now()

					if m.DB != nil {
						m.DB.UpdateTask(t)
						m.refreshTasks()
					} else {
						m.updateTaskInMemory(t)
					}

					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmTask = model.Task{}
					m.CurrentView = DayView
					m.SelectedDay = now
					m.SelectedTaskUUID = t.UUID
					m.TimelineHour = startTime.Hour()
					m.ScrollOffset = 0
					m.SidebarFocus = false
					m.TodoShelfFocus = false
					m.AutoScrollToSelectedTask()
					m.triggerGCalPush(t)
					m.StatusMsg = fmt.Sprintf("Task '%s' anchored to today's timeline at %s.", t.Title, startTime.Format("15:04"))
					return true, nil
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmTask = model.Task{}
					m.StatusMsg = "Anchoring canceled."
					return true, nil
				}
			case "clear_shelf_section":
				if m.ConfirmSelectedIndex == 0 {
					m.ConfirmClearShelfSection()
				} else {
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.ConfirmShelfSection = ShelfSection{}
					m.StatusMsg = "Clear section canceled."
				}
			case "edit_recurring":
				if m.ConfirmSelectedIndex == 0 {
					m.DB.UpdateTask(m.PendingEditTask)
					if m.LogRemainingOnConfirm {
						remainingTitle := m.PendingEditTask.Title
						if !strings.HasSuffix(remainingTitle, " (remaining)") {
							remainingTitle += " (remaining)"
						}
						remainingTask := model.Task{
							UUID:                  uuid.New().String(),
							WorkspaceUUID:         m.PendingEditTask.WorkspaceUUID,
							Title:                 remainingTitle,
							Description:           m.PendingEditTask.Description,
							Priority:              m.PendingEditTask.Priority,
							StoryPoints:           m.PendingEditTask.StoryPoints,
							SchedulingType:        model.Floating,
							LifecycleState:        model.StateReady,
							EstimatedDurationMins: m.ShrinkRemainingMins,
							Tags:                  m.PendingEditTask.Tags,
							Notes:                 m.PendingEditTask.Notes,
							CreatedAt:             time.Now(),
							UpdatedAt:             time.Now(),
						}
						m.DB.AddTask(remainingTask)
						m.StatusMsg = fmt.Sprintf("Occurrence of '%s' updated; remaining %dm logged to Todo Shelf.", m.PendingEditTask.Title, m.ShrinkRemainingMins)
					} else {
						m.StatusMsg = fmt.Sprintf("Occurrence of '%s' updated.", m.PendingEditTask.Title)
					}
					m.refreshTasks()
					m.triggerGCalPush(m.PendingEditTask)
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					m.RecurringEditFromForm = false
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
						if m.LogRemainingOnConfirm {
							remainingTitle := m.PendingEditTask.Title
							if !strings.HasSuffix(remainingTitle, " (remaining)") {
								remainingTitle += " (remaining)"
							}
							remainingTask := model.Task{
								UUID:                  uuid.New().String(),
								WorkspaceUUID:         m.PendingEditTask.WorkspaceUUID,
								Title:                 remainingTitle,
								Description:           m.PendingEditTask.Description,
								Priority:              m.PendingEditTask.Priority,
								StoryPoints:           m.PendingEditTask.StoryPoints,
								SchedulingType:        model.Floating,
								LifecycleState:        model.StateReady,
								EstimatedDurationMins: m.ShrinkRemainingMins,
								Tags:                  m.PendingEditTask.Tags,
								Notes:                 m.PendingEditTask.Notes,
								CreatedAt:             time.Now(),
								UpdatedAt:             time.Now(),
							}
							m.DB.AddTask(remainingTask)
							m.StatusMsg = fmt.Sprintf("This and all future occurrences updated; remaining %dm logged to Todo Shelf.", m.ShrinkRemainingMins)
						} else {
							m.StatusMsg = "This and all future occurrences updated."
						}
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
			case "shrink_remaining_confirm":
				if m.ConfirmSelectedIndex == 0 {
					m.LogRemainingOnConfirm = true
				} else {
					m.LogRemainingOnConfirm = false
				}

				origDur := m.ConfirmTask.TimeWindow.End.Sub(m.ConfirmTask.TimeWindow.Start)
				newDur := m.PendingEditTask.TimeWindow.End.Sub(m.PendingEditTask.TimeWindow.Start)
				m.ShrinkRemainingMins = int((origDur - newDur).Minutes())

				if m.PendingEditTask.RecurringParentUUID != "" {
					m.ConfirmActionType = "edit_recurring"
					m.ConfirmSelectedIndex = 0
					m.RecurringEditFromForm = false
					m.StatusMsg = "Choose recurring update option."
				} else {
					m.DB.UpdateTask(m.PendingEditTask)
					if m.LogRemainingOnConfirm {
						remainingTitle := m.PendingEditTask.Title
						if !strings.HasSuffix(remainingTitle, " (remaining)") {
							remainingTitle += " (remaining)"
						}
						remainingTask := model.Task{
							UUID:                  uuid.New().String(),
							WorkspaceUUID:         m.PendingEditTask.WorkspaceUUID,
							Title:                 remainingTitle,
							Description:           m.PendingEditTask.Description,
							Priority:              m.PendingEditTask.Priority,
							StoryPoints:           m.PendingEditTask.StoryPoints,
							SchedulingType:        model.Floating,
							LifecycleState:        model.StateReady,
							EstimatedDurationMins: m.ShrinkRemainingMins,
							Tags:                  m.PendingEditTask.Tags,
							Notes:                 m.PendingEditTask.Notes,
							CreatedAt:             time.Now(),
							UpdatedAt:             time.Now(),
						}
						m.DB.AddTask(remainingTask)
					}
					m.refreshTasks()
					m.triggerGCalPush(m.PendingEditTask)
					m.ConfirmOpen = false
					m.ConfirmActionType = ""
					if m.LogRemainingOnConfirm {
						m.StatusMsg = fmt.Sprintf("Task '%s' shrunk; remaining %dm logged to Todo Shelf.", m.PendingEditTask.Title, m.ShrinkRemainingMins)
					} else {
						m.StatusMsg = fmt.Sprintf("Task '%s' shrunk; remaining duration discarded.", m.PendingEditTask.Title)
					}
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
			} else if m.ConfirmActionType == "shrink_remaining_confirm" {
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
				m.StatusMsg = "Shrink canceled."
			} else if m.ConfirmActionType == "factory_reset" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.FactoryResetCountdown = 0
				m.StatusMsg = "Factory reset cancelled."
			} else if m.ConfirmActionType == "delete_sprint" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.ConfirmSprint = model.Sprint{}
				m.StatusMsg = "Sprint deletion canceled."
			} else if m.ConfirmActionType == "clear_shelf_section" {
				m.ConfirmOpen = false
				m.ConfirmActionType = ""
				m.ConfirmShelfSection = ShelfSection{}
				m.StatusMsg = "Clear section canceled."
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

