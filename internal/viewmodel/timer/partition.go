package timer

import (
	"time"

	"stream/constant"
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

// PartitionTask segments remaining focus duration into optimal Focus and Break sessions
func PartitionTask(total time.Duration) []Session {
	var sessions []Session
	rem := total

	// Default fallback for invalid or zero duration
	if rem <= 0 {
		return []Session{
			{Type: FocusSession, Duration: constant.TimerDefaultFocusDuration},
			{Type: BreakSession, Duration: constant.TimerDefaultBreakDuration},
		}
	}

	// 1. Process focus time using 50-min and 25-min blocks
	for rem > 0 {
		if rem >= constant.TimerLongFocusDuration {
			sessions = append(sessions, Session{Type: FocusSession, Duration: constant.TimerLongFocusDuration})
			sessions = append(sessions, Session{Type: BreakSession, Duration: constant.TimerLongBreakDuration})
			rem -= constant.TimerLongFocusDuration
		} else if rem >= constant.TimerDefaultFocusDuration {
			sessions = append(sessions, Session{Type: FocusSession, Duration: constant.TimerDefaultFocusDuration})
			sessions = append(sessions, Session{Type: BreakSession, Duration: constant.TimerDefaultBreakDuration})
			rem -= constant.TimerDefaultFocusDuration
		} else {
			// 2. Handle trailing leftover focus time (less than 25 minutes)
			if len(sessions) > 0 {
				// Distribute leftover focus time to an existing Focus session,
				// ensuring we don't burst past our cap (50m).
				appended := false
				for i := len(sessions) - 1; i >= 0; i-- {
					if sessions[i].Type == FocusSession {
						if sessions[i].Duration == constant.TimerLongFocusDuration {
							continue
						}
						// If adding it keeps it under or equal to a 50m cap, add it
						if sessions[i].Duration+rem <= constant.TimerLongFocusDuration {
							sessions[i].Duration += rem
							appended = true
							break
						}
					}
				}
				// If it couldn't be cleanly attached to a flexible session, make it its own small block
				if !appended {
					sessions = append(sessions, Session{Type: FocusSession, Duration: rem})
				}
			} else {
				// If the total initial task time was less than 25 minutes to begin with
				sessions = append(sessions, Session{Type: FocusSession, Duration: rem})
			}
			rem = 0 // All remaining work time accounted for
		}
	}

	return sessions
}
