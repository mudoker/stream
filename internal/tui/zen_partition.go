package tui

import (
	"time"
)

type SessionType string

const (
	FocusSession SessionType = "Focus"
	BreakSession SessionType = "Break"
)

type Session struct {
	Type     SessionType
	Duration time.Duration
}

// PartitionTask segments remaining duration into optimal Focus and Break sessions
func PartitionTask(total time.Duration) []Session {
	var sessions []Session
	rem := total

	if rem <= 0 {
		return []Session{
			{Type: FocusSession, Duration: 25 * time.Minute},
			{Type: BreakSession, Duration: 5 * time.Minute},
		}
	}

	for rem > 0 {
		if rem >= 110*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 90 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 20 * time.Minute})
			rem -= 110 * time.Minute
		} else if rem >= 60*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 50 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 10 * time.Minute})
			rem -= 60 * time.Minute
		} else {
			if len(sessions) > 0 {
				if rem < 30*time.Minute {
					found := false
					for i := len(sessions) - 1; i >= 0; i-- {
						if sessions[i].Type == FocusSession {
							sessions[i].Duration += rem
							found = true
							break
						}
					}
					if !found {
						sessions = append(sessions, Session{Type: FocusSession, Duration: rem})
					}
					rem = 0
				} else {
					sessions = append(sessions, Session{Type: FocusSession, Duration: 25 * time.Minute})
					sessions = append(sessions, Session{Type: BreakSession, Duration: 5 * time.Minute})
					rem -= 30 * time.Minute
				}
			} else {
				if rem < 30*time.Minute {
					sessions = append(sessions, Session{Type: FocusSession, Duration: rem})
				} else {
					focusDur := rem - 5*time.Minute
					sessions = append(sessions, Session{Type: FocusSession, Duration: focusDur})
					sessions = append(sessions, Session{Type: BreakSession, Duration: 5 * time.Minute})
				}
				rem = 0
			}
		}
	}
	return sessions
}
