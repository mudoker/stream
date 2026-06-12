package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handlePromptDialogKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.AnchorPromptOpen {
		switch msg.String() {
		case "tab", "down":
			m.AnchorActiveField = (m.AnchorActiveField + 1) % 2
			m.focusAnchorPromptFields()
			return true, nil
		case "shift+tab", "up":
			m.AnchorActiveField = (m.AnchorActiveField - 1 + 2) % 2
			m.focusAnchorPromptFields()
			return true, nil
		case "enter":
			timeStr := m.AnchorTimeInput.Value()
			hour, min := ParseFlexibleTime(timeStr, 9, 0)

			now := time.Now()
			startTime := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())

			durStr := m.AnchorDurationInput.Value()
			durationMins := 60
			if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
				durationMins = d
			}
			dur := time.Duration(durationMins) * time.Minute

			t := m.AnchorPromptTask
			t.SchedulingType = model.Anchored
			t.TimeWindow = model.TimeWindow{
				Start: startTime,
				End:   startTime.Add(dur),
			}
			t.LifecycleState = model.StateScheduled

			if m.DB != nil {
				m.DB.UpdateTask(t)
				m.refreshTasks()
			} else {
				m.updateTaskInMemory(t)
			}
			m.SelectedTaskUUID = t.UUID
			m.AutoScrollToSelectedTask()
			m.triggerGCalPush(t)

			m.StatusMsg = fmt.Sprintf("Task '%s' anchored to %s.", t.Title, startTime.Format("15:04"))
			m.AnchorPromptOpen = false
			return true, nil

		case "esc":
			m.AnchorPromptOpen = false
			m.StatusMsg = "Anchoring canceled."
			return true, nil
		}

		var cmd tea.Cmd
		if m.AnchorActiveField == 0 {
			m.AnchorTimeInput, cmd = m.AnchorTimeInput.Update(msg)
		} else {
			m.AnchorDurationInput, cmd = m.AnchorDurationInput.Update(msg)
		}
		return true, cmd
	}

	if m.PromptOpen {
		switch msg.String() {
		case "left", "up", "shift+tab":
			m.PromptSelectedIdx = (m.PromptSelectedIdx - 1 + 3) % 3
			return true, nil
		case "right", "down", "tab":
			m.PromptSelectedIdx = (m.PromptSelectedIdx + 1) % 3
			return true, nil
		case "s", "S":
			m.PromptSelectedIdx = 1
			return true, nil
		case "d", "D":
			m.PromptSelectedIdx = 2
			return true, nil
		case "f", "F":
			m.PromptSelectedIdx = 0
			return true, nil
		case "esc":
			m.cancelPromptTask()
			m.StatusMsg = "Task prompt dismissed."
			return true, nil
		case "enter":
			switch m.PromptSelectedIdx {
			case 0: // Start Focus (or Dismiss if Reminder)
				if m.PromptTask.SchedulingType == model.Reminder {
					m.PromptOpen = false
					m.StatusMsg = fmt.Sprintf("Reminder '%s' dismissed.", m.PromptTask.Title)
					return true, nil
				}
				m.StartZenMode(m.PromptTask)
				m.PromptOpen = false
				return true, nil
			case 1: // Snooze 5m
				m.PromptTask.TimeWindow.Start = m.PromptTask.TimeWindow.Start.Add(5 * time.Minute)
				if m.PromptTask.SchedulingType != model.Reminder {
					m.PromptTask.TimeWindow.End = m.PromptTask.TimeWindow.End.Add(5 * time.Minute)
				}
				m.DB.UpdateTask(m.PromptTask)
				m.refreshTasks()
				m.PromptOpen = false
				m.StatusMsg = "Task start snoozed by 5m."
				return true, nil
			case 2: // Dismiss
				m.PromptTask.LifecycleState = model.StateReady
				m.DB.UpdateTask(m.PromptTask)
				m.refreshTasks()
				m.PromptOpen = false
				m.StatusMsg = "Task prompt dismissed."
				return true, nil
			}
		}
		return true, nil
	}

	if m.ReviewOpen {
		switch msg.String() {
		case "y", "enter":
			today := time.Now()
			shifted := 0
			for _, t := range m.Tasks {
				if t.SchedulingType == model.Anchored &&
					t.TimeWindow.Start.Year() == today.Year() && t.TimeWindow.Start.Month() == today.Month() && t.TimeWindow.Start.Day() == today.Day() &&
					t.LifecycleState != model.StateCompleted {

					t.TimeWindow.Start = t.TimeWindow.Start.AddDate(0, 0, 1)
					t.TimeWindow.End = t.TimeWindow.End.AddDate(0, 0, 1)
					t.LifecycleState = model.StateScheduled
					m.DB.UpdateTask(t)
					shifted++
				}
			}
			m.refreshTasks()
			m.ReviewOpen = false
			m.StatusMsg = fmt.Sprintf("Moved %d incomplete tasks to tomorrow.", shifted)
			return true, nil
		case "n", "esc":
			m.ReviewOpen = false
			m.StatusMsg = "Daily review closed."
			return true, nil
		}
		return true, nil
	}

	return false, nil
}

func (m *Model) cancelPromptTask() {
	if m.PromptTask.SchedulingType != model.Reminder {
		m.PromptTask.LifecycleState = model.StateReady
		if m.DB != nil {
			m.DB.UpdateTask(m.PromptTask)
			m.refreshTasks()
		}
	}
	m.PromptOpen = false
}
