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
		CommandEntry{"sprint", "Switch to Sprint view (3)"},
		CommandEntry{"week", "Switch to Week view (4)"},
		CommandEntry{"day", "Switch to Day view (5)"},
		CommandEntry{"analytics", "Switch to Analytics view (6)"},
		CommandEntry{"sprint-create", "Create a new sprint"},
		CommandEntry{"sprint-edit", "Edit active sprint settings"},
		CommandEntry{"sprint-delete", "Delete active sprint"},
		CommandEntry{"sprint-generate", "Generate recurring sprints"},
		CommandEntry{"feature <title>", "Create new feature in active sprint"},
		CommandEntry{"defect <title>", "Create new defect in active sprint"},
		CommandEntry{"improvement <title>", "Create new improvement in active sprint"},
		CommandEntry{"task <title>", "Create new backlog task"},
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
		CommandEntry{"factory-reset", "Wipe all data and reset to factory state"},
		CommandEntry{"reset", "Wipe all data and reset to factory state"},
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
	val := strings.TrimSpace(m.CommandInput.Value())
	if strings.HasPrefix(val, ":") {
		val = strings.TrimSpace(strings.TrimPrefix(val, ":"))
	}
	val = strings.ToLower(val)
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
		if val == "" {
			return src
		}
		var exact []CommandEntry
		var prefixName []CommandEntry
		var containsName []CommandEntry
		var containsDesc []CommandEntry

		for _, c := range src {
			nameLower := strings.ToLower(c.Name)
			descLower := strings.ToLower(c.Desc)

			cleanName := nameLower
			if idx := strings.Index(cleanName, "<"); idx != -1 {
				cleanName = strings.TrimSpace(cleanName[:idx])
			}

			if nameLower == val || cleanName == val {
				exact = append(exact, c)
			} else if strings.HasPrefix(nameLower, val) || strings.HasPrefix(cleanName, val) {
				prefixName = append(prefixName, c)
			} else if strings.Contains(nameLower, val) {
				containsName = append(containsName, c)
			} else if strings.Contains(descLower, val) {
				containsDesc = append(containsDesc, c)
			}
		}

		var out []CommandEntry
		out = append(out, exact...)
		out = append(out, prefixName...)
		out = append(out, containsName...)
		out = append(out, containsDesc...)
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

	effectiveIdx := m.CommandSelectedIndex
	if effectiveIdx < 0 && strings.TrimSpace(m.CommandInput.Value()) != "" && len(cmdList) > 0 {
		effectiveIdx = 0
	}

	switch key {
	case "up":
		if len(cmdList) > 0 {
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = len(cmdList) - 1
			} else {
				m.CommandSelectedIndex--
				if m.CommandSelectedIndex < 0 {
					m.CommandSelectedIndex = len(cmdList) - 1
				}
			}
		}
		return m, nil
	case "down":
		if len(cmdList) > 0 {
			if m.CommandSelectedIndex < 0 {
				if effectiveIdx == 0 && len(cmdList) > 1 {
					m.CommandSelectedIndex = 1
				} else {
					m.CommandSelectedIndex = 0
				}
			} else {
				m.CommandSelectedIndex = (m.CommandSelectedIndex + 1) % len(cmdList)
			}
		}
		return m, nil
	case "tab":
		if len(cmdList) > 0 {
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = 0
			} else {
				m.CommandSelectedIndex = (m.CommandSelectedIndex + 1) % len(cmdList)
			}
			val := cmdList[m.CommandSelectedIndex].Name
			if idx := strings.Index(val, "<"); idx != -1 {
				val = strings.TrimRight(val[:idx], " ") + " "
			}
			m.CommandInput.SetValue(val)
			m.CommandInput.SetCursor(len(val))
		}
		return m, nil
	case "shift+tab":
		if len(cmdList) > 0 {
			if m.CommandSelectedIndex < 0 {
				m.CommandSelectedIndex = len(cmdList) - 1
			} else {
				m.CommandSelectedIndex--
				if m.CommandSelectedIndex < 0 {
					m.CommandSelectedIndex = len(cmdList) - 1
				}
			}
			val := cmdList[m.CommandSelectedIndex].Name
			if idx := strings.Index(val, "<"); idx != -1 {
				val = strings.TrimRight(val[:idx], " ") + " "
			}
			m.CommandInput.SetValue(val)
			m.CommandInput.SetCursor(len(val))
		}
		return m, nil
	case "enter":
		if strings.TrimSpace(m.CommandInput.Value()) == "" && m.CommandSelectedIndex < 0 {
			m.CurrentMode = ModeNormal
			return m, nil
		}

		if len(cmdList) > 0 && effectiveIdx >= 0 && effectiveIdx < len(cmdList) {
			matched := cmdList[effectiveIdx]
			val := matched.Name
			if idx := strings.Index(val, "<"); idx != -1 {
				prefix := strings.TrimRight(val[:idx], " ") + " "
				inputVal := strings.TrimSpace(m.CommandInput.Value())
				cmdBase := strings.TrimSpace(val[:idx])
				if inputVal == "" || strings.HasPrefix(cmdBase, inputVal) || inputVal == cmdBase {
					m.CommandInput.SetValue(prefix)
					m.CommandInput.SetCursor(len(prefix))
					m.CommandSelectedIndex = -1
					return m, nil
				}
				val = prefix
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
