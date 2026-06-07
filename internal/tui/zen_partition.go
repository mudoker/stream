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

// PartitionTask segments remaining focus duration into optimal Focus and Break sessions
func PartitionTask(total time.Duration) []Session {
	var sessions []Session
	rem := total

	// Default fallback for invalid or zero duration
	if rem <= 0 {
		return []Session{
			{Type: FocusSession, Duration: 25 * time.Minute},
			{Type: BreakSession, Duration: 5 * time.Minute},
		}
	}

	// 1. Exactly one 90-minute Focus session allowed maximum
	if rem >= 90*time.Minute {
		sessions = append(sessions, Session{Type: FocusSession, Duration: 90 * time.Minute})
		sessions = append(sessions, Session{Type: BreakSession, Duration: 20 * time.Minute})
		rem -= 90 * time.Minute // Subtracting ONLY work time
	}

	// 2. Process remaining focus time using 50-min and 25-min blocks
	for rem > 0 {
		if rem >= 50*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 50 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 10 * time.Minute})
			rem -= 50 * time.Minute
		} else if rem >= 25*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 25 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 5 * time.Minute})
			rem -= 25 * time.Minute
		} else {
			// 3. Handle trailing leftover focus time (less than 25 minutes)
			if len(sessions) > 0 {
				// Distribute leftover focus time to an existing Focus session,
				// ensuring we don't burst past our caps (90m, 50m).
				appended := false
				for i := len(sessions) - 1; i >= 0; i-- {
					if sessions[i].Type == FocusSession {
						if sessions[i].Duration == 90*time.Minute || sessions[i].Duration == 50*time.Minute {
							continue
						}
						// If adding it keeps it under or equal to a 50m cap, add it
						if sessions[i].Duration+rem <= 50*time.Minute {
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

	// Clean up trailing break check: If the last session is a break, 
	// it's usually preferred, but if you don't want a break at the absolute end of a task, 
	// you can optionally strip the final BreakSession here.

	return sessions
}