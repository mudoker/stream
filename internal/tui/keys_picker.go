package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleWorkspacePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.Workspaces) == 0 {
		m.CurrentMode = ModeNormal
		return m, nil
	}

	switch msg.String() {
	case "j", "down":
		m.WorkspacePickerIdx = (m.WorkspacePickerIdx + 1) % len(m.Workspaces)
		return m, nil

	case "k", "up":
		m.WorkspacePickerIdx = (m.WorkspacePickerIdx - 1 + len(m.Workspaces)) % len(m.Workspaces)
		return m, nil

	case "enter":
		idx := m.WorkspacePickerIdx
		if idx >= 0 && idx < len(m.Workspaces) {
			targetWS := m.Workspaces[idx]
			m.ActiveWorkspaceUUID = targetWS.UUID
			m.refreshTasks()
			m.selectDefaultTaskForSelectedDay()
			m.StatusMsg = fmt.Sprintf("Switched to workspace '%s'.", targetWS.Name)
		}
		m.CurrentMode = ModeNormal
		return m, nil

	case "esc", "q":
		m.CurrentMode = ModeNormal
		m.StatusMsg = "Workspace switch canceled."
		return m, nil
	}

	return m, nil
}
