package viewmodel

import (
	"fmt"
	"sort"
	"strings"

	"stream/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleTagsCRUDKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tags := m.DB.GetTags()
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Frequency != tags[j].Frequency {
			return tags[i].Frequency > tags[j].Frequency
		}
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	if m.TagsCRUDState == "CREATE" || m.TagsCRUDState == "EDIT" {
		switch msg.String() {
		case "esc":
			m.TagsCRUDState = "BROWSE"
			m.TagsCRUDInput.Blur()
			m.StatusMsg = "Cancelled."
			return m, nil
		case "enter":
			newName := strings.TrimSpace(m.TagsCRUDInput.Value())
			if newName == "" {
				m.StatusMsg = "Tag name cannot be empty."
				return m, nil
			}

			if m.TagsCRUDState == "CREATE" {
				parts := strings.Split(newName, ",")
				addedAny := false
				var addedList []string
				for _, part := range parts {
					trimmedPart := strings.TrimSpace(part)
					if trimmedPart == "" {
						continue
					}
					exists := false
					for _, t := range tags {
						if strings.EqualFold(t.Name, trimmedPart) {
							exists = true
							break
						}
					}
					if !exists {
						tags = append(tags, model.TagInfo{Name: trimmedPart, Frequency: 1})
						addedList = append(addedList, trimmedPart)
						addedAny = true
					}
				}
				if addedAny {
					m.DB.SaveTags(tags)
					m.StatusMsg = fmt.Sprintf("Added tag(s): %s", strings.Join(addedList, ", "))
				} else {
					m.StatusMsg = "No new tags were added (already exist)."
				}
			} else { // EDIT
				oldTag := tags[m.TagsCRUDSelectedIndex]
				oldName := oldTag.Name
				// Remove old tag first from our working copy
				tags = append(tags[:m.TagsCRUDSelectedIndex], tags[m.TagsCRUDSelectedIndex+1:]...)

				parts := strings.Split(newName, ",")
				addedAny := false
				var addedList []string
				for _, part := range parts {
					trimmedPart := strings.TrimSpace(part)
					if trimmedPart == "" {
						continue
					}
					exists := false
					for _, t := range tags {
						if strings.EqualFold(t.Name, trimmedPart) {
							exists = true
							break
						}
					}
					if !exists {
						tags = append(tags, model.TagInfo{Name: trimmedPart, Frequency: oldTag.Frequency})
						addedList = append(addedList, trimmedPart)
						addedAny = true
					}
				}
				m.DB.SaveTags(tags)
				if addedAny {
					m.StatusMsg = fmt.Sprintf("Renamed '%s' to: %s", oldName, strings.Join(addedList, ", "))
				} else {
					m.StatusMsg = fmt.Sprintf("Deleted '%s' (renamed to existing tags).", oldName)
				}
			}
			m.TagsCRUDState = "BROWSE"
			m.TagsCRUDInput.Blur()
			return m, nil
		}

		var cmd tea.Cmd
		m.TagsCRUDInput, cmd = m.TagsCRUDInput.Update(msg)
		return m, cmd
	}

	// BROWSE state
	key := msg.String()
	switch key {
	case "up", "k":
		if len(tags) > 0 {
			m.TagsCRUDSelectedIndex--
			if m.TagsCRUDSelectedIndex < 0 {
				m.TagsCRUDSelectedIndex = len(tags) - 1
			}
		}
	case "down", "j":
		if len(tags) > 0 {
			m.TagsCRUDSelectedIndex = (m.TagsCRUDSelectedIndex + 1) % len(tags)
		}
	case "c", "C":
		m.TagsCRUDState = "CREATE"
		m.TagsCRUDInput.SetValue("")
		m.TagsCRUDInput.Focus()
		m.StatusMsg = "Create new tag: enter tag name."
	case "e", "E":
		if len(tags) > 0 && m.TagsCRUDSelectedIndex >= 0 && m.TagsCRUDSelectedIndex < len(tags) {
			m.TagsCRUDState = "EDIT"
			m.TagsCRUDInput.SetValue(tags[m.TagsCRUDSelectedIndex].Name)
			m.TagsCRUDInput.Focus()
			m.StatusMsg = "Edit tag name: enter new name."
		}
	case "d", "D", "backspace", "delete":
		if len(tags) > 0 && m.TagsCRUDSelectedIndex >= 0 && m.TagsCRUDSelectedIndex < len(tags) {
			deletedName := tags[m.TagsCRUDSelectedIndex].Name
			tags = append(tags[:m.TagsCRUDSelectedIndex], tags[m.TagsCRUDSelectedIndex+1:]...)
			m.DB.SaveTags(tags)
			m.StatusMsg = fmt.Sprintf("Tag '%s' deleted.", deletedName)
			if m.TagsCRUDSelectedIndex >= len(tags) {
				m.TagsCRUDSelectedIndex = len(tags) - 1
			}
			if m.TagsCRUDSelectedIndex < 0 {
				m.TagsCRUDSelectedIndex = 0
			}
		}
	case "+", "=":
		if len(tags) > 0 && m.TagsCRUDSelectedIndex >= 0 && m.TagsCRUDSelectedIndex < len(tags) {
			tags[m.TagsCRUDSelectedIndex].Frequency++
			m.DB.SaveTags(tags)
			m.StatusMsg = fmt.Sprintf("Frequency of '%s' increased to %d.", tags[m.TagsCRUDSelectedIndex].Name, tags[m.TagsCRUDSelectedIndex].Frequency)
		}
	case "-", "_":
		if len(tags) > 0 && m.TagsCRUDSelectedIndex >= 0 && m.TagsCRUDSelectedIndex < len(tags) {
			if tags[m.TagsCRUDSelectedIndex].Frequency > 0 {
				tags[m.TagsCRUDSelectedIndex].Frequency--
				m.DB.SaveTags(tags)
				m.StatusMsg = fmt.Sprintf("Frequency of '%s' decreased to %d.", tags[m.TagsCRUDSelectedIndex].Name, tags[m.TagsCRUDSelectedIndex].Frequency)
			}
		}
	case "esc":
		m.CurrentMode = ModeNormal
		m.StatusMsg = "Manage Tags closed."
	}

	return m, nil
}
