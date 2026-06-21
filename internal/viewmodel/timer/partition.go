package timer

import (
	"time"

	"stream/internal/viewmodel/common/constants"
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
			{Type: FocusSession, Duration: constants.TimerDefaultFocusDuration},
		}
	}

	// 1. Process focus time using 50-min and 25-min blocks
	for rem > 0 {
		if rem >= constants.TimerLongFocusDuration {
			sessions = append(sessions, Session{Type: FocusSession, Duration: constants.TimerLongFocusDuration})
			sessions = append(sessions, Session{Type: BreakSession, Duration: constants.TimerLongBreakDuration})
			rem -= constants.TimerLongFocusDuration
		} else if rem >= constants.TimerDefaultFocusDuration {
			sessions = append(sessions, Session{Type: FocusSession, Duration: constants.TimerDefaultFocusDuration})
			sessions = append(sessions, Session{Type: BreakSession, Duration: constants.TimerDefaultBreakDuration})
			rem -= constants.TimerDefaultFocusDuration
		} else {
			// 2. Handle trailing leftover focus time (less than 25 minutes)
			if len(sessions) > 0 {
				// Distribute leftover focus time to an existing Focus session,
				// ensuring we don't burst past our cap (50m).
				appended := false
				for i := len(sessions) - 1; i >= 0; i-- {
					if sessions[i].Type == FocusSession {
						if sessions[i].Duration == constants.TimerLongFocusDuration {
							continue
						}
						// If adding it keeps it under or equal to a 50m cap, add it
						if sessions[i].Duration+rem <= constants.TimerLongFocusDuration {
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

	// Remove trailing break session if it exists
	if len(sessions) > 0 && sessions[len(sessions)-1].Type == BreakSession {
		sessions = sessions[:len(sessions)-1]
	}

	return sessions
}
