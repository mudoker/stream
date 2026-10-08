package viewmodel

import (
	"fmt"

	"stream/internal/model"
	"stream/internal/viewmodel/tasks"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) EnterSprintTaskMoveMode() {
	task, exists := m.GetActiveTask()
	if !exists {
		m.StatusMsg = "No feature or task selected to move."
		return
	}
	activeSprint, ok := m.GetActiveSprint()
	if !ok {
		m.StatusMsg = "No active sprint."
		return
	}

	// Backup all tasks
	m.SprintMoveOriginalTasks = make([]model.Task, len(m.Tasks))
	copy(m.SprintMoveOriginalTasks, m.Tasks)
	m.SprintMoveTaskUUID = task.UUID
	m.CurrentMode = ModeSprintTaskMove

	// Ensure all tasks in active sprint lanes have deterministic sprint orders assigned
	m.normalizeSprintOrders(activeSprint.UUID)

	m.StatusMsg = fmt.Sprintf("Moving '%s'. Use j/k to reorder, h/l to move swimlane. Enter to confirm, Esc to cancel.", task.Title)
}

func (m *Model) normalizeSprintOrders(sprintUUID string) {
	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, sprintUUID)
	lanes := [][]model.Task{defined, inProgress, review, testing, completed}
	for _, lane := range lanes {
		for idx, t := range lane {
			t.SprintOrder = (idx + 1) * 10
			m.updateTaskInMemory(t)
		}
	}
}

func (m *Model) HandleSprintTaskMoveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "j", "down":
		m.moveSprintTaskVertical(1)
	case "k", "up":
		m.moveSprintTaskVertical(-1)
	case "h", "left":
		m.moveSprintTaskHorizontal(-1)
	case "l", "right":
		m.moveSprintTaskHorizontal(1)
	case "enter":
		m.confirmSprintTaskMove()
	case "esc", "q":
		m.cancelSprintTaskMove()
	}
	return m, nil
}

func (m *Model) moveSprintTaskVertical(dir int) {
	activeSprint, ok := m.GetActiveSprint()
	if !ok {
		return
	}
	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	lanes := [][]model.Task{defined, inProgress, review, testing, completed}
	curLaneIdx := m.SprintSwimlaneIdx
	if curLaneIdx < 0 || curLaneIdx >= len(lanes) {
		return
	}
	curLane := lanes[curLaneIdx]
	curPos := -1
	for i, t := range curLane {
		if t.UUID == m.SprintMoveTaskUUID {
			curPos = i
			break
		}
	}
	if curPos == -1 {
		return
	}
	targetPos := curPos + dir
	if targetPos < 0 || targetPos >= len(curLane) {
		return
	}

	// Swap SprintOrder
	taskA := curLane[curPos]
	taskB := curLane[targetPos]
	orderA := taskA.SprintOrder
	orderB := taskB.SprintOrder
	if orderA == orderB {
		orderA = (curPos + 1) * 10
		orderB = (targetPos + 1) * 10
	}
	taskA.SprintOrder = orderB
	taskB.SprintOrder = orderA
	m.updateTaskInMemory(taskA)
	m.updateTaskInMemory(taskB)
	m.SelectedTaskUUID = taskA.UUID
}

func (m *Model) moveSprintTaskHorizontal(dir int) {
	activeSprint, ok := m.GetActiveSprint()
	if !ok {
		return
	}
	curLaneIdx := m.SprintSwimlaneIdx
	targetLaneIdx := curLaneIdx + dir
	if targetLaneIdx < 0 || targetLaneIdx > 4 {
		return
	}

	task, exists := m.GetActiveTask()
	if !exists {
		return
	}

	var newState model.LifecycleState
	switch targetLaneIdx {
	case 0:
		newState = model.StateReady
	case 1:
		newState = model.StateActive
	case 2:
		newState = model.StateReview
	case 3:
		newState = model.StateTesting
	case 4:
		newState = model.StateCompleted
	}

	task.LifecycleState = newState

	// Assign order in the target lane (place at the end)
	defined, inProgress, review, testing, completed := tasks.GetSprintSwimlaneTasks(m.Tasks, activeSprint.UUID)
	lanes := [][]model.Task{defined, inProgress, review, testing, completed}
	targetLane := lanes[targetLaneIdx]
	maxOrder := 0
	for _, t := range targetLane {
		if t.SprintOrder > maxOrder {
			maxOrder = t.SprintOrder
		}
	}
	task.SprintOrder = maxOrder + 10

	m.updateTaskInMemory(task)
	m.SprintSwimlaneIdx = targetLaneIdx
	m.SelectedTaskUUID = task.UUID
	m.AutoScrollSprintLane()
	m.StatusMsg = fmt.Sprintf("Moved '%s' to %s swimlane. Enter to confirm, Esc to cancel.", task.Title, sprintLaneName(targetLaneIdx))
}

func (m *Model) confirmSprintTaskMove() {
	task, exists := m.GetActiveTask()
	if exists {
		_ = m.DB.UpdateTask(task)
	}
	// Persist all sprint tasks that might have had sprint order changes
	if activeSprint, ok := m.GetActiveSprint(); ok {
		for _, t := range m.Tasks {
			if t.SprintUUID == activeSprint.UUID {
				_ = m.DB.UpdateTask(t)
			}
		}
	}
	m.CurrentMode = ModeNormal
	m.SprintMoveOriginalTasks = nil
	m.SprintMoveTaskUUID = ""
	if exists {
		m.StatusMsg = fmt.Sprintf("Confirmed position for '%s'.", task.Title)
	} else {
		m.StatusMsg = "Move confirmed."
	}
}

func (m *Model) cancelSprintTaskMove() {
	if len(m.SprintMoveOriginalTasks) > 0 {
		m.Tasks = m.SprintMoveOriginalTasks
		m.SprintMoveOriginalTasks = nil
	}
	m.CurrentMode = ModeNormal
	m.SprintMoveTaskUUID = ""
	m.StatusMsg = "Move cancelled."
}
