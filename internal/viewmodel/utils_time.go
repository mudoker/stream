package viewmodel

import (
	"strconv"
	"strings"
	"time"

	"stream/internal/viewmodel/common/constants"
)

const (
	RowsPerHour = constants.RowsPerHour
	TotalRows   = constants.TotalRows
	GutterWidth = constants.GutterWidth
)

// TimeToRow converts a time.Time to its local day row index (0 to TotalRows-1).
func TimeToRow(t time.Time) int {
	local := t.Local()
	return (local.Hour() * RowsPerHour) + (local.Minute() * RowsPerHour / 60)
}

// SameDay returns true if a and b are on the same calendar day in local time.
func SameDay(a, b time.Time) bool {
	aLocal := a.Local()
	bLocal := b.Local()
	return aLocal.Year() == bLocal.Year() && aLocal.Month() == bLocal.Month() && aLocal.Day() == bLocal.Day()
}

// ParseFlexibleTime parses time input in multiple formats:
// - "14" -> 14:00
// - "14:30" -> 14:30
// - "9" -> 09:00
// - "09:30" -> 09:30
// If parsing fails, returns the provided default values.
func ParseFlexibleTime(timeStr string, defaultHour, defaultMin int) (int, int) {
	hour, min := defaultHour, defaultMin

	if strings.Contains(timeStr, ":") {
		// Format: HH:MM - only apply if both hour and minute are valid
		parts := strings.Split(timeStr, ":")
		if len(parts) == 2 {
			h, errH := strconv.Atoi(parts[0])
			m, errM := strconv.Atoi(parts[1])
			// Only update if both parse successfully and are in valid ranges
			if errH == nil && errM == nil && h >= 0 && h < 24 && m >= 0 && m < 60 {
				hour = h
				min = m
			}
		}
	} else {
		// Format: H or HH
		if h, err := strconv.Atoi(strings.TrimSpace(timeStr)); err == nil && h >= 0 && h < 24 {
			hour = h
			min = 0
		}
	}

	return hour, min
}
