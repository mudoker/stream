package modals

import (
	"fmt"
	"sort"
	"strings"

	"stream/internal/view/components"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	"github.com/charmbracelet/lipgloss"
)

func RenderTagsCRUDModal(m *viewmodel.Model, t theme.Theme) string {
	const innerW = 52
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	sb.WriteString(titleStyle.Render("🏷️  MANAGE SYSTEM TAGS") + "\n")
	sb.WriteString(components.ModalSep(innerW) + "\n\n")

	tags := m.DB.GetTags()
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Frequency != tags[j].Frequency {
			return tags[i].Frequency > tags[j].Frequency
		}
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	if m.TagsCRUDState == "CREATE" || m.TagsCRUDState == "EDIT" {
		promptText := "Create New Tag:"
		if m.TagsCRUDState == "EDIT" {
			promptText = "Rename Selected Tag:"
		}
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(t.Fg).Bold(true).Render(promptText) + "\n\n")
		inputView := m.TagsCRUDInput.View()
		sb.WriteString("  " + inputView + "\n\n")
		sb.WriteString(components.ModalSep(innerW) + "\n")
		hint := lipgloss.NewStyle().Foreground(t.Muted).Render("  enter save  •  esc cancel")
		sb.WriteString(hint)
	} else {
		if len(tags) == 0 {
			sb.WriteString("  " + lipgloss.NewStyle().Foreground(t.Muted).Render("(no tags found)") + "\n\n")
		} else {
			maxVisible := 8
			startIndex := 0
			if len(tags) > maxVisible {
				startIndex = m.TagsCRUDSelectedIndex - (maxVisible / 2)
				if startIndex < 0 {
					startIndex = 0
				}
				if startIndex+maxVisible > len(tags) {
					startIndex = len(tags) - maxVisible
				}
			}
			endIndex := startIndex + maxVisible
			if endIndex > len(tags) {
				endIndex = len(tags)
			}

			if startIndex > 0 {
				sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  ▲  (more tags above)") + "\n")
			} else {
				sb.WriteString("\n")
			}

			for i := startIndex; i < endIndex; i++ {
				tag := tags[i]
				if i == m.TagsCRUDSelectedIndex {
					tagStr := fmt.Sprintf(" ▶ %-20s (freq: %d)", tag.Name, tag.Frequency)
					sb.WriteString(lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(tagStr) + "\n")
				} else {
					tagStr := fmt.Sprintf("   %-20s (freq: %d)", tag.Name, tag.Frequency)
					sb.WriteString(lipgloss.NewStyle().Foreground(t.Fg).Render(tagStr) + "\n")
				}
			}

			if endIndex < len(tags) {
				sb.WriteString(lipgloss.NewStyle().Foreground(t.Muted).Render("  ▼  (more tags below)") + "\n")
			} else {
				sb.WriteString("\n")
			}
		}

		sb.WriteString("\n" + components.ModalSep(innerW) + "\n")
		hint := lipgloss.NewStyle().Foreground(t.Muted).Render("c create • e edit • d delete • +/- freq • esc close")
		sb.WriteString("  " + hint)
	}

	return t.ModalStyle.Render(components.PrepareModalContent(sb.String(), innerW))
}
