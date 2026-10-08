package viewmodel

import (
	"fmt"

	"stream/internal/model"
)

type ShelfSectionType string

const (
	SectionToday     ShelfSectionType = "TODAY"
	SectionReminders ShelfSectionType = "REMINDERS"
	SectionHabits    ShelfSectionType = "HABITS"
	SectionBacklog   ShelfSectionType = "BACKLOG"
	SectionCompleted ShelfSectionType = "COMPLETED"
)

type ShelfSection struct {
	Type  ShelfSectionType
	Title string
	Icon  string
	Tasks []model.Task
}

type ShelfData struct {
	Title           string
	IsGlobalBacklog bool
	Tasks           []model.Task
	Sections        []ShelfSection
}

// GetCurrentShelfTasks returns the tasks for the currently active right shelf
// (Global Backlog for SprintView, Todo Shelf for DayView).
func (m *Model) GetCurrentShelfTasks() []model.Task {
	if m.CurrentView == SprintView {
		return m.GetGlobalBacklogTasks()
	}
	return m.GetTodoShelfTasks()
}

// GetShelfData constructs the headless UI state for the right shelf,
// organizing tasks into sections with headers, counts, and flat task sequences.
func (m *Model) GetShelfData() ShelfData {
	isGlobal := m.CurrentView == SprintView
	shelfTitle := "TODO SHELF"
	if isGlobal {
		shelfTitle = "GLOBAL BACKLOG"
	}

	shelfTasks := m.GetCurrentShelfTasks()

	var todayTasks []model.Task
	todayUUIDs := make(map[string]bool)

	if isGlobal {
		activeSprint, hasSprint := m.GetActiveSprint()
		if hasSprint {
			activeFeatIDs := make(map[string]bool)
			for _, t := range m.Tasks {
				if t.SprintUUID == activeSprint.UUID && t.ID != "" {
					activeFeatIDs[t.ID] = true
				}
			}
			for _, t := range m.Tasks {
				if m.ActiveWorkspaceUUID != "ALL_WORKSPACES" && t.WorkspaceUUID != m.ActiveWorkspaceUUID {
					continue
				}
				if t.AddedToToday && t.LifecycleState != model.StateCompleted {
					if (t.LinkedFeatureID != "" && activeFeatIDs[t.LinkedFeatureID]) || t.SprintUUID == activeSprint.UUID {
						todayTasks = append(todayTasks, t)
						todayUUIDs[t.UUID] = true
					}
				}
			}
		}
	}

	var reminders []model.Task
	var habits []model.Task
	var backlog []model.Task
	var completed []model.Task

	for _, task := range shelfTasks {
		if todayUUIDs[task.UUID] {
			continue
		}

		isDone := false
		if task.SchedulingType == model.Habit {
			dateStr := m.SelectedDay.Format("2006-01-02")
			for _, d := range task.CompletedDates {
				if d == dateStr {
					isDone = true
					break
				}
			}
		} else {
			isDone = task.LifecycleState == model.StateCompleted
		}

		if isDone {
			completed = append(completed, task)
		} else if task.SchedulingType == model.Reminder {
			reminders = append(reminders, task)
		} else if task.SchedulingType == model.Habit {
			habits = append(habits, task)
		} else {
			backlog = append(backlog, task)
		}
	}

	var sections []ShelfSection
	if isGlobal {
		sections = append(sections, ShelfSection{
			Type:  SectionToday,
			Title: fmt.Sprintf("⚡ TODAY SHELF (%d)", len(todayTasks)),
			Icon:  "⚡",
			Tasks: todayTasks,
		})
	} else {
		sections = append(sections, ShelfSection{
			Type:  SectionReminders,
			Title: fmt.Sprintf("⏰ REMINDERS (%d)", len(reminders)),
			Icon:  "⏰",
			Tasks: reminders,
		})
		sections = append(sections, ShelfSection{
			Type:  SectionHabits,
			Title: fmt.Sprintf("🔁 HABITS (%d)", len(habits)),
			Icon:  "🔁",
			Tasks: habits,
		})
	}

	sections = append(sections, ShelfSection{
		Type:  SectionBacklog,
		Title: fmt.Sprintf("☱ BACKLOG (%d)", len(backlog)),
		Icon:  "☱",
		Tasks: backlog,
	})

	sections = append(sections, ShelfSection{
		Type:  SectionCompleted,
		Title: fmt.Sprintf("✓ COMPLETED (%d)", len(completed)),
		Icon:  "✓",
		Tasks: completed,
	})

	var flatTasks []model.Task
	for _, sec := range sections {
		flatTasks = append(flatTasks, sec.Tasks...)
	}

	return ShelfData{
		Title:           shelfTitle,
		IsGlobalBacklog: isGlobal,
		Tasks:           flatTasks,
		Sections:        sections,
	}
}

// MoveShelfSection switches the selection to the first task of the next (dir > 0)
// or previous (dir < 0) non-empty section on the shelf.
func (m *Model) MoveShelfSection(dir int) {
	shelfData := m.GetShelfData()
	if len(shelfData.Tasks) == 0 {
		return
	}

	// 1. Identify which section currently holds the selected task
	currentSecIdx := -1
	for secIdx, sec := range shelfData.Sections {
		for _, task := range sec.Tasks {
			if task.UUID == m.SelectedTaskUUID {
				currentSecIdx = secIdx
				break
			}
		}
		if currentSecIdx != -1 {
			break
		}
	}

	numSecs := len(shelfData.Sections)
	if currentSecIdx == -1 {
		// Select the first non-empty section's first task
		for _, sec := range shelfData.Sections {
			if len(sec.Tasks) > 0 {
				m.SelectedTaskUUID = sec.Tasks[0].UUID
				m.StatusMsg = fmt.Sprintf("Switched to %s.", sec.Type)
				return
			}
		}
		return
	}

	// 2. Scan in the specified direction for the next non-empty section
	for step := 1; step <= numSecs; step++ {
		var targetIdx int
		if dir >= 0 {
			targetIdx = (currentSecIdx + step) % numSecs
		} else {
			targetIdx = (currentSecIdx - step + numSecs*100) % numSecs
		}

		targetSec := shelfData.Sections[targetIdx]
		if len(targetSec.Tasks) > 0 {
			m.SelectedTaskUUID = targetSec.Tasks[0].UUID
			m.StatusMsg = fmt.Sprintf("Switched to %s.", targetSec.Type)
			return
		}
	}
}

// GetActiveShelfSection returns the currently focused shelf section based on SelectedTaskUUID.
func (m *Model) GetActiveShelfSection() (ShelfSection, bool) {
	shelfData := m.GetShelfData()
	if len(shelfData.Sections) == 0 {
		return ShelfSection{}, false
	}

	// 1. If SelectedTaskUUID belongs to a section, return that section
	if m.SelectedTaskUUID != "" {
		for _, sec := range shelfData.Sections {
			for _, task := range sec.Tasks {
				if task.UUID == m.SelectedTaskUUID {
					return sec, true
				}
			}
		}
	}

	// 2. Otherwise return the first section that has tasks
	for _, sec := range shelfData.Sections {
		if len(sec.Tasks) > 0 {
			return sec, true
		}
	}

	// 3. Fallback to the first section
	return shelfData.Sections[0], true
}

func (m *Model) InitiateClearShelfSection(sec ShelfSection) {
	m.ConfirmShelfSection = sec
	m.ConfirmOpen = true
	m.ConfirmSelectedIndex = 0
	m.ConfirmFocusArea = 0
	m.ConfirmActionType = "clear_shelf_section"
}

func (m *Model) ConfirmClearShelfSection() {
	sec := m.ConfirmShelfSection
	if m.DB != nil {
		for _, task := range sec.Tasks {
			m.DB.DeleteTask(task.UUID)
		}
	}
	m.refreshTasks()

	// Re-adjust selection to the first available task on the shelf
	shelf := m.GetCurrentShelfTasks()
	if len(shelf) > 0 {
		m.SelectedTaskUUID = shelf[0].UUID
	} else {
		m.SelectedTaskUUID = ""
	}

	m.ConfirmOpen = false
	m.ConfirmActionType = ""
	m.ConfirmShelfSection = ShelfSection{}
	m.StatusMsg = fmt.Sprintf("Cleared %d tasks from %s.", len(sec.Tasks), sec.Type)
}

