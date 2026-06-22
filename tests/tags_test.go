package tests

import (
	"sort"
	"strings"
	"testing"

	"stream/internal/db"
	"stream/internal/sync"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTagsAutocompleteAndConsent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	modelVal := viewmodel.NewModel(database, syncEngine)
	m := &modelVal

	// 1. Verify default tags are seeded
	tags := database.GetTags()
	if len(tags) == 0 {
		t.Fatal("expected default tags to be seeded")
	}

	// Find "work" and "personal" frequencies
	var workFreq, personalFreq int
	for _, tag := range tags {
		if tag.Name == "work" {
			workFreq = tag.Frequency
		}
		if tag.Name == "personal" {
			personalFreq = tag.Frequency
		}
	}

	t.Logf("Seeded tags: work frequency = %d, personal frequency = %d", workFreq, personalFreq)

	// 2. Test prefix autocompletion
	m.Form.TagsInput.SetValue("work, per")
	sug := m.GetTagsAutocompleteSuggestion()
	if sug != "sonal" {
		t.Errorf("expected autocomplete suggestion 'sonal', got '%s'", sug)
	}

	m.AutocompleteTag(sug)
	if m.Form.TagsInput.Value() != "work, personal, " {
		t.Errorf("expected autocomplete input value to be 'work, personal, ', got '%s'", m.Form.TagsInput.Value())
	}

	// 3. Submit task with a new tag: should ask for consent
	m.Form.TitleInput.SetValue("New task with new tag")
	m.Form.TagsInput.SetValue("work, learning, extremelynewtag")
	m.SubmitForm()

	if !m.ConfirmOpen || m.ConfirmActionType != "save_tag_confirm" {
		t.Fatal("expected save_tag_confirm modal to be open")
	}

	if len(m.PendingNewTags) != 1 || m.PendingNewTags[0] != "extremelynewtag" {
		t.Fatalf("expected pending new tag to be 'extremelynewtag', got %v", m.PendingNewTags)
	}

	// Select "No, Just Submit" (index 1) and press enter
	m.ConfirmSelectedIndex = 1
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.ConfirmOpen {
		t.Error("expected confirm modal to close")
	}

	// Check that the task is created
	tasks := database.GetTasks()
	if len(tasks) == 0 {
		t.Fatal("expected task to be submitted and saved")
	}
	foundNewTask := false
	for _, tsk := range tasks {
		if tsk.Title == "New task with new tag" {
			foundNewTask = true
			hasTag := false
			for _, tg := range tsk.Tags {
				if tg == "extremelynewtag" {
					hasTag = true
				}
			}
			if !hasTag {
				t.Error("expected task to keep the new tag")
			}
		}
	}
	if !foundNewTask {
		t.Error("expected to find the submitted task in DB")
	}

	// Verify the new tag was NOT persisted in settings
	tagsAfterNo := database.GetTags()
	for _, tag := range tagsAfterNo {
		if tag.Name == "extremelynewtag" {
			t.Error("expected extremelynewtag NOT to be saved in system tags after choosing No")
		}
	}

	// 4. Submit another task with another new tag, but select "Yes, Save and Submit" (index 0)
	m.Form.TitleInput.SetValue("Another task with new tag")
	m.Form.TagsInput.SetValue("yetsimplertag")
	m.SubmitForm()

	if !m.ConfirmOpen || m.ConfirmActionType != "save_tag_confirm" {
		t.Fatal("expected save_tag_confirm modal to be open for yetsimplertag")
	}

	m.ConfirmSelectedIndex = 0 // Yes, Save
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	// Verify the new tag WAS persisted in settings
	tagsAfterYes := database.GetTags()
	foundTagInSystem := false
	for _, tag := range tagsAfterYes {
		if tag.Name == "yetsimplertag" {
			foundTagInSystem = true
		}
	}
	if !foundTagInSystem {
		t.Error("expected yetsimplertag to be saved in system tags after choosing Yes")
	}

	// Frequency check: "work" frequency should have incremented since it was in the system tags list
	tagsFinal := database.GetTags()
	for _, tag := range tagsFinal {
		if tag.Name == "work" && tag.Frequency != workFreq+1 {
			t.Errorf("expected 'work' tag frequency to increment from %d to %d, got %d", workFreq, workFreq+1, tag.Frequency)
		}
	}
}

func TestTagsCRUDModal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	modelVal := viewmodel.NewModel(database, syncEngine)
	m := &modelVal

	// 1. Run tags command
	res, _ := m.RunCommand("tags")
	m = res.(*viewmodel.Model)

	if m.CurrentMode != viewmodel.ModeTagsCRUD {
		t.Fatalf("expected mode to be ModeTagsCRUD, got %s", m.CurrentMode)
	}
	if m.TagsCRUDState != "BROWSE" {
		t.Errorf("expected initial state to be BROWSE, got %s", m.TagsCRUDState)
	}

	// Get initial list size
	initialCount := len(database.GetTags())

	// 2. Press "c" to Create a new tag
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(*viewmodel.Model)
	if m.TagsCRUDState != "CREATE" {
		t.Errorf("expected state to change to CREATE, got %s", m.TagsCRUDState)
	}

	// Type tag name and press Enter
	m.TagsCRUDInput.SetValue("brandnewtag")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.TagsCRUDState != "BROWSE" {
		t.Errorf("expected state to return to BROWSE, got %s", m.TagsCRUDState)
	}

	tags := database.GetTags()
	if len(tags) != initialCount+1 {
		t.Errorf("expected %d tags, got %d", initialCount+1, len(tags))
	}

	// Find brandnewtag in sorted slice to get its selection index
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Frequency != tags[j].Frequency {
			return tags[i].Frequency > tags[j].Frequency
		}
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	foundIdx := -1
	for idx, tag := range tags {
		if tag.Name == "brandnewtag" {
			foundIdx = idx
			break
		}
	}
	if foundIdx == -1 {
		t.Fatal("could not find brandnewtag in settings tags")
	}

	// 3. Highlight the new tag and increment its frequency with "+"
	m.TagsCRUDSelectedIndex = foundIdx
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	m = res.(*viewmodel.Model)

	updatedTags := database.GetTags()
	foundBrandNew := false
	for _, tag := range updatedTags {
		if tag.Name == "brandnewtag" {
			foundBrandNew = true
			if tag.Frequency != 2 {
				t.Errorf("expected brandnewtag frequency to increment to 2, got %d", tag.Frequency)
			}
		}
	}
	if !foundBrandNew {
		t.Error("expected to find brandnewtag in updated tags")
	}

	// Find brandnewtag index in new sorted tags to select it for editing
	sort.Slice(updatedTags, func(i, j int) bool {
		if updatedTags[i].Frequency != updatedTags[j].Frequency {
			return updatedTags[i].Frequency > updatedTags[j].Frequency
		}
		return strings.ToLower(updatedTags[i].Name) < strings.ToLower(updatedTags[j].Name)
	})
	for idx, tag := range updatedTags {
		if tag.Name == "brandnewtag" {
			m.TagsCRUDSelectedIndex = idx
			break
		}
	}

	// 4. Edit the name of the new tag
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(*viewmodel.Model)
	if m.TagsCRUDState != "EDIT" {
		t.Errorf("expected state to change to EDIT, got %s", m.TagsCRUDState)
	}

	m.TagsCRUDInput.SetValue("renamedtag")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	if m.TagsCRUDState != "BROWSE" {
		t.Errorf("expected state to return to BROWSE after edit, got %s", m.TagsCRUDState)
	}

	tagsAfterEdit := database.GetTags()
	foundRenamed := false
	for _, tag := range tagsAfterEdit {
		if tag.Name == "renamedtag" {
			foundRenamed = true
			if tag.Frequency != 2 {
				t.Errorf("expected frequency of renamed tag to be preserved (2), got %d", tag.Frequency)
			}
		}
		if tag.Name == "brandnewtag" {
			t.Error("old tag brandnewtag should not exist anymore")
		}
	}
	if !foundRenamed {
		t.Error("expected to find renamedtag in settings tags")
	}

	// Find renamedtag index in sorted tags
	sort.Slice(tagsAfterEdit, func(i, j int) bool {
		if tagsAfterEdit[i].Frequency != tagsAfterEdit[j].Frequency {
			return tagsAfterEdit[i].Frequency > tagsAfterEdit[j].Frequency
		}
		return strings.ToLower(tagsAfterEdit[i].Name) < strings.ToLower(tagsAfterEdit[j].Name)
	})
	for idx, tag := range tagsAfterEdit {
		if tag.Name == "renamedtag" {
			m.TagsCRUDSelectedIndex = idx
			break
		}
	}

	// 5. Delete the renamed tag using "d"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = res.(*viewmodel.Model)

	tagsAfterDelete := database.GetTags()
	if len(tagsAfterDelete) != initialCount {
		t.Errorf("expected tag count to return to %d, got %d", initialCount, len(tagsAfterDelete))
	}
	for _, tag := range tagsAfterDelete {
		if tag.Name == "renamedtag" {
			t.Error("expected renamedtag to be deleted")
		}
	}

	// 6. Press ESC to close modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(*viewmodel.Model)
	if m.CurrentMode != viewmodel.ModeNormal {
		t.Errorf("expected mode to return to ModeNormal, got %s", m.CurrentMode)
	}
}

func TestTagsCRUDModalCommaSeparated(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	database, err := db.NewJSONDB()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	syncEngine, err := sync.NewSyncEngine(database, nil, nil)
	if err != nil {
		t.Fatalf("failed to create sync engine: %v", err)
	}

	modelVal := viewmodel.NewModel(database, syncEngine)
	m := &modelVal

	// Open CRUD tags modal
	res, _ := m.RunCommand("tags")
	m = res.(*viewmodel.Model)

	initialCount := len(database.GetTags())

	// Press "c" to Create new tags with comma
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(*viewmodel.Model)

	m.TagsCRUDInput.SetValue("engineering, business")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	tags := database.GetTags()
	if len(tags) != initialCount+2 {
		t.Errorf("expected %d tags, got %d", initialCount+2, len(tags))
	}

	foundEngineering := false
	foundBusiness := false
	for _, tag := range tags {
		if tag.Name == "engineering" {
			foundEngineering = true
		}
		if tag.Name == "business" {
			foundBusiness = true
		}
	}
	if !foundEngineering || !foundBusiness {
		t.Error("expected both 'engineering' and 'business' tags to be added")
	}

	// Now try renaming one tag into comma-separated tags
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Frequency != tags[j].Frequency {
			return tags[i].Frequency > tags[j].Frequency
		}
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})
	businessIdx := -1
	for idx, tag := range tags {
		if tag.Name == "business" {
			businessIdx = idx
			break
		}
	}
	if businessIdx == -1 {
		t.Fatal("could not find business tag index")
	}

	m.TagsCRUDSelectedIndex = businessIdx
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(*viewmodel.Model)

	m.TagsCRUDInput.SetValue("marketing, finance")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*viewmodel.Model)

	tagsAfterEdit := database.GetTags()
	foundMarketing := false
	foundFinance := false
	for _, tag := range tagsAfterEdit {
		if tag.Name == "marketing" {
			foundMarketing = true
		}
		if tag.Name == "finance" {
			foundFinance = true
		}
		if tag.Name == "business" {
			t.Error("expected 'business' tag to be removed")
		}
	}
	if !foundMarketing || !foundFinance {
		t.Error("expected both 'marketing' and 'finance' tags to be added after renaming")
	}
}

