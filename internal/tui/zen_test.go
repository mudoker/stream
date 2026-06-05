package tui

import (
	"testing"
	"time"
)

func TestPartitionTask(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected []Session
	}{
		{
			name:     "180 minutes task (>= 110m)",
			duration: 180 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 90 * time.Minute},
				{Type: BreakSession, Duration: 20 * time.Minute},
				{Type: FocusSession, Duration: 60 * time.Minute}, // 10m trailing merged
				{Type: BreakSession, Duration: 10 * time.Minute},
			},
		},
		{
			name:     "80 minutes task (60m - 110m)",
			duration: 80 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 70 * time.Minute}, // 20m trailing merged
				{Type: BreakSession, Duration: 10 * time.Minute},
			},
		},
		{
			name:     "40 minutes task (< 60m)",
			duration: 40 * time.Minute,
			expected: []Session{
				{Type: FocusSession, Duration: 35 * time.Minute},
				{Type: BreakSession, Duration: 5 * time.Minute},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := PartitionTask(tt.duration)
			if len(sessions) != len(tt.expected) {
				t.Fatalf("Expected %d sessions, got %d", len(tt.expected), len(sessions))
			}
			for i := range sessions {
				if sessions[i].Type != tt.expected[i].Type {
					t.Errorf("Session %d: expected type %v, got %v", i, tt.expected[i].Type, sessions[i].Type)
				}
				if sessions[i].Duration != tt.expected[i].Duration {
					t.Errorf("Session %d: expected duration %v, got %v", i, tt.expected[i].Duration, sessions[i].Duration)
				}
			}
		})
	}
}
