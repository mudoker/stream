package viewmodel

import (
	"strings"
	"time"

	"stream/internal/model"
	"stream/internal/viewmodel/timer"
)

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func CalculateTaskRestTime(t model.Task) time.Duration {
	if t.SchedulingType == model.Event {
		return 0
	}
	workDur := t.TimeWindow.End.Sub(t.TimeWindow.Start)
	if t.SchedulingType != model.Anchored {
		workDur = time.Duration(t.StoryPoints) * 45 * time.Minute
	}
	sessions := timer.PartitionTask(workDur)
	var rest time.Duration
	for _, s := range sessions {
		if s.Type == timer.BreakSession {
			rest += s.Duration
		}
	}
	return rest
}

func (m *Model) HasPriorityOverlapCollision(t model.Task) bool {
	if m.CurrentMode == ModeTaskMove || m.CurrentMode == ModeTaskDurationAdjust || strings.HasSuffix(t.UUID, "_moving") || strings.HasSuffix(t.UUID, "_adjusting") {
		return false
	}
	if t.SchedulingType != model.Anchored {
		return false
	}
	taskUUID := movingTaskBaseUUID(t.UUID)
	for _, t2 := range m.Tasks {
		if movingTaskBaseUUID(t2.UUID) == taskUUID || t2.SchedulingType != model.Anchored {
			continue
		}
		if !SameDay(t.TimeWindow.Start, t2.TimeWindow.Start) {
			continue
		}
		overlap := t.TimeWindow.Start.Before(t2.TimeWindow.End) && t2.TimeWindow.Start.Before(t.TimeWindow.End)
		if overlap {
			if t.Priority == model.P0 || t.Priority == model.P1 || t2.Priority == model.P0 || t2.Priority == model.P1 {
				return true
			}
		}
	}
	return false
}

func movingTaskBaseUUID(uuid string) string {
	uuid = strings.TrimSuffix(uuid, "_moving")
	uuid = strings.TrimSuffix(uuid, "_adjusting")
	return uuid
}
