package tui

import (
	"fmt"
	"strings"
	"time"
)

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

	sb.WriteString(strings.Repeat("█", wholeBlocks))

	if wholeBlocks < width {
		blocks := []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
		idx := int(remainder * 8)
		if idx > 0 && idx < len(blocks) {
			sb.WriteString(blocks[idx])
			wholeBlocks++
		}
	}

	if wholeBlocks < width {
		sb.WriteString(strings.Repeat("░", width-wholeBlocks))
	}

	return sb.String()
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
