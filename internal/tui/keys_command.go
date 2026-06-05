package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func (m *Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := m.CommandInput.Value()
		m.CurrentMode = ModeNormal
		return m.runCommand(val)
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.CommandInput, cmd = m.CommandInput.Update(msg)
	return m, cmd
}

func (m *Model) runCommand(val string) (tea.Model, tea.Cmd) {
	val = strings.TrimSpace(val)
	if val == "" {
		return m, nil
	}

	parts := strings.Fields(val)
	cmdName := parts[0]

	switch cmdName {
	case "q", "quit", "exit":
		return m, tea.Quit

	case "dashboard":
		m.CurrentView = DashboardView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Dashboard view."
	case "month":
		m.CurrentView = MonthView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Month view."
	case "week":
		m.CurrentView = WeekView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Week view."
	case "day":
		m.CurrentView = DayView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Day view."
	case "analytics":
		m.CurrentView = AnalyticsView
		m.ScrollOffset = 0
		m.ShelfScrollOffset = 0
		m.StatusMsg = "Switched to Analytics view."
	case "help", "h", "?":
		m.HelpOpen = true
		m.StatusMsg = "Help opened. Press Esc/Enter to exit."

	case "create", "todo":
		if len(parts) < 2 {
			m.StatusMsg = "Syntax: create <task title>"
			return m, nil
		}
		title := strings.Join(parts[1:], " ")
		newTask := model.Task{
			UUID:           uuid.New().String(),
			Title:          title,
			Priority:       model.P2,
			StoryPoints:    3,
			SchedulingType: model.Floating,
			LifecycleState: model.StateReady,
		}
		if cmdName == "create" {
			// Anchor it for today 9:00 - 10:00
			now := time.Now()
			start := time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), 9, 0, 0, 0, now.Location())
			end := start.Add(1 * time.Hour)
			newTask.SchedulingType = model.Anchored
			newTask.TimeWindow = model.TimeWindow{Start: start, End: end}
			newTask.LifecycleState = model.StateScheduled
		}
		m.DB.AddTask(newTask)
		m.refreshTasks()
		m.Sync.TriggerSync()
		m.StatusMsg = fmt.Sprintf("Task '%s' created.", title)

	case "review":
		today := time.Now()
		completed := 0
		deferred := 0
		secs := 0
		for _, t := range m.Tasks {
			isToday := false
			if t.SchedulingType == model.Anchored {
				isToday = t.TimeWindow.Start.Year() == today.Year() && t.TimeWindow.Start.Month() == today.Month() && t.TimeWindow.Start.Day() == today.Day()
			} else {
				isToday = t.CreatedAt.Year() == today.Year() && t.CreatedAt.Month() == today.Month() && t.CreatedAt.Day() == today.Day()
			}

			if isToday {
				if t.LifecycleState == model.StateCompleted {
					completed++
				} else {
					deferred++
				}
				secs += t.ExecutionMetrics.ElapsedFocusSeconds
			}
		}
		m.ReviewTasksCompleted = completed
		m.ReviewTasksDeferred = deferred
		m.ReviewFocusSeconds = secs
		m.ReviewOpen = true
		m.StatusMsg = "Daily Shutdown Review opened."

	case "complete":
		task, exists := m.getActiveTask()
		if exists {
			task.LifecycleState = model.StateCompleted
			m.DB.UpdateTask(task)
			m.refreshTasks()
			m.Sync.TriggerSync()
			m.StatusMsg = fmt.Sprintf("Task '%s' completed.", task.Title)
		}

	case "delete":
		task, exists := m.getActiveTask()
		if exists {
			m.DB.DeleteTask(task.UUID)
			m.refreshTasks()
			m.Sync.TriggerSync()
			m.StatusMsg = fmt.Sprintf("Task '%s' deleted.", task.Title)
		}

	case "sync":
		m.Sync.TriggerSync()
		m.StatusMsg = "Triggering Google Calendar sync..."

	case "auth":
		url, err := m.Sync.StartAuthServer(8080)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Auth server error: %v", err)
		} else {
			m.StatusMsg = "Go to: " + url
		}

	default:
		m.StatusMsg = fmt.Sprintf("Unknown command: %s", cmdName)
	}

	return m, nil
}
