package tests

import (
	"strings"
	"testing"

	"stream/internal/view/modals"
	"stream/internal/view/theme"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpModalNavigationAndScrolling(t *testing.T) {
	database, cleanup := setupTestSprintDB(t)
	defer cleanup()

	m := viewmodel.NewModel(database, nil)
	m.Width = 120
	m.Height = 40
	m.Layout = viewmodel.ComputeLayout(120, 40)

	// 1. Open help modal via '?'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !m.HelpOpen {
		t.Fatal("expected HelpOpen to be true after pressing '?'")
	}
	if m.HelpScrollOffset != 0 {
		t.Fatalf("expected initial HelpScrollOffset 0, got %d", m.HelpScrollOffset)
	}

	// 2. Render initial modal (0%)
	th := theme.NewTheme()
	rendered := modals.RenderHelpModal(&m, th)
	if !strings.Contains(rendered, "0%") {
		t.Errorf("expected 0%% in rendered modal header/footer, got: %s", rendered)
	}

	// 3. Scroll down using 'j'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.HelpScrollOffset != 1 {
		t.Fatalf("expected HelpScrollOffset 1 after pressing 'j', got %d", m.HelpScrollOffset)
	}

	// 4. Scroll down with 'ctrl+d' or 'd'
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.HelpScrollOffset != 7 {
		t.Fatalf("expected HelpScrollOffset 7 after pressing ctrl+d, got %d", m.HelpScrollOffset)
	}

	// 5. Scroll up using 'k'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.HelpScrollOffset != 6 {
		t.Fatalf("expected HelpScrollOffset 6 after pressing 'k', got %d", m.HelpScrollOffset)
	}

	// 6. Jump to bottom using 'G'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	modals.RenderHelpModal(&m, th) // Triggers clamp and sync
	bottomOffset := m.HelpScrollOffset
	if bottomOffset == 0 {
		t.Fatal("expected non-zero HelpScrollOffset at bottom")
	}

	// 7. Verify that pressing 'k' immediately scrolls UP from the bottom (no stuck offset!)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.HelpScrollOffset != bottomOffset-1 {
		t.Fatalf("expected HelpScrollOffset to immediately decrement to %d after 'k', got %d", bottomOffset-1, m.HelpScrollOffset)
	}

	// 8. Jump to top using 'g'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.HelpScrollOffset != 0 {
		t.Fatalf("expected HelpScrollOffset 0 after pressing 'g', got %d", m.HelpScrollOffset)
	}

	// 9. Close help modal using 'q'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.HelpOpen {
		t.Fatal("expected HelpOpen to be false after pressing 'q'")
	}
}
