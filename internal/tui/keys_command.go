package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type CommandEntry struct {
	Name string
	Desc string
}

func (m *Model) getCommandList() []CommandEntry {
	var list []CommandEntry
	list = append(list,
		CommandEntry{"q", "Quit Stream"},
		CommandEntry{"quit", "Quit Stream"},
		CommandEntry{"exit", "Quit Stream"},
		CommandEntry{"dashboard", "Switch to Dashboard view (1)"},
		CommandEntry{"month", "Switch to Month view (2)"},
		CommandEntry{"week", "Switch to Week view (3)"},
		CommandEntry{"day", "Switch to Day view (4)"},
		CommandEntry{"analytics", "Switch to Analytics view (5)"},
		CommandEntry{"profile", "Edit profile & security settings"},
		CommandEntry{"help", "Toggle command reference help overlay"},
		CommandEntry{"create <title>", "Create anchored task starting 9:00 AM"},
		CommandEntry{"todo <title>", "Create backlog floating task"},
		CommandEntry{"habit <title>", "Create daily repeatable habit"},
		CommandEntry{"ws-create", "Create workspace"},
		CommandEntry{"ws-edit", "Edit active workspace settings"},
		CommandEntry{"ws-delete", "Delete workspace"},
		CommandEntry{"ws-switch", "Open workspace selector picker"},
		CommandEntry{"review", "Trigger daily shutdown evaluation"},
		CommandEntry{"complete", "Mark selected task completed"},
		CommandEntry{"delete", "Delete selected task permanently"},
		CommandEntry{"sync", "Trigger calendar sync"},
		CommandEntry{"auth", "Start Google Calendar authorization"},
		CommandEntry{"stop", "Stop active focus timer session"},
	)

	for _, ws := range m.Workspaces {
		list = append(list, CommandEntry{
			Name: fmt.Sprintf("ws-switch %s", ws.Name),
			Desc: fmt.Sprintf("Switch directly to workspace '%s'", ws.Name),
		})
	}
	return list
}

func (m *Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up":
		m.CommandSelectedIndex--
		if m.CommandSelectedIndex < 0 {
			m.CommandSelectedIndex = len(m.getCommandList()) - 1
		}
		return m, nil
	case "down":
		m.CommandSelectedIndex = (m.CommandSelectedIndex + 1) % len(m.getCommandList())
		return m, nil
	case "enter":
		// Check if selected entry exists
		cmdList := m.getCommandList()
		if len(cmdList) > 0 && m.CommandSelectedIndex >= 0 && m.CommandSelectedIndex < len(cmdList) {
			val := cmdList[m.CommandSelectedIndex].Name
			m.CommandInput.SetValue(val)
		}
		m.CurrentMode = ModeNormal
		return m.runCommand(m.CommandInput.Value())
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.CommandInput, cmd = m.CommandInput.Update(msg)
	return m, cmd
}
