package viewmodel

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleSyncFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "shift+tab":
		m.SyncForm.ActiveField = (m.SyncForm.ActiveField - 1 + 3) % 3
		m.focusSyncFormFields()
		return m, nil
	case "down", "tab":
		m.SyncForm.ActiveField = (m.SyncForm.ActiveField + 1) % 3
		m.focusSyncFormFields()
		return m, nil
	case "left", "h":
		if m.SyncForm.ActiveField == 0 {
			m.SyncForm.ModeIdx = (m.SyncForm.ModeIdx - 1 + len(SyncModeOptions)) % len(SyncModeOptions)
		}
		return m, nil
	case "right", "l":
		if m.SyncForm.ActiveField == 0 {
			m.SyncForm.ModeIdx = (m.SyncForm.ModeIdx + 1) % len(SyncModeOptions)
		}
		return m, nil
	case "enter":
		if m.SyncForm.ActiveField == 2 {
			m.submitSyncForm()
			m.CurrentMode = ModeNormal
			return m, nil
		}
		m.SyncForm.ActiveField = (m.SyncForm.ActiveField + 1) % 3
		m.focusSyncFormFields()
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		m.StatusMsg = "Sync settings unchanged."
		return m, nil
	}

	var cmd tea.Cmd
	if m.SyncForm.ActiveField == 1 {
		m.SyncForm.IntervalInput, cmd = m.SyncForm.IntervalInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) focusSyncFormFields() {
	m.SyncForm.IntervalInput.Blur()
	if m.SyncForm.ActiveField == 1 {
		m.SyncForm.IntervalInput.Focus()
	}
}

func (m *Model) submitSyncForm() {
	intervalSecs := 5
	if val, err := strconv.Atoi(m.SyncForm.IntervalInput.Value()); err == nil && val > 0 {
		intervalSecs = val
	}

	settings := m.DB.GetUserSettings()
	settings.GCalSyncMode = m.SyncForm.ModeValue()
	settings.GCalSyncIntervalSeconds = intervalSecs
	settings.GCalSyncIntervalMinutes = 0

	if err := m.DB.UpdateUserSettings(settings); err != nil {
		m.StatusMsg = fmt.Sprintf("Error saving sync settings: %v", err)
		return
	}

	m.SyncForm.IntervalSecs = intervalSecs
	if m.Sync != nil {
		m.Sync.NotifySettingsChanged()
	}

	modeLabel := SyncModeOptions[m.SyncForm.ModeIdx]
	m.StatusMsg = fmt.Sprintf("GCal sync: %s, auto-sync every %d sec.", modeLabel, intervalSecs)
}
