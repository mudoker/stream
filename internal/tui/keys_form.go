package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func (m *Model) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab", "down":
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
		m.focusFormFields()
		return m, nil
	case "shift+tab", "up":
		m.Form.ActiveField = (m.Form.ActiveField - 1 + 8) % 8
		m.focusFormFields()
		return m, nil
	case "enter":
		if m.Form.ActiveField == 7 { // Submit
			m.submitForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		m.Form.ActiveField = (m.Form.ActiveField + 1) % 8
		m.focusFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput, cmd = m.Form.TitleInput.Update(msg)
	case 1:
		m.Form.DescInput, cmd = m.Form.DescInput.Update(msg)
	case 2:
		m.Form.PriorityInput, cmd = m.Form.PriorityInput.Update(msg)
	case 3:
		m.Form.SPInput, cmd = m.Form.SPInput.Update(msg)
	case 4:
		m.Form.AnchorInput, cmd = m.Form.AnchorInput.Update(msg)
	case 5:
		m.Form.StartTimeInput, cmd = m.Form.StartTimeInput.Update(msg)
	case 6:
		m.Form.DurationInput, cmd = m.Form.DurationInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusFormFields() {
	m.Form.TitleInput.Blur()
	m.Form.DescInput.Blur()
	m.Form.PriorityInput.Blur()
	m.Form.SPInput.Blur()
	m.Form.AnchorInput.Blur()
	m.Form.StartTimeInput.Blur()
	m.Form.DurationInput.Blur()

	switch m.Form.ActiveField {
	case 0:
		m.Form.TitleInput.Focus()
	case 1:
		m.Form.DescInput.Focus()
	case 2:
		m.Form.PriorityInput.Focus()
	case 3:
		m.Form.SPInput.Focus()
	case 4:
		m.Form.AnchorInput.Focus()
	case 5:
		m.Form.StartTimeInput.Focus()
	case 6:
		m.Form.DurationInput.Focus()
	}
}

func (m *Model) submitForm() {
	title := m.Form.TitleInput.Value()
	if title == "" {
		m.StatusMsg = "Title cannot be empty."
		return
	}

	priorityVal := model.Priority(strings.ToUpper(m.Form.PriorityInput.Value()))
	if priorityVal != model.P0 && priorityVal != model.P1 && priorityVal != model.P2 && priorityVal != model.P3 {
		priorityVal = model.P2
	}

	spVal, err := strconv.Atoi(m.Form.SPInput.Value())
	if err != nil || spVal <= 0 {
		spVal = 3
	}

	anchored := true
	if strings.ToUpper(m.Form.AnchorInput.Value()) == "N" {
		anchored = false
	}

	var startTime time.Time
	duration := 60

	if anchored {
		timeStr := m.Form.StartTimeInput.Value()
		durStr := m.Form.DurationInput.Value()

		hour, min := 9, 0
		if parts := strings.Split(timeStr, ":"); len(parts) == 2 {
			h, _ := strconv.Atoi(parts[0])
			mVal, _ := strconv.Atoi(parts[1])
			if h >= 0 && h < 24 {
				hour = h
			}
			if mVal >= 0 && mVal < 60 {
				min = mVal
			}
		}

		if d, err := strconv.Atoi(durStr); err == nil && d > 0 {
			duration = d
		}

		now := time.Now()
		startTime = time.Date(m.SelectedDay.Year(), m.SelectedDay.Month(), m.SelectedDay.Day(), hour, min, 0, 0, now.Location())
	}

	newTask := model.Task{
		UUID:        uuid.New().String(),
		Title:       title,
		Description: m.Form.DescInput.Value(),
		Priority:    priorityVal,
		StoryPoints: spVal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if anchored {
		newTask.SchedulingType = model.Anchored
		newTask.TimeWindow = model.TimeWindow{
			Start: startTime,
			End:   startTime.Add(time.Duration(duration) * time.Minute),
		}
		newTask.LifecycleState = model.StateScheduled
	} else {
		newTask.SchedulingType = model.Floating
		newTask.LifecycleState = model.StateReady
	}

	m.DB.AddTask(newTask)
	m.refreshTasks()
	m.Sync.TriggerSync()
	m.StatusMsg = fmt.Sprintf("Task '%s' created successfully.", title)
}
