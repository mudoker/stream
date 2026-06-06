package tui

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.SessionExpiryPromptOpen {
		switch msg.String() {
		case "y", "Y", "enter":
			m.SessionTimeRemainingSeconds = m.DB.GetUserSettings().LockTimeoutMinutes * 60
			m.SessionExpiryPromptOpen = false
			m.StatusMsg = "Session timer reset."
			return m, nil
		case "n", "N", "esc":
			m.SessionExpiryPromptOpen = false
			m.StatusMsg = "Session will lock in 1 minute."
			return m, nil
		}
		return m, nil
	}

	if m.ConfirmOpen {
		switch msg.String() {
		case "y", "Y":
			m.DB.DeleteTask(m.ConfirmTask.UUID)
			m.refreshTasks()
			m.Sync.TriggerSync()
			if m.DetailOpen && m.DetailTask.UUID == m.ConfirmTask.UUID {
				m.DetailOpen = false
			}
			m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", m.ConfirmTask.Title)
			m.ConfirmOpen = false
			return m, nil
		case "n", "N", "esc", "enter":
			m.ConfirmOpen = false
			m.StatusMsg = "Deletion canceled."
			return m, nil
		}
		return m, nil
	}

	if m.HelpOpen {
		switch msg.String() {
		case "esc", "q", "?":
			m.HelpOpen = false
			m.HelpScrollOffset = 0
			return m, nil
		case "j", "down":
			m.HelpScrollOffset++
			return m, nil
		case "k", "up":
			if m.HelpScrollOffset > 0 {
				m.HelpScrollOffset--
			}
			return m, nil
		case "ctrl+d":
			m.HelpScrollOffset += 5
			return m, nil
		case "ctrl+u":
			m.HelpScrollOffset -= 5
			if m.HelpScrollOffset < 0 {
				m.HelpScrollOffset = 0
			}
			return m, nil
		case "g":
			m.HelpScrollOffset = 0
			return m, nil
		case "G":
			m.HelpScrollOffset = 9999
			return m, nil
		}
		return m, nil
	}

	if m.AnchorPromptOpen {
		switch msg.String() {
		case "tab", "down":
			m.AnchorActiveField = (m.AnchorActiveField + 1) % 2
			m.focusAnchorPromptFields()
			return m, nil
		case "shift+tab", "up":
			m.AnchorActiveField = (m.AnchorActiveField - 1 + 2) % 2
			m.focusAnchorPromptFields()
			return m, nil
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
			if m.Sync != nil {
				m.Sync.TriggerSync()
			}

			m.StatusMsg = fmt.Sprintf("Task '%s' anchored to %s.", t.Title, startTime.Format("15:04"))
			m.AnchorPromptOpen = false
			return m, nil

		case "esc":
			m.AnchorPromptOpen = false
			m.StatusMsg = "Anchoring canceled."
			return m, nil
		}

		var cmd tea.Cmd
		if m.AnchorActiveField == 0 {
			m.AnchorTimeInput, cmd = m.AnchorTimeInput.Update(msg)
		} else {
			m.AnchorDurationInput, cmd = m.AnchorDurationInput.Update(msg)
		}
		return m, cmd
	}

	if m.PromptOpen {
		switch msg.String() {
		case "enter":
			if m.PromptTask.SchedulingType == model.Reminder {
				m.PromptOpen = false
				m.StatusMsg = fmt.Sprintf("Reminder '%s' dismissed.", m.PromptTask.Title)
				return m, nil
			}
			m.startZenMode(m.PromptTask)
			m.PromptOpen = false
			return m, nil
		case "s":
			m.PromptTask.TimeWindow.Start = m.PromptTask.TimeWindow.Start.Add(5 * time.Minute)
			if m.PromptTask.SchedulingType != model.Reminder {
				m.PromptTask.TimeWindow.End = m.PromptTask.TimeWindow.End.Add(5 * time.Minute)
			}
			m.DB.UpdateTask(m.PromptTask)
			m.refreshTasks()
			m.PromptOpen = false
			m.StatusMsg = "Task start snoozed by 5m."
			return m, nil
		case "d", "esc":
			m.PromptTask.LifecycleState = model.StateReady
			m.DB.UpdateTask(m.PromptTask)
			m.refreshTasks()
			m.PromptOpen = false
			m.StatusMsg = "Task prompt dismissed."
			return m, nil
		}
		return m, nil
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
			return m, nil
		case "n", "esc":
			m.ReviewOpen = false
			m.StatusMsg = "Daily review closed."
			return m, nil
		}
		return m, nil
	}

	if m.DetailOpen {
		switch msg.String() {
		case "esc", "enter":
			m.DetailOpen = false
			return m, nil
		case "z":
			m.startZenMode(m.DetailTask)
			m.DetailOpen = false
			return m, nil
		case "x":
			m.DetailTask.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(m.DetailTask)
			m.refreshTasks()
			m.DetailOpen = false
			m.StatusMsg = fmt.Sprintf("Task '%s' completed!", m.DetailTask.Title)
			return m, nil
		case "d":
			m.ConfirmTask = m.DetailTask
			m.ConfirmOpen = true
			return m, nil
		case "e":
			m.startEditMode(m.DetailTask)
			m.DetailOpen = false
			return m, nil
		}
		return m, nil
	}

	if msg.String() == "esc" {
		if m.DetailOpen {
			m.DetailOpen = false
			return m, nil
		}
		if m.CurrentMode == ModeZen {
			m.CurrentMode = ModeNormal
			m.StatusMsg = "Focus Session running in background. Press 'z' to return."
			return m, nil
		}
		m.CurrentMode = ModeNormal
		return m, nil
	}

	switch m.CurrentMode {
	case ModeZen:
		return m.handleZenKeys(msg)
	case ModeCommand:
		return m.handleCommandKeys(msg)
	case ModeForm:
		return m.handleFormKeys(msg)
	case ModeTaskMove:
		return m.handleTaskMoveKeys(msg)
	case ModeWorkspaceForm:
		return m.handleWorkspaceFormKeys(msg)
	case ModeProfileForm:
		return m.handleProfileFormKeys(msg)
	case ModeWorkspacePicker:
		return m.handleWorkspacePickerKeys(msg)
	case ModeNormal:
		return m.handleNormalKeys(msg)
	}

	return m, nil
}
