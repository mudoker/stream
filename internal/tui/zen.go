package tui

import (
	"fmt"
	"strings"
	"time"

	"stream/internal/model"
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
		// Default to 25m Focus if no duration
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
			// Under 60m remaining
			if len(sessions) > 0 {
				if rem < 30*time.Minute {
					// Merge into the last Focus session to avoid short intervals
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
				// First block, and total duration is < 60m
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

var blockDigits = map[rune][]string{
	'0': {
		"██████",
		"██  ██",
		"██  ██",
		"██  ██",
		"██████",
	},
	'1': {
		"    ██",
		"    ██",
		"    ██",
		"    ██",
		"    ██",
	},
	'2': {
		"██████",
		"    ██",
		"██████",
		"██    ",
		"██████",
	},
	'3': {
		"██████",
		"    ██",
		"██████",
		"    ██",
		"██████",
	},
	'4': {
		"██  ██",
		"██  ██",
		"██████",
		"    ██",
		"    ██",
	},
	'5': {
		"██████",
		"██    ",
		"██████",
		"    ██",
		"██████",
	},
	'6': {
		"██████",
		"██    ",
		"██████",
		"██  ██",
		"██████",
	},
	'7': {
		"██████",
		"    ██",
		"    ██",
		"    ██",
		"    ██",
	},
	'8': {
		"██████",
		"██  ██",
		"██████",
		"██  ██",
		"██████",
	},
	'9': {
		"██████",
		"██  ██",
		"██████",
		"    ██",
		"██████",
	},
	':': {
		"      ",
		"  ▄▄  ",
		"      ",
		"  ▄▄  ",
		"      ",
	},
}

func RenderLargeTime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	timeStr := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	lines := make([]string, 5)

	for _, char := range timeStr {
		glyph, exists := blockDigits[char]
		if !exists {
			continue
		}
		for i := 0; i < 5; i++ {
			lines[i] = lines[i] + glyph[i] + "  "
		}
	}

	return strings.Join(lines, "\n")
}

// RenderProgressBar renders a fluid progress bar with fractional block characters
func RenderProgressBar(width int, percent float64) string {
	if percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}

	totalBlocks := float64(width)
	filledBlocks := percent * totalBlocks
	wholeBlocks := int(filledBlocks)
	remainder := filledBlocks - float64(wholeBlocks)

	var sb strings.Builder

	// Full blocks
	sb.WriteString(strings.Repeat("█", wholeBlocks))

	// Fractional block
	if wholeBlocks < width {
		// Unicode block weights: Left to right
		blocks := []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
		idx := int(remainder * 8)
		if idx > 0 && idx < len(blocks) {
			sb.WriteString(blocks[idx])
			wholeBlocks++
		}
	}

	// Empty spaces
	if wholeBlocks < width {
		sb.WriteString(strings.Repeat("░", width-wholeBlocks))
	}

	return sb.String()
}

type ZenTimer struct {
	Task             model.Task
	Sessions         []Session
	CurrentSessionIdx int
	TimeRemaining    time.Duration
	TotalDuration    time.Duration // Total duration of the current session
	IsPaused         bool
	Running          bool
}

func NewZenTimer(t model.Task) *ZenTimer {
	// Calculate total task duration: SP * 45m or scheduled window
	dur := time.Duration(t.StoryPoints) * 45 * time.Minute
	if t.SchedulingType == model.Anchored {
		dur = t.TimeWindow.End.Sub(t.TimeWindow.Start)
	}

	// Deduct elapsed focus seconds to resume from where we left off
	elapsed := time.Duration(t.ExecutionMetrics.ElapsedFocusSeconds) * time.Second
	dur -= elapsed
	if dur < 0 {
		dur = 0
	}

	sessions := PartitionTask(dur)
	var timeRemaining time.Duration
	if len(sessions) > 0 {
		timeRemaining = sessions[0].Duration
	}

	return &ZenTimer{
		Task:             t,
		Sessions:         sessions,
		CurrentSessionIdx: 0,
		TimeRemaining:    timeRemaining,
		TotalDuration:    timeRemaining,
		IsPaused:         false,
		Running:          true,
	}
}

func (zt *ZenTimer) RecordElapsedTimes() int {
	if zt.CurrentSessionIdx >= 0 && zt.CurrentSessionIdx < len(zt.Sessions) {
		sess := zt.Sessions[zt.CurrentSessionIdx]
		elapsed := zt.TotalDuration - zt.TimeRemaining
		if elapsed > 0 {
			elapsedSecs := int(elapsed.Seconds())
			if sess.Type == FocusSession {
				zt.Task.ExecutionMetrics.ElapsedFocusSeconds += elapsedSecs
			} else if sess.Type == BreakSession {
				zt.Task.ExecutionMetrics.ElapsedBreakSeconds += elapsedSecs
			}
			// Set TotalDuration equal to TimeRemaining to avoid double counting
			zt.TotalDuration = zt.TimeRemaining
			return elapsedSecs
		}
	}
	return 0
}

func (zt *ZenTimer) Tick() bool {
	if zt.IsPaused || !zt.Running {
		return false
	}

	zt.TimeRemaining -= time.Second
	if zt.TimeRemaining <= 0 {
		// Record elapsed focus/break time for the session block we are leaving
		zt.RecordElapsedTimes()

		// Increment completed pomodoros if leaving a focus session
		if zt.CurrentSessionIdx >= 0 && zt.CurrentSessionIdx < len(zt.Sessions) {
			if zt.Sessions[zt.CurrentSessionIdx].Type == FocusSession {
				zt.Task.ExecutionMetrics.TotalCompletedPomodoros++
			}
		}

		return zt.NextSession()
	}
	return false
}

func (zt *ZenTimer) NextSession() bool {
	zt.CurrentSessionIdx++
	if zt.CurrentSessionIdx >= len(zt.Sessions) {
		zt.Running = false
		zt.TimeRemaining = 0
		return true // Terminated naturally
	}
	zt.TimeRemaining = zt.Sessions[zt.CurrentSessionIdx].Duration
	zt.TotalDuration = zt.TimeRemaining
	return false
}

func (zt *ZenTimer) AddTime(d time.Duration) {
	zt.TimeRemaining += d
	if zt.TimeRemaining < 0 {
		zt.TimeRemaining = 0
	}
	zt.TotalDuration += d
	if zt.TotalDuration < 0 {
		zt.TotalDuration = 0
	}
}

func (zt *ZenTimer) UpdateTaskDuration(newTask model.Task) {
	zt.Task = newTask
	var newDur time.Duration
	if newTask.SchedulingType == model.Anchored {
		newDur = newTask.TimeWindow.End.Sub(newTask.TimeWindow.Start)
	} else {
		newDur = time.Duration(newTask.StoryPoints) * 45 * time.Minute
	}

	elapsedTotal := time.Duration(0)
	for i := 0; i < zt.CurrentSessionIdx; i++ {
		elapsedTotal += zt.Sessions[i].Duration
	}
	elapsedCurrent := zt.TotalDuration - zt.TimeRemaining
	elapsedTotal += elapsedCurrent

	if newDur <= elapsedTotal {
		// Truncate current session to what was already done
		zt.Sessions[zt.CurrentSessionIdx].Duration = elapsedCurrent
		zt.TotalDuration = elapsedCurrent
		zt.TimeRemaining = 0
		zt.Sessions = zt.Sessions[:zt.CurrentSessionIdx+1]
		zt.Running = false
		return
	}

	remainingToSchedule := newDur - elapsedTotal
	if remainingToSchedule <= zt.TimeRemaining {
		zt.TimeRemaining = remainingToSchedule
		zt.TotalDuration = elapsedCurrent + remainingToSchedule
		zt.Sessions[zt.CurrentSessionIdx].Duration = zt.TotalDuration
		zt.Sessions = zt.Sessions[:zt.CurrentSessionIdx+1]
	} else {
		// Keep current session's remaining time as is
		subRemaining := remainingToSchedule - zt.TimeRemaining
		keepCount := zt.CurrentSessionIdx + 1

		for j := zt.CurrentSessionIdx + 1; j < len(zt.Sessions); j++ {
			if subRemaining <= 0 {
				break
			}
			if zt.Sessions[j].Duration <= subRemaining {
				subRemaining -= zt.Sessions[j].Duration
				keepCount++
			} else {
				zt.Sessions[j].Duration = subRemaining
				subRemaining = 0
				keepCount++
				break
			}
		}
		zt.Sessions = zt.Sessions[:keepCount]

		if subRemaining > 0 {
			extraSessions := PartitionTask(subRemaining)
			zt.Sessions = append(zt.Sessions, extraSessions...)
		}
	}
}


var blockDigits3 = map[rune][]string{
	'0': {
		"█▀▀▀█",
		"█   █",
		"█▄▄▄█",
	},
	'1': {
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'2': {
		"█▀▀▀█",
		"  ▄█▀",
		"█▄▄▄█",
	},
	'3': {
		"█▀▀▀█",
		" ▀▀▀█",
		"█▄▄▄█",
	},
	'4': {
		"█  █ ",
		"█▄▄█▀",
		"   █ ",
	},
	'5': {
		"█▀▀▀▀",
		"▀▀▀▀█",
		"▄▄▄▄█",
	},
	'6': {
		"█▀▀▀▀",
		"█▄▄▄█",
		"█▄▄▄█",
	},
	'7': {
		"█▀▀▀█",
		"   █▀",
		"  █▀ ",
	},
	'8': {
		"█▀▀▀█",
		"█▄▄▄█",
		"█▄▄▄█",
	},
	'9': {
		"█▀▀▀█",
		"▀▀▀██",
		"▄▄▄▄█",
	},
	':': {
		"  ▄  ",
		"     ",
		"  ▄  ",
	},
}

func RenderLargeTime3(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	timeStr := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	lines := make([]string, 3)

	for _, char := range timeStr {
		glyph, exists := blockDigits3[char]
		if !exists {
			continue
		}
		for i := 0; i < 3; i++ {
			lines[i] = lines[i] + glyph[i] + "  "
		}
	}

	return strings.Join(lines, "\n")
}
