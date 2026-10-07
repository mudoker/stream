package viewmodel

import (
	"fmt"
	"strconv"
	"time"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func (m *Model) handleSprintFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab", "down":
		m.SprintForm.ActiveField = (m.SprintForm.ActiveField + 1) % 5
		m.focusSprintFormFields()
		return m, nil
	case "shift+tab", "up":
		m.SprintForm.ActiveField = (m.SprintForm.ActiveField - 1 + 5) % 5
		m.focusSprintFormFields()
		return m, nil
	case "enter":
		m.submitSprintForm()
		m.CurrentMode = ModeNormal
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	switch m.SprintForm.ActiveField {
	case 0:
		m.SprintForm.NameInput, cmd = m.SprintForm.NameInput.Update(msg)
	case 1:
		m.SprintForm.StartDateInput, cmd = m.SprintForm.StartDateInput.Update(msg)
	case 2:
		m.SprintForm.EndDateInput, cmd = m.SprintForm.EndDateInput.Update(msg)
	case 3:
		m.SprintForm.RecurringCountInput, cmd = m.SprintForm.RecurringCountInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusSprintFormFields() {
	m.SprintForm.NameInput.Blur()
	m.SprintForm.StartDateInput.Blur()
	m.SprintForm.EndDateInput.Blur()
	m.SprintForm.RecurringCountInput.Blur()

	switch m.SprintForm.ActiveField {
	case 0:
		m.SprintForm.NameInput.Focus()
	case 1:
		m.SprintForm.StartDateInput.Focus()
	case 2:
		m.SprintForm.EndDateInput.Focus()
	case 3:
		m.SprintForm.RecurringCountInput.Focus()
	}
}

func (m *Model) submitSprintForm() {
	name := m.SprintForm.NameInput.Value()
	if name == "" {
		name = "Sprint"
	}

	startStr := m.SprintForm.StartDateInput.Value()
	endStr := m.SprintForm.EndDateInput.Value()
	recStr := m.SprintForm.RecurringCountInput.Value()

	now := time.Now()
	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil || !endDate.After(startDate) {
		endDate = startDate.AddDate(0, 0, 14)
	}

	recCount, _ := strconv.Atoi(recStr)

	wsUUID := m.ActiveWorkspaceUUID
	if wsUUID == "ALL_WORKSPACES" {
		for _, ws := range m.Workspaces {
			if ws.UUID != "ALL_WORKSPACES" {
				wsUUID = ws.UUID
				break
			}
		}
	}

	if m.SprintForm.IsEditing {
		sprint, exists := m.DB.GetSprint(m.SprintForm.EditingSprintUUID)
		if exists {
			sprint.Name = name
			sprint.StartDate = startDate
			sprint.EndDate = endDate
			sprint.UpdatedAt = time.Now()
			m.DB.UpdateSprint(sprint)
			m.ActiveSprintUUID = sprint.UUID
			m.StatusMsg = fmt.Sprintf("Sprint '%s' updated successfully.", name)

			if recCount > 0 {
				m.DB.GenerateRecurringSprints(sprint, recCount)
				m.StatusMsg = fmt.Sprintf("Sprint updated and %d recurring sprints generated.", recCount)
			}
		}
	} else {
		newSprint := model.Sprint{
			UUID:          uuid.New().String(),
			WorkspaceUUID: wsUUID,
			Name:          name,
			StartDate:     startDate,
			EndDate:       endDate,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		m.DB.AddSprint(newSprint)
		m.ActiveSprintUUID = newSprint.UUID
		m.StatusMsg = fmt.Sprintf("Sprint '%s' created successfully.", name)

		if recCount > 0 {
			m.DB.GenerateRecurringSprints(newSprint, recCount)
			m.StatusMsg = fmt.Sprintf("Sprint created and %d recurring sprints generated.", recCount)
		}
	}

	m.refreshSprints()
}
