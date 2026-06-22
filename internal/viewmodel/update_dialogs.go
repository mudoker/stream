package viewmodel

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.UpdatePromptOpen {
		if handled, cmd := m.HandleUpdatePromptKeys(msg); handled {
			return m, cmd
		}
	}

	if msg.String() == "esc" {
		if m.ConfirmOpen {
			if handled, cmd := m.handleConfirmDialogKeys(msg); handled {
				return m, cmd
			}
		}

		m.WarningOpen = false
		m.WarningMsg = ""

		m.AuthNoticeOpen = false
		m.AuthNoticeMsg = ""

		m.SessionExpiryPromptOpen = false

		m.ConfirmOpen = false

		m.HelpOpen = false
		m.HelpScrollOffset = 0

		m.AnchorPromptOpen = false
		m.LogSessionPromptOpen = false
		m.UpdatePromptOpen = false

		if m.PromptOpen {
			m.cancelPromptTask()
		}

		m.ReviewOpen = false
		m.DetailOpen = false
		m.JazzLoungeOpen = false

		m.StatusMsg = ""

		if m.CurrentMode == ModeTaskMove {
			m.cancelTaskMove()
			return m, nil
		}
		if m.CurrentMode == ModeTaskDurationAdjust {
			m.cancelTaskDurationAdjust()
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

	// Route based on open modal/state
	if m.JazzLoungeOpen {
		if handled, cmd := m.handleJazzLoungeKeys(msg); handled {
			return m, cmd
		}
	}

	if m.WarningOpen || m.AuthNoticeOpen || m.SessionExpiryPromptOpen || m.ConfirmOpen {
		if handled, cmd := m.handleConfirmDialogKeys(msg); handled {
			return m, cmd
		}
	}

	if m.HelpOpen || m.AnchorPromptOpen || m.LogSessionPromptOpen || m.PromptOpen || m.ReviewOpen || m.DetailOpen {
		if handled, cmd := m.handleConfirmDialogKeys(msg); handled {
			return m, cmd
		}
		if handled, cmd := m.handleHelpAndDetailKeys(msg); handled {
			return m, cmd
		}
		if handled, cmd := m.handlePromptDialogKeys(msg); handled {
			return m, cmd
		}
	}

	switch m.CurrentMode {
	case ModeZen:
		return m.HandleZenKeys(msg)
	case ModeCommand:
		return m.handleCommandKeys(msg)
	case ModeForm:
		return m.handleFormKeys(msg)
	case ModeTaskMove:
		return m.HandleTaskMoveKeys(msg)
	case ModeTaskDurationAdjust:
		return m.HandleTaskDurationAdjustKeys(msg)
	case ModeWorkspaceForm:
		return m.handleWorkspaceFormKeys(msg)
	case ModeProfileForm:
		return m.handleProfileFormKeys(msg)
	case ModeSyncForm:
		return m.handleSyncFormKeys(msg)
	case ModeWorkspacePicker:
		return m.handleWorkspacePickerKeys(msg)
	case ModeTagsCRUD:
		return m.handleTagsCRUDKeys(msg)
	case ModeNormal:
		return m.HandleNormalKeys(msg)
	}

	return m, nil
}
