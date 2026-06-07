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

	// 1. Exactly one 90-minute block allowance, but ONLY if we don't leave 
	// an awkwardly small remainder (like 10 mins) that distorts the session.
	// We only take the 90-min block if remaining time is exactly 110m, or >= 140m (110 + 30).
	if rem == 110*time.Minute || rem >= 140*time.Minute {
		sessions = append(sessions, Session{Type: FocusSession, Duration: 90 * time.Minute})
		sessions = append(sessions, Session{Type: BreakSession, Duration: 20 * time.Minute})
		rem -= 110 * time.Minute
	}

	// 2. Loop for standard 50-min and 25-min intervals
	for rem > 0 {
		if rem >= 60*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 50 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 10 * time.Minute})
			rem -= 60 * time.Minute
		} else if rem >= 30*time.Minute {
			sessions = append(sessions, Session{Type: FocusSession, Duration: 25 * time.Minute})
			sessions = append(sessions, Session{Type: BreakSession, Duration: 5 * time.Minute})
			rem -= 30 * time.Minute
		} else {
			// 3. Handle specific cleanups for tail end durations
			if rem >= 25*time.Minute {
				sessions = append(sessions, Session{Type: FocusSession, Duration: 25 * time.Minute})
				rem -= 25 * time.Minute
			} else {
				// For small leftovers, try to append to a 25 or 50 min session, 
				// but NEVER let a session exceed its structural caps.
				appended := false
				for i := len(sessions) - 1; i >= 0; i-- {
					if sessions[i].Type == FocusSession {
						// Don't let 50m go past 50m, or 25m go past 50m
						if (sessions[i].Duration == 50*time.Minute) || 
						   (sessions[i].Duration == 90*time.Minute) {
							continue 
						}
						sessions[i].Duration += rem
						appended = true
						break
					}
				}
				if !appended {
					sessions = append(sessions, Session{Type: FocusSession, Duration: rem})
				}
				rem = 0
			}
		}
	}

	return sessions
}