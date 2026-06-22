package viewmodel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type CommandEntry struct {
	Name string
	Desc string
}

func (m *Model) GetCommandList() []CommandEntry {
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
		CommandEntry{"pull", "Pull anchored tasks from Google Calendar"},
		CommandEntry{"push", "Push local anchored tasks to Google Calendar"},
		CommandEntry{"sync-settings", "Configure GCal sync mode and interval (seconds)"},
		CommandEntry{"auth", "Start Google Calendar authorization"},
		CommandEntry{"stop", "Stop active focus timer session"},
		CommandEntry{"music", "Open jazz lounge music player modal"},
		CommandEntry{"tags", "Manage system tags (CRUD)"},
	)

	for _, ws := range m.Workspaces {
		list = append(list, CommandEntry{
			Name: fmt.Sprintf("ws-switch %s", ws.Name),
			Desc: fmt.Sprintf("Switch directly to workspace '%s'", ws.Name),
		})
	}
	return list
}

func (m *Model) GetFilteredCommandList() []CommandEntry {
	val := strings.ToLower(m.CommandInput.Value())
	allCommands := m.GetCommandList()

	var genericEntries []CommandEntry
	var wsEntries []CommandEntry
	for _, c := range allCommands {
		if strings.HasPrefix(c.Name, "ws-switch ") {
			wsEntries = append(wsEntries, c)
		} else {
			genericEntries = append(genericEntries, c)
		}
	}

	filterGroup := func(src []CommandEntry) []CommandEntry {
		var out []CommandEntry
		for _, c := range src {
			if strings.Contains(strings.ToLower(c.Name), val) ||
				strings.Contains(strings.ToLower(c.Desc), val) {
				out = append(out, c)
			}
		}
		return out
	}
	filteredGeneric := filterGroup(genericEntries)
	filteredWS := filterGroup(wsEntries)

	var res []CommandEntry
	res = append(res, filteredGeneric...)
	res = append(res, filteredWS...)
	return res
}

func (m *Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	cmdList := m.GetFilteredCommandList()

	switch key {
	case "up":
		if len(cmdList) > 0 {
			m.CommandSelectedIndex--
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = len(cmdList) - 1
			}
		}
		return m, nil
	case "down":
		if len(cmdList) > 0 {
			m.CommandSelectedIndex = (m.CommandSelectedIndex + 1) % len(cmdList)
		}
		return m, nil
	case "tab":
		if len(cmdList) > 0 {
			m.CommandSelectedIndex = (m.CommandSelectedIndex + 1) % len(cmdList)
			val := cmdList[m.CommandSelectedIndex].Name
			if idx := strings.Index(val, "<"); idx != -1 {
				val = strings.TrimRight(val[:idx], " ") + " "
			}
			m.CommandInput.SetValue(val)
		}
		return m, nil
	case "shift+tab":
		if len(cmdList) > 0 {
			m.CommandSelectedIndex--
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = len(cmdList) - 1
			}
			val := cmdList[m.CommandSelectedIndex].Name
			if idx := strings.Index(val, "<"); idx != -1 {
				val = strings.TrimRight(val[:idx], " ") + " "
			}
			m.CommandInput.SetValue(val)
		}
		return m, nil
	case "enter":
		if len(cmdList) > 0 && m.CommandSelectedIndex >= 0 && m.CommandSelectedIndex < len(cmdList) {
			val := cmdList[m.CommandSelectedIndex].Name
			if idx := strings.Index(val, "<"); idx != -1 {
				val = strings.TrimRight(val[:idx], " ") + " "
			}
			m.CommandInput.SetValue(val)
		}
		m.CurrentMode = ModeNormal
		return m.RunCommand(m.CommandInput.Value())
	case "esc":
		m.CurrentMode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.CommandInput, cmd = m.CommandInput.Update(msg)
	// Reset selected index if input changed so we don't accidentally preserve old selection on new list
	m.CommandSelectedIndex = -1
	return m, cmd
}
