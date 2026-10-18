package tests

import (
	"strings"
	"testing"
	"time"

	"stream/internal/db"
	"stream/internal/model"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPalettePartialSearchHelp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, _ := db.NewJSONDB()
	modelVal := viewmodel.NewModel(database, nil)
	m := &modelVal

	// Open command palette with ':'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeCommand {
		t.Fatalf("expected ModeCommand, got %v", m.CurrentMode)
	}

	// Type "hel"
	for _, r := range "hel" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}

	// Filtered list should have "help" as the top match
	filtered := m.GetFilteredCommandList()
	if len(filtered) == 0 {
		t.Fatalf("expected filtered results for 'hel', got none")
	}
	if filtered[0].Name != "help" {
		t.Fatalf("expected top filtered match to be 'help', got '%s'", filtered[0].Name)
	}

	// Press Enter without moving arrow keys
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected ModeNormal after enter, got %v", m.CurrentMode)
	}
	if !m.HelpOpen {
		t.Errorf("expected HelpOpen to be true after pressing enter on partial match 'hel'")
	}
	if strings.Contains(m.StatusMsg, "Unknown command") || strings.Contains(m.StatusMsg, "not found") {
		t.Errorf("expected statusMsg to not show unknown command error, got: %s", m.StatusMsg)
	}
}

func TestCommandPalettePartialSearchViews(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, _ := db.NewJSONDB()
	modelVal := viewmodel.NewModel(database, nil)
	m := &modelVal

	// Test "dash" -> DashboardView
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)
	for _, r := range "dash" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)
	if m.CurrentView != viewmodel.DashboardView {
		t.Errorf("expected DashboardView, got %v", m.CurrentView)
	}

	// Test "spr" -> SprintView
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)
	for _, r := range "spr" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)
	if m.CurrentView != viewmodel.SprintView {
		t.Errorf("expected SprintView, got %v", m.CurrentView)
	}

	// Test "mon" -> MonthView
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)
	for _, r := range "mon" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)
	if m.CurrentView != viewmodel.MonthView {
		t.Errorf("expected MonthView, got %v", m.CurrentView)
	}
}

func TestCommandPaletteEmptyEnterClosesSafely(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, _ := db.NewJSONDB()
	modelVal := viewmodel.NewModel(database, nil)
	m := &modelVal

	// Open command palette with ':'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)

	// Immediately press Enter on empty input
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected ModeNormal after enter on empty command palette, got %v", m.CurrentMode)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd (not quit), got non-nil")
	}
}

func TestCommandPaletteArgumentPrefixPopulation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, _ := db.NewJSONDB()
	modelVal := viewmodel.NewModel(database, nil)
	m := &modelVal

	// Create an active sprint
	sprint := model.Sprint{
		UUID:      "sprint-test-cmd",
		Name:      "Sprint 1",
		StartDate: time.Now(),
		EndDate:   time.Now().Add(14 * 24 * time.Hour),
	}
	database.AddSprint(sprint)
	m.Sprints = database.GetSprints()

	// Open command palette with ':'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	m = res.(*viewmodel.Model)

	// Type "feat"
	for _, r := range "feat" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}

	// Press Enter on "feat" -> should populate "feature " and keep command palette open
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeCommand {
		t.Fatalf("expected ModeCommand to remain open for argument input, got %v", m.CurrentMode)
	}
	if m.CommandInput.Value() != "feature " {
		t.Fatalf("expected input to be 'feature ', got '%s'", m.CommandInput.Value())
	}
	if m.CommandInput.Position() != len("feature ") {
		t.Errorf("expected cursor position at %d, got %d", len("feature "), m.CommandInput.Position())
	}

	// Now type title
	for _, r := range "Dark Mode" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(*viewmodel.Model)
	}

	// Press Enter to submit
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected ModeNormal after feature creation, got %v", m.CurrentMode)
	}

	tasks := database.GetTasks()
	found := false
	for _, tk := range tasks {
		if tk.Title == "Dark Mode" && tk.WorkItemType == model.WorkItemFeature {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected feature task 'Dark Mode' to be created in database")
	}
}
