package tui

import (
	"fmt"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Layout = computeLayout(msg.Width, msg.Height)
		return m, nil

	case TickMsg:
		// Tick Zen Timer if active
		if m.CurrentMode == ModeZen && m.ZenTimer != nil {
			finished := m.ZenTimer.Tick()
			if finished {
				// Record completion metrics
				t := m.ZenTimer.Task
				t.LifecycleState = model.StateCompleted
				t.ExecutionMetrics.ElapsedFocusSeconds += int(m.ZenTimer.TotalDuration.Seconds())
				t.ExecutionMetrics.TotalCompletedPomodoros += 1
				m.DB.UpdateTask(t)
				m.refreshTasks()

				m.CurrentMode = ModeNormal
				m.StatusMsg = fmt.Sprintf("Completed Focus Session for %s!", t.Title)
			}
		}

		// Check for auto-activation tasks
		if m.CurrentMode == ModeNormal && !m.PromptOpen && !m.ReviewOpen {
			now := time.Now()
			for _, t := range m.Tasks {
				if t.SchedulingType == model.Anchored &&
					t.LifecycleState == model.StateScheduled &&
					t.TimeWindow.Start.Year() == now.Year() && t.TimeWindow.Start.Month() == now.Month() && t.TimeWindow.Start.Day() == now.Day() &&
					t.TimeWindow.Start.Hour() == now.Hour() && t.TimeWindow.Start.Minute() == now.Minute() {

					m.PromptTask = t
					m.PromptOpen = true
					break
				}
			}
		}

		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case SyncLogMsg:
		m.SyncLogs = append(m.SyncLogs, msg.Message)
		if len(m.SyncLogs) > 5 {
			m.SyncLogs = m.SyncLogs[1:]
		}
		m.StatusMsg = msg.Message
		m.LastSyncTime = time.Now()
		m.refreshTasks()
		return m, nil

	case tea.KeyMsg:
		if m.HelpOpen {
			switch msg.String() {
			case "esc", "enter", "q", " ", "?":
				m.HelpOpen = false
				return m, nil
			}
			return m, nil
		}

		if m.PromptOpen {
			switch msg.String() {
			case "enter":
				m.startZenMode(m.PromptTask)
				m.PromptOpen = false
				return m, nil
			case "s":
				// Snooze 5 minutes
				m.PromptTask.TimeWindow.Start = m.PromptTask.TimeWindow.Start.Add(5 * time.Minute)
				m.PromptTask.TimeWindow.End = m.PromptTask.TimeWindow.End.Add(5 * time.Minute)
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
				m.DB.DeleteTask(m.DetailTask.UUID)
				m.refreshTasks()
				m.DetailOpen = false
				m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", m.DetailTask.Title)
				return m, nil
			}
			return m, nil
		}

		// ESC is universal: drop back to NORMAL mode, dismiss overlays
		if msg.String() == "esc" {
			if m.DetailOpen {
				m.DetailOpen = false
				return m, nil
			}
			if m.CurrentMode == ModeZen {
				// Abort session
				if m.ZenTimer != nil {
					t := m.ZenTimer.Task
					t.LifecycleState = model.StatePaused
					t.ExecutionMetrics.ElapsedFocusSeconds += int((m.ZenTimer.TotalDuration - m.ZenTimer.TimeRemaining).Seconds())
					m.DB.UpdateTask(t)
					m.refreshTasks()
				}
				m.CurrentMode = ModeNormal
				m.StatusMsg = "Focus Session aborted."
				return m, nil
			}
			m.CurrentMode = ModeNormal
			return m, nil
		}

		// Mode-Specific Key Handlers
		switch m.CurrentMode {
		case ModeZen:
			return m.handleZenKeys(msg)

		case ModeCommand:
			return m.handleCommandKeys(msg)

		case ModeForm:
			return m.handleFormKeys(msg)

		case ModeNormal:
			return m.handleNormalKeys(msg)
		}
	}

	return m, nil
}
