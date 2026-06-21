package theme

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"stream/internal/model"
	"stream/internal/viewmodel/common/constants"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)


type Theme struct {
	// Colors (Linear / Arc Palette)
	CanvasBg    lipgloss.Color // Layer 0: Charcoal Black (#121318)
	PanelBg     lipgloss.Color // Layer 1: Elevated Slate (#1c1d24)
	SelectedBg  lipgloss.Color // Layer 2: Selected Focus (#2b2d38)
	ModalBg     lipgloss.Color // Layer 3: Modal Elevated (#252730)
	Fg          lipgloss.Color // Primary Text (#e2e8f0)
	Muted       lipgloss.Color // slate gray helper text (#626875)
	Accent      lipgloss.Color // Linear Signature Indigo (#5e6ad2)
	FocusPurple lipgloss.Color // Focus Highlight (#8b5cf6)

	// Priorities
	P0Color lipgloss.Color // Crimson Rose (#f43f5e)
	P1Color lipgloss.Color // Amber (#f59e0b)
	P2Color lipgloss.Color // Soft Blue (#3b82f6)
	P3Color lipgloss.Color // Gray (#6b7280)

	// Statuses
	SuccessColor lipgloss.Color // Emerald Green (#10b981)

	// Styling templates
	BaseStyle      lipgloss.Style
	PanelStyle     lipgloss.Style
	SelectedPanel  lipgloss.Style
	ModalStyle     lipgloss.Style
	HeaderStyle    lipgloss.Style
	FooterStyle    lipgloss.Style
	TitleHeroStyle lipgloss.Style
	MetadataStyle  lipgloss.Style
}

func NewTheme() Theme {
	canvasBg := lipgloss.Color(constants.ColorCanvasBg)
	panelBg := lipgloss.Color(constants.ColorPanelBg)
	selectedBg := lipgloss.Color(constants.ColorSelectedBg)
	modalBg := lipgloss.Color(constants.ColorModalBg)
	fg := lipgloss.Color(constants.ColorFg)
	muted := lipgloss.Color(constants.ColorMuted)
	accent := lipgloss.Color(constants.ColorAccent)
	focusPurple := lipgloss.Color(constants.ColorFocusPurple)

	p0 := lipgloss.Color(constants.ColorP0)
	p1 := lipgloss.Color(constants.ColorP1)
	p2 := lipgloss.Color(constants.ColorP2)
	p3 := lipgloss.Color(constants.ColorP3)

	success := lipgloss.Color(constants.ColorSuccess)

	return Theme{
		CanvasBg:     canvasBg,
		PanelBg:      panelBg,
		SelectedBg:   selectedBg,
		ModalBg:      modalBg,
		Fg:           fg,
		Muted:        muted,
		Accent:       accent,
		FocusPurple:  focusPurple,
		P0Color:      p0,
		P1Color:      p1,
		P2Color:      p2,
		P3Color:      p3,
		SuccessColor: success,

		BaseStyle: lipgloss.NewStyle().
			Foreground(fg),

		PanelStyle: lipgloss.NewStyle().
			Foreground(fg).
			Padding(1, 2),

		SelectedPanel: lipgloss.NewStyle().
			Foreground(fg).
			Padding(1, 2),

		ModalStyle: lipgloss.NewStyle().
			Foreground(fg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2),

		HeaderStyle: lipgloss.NewStyle().
			Foreground(fg).
			Padding(0, 1).
			Bold(true),

		FooterStyle: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),

		TitleHeroStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(fg),

		MetadataStyle: lipgloss.NewStyle().
			Foreground(muted),
	}
}

func (t Theme) PriorityColor(p model.Priority) lipgloss.Color {
	switch p {
	case model.P0:
		return t.P0Color
	case model.P1:
		return t.P1Color
	case model.P2:
		return t.P2Color
	default:
		return t.P3Color
	}
}

func SentenceCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func SliceAnsi(s string, start, end int) string {
	var sb strings.Builder
	var runes = []rune(s)
	var inEscape = false
	var visualCount = 0

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			sb.WriteRune(r)
			continue
		}
		if inEscape {
			sb.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		if visualCount >= start && visualCount < end {
			sb.WriteRune(r)
		}
		visualCount++
	}
	return sb.String()
}

type Cell struct {
	Text           string
	Style          string
	IsContinuation bool
}

func ParseLineToCells(s string) []Cell {
	var cells []Cell
	var currentStyle strings.Builder
	var runes = []rune(s)
	var inEscape = false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEscape = true
			currentStyle.WriteRune(r)
			continue
		}
		if inEscape {
			currentStyle.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		w := runewidth.RuneWidth(r)
		if w <= 0 {
			if len(cells) > 0 {
				idx := len(cells) - 1
				for idx > 0 && cells[idx].IsContinuation {
					idx--
				}
				cells[idx].Text += string(r)
			}
			continue
		}

		styleStr := currentStyle.String()
		cells = append(cells, Cell{
			Text:           string(r),
			Style:          styleStr,
			IsContinuation: false,
		})
		for k := 1; k < w; k++ {
			cells = append(cells, Cell{
				Text:           "",
				Style:          styleStr,
				IsContinuation: true,
			})
		}
	}
	return cells
}

func CellsToLine(cells []Cell) string {
	var sb strings.Builder
	var lastStyle string

	for _, cell := range cells {
		if cell.IsContinuation {
			continue
		}
		if cell.Style != lastStyle {
			if cell.Style == "" {
				sb.WriteString("\x1b[0m")
			} else {
				sb.WriteString(cell.Style)
			}
			lastStyle = cell.Style
		}
		if cell.Text != "" {
			sb.WriteString(cell.Text)
		} else {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\x1b[0m")
	return sb.String()
}

func OverlayString(base string, overlay string, x int, y int, baseWidth int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := 0
	for _, l := range overlayLines {
		cells := ParseLineToCells(l)
		if len(cells) > overlayWidth {
			overlayWidth = len(cells)
		}
	}

	for i, oLine := range overlayLines {
		targetY := y + i
		if targetY >= len(baseLines) {
			break
		}
		bLine := baseLines[targetY]
		baseCells := ParseLineToCells(bLine)
		overlayCells := ParseLineToCells(oLine)

		// Pad base line to baseWidth
		for len(baseCells) < baseWidth {
			baseCells = append(baseCells, Cell{Text: " "})
		}
		if len(baseCells) > baseWidth {
			baseCells = baseCells[:baseWidth]
		}

		// Pad overlay line to overlayWidth
		for len(overlayCells) < overlayWidth {
			overlayCells = append(overlayCells, Cell{Text: " "})
		}
		if len(overlayCells) > overlayWidth {
			overlayCells = overlayCells[:overlayWidth]
		}

		// Overwrite base line cells starting at x
		for col := 0; col < overlayWidth; col++ {
			targetX := x + col
			if targetX >= 0 && targetX < baseWidth {
				baseCells[targetX] = overlayCells[col]
			}
		}

		baseLines[targetY] = CellsToLine(baseCells)
	}
	return strings.Join(baseLines, "\n")
}

func WrapText(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	var words = strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var res []string
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > limit {
			if len(currentLine) > 0 {
				res = append(res, currentLine)
			}
			currentLine = word
		} else {
			if len(currentLine) > 0 {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if len(currentLine) > 0 {
		res = append(res, currentLine)
	}
	return strings.Join(res, "\n")
}

func IndentText(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = indent + l
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

var blockDigits = map[rune][]string{
	'0': {"██████", "██  ██", "██  ██", "██  ██", "██████"},
	'1': {"    ██", "    ██", "    ██", "    ██", "    ██"},
	'2': {"██████", "    ██", "██████", "██    ", "██████"},
	'3': {"██████", "    ██", "██████", "    ██", "██████"},
	'4': {"██  ██", "██  ██", "██████", "    ██", "    ██"},
	'5': {"██████", "██    ", "██████", "    ██", "██████"},
	'6': {"██████", "██    ", "██████", "██  ██", "██████"},
	'7': {"██████", "    ██", "    ██", "    ██", "    ██"},
	'8': {"██████", "██  ██", "██████", "██  ██", "██████"},
	'9': {"██████", "██  ██", "██████", "    ██", "██████"},
	':': {"      ", "  ▄▄  ", "      ", "  ▄▄  ", "      "},
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

var blockDigits3 = map[rune][]string{
	'0': {"█▀▀▀█", "█   █", "█▄▄▄█"},
	'1': {"  █  ", "  █  ", "  █  "},
	'2': {"█▀▀▀█", "  ▄█▀", "█▄▄▄█"},
	'3': {"█▀▀▀█", " ▀▀▀█", "█▄▄▄█"},
	'4': {"█  █ ", "█▄▄█▀", "   █ "},
	'5': {"█▀▀▀▀", "▀▀▀▀█", "▄▄▄▄█"},
	'6': {"█▀▀▀▀", "█▄▄▄█", "█▄▄▄█"},
	'7': {"█▀▀▀█", "   █▀", "  █▀ "},
	'8': {"█▀▀▀█", "█▄▄▄█", "█▄▄▄█"},
	'9': {"█▀▀▀█", "▀▀▀██", "▄▄▄▄█"},
	':': {"  ▄  ", "     ", "  ▄  "},
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

