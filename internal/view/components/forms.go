package components

import (
	"fmt"
	"strings"

	"stream/internal/view/theme"

	"github.com/charmbracelet/lipgloss"
)

// FormFieldConfig defines the configuration for a reusable form field component.
type FormFieldConfig struct {
	Num        string
	Label      string
	ValueView  string
	LabelWidth int
	IsActive   bool
	Theme      theme.Theme

	// Optional style overrides
	NumStyle   *lipgloss.Style
	LabelStyle *lipgloss.Style
	ValueStyle *lipgloss.Style
}

// RenderBaseFormField draws a numbered label-value row inside a form modal or panel.
func RenderBaseFormField(cfg FormFieldConfig) string {
	lblW := cfg.LabelWidth
	if lblW <= 0 {
		lblW = 16
	}

	// 1. Prefix number style
	numStr := fmt.Sprintf("%2s", cfg.Num)
	var numView string
	if cfg.NumStyle != nil {
		numView = cfg.NumStyle.Render(numStr)
	} else {
		numView = lipgloss.NewStyle().Foreground(cfg.Theme.Muted).Render(numStr)
	}

	// 2. Label style
	var lblView string
	if cfg.LabelStyle != nil {
		lblView = cfg.LabelStyle.Render(cfg.Label)
	} else {
		lblStyle := lipgloss.NewStyle().Foreground(cfg.Theme.Fg)
		if cfg.IsActive {
			lblStyle = lblStyle.Foreground(cfg.Theme.Accent).Bold(true)
		}
		lblView = lblStyle.Render(cfg.Label)
	}

	// 3. Value/Input view
	var valView string
	if cfg.ValueStyle != nil {
		valView = cfg.ValueStyle.Render(cfg.ValueView)
	} else {
		valView = cfg.ValueView
	}

	formatStr := fmt.Sprintf("  %%s  %%-%ds %%s", lblW)
	return fmt.Sprintf(formatStr, numView, lblView, valView)
}

// RenderFormField draws a numbered input field inside a form modal.
func RenderFormField(num, label, input string, isActive bool, t theme.Theme) string {
	return RenderBaseFormField(FormFieldConfig{
		Num:       num,
		Label:     label,
		ValueView: input,
		IsActive:  isActive,
		Theme:     t,
	})
}

// RenderFormFieldWide draws a wider numbered input field inside a form modal (e.g. settings).
func RenderFormFieldWide(num, label, input string, labelWidth int, isActive bool, t theme.Theme) string {
	return RenderBaseFormField(FormFieldConfig{
		Num:        num,
		Label:      label,
		ValueView:  input,
		LabelWidth: labelWidth,
		IsActive:   isActive,
		Theme:      t,
	})
}

// RenderFormDropdown draws a dropdown selection option in forms.
func RenderFormDropdown(num, label, value string, isActive bool, t theme.Theme) string {
	var valStr string
	if isActive {
		valStr = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
	} else {
		valStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
	}
	return RenderBaseFormField(FormFieldConfig{
		Num:       num,
		Label:     label,
		ValueView: valStr,
		IsActive:  isActive,
		Theme:     t,
	})
}

// RenderFormDropdownWide draws a wider dropdown selection option (e.g. settings).
func RenderFormDropdownWide(num, label, value string, labelWidth int, isActive bool, t theme.Theme) string {
	var valStr string
	if isActive {
		valStr = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(fmt.Sprintf("◀ %s ▶", value))
	} else {
		valStr = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("  %s  ", value))
	}
	return RenderBaseFormField(FormFieldConfig{
		Num:        num,
		Label:      label,
		ValueView:  valStr,
		LabelWidth: labelWidth,
		IsActive:   isActive,
		Theme:      t,
	})
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
	return RenderBaseFormField(FormFieldConfig{
		Num:       num,
		Label:     label,
		ValueView: daysRow,
		IsActive:  isActiveField,
		Theme:     t,
	})
}
