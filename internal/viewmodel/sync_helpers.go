package viewmodel

import (
	"stream/internal/model"
)

// triggerGCalPush queues a one-way push for anchored calendar changes.
func (m *Model) triggerGCalPush(task model.Task) {
	// No-op: all sync is manual
}

// triggerGCalPushIfAnchored queues push when the task is or was anchored for GCal.
func (m *Model) triggerGCalPushIfAnchored(task model.Task) {
	// No-op: all sync is manual
}
