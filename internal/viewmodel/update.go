package viewmodel

import (
	"fmt"
	"time"

	"stream/internal/db"
	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.IsLocked {
		switch msg := msg.(type) {
		case TickMsg:
			cmds = append(cmds, tickCmd())
			return m, tea.Batch(cmds...)
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height
			m.Layout = ComputeLayout(msg.Width, msg.Height)
			return m, nil
		case tea.KeyMsg:
			key := msg.String()
			if key == "enter" {
				entered := m.LockPasswordInput.Value()
				storedHash := m.DB.GetUserSettings().PasswordHash
				if db.HashPassword(entered) == storedHash {
					m.IsLocked = false
					m.SessionTimeRemainingSeconds = m.DB.GetUserSettings().LockTimeoutMinutes * 60
					m.LockPasswordInput.SetValue("")
					m.StatusMsg = "Session unlocked."
				} else {
					m.StatusMsg = "❌ Incorrect password"
					m.LockPasswordInput.SetValue("")
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.LockPasswordInput, cmd = m.LockPasswordInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Layout = ComputeLayout(msg.Width, msg.Height)
		return m, nil

	case TickMsg:
		if m.ZenTimer != nil {
			oldIdx := m.ZenTimer.CurrentSessionIdx
			finished := m.ZenTimer.Tick()

			if m.ZenTimer.CurrentSessionIdx != oldIdx || finished {
				if m.DB != nil {
					m.DB.UpdateTask(m.ZenTimer.Task)
					m.refreshTasks()
				}
			}

			if finished {
				t := m.ZenTimer.Task
				t.LifecycleState = model.StateCompleted
				if m.DB != nil {
					m.DB.UpdateTask(t)
					m.refreshTasks()
				}
				m.ZenTimer = nil

				if m.CurrentMode == ModeZen {
					m.CurrentMode = ModeNormal
				}
				m.StatusMsg = fmt.Sprintf("Completed Focus Session for %s!", t.Title)
			}
		}

		if m.DB != nil {
			storedHash := m.DB.GetUserSettings().PasswordHash
			if storedHash != "" {
				if m.CurrentMode == ModeZen {
					m.SessionTimeRemainingSeconds = m.DB.GetUserSettings().LockTimeoutMinutes * 60
					m.SessionExpiryPromptOpen = false
				} else {
					m.SessionTimeRemainingSeconds--
					if m.SessionTimeRemainingSeconds <= 0 {
						m.IsLocked = true
						m.SessionExpiryPromptOpen = false
						m.CurrentMode = ModeNormal
					} else if m.SessionTimeRemainingSeconds == 60 {
						m.SessionExpiryPromptOpen = true
					}
				}
			}
		}

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
			if !m.PromptOpen {
				for _, t := range m.Tasks {
					if t.SchedulingType == model.Reminder && t.LifecycleState == model.StateReady {
						due := t.TimeWindow.Start
						if !due.Before(now.Add(-5*time.Minute)) && due.Before(now.Add(1*time.Minute)) && m.PromptTask.UUID != t.UUID {
							m.PromptTask = t
							m.PromptOpen = true
							break
						}
					}
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
		if !m.AuthNoticeOpen {
			m.StatusMsg = msg.Message
		}
		m.LastSyncTime = time.Now()
		m.refreshTasks()
		return m, nil

	case AuthCompleteMsg:
		m.AuthNoticeOpen = false
		m.AuthNoticeMsg = ""
		m.StatusMsg = "Google Calendar authorization successful."
		m.refreshTasks()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}
