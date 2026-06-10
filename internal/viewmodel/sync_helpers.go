package viewmodel

import (
	"stream/internal/model"
)

// triggerGCalPush queues a one-way push for anchored calendar changes.
func (m *Model) triggerGCalPush(task model.Task) {
	if m.Sync == nil || !model.IsGCalSyncable(task) {
		return
	}
	m.Sync.TriggerPushSync()
}

// triggerGCalPushIfAnchored queues push when the task is or was anchored for GCal.
func (m *Model) triggerGCalPushIfAnchored(task model.Task) {
	if m.Sync == nil {
		return
	}
	if model.IsGCalSyncable(task) {
		m.Sync.TriggerPushSync()
	}
}
