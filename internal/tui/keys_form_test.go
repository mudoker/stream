package tui

import (
	"testing"
	"time"
)

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHour  int
		wantMin   int
		defaultH  int
		defaultM  int
	}{
		{"single digit hour", "14", 14, 0, 9, 0},
		{"single digit hour with default", "9", 9, 0, 9, 0},
		{"hour with colon and minutes", "14:30", 14, 30, 9, 0},
		{"leading zero", "09:15", 9, 15, 9, 0},
		{"invalid hour out of range", "25", 9, 0, 9, 0},
		{"invalid hour with colon", "25:30", 9, 0, 9, 0},
		{"invalid minutes", "14:75", 9, 0, 9, 0},
		{"zero hour", "0", 0, 0, 9, 0},
		{"zero hour with minutes", "0:15", 0, 15, 9, 0},
		{"empty string", "", 9, 0, 9, 0},
		{"whitespace", "  ", 9, 0, 9, 0},
		{"hour with whitespace", "  14  ", 14, 0, 9, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := ParseFlexibleTime(tt.input, tt.defaultH, tt.defaultM)
			if h != tt.wantHour || m != tt.wantMin {
				t.Errorf("ParseFlexibleTime(%q) = (%d, %d), want (%d, %d)",
					tt.input, h, m, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestFlexibleTimeInContext(t *testing.T) {
	// Simulate form submission with various time inputs
	tests := []struct {
		name        string
		timeInput   string
		expectedHour int
		expectedMin  int
	}{
		{"14 becomes 14:00", "14", 14, 0},
		{"14:30 stays 14:30", "14:30", 14, 30},
		{"9 becomes 9:00", "9", 9, 0},
		{"09:15 stays 09:15", "09:15", 9, 15},
	}

	selectedDay := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, min := ParseFlexibleTime(tt.timeInput, 9, 0)
			
			// Create a time with the parsed hour and minute
			result := time.Date(selectedDay.Year(), selectedDay.Month(), selectedDay.Day(), hour, min, 0, 0, time.UTC)
			
			if result.Hour() != tt.expectedHour || result.Minute() != tt.expectedMin {
				t.Errorf("Expected %02d:%02d, got %02d:%02d",
					tt.expectedHour, tt.expectedMin, result.Hour(), result.Minute())
			}
		})
	}
}

