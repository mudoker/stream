package tui

import (
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) startZenMode(task model.Task) {
	task.LifecycleState = model.StateActive
	m.DB.UpdateTask(task)
	m.refreshTasks()

	m.ZenTimer = NewZenTimer(task)
	m.CurrentMode = ModeZen
}

func (m *Model) handleZenKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ZenTimer == nil {
		m.CurrentMode = ModeNormal
		return m, nil
	}

	switch msg.String() {
	case " ", "space":
		m.ZenTimer.IsPaused = !m.ZenTimer.IsPaused
		if m.ZenTimer.IsPaused {
			m.StatusMsg = "Timer PAUSED"
			m.ZenTimer.Task.ExecutionMetrics.InterruptionCount++
			m.DB.UpdateTask(m.ZenTimer.Task)
			m.refreshTasks()
		} else {
			m.StatusMsg = "Timer RUNNING"
		}
	case "+":
		m.ZenTimer.AddTime(5 * time.Minute)
		m.StatusMsg = "Added 5 minutes to countdown."
	case "r":
		// Restart current session block
		sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]
		m.ZenTimer.TimeRemaining = sess.Duration
		m.ZenTimer.TotalDuration = sess.Duration
		m.StatusMsg = "Timer RESTARTED"
	case "q":
		// Stop/Abort focus session completely
		t := m.ZenTimer.Task
		t.LifecycleState = model.StateReady
		m.DB.UpdateTask(t)
		m.refreshTasks()
		m.ZenTimer = nil
		m.CurrentMode = ModeNormal
		m.StatusMsg = "Timer STOPPED"
	case "b":
		// Force Break
		finished := m.ZenTimer.NextSession()
		if finished {
			t := m.ZenTimer.Task
			t.LifecycleState = model.StateCompleted
			t.ExecutionMetrics.ElapsedFocusSeconds += int(m.ZenTimer.TotalDuration.Seconds())
			m.DB.UpdateTask(t)
			m.refreshTasks()
			m.CurrentMode = ModeNormal
			m.StatusMsg = "Focus sessions completed!"
		} else {
			m.StatusMsg = "Skipped to next block."
		}
	}
	return m, nil
}
