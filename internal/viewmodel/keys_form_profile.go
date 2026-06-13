package viewmodel

import (
	"fmt"
	"strconv"
	"strings"

	"stream/internal/db"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleProfileFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "shift+tab":
		m.ProfileForm.ActiveField = (m.ProfileForm.ActiveField - 1 + 4) % 4
		m.focusProfileFormFields()
		return m, nil
	case "down", "tab":
		m.ProfileForm.ActiveField = (m.ProfileForm.ActiveField + 1) % 4
		m.focusProfileFormFields()
		return m, nil
	case "enter":
		m.submitProfileForm()
		m.CurrentMode = ModeNormal
		return m, nil
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	switch m.ProfileForm.ActiveField {
	case 0:
		m.ProfileForm.UsernameInput, cmd = m.ProfileForm.UsernameInput.Update(msg)
	case 1:
		m.ProfileForm.PasswordInput, cmd = m.ProfileForm.PasswordInput.Update(msg)
	case 2:
		m.ProfileForm.LockTimeoutInput, cmd = m.ProfileForm.LockTimeoutInput.Update(msg)
	}

	return m, cmd
}

func (m *Model) focusProfileFormFields() {
	m.ProfileForm.UsernameInput.Blur()
	m.ProfileForm.PasswordInput.Blur()
	m.ProfileForm.LockTimeoutInput.Blur()

	switch m.ProfileForm.ActiveField {
	case 0:
		m.ProfileForm.UsernameInput.Focus()
	case 1:
		m.ProfileForm.PasswordInput.Focus()
	case 2:
		m.ProfileForm.LockTimeoutInput.Focus()
	}
}

func (m *Model) submitProfileForm() {
	user := strings.TrimSpace(m.ProfileForm.UsernameInput.Value())
	if user == "" {
		m.StatusMsg = "Username cannot be empty."
		return
	}

	passVal := m.ProfileForm.PasswordInput.Value()
	timeoutStr := m.ProfileForm.LockTimeoutInput.Value()

	timeoutVal := 5
	if val, err := strconv.Atoi(timeoutStr); err == nil && val > 0 {
		timeoutVal = val
	}

	settings := m.DB.GetUserSettings()
	settings.Username = user
	settings.LockTimeoutMinutes = timeoutVal

	if passVal != "" {
		if strings.EqualFold(passVal, "none") {
			settings.PasswordHash = ""
			m.IsLocked = false
			m.StatusMsg = "Display username and settings updated. Password lock disabled."
		} else {
			settings.PasswordHash = db.HashPassword(passVal)
			m.StatusMsg = "Display username, settings, and password updated successfully."
		}
	} else {
		m.StatusMsg = "Display username and lock settings updated successfully."
	}

	if err := m.DB.UpdateUserSettings(settings); err != nil {
		m.StatusMsg = fmt.Sprintf("Error saving settings: %v", err)
	} else {
		m.SessionTimeRemainingSeconds = timeoutVal * 60
	}

	m.ProfileForm.Username = user
	m.ProfileForm.LockTimeoutMins = timeoutVal
	m.ProfileForm.PasswordInput.SetValue("")
}
