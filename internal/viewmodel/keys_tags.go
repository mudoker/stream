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
				// Check for duplicates
				exists := false
				for _, t := range tags {
					if strings.EqualFold(t.Name, newName) {
						exists = true
						break
					}
				}
				if exists {
					m.StatusMsg = fmt.Sprintf("Tag '%s' already exists.", newName)
					return m, nil
				}
				tags = append(tags, model.TagInfo{Name: newName, Frequency: 1})
				m.DB.SaveTags(tags)
				m.StatusMsg = fmt.Sprintf("Tag '%s' created.", newName)
			} else { // EDIT
				// Check for duplicates (excluding current item)
				existsIdx := -1
				for i, t := range tags {
					if i != m.TagsCRUDSelectedIndex && strings.EqualFold(t.Name, newName) {
						existsIdx = i
						break
					}
				}
				if existsIdx != -1 {
					m.StatusMsg = fmt.Sprintf("Tag name '%s' already exists.", newName)
					return m, nil
				}
				oldName := tags[m.TagsCRUDSelectedIndex].Name
				tags[m.TagsCRUDSelectedIndex].Name = newName
				m.DB.SaveTags(tags)
				m.StatusMsg = fmt.Sprintf("Tag '%s' renamed to '%s'.", oldName, newName)
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
