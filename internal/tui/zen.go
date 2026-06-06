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

func (zt *ZenTimer) Tick() bool {
	if zt.IsPaused || !zt.Running {
		return false
	}

	zt.TimeRemaining -= time.Second
	if zt.TimeRemaining <= 0 {
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
	zt.TotalDuration += d
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
