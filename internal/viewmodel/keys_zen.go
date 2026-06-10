package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/timer"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) StartZenMode(task model.Task) {
	task.LifecycleState = model.StateActive
	m.DB.UpdateTask(task)
	m.refreshTasks()

	m.ZenTimer = timer.NewZenTimer(task)
	m.CurrentMode = ModeZen
}

func (m *Model) HandleZenKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ZenTimer == nil {
		m.CurrentMode = ModeNormal
		return m, nil
	}

	key := msg.String()
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		m.ZenPrefix += key
		m.StatusMsg = fmt.Sprintf("Multiplier: %s", m.ZenPrefix)
		return m, nil
	}

	switch key {
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
		multiplier := 1
		if m.ZenPrefix != "" {
			if val, err := strconv.Atoi(m.ZenPrefix); err == nil && val > 0 {
				multiplier = val
			}
		}
		m.ZenPrefix = ""
		dur := time.Duration(multiplier) * 30 * time.Second
		m.ZenTimer.AddTime(dur)
		m.StatusMsg = fmt.Sprintf("Added %s to countdown.", formatDuration(dur))
	case "-":
		multiplier := 1
		if m.ZenPrefix != "" {
			if val, err := strconv.Atoi(m.ZenPrefix); err == nil && val > 0 {
				multiplier = val
			}
		}
		m.ZenPrefix = ""
		dur := time.Duration(multiplier) * 30 * time.Second
		m.ZenTimer.AddTime(-dur)
		m.StatusMsg = fmt.Sprintf("Subtracted %s from countdown.", formatDuration(dur))
	case "r":
		// Restart current session block
		m.ZenTimer.RecordElapsedTimes()
		if m.DB != nil {
			m.DB.UpdateTask(m.ZenTimer.Task)
			m.refreshTasks()
		}
		sess := m.ZenTimer.Sessions[m.ZenTimer.CurrentSessionIdx]
		m.ZenTimer.TimeRemaining = sess.Duration
		m.ZenTimer.TotalDuration = sess.Duration
		m.StatusMsg = "Timer RESTARTED"
	case "q":
		// Stop/Abort focus session completely
		m.ZenTimer.RecordElapsedTimes()
		t := m.ZenTimer.Task
		t.LifecycleState = model.StateReady
		if m.DB != nil {
			m.DB.UpdateTask(t)
			m.refreshTasks()
		}
		m.ZenTimer = nil
		m.CurrentMode = ModeNormal
		m.StatusMsg = "Timer STOPPED"
	case "b":
		// Force Break
		m.ZenTimer.RecordElapsedTimes()
		finished := m.ZenTimer.NextSession()
		if m.DB != nil {
			m.DB.UpdateTask(m.ZenTimer.Task)
			m.refreshTasks()
		}
		if finished {
			t := m.ZenTimer.Task
			t.LifecycleState = model.StateCompleted
			if m.DB != nil {
				m.DB.UpdateTask(t)
				m.refreshTasks()
			}
			m.ZenTimer = nil
			m.CurrentMode = ModeNormal
			m.StatusMsg = "Focus sessions completed!"
		} else {
			m.StatusMsg = "Skipped to next block."
		}
	}
	return m, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if secs == 0 {
		return fmt.Sprintf("%d minutes", mins)
	}
	return fmt.Sprintf("%dm %ds", mins, secs)
}

