package components

import (
	"fmt"
	"strings"

	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderFormField draws a numbered input field inside a form modal.
func RenderFormField(num, label, input string, isActive bool, t theme.Theme) string {
	numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
	lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
	if isActive {
		lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
	}
	return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), input)
}

// RenderFormFieldWide draws a wider numbered input field inside a form modal (e.g. settings).
func RenderFormFieldWide(num, label, input string, labelWidth int, isActive bool, t theme.Theme) string {
	numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
	lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
	if isActive {
		lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
	}
	formatStr := fmt.Sprintf("  %%s  %%-%ds %%s", labelWidth)
	return fmt.Sprintf(formatStr, numStyle, lblStyle.Render(label), input)
}

// RenderFormDropdown draws a dropdown selection option in forms.
func RenderFormDropdown(num, label, value string, isActive bool, t theme.Theme) string {
	numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
	lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
	if isActive {
		lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		valStr := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
		return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
	}
	valStr := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
	return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), valStr)
}

// RenderFormDropdownWide draws a wider dropdown selection option (e.g. settings).
func RenderFormDropdownWide(num, label, value string, labelWidth int, isActive bool, t theme.Theme) string {
	numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
	lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
	formatStr := fmt.Sprintf("  %%s  %%-%ds %%s", labelWidth)
	var valStr string
	if isActive {
		lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
		valStr = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
	} else {
		valStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
	}
	return fmt.Sprintf(formatStr, numStyle, lblStyle.Render(label), valStr)
}

// RenderFormSubmitButton draws the highlighted/dimmed submit button.
func RenderFormSubmitButton(text string, isActive bool, t theme.Theme) string {
	submitFg := t.Muted
	submitText := fmt.Sprintf("  %s  ", text)
	if isActive {
		submitFg = t.SuccessColor
		submitText = fmt.Sprintf("[ %s ]", text)
	}
	return lipgloss.NewStyle().Foreground(submitFg).Bold(true).Render(submitText)
}

// RenderDaysSelect draws a recurring days picker.
func RenderDaysSelect(num, label string, selectedDays []bool, activeSubIdx int, isActiveField bool, t theme.Theme) string {
	numStyle := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%2s", num))
	lblStyle := lipgloss.NewStyle().Foreground(t.Fg)
	if isActiveField {
		lblStyle = lblStyle.Foreground(t.Accent).Bold(true)
	}

	dayNames := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	var dayStrs []string
	for i, name := range dayNames {
		sel := selectedDays[i]
		isCursor := isActiveField && activeSubIdx == i

		var dStr string
		if sel && isCursor {
			dStr = lipgloss.NewStyle().
				Foreground(t.CanvasBg).
				Background(t.Accent).
				Bold(true).
				Render(" ✓" + name + " ")
		} else if sel && !isCursor {
			dStr = lipgloss.NewStyle().
				Foreground(t.SuccessColor).
				Bold(true).
				Render(" ✓" + name + " ")
		} else if !sel && isCursor {
			dStr = lipgloss.NewStyle().
				Foreground(t.CanvasBg).
				Background(t.Muted).
				Render(" ·" + strings.ToLower(name) + " ")
		} else {
			dStr = lipgloss.NewStyle().
				Foreground(t.Muted).
				Render("  " + strings.ToLower(name) + " ")
		}
		dayStrs = append(dayStrs, dStr)
	}

	daysRow := strings.Join(dayStrs, "")
	return fmt.Sprintf("  %s  %-16s %s", numStyle, lblStyle.Render(label), daysRow)
}
