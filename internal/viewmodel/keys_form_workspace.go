package viewmodel

import (
	"fmt"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

func (m *Model) handleWorkspaceFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab", "down":
		m.WorkspaceForm.ActiveField = (m.WorkspaceForm.ActiveField + 1) % 4
		m.focusWorkspaceFormFields()
		return m, nil
	case "shift+tab", "up":
		m.WorkspaceForm.ActiveField = (m.WorkspaceForm.ActiveField - 1 + 4) % 4
		m.focusWorkspaceFormFields()
		return m, nil
	case "enter":
		m.submitWorkspaceForm()
		m.CurrentMode = ModeNormal
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		m.IsEditingWorkspace = false
		return m, nil
	}

	var cmd tea.Cmd
	switch m.WorkspaceForm.ActiveField {
	case 0:
		m.WorkspaceForm.NameInput, cmd = m.WorkspaceForm.NameInput.Update(msg)
	case 1:
		m.WorkspaceForm.IconInput, cmd = m.WorkspaceForm.IconInput.Update(msg)
	case 2:
		m.WorkspaceForm.BadgeInput, cmd = m.WorkspaceForm.BadgeInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusWorkspaceFormFields() {
	m.WorkspaceForm.NameInput.Blur()
	m.WorkspaceForm.IconInput.Blur()
	m.WorkspaceForm.BadgeInput.Blur()

	switch m.WorkspaceForm.ActiveField {
	case 0:
		m.WorkspaceForm.NameInput.Focus()
	case 1:
		m.WorkspaceForm.IconInput.Focus()
	case 2:
		m.WorkspaceForm.BadgeInput.Focus()
	}
}

func (m *Model) submitWorkspaceForm() {
	name := m.WorkspaceForm.NameInput.Value()
	icon := m.WorkspaceForm.IconInput.Value()
	badge := m.WorkspaceForm.BadgeInput.Value()

	if name == "" {
		m.StatusMsg = "Workspace name cannot be empty."
		return
	}
	if icon == "" {
		icon = "💼"
	}

	if m.IsEditingWorkspace {
		var ws model.Workspace
		found := false
		for _, w := range m.Workspaces {
			if w.UUID == m.ActiveWorkspaceUUID {
				ws = w
				found = true
				break
			}
		}
		if found {
			ws.Name = name
			ws.Icon = icon
			ws.Badge = badge
			m.DB.UpdateWorkspace(ws)
			m.StatusMsg = fmt.Sprintf("Workspace '%s' updated successfully.", name)
		}
		m.IsEditingWorkspace = false
	} else {
		newWS := model.Workspace{
			UUID:  uuid.New().String(),
			Name:  name,
			Icon:  icon,
			Badge: badge,
		}
		m.DB.AddWorkspace(newWS)
		m.StatusMsg = fmt.Sprintf("Workspace '%s' created successfully.", name)
	}

	m.refreshWorkspaces()
	m.refreshTasks()
	m.selectDefaultTaskForSelectedDay()
}
