package dub

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func TestNewSkillPromptKeyMap(t *testing.T) {
	t.Parallel()

	keys := newSkillPromptKeyMap(skillLanguageEN)
	if !slices.Contains(keys.MultiSelect.SelectAll.Keys(), "a") {
		t.Fatal("select-all does not bind a")
	}
	if !slices.Contains(keys.MultiSelect.SelectNone.Keys(), "n") {
		t.Fatal("select-none does not bind n")
	}
	if !slices.Contains(keys.MultiSelect.Toggle.Keys(), "space") {
		t.Fatal("toggle does not bind space")
	}
	if !slices.Contains(keys.MultiSelect.Filter.Keys(), "/") {
		t.Fatal("filter does not bind /")
	}
}

func TestNewSkillPromptFilterHelpLocalized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language skillLanguage
		apply    string
		clear    string
	}{
		{language: skillLanguageEN, apply: "apply filter", clear: "clear filter"},
		{language: skillLanguageZH, apply: "应用搜索", clear: "清除搜索"},
	}
	for _, test := range tests {
		keys := newSkillPromptKeyMap(test.language)
		if got := keys.MultiSelect.SetFilter.Help().Desc; got != test.apply {
			t.Errorf("language=%s apply help=%q", test.language, got)
		}
		if got := keys.MultiSelect.ClearFilter.Help().Desc; got != test.clear {
			t.Errorf("language=%s clear help=%q", test.language, got)
		}
	}
}

func TestClearableSkillMultiSelectShowsLocalizedFilterStateHelp(t *testing.T) {
	t.Parallel()

	selected := []string{"codex"}
	multiSelect := huh.NewMultiSelect[string]().
		Options(huh.NewOption("Codex", "codex")).
		Value(&selected).
		Filterable(true)
	field := &clearableSkillMultiSelect{Field: multiSelect}
	field.WithKeyMap(newSkillPromptKeyMap(skillLanguageZH))
	field.Focus()

	_, _ = field.Update(skillPromptKeyPress("/"))
	if !helpBindingsContain(field.KeyBinds(), "应用搜索") {
		t.Fatal("filter-entry help lacks localized apply text")
	}
	for _, msg := range []tea.KeyPressMsg{
		skillPromptKeyPress("c"),
		skillPromptKeyPress("o"),
		skillPromptKeyPress("d"),
		tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}),
	} {
		_, _ = field.Update(msg)
	}
	if !helpBindingsContain(field.KeyBinds(), "清除搜索") {
		t.Fatal("active-filter help lacks localized clear text")
	}
}

func helpBindingsContain(bindings []key.Binding, description string) bool {
	for _, binding := range bindings {
		if binding.Enabled() && binding.Help().Desc == description {
			return true
		}
	}
	return false
}

func TestSkillPromptOptionsIncludeDetectedLabelAndDefaults(t *testing.T) {
	t.Parallel()

	got, defaults := buildHuhSkillPlatformOptions(
		skillLanguageZH,
		orderedSkillPromptPlatforms([]string{"codex"}),
	)
	if len(got) != len(skillPlatforms) || !slices.Equal(defaults, []string{"codex"}) {
		t.Fatalf("options=%d defaults=%v", len(got), defaults)
	}
	if !strings.Contains(got[0].String(), "已检测") {
		t.Fatalf("detected option label = %q", got[0].String())
	}
}

func TestSkillPromptRejectsEmptyPlatforms(t *testing.T) {
	t.Parallel()

	err := validatePromptPlatforms(skillLanguageEN, nil)
	if err == nil || err.Error() != "Select at least one platform" {
		t.Fatalf("error = %v", err)
	}
}

func TestOrcaDubSkillThemeUsesCheckboxPrefixes(t *testing.T) {
	t.Parallel()

	styles := newOrcaDubSkillTheme().Theme(true)
	if got := styles.Focused.SelectedPrefix.Value(); got != "[✓] " {
		t.Fatalf("selected prefix = %q", got)
	}
	if got := styles.Focused.UnselectedPrefix.Value(); got != "[ ] " {
		t.Fatalf("unselected prefix = %q", got)
	}
}

func TestClearableSkillMultiSelectClearsWithN(t *testing.T) {
	t.Parallel()

	selected := []string{"codex"}
	multiSelect := huh.NewMultiSelect[string]().
		Options(
			huh.NewOption("Claude Code", "claude"),
			huh.NewOption("Codex", "codex").Selected(true),
		).
		Value(&selected)
	field := &clearableSkillMultiSelect{Field: multiSelect}
	field.WithKeyMap(newSkillPromptKeyMap(skillLanguageEN))
	field.Focus()

	_, _ = field.Update(skillPromptKeyPress("n"))
	if len(selected) != 0 {
		t.Fatalf("selected = %v, want empty", selected)
	}
}

func TestClearableSkillMultiSelectLeavesNToFilter(t *testing.T) {
	t.Parallel()

	selected := []string{"codex"}
	multiSelect := huh.NewMultiSelect[string]().
		Options(
			huh.NewOption("Claude Code", "claude"),
			huh.NewOption("Codex", "codex").Selected(true),
		).
		Value(&selected).
		Filterable(true)
	field := &clearableSkillMultiSelect{Field: multiSelect}
	field.WithKeyMap(newSkillPromptKeyMap(skillLanguageEN))
	field.Focus()

	_, _ = field.Update(skillPromptKeyPress("/"))
	_, _ = field.Update(tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'}))
	if !slices.Equal(selected, []string{"codex"}) {
		t.Fatalf("selected = %v, want filter typing to preserve selection", selected)
	}
}

func TestClearableSkillMultiSelectAlwaysShowsAllAndNoneHelp(t *testing.T) {
	t.Parallel()

	selected := []string{"codex"}
	multiSelect := huh.NewMultiSelect[string]().
		Options(
			huh.NewOption("Claude Code", "claude"),
			huh.NewOption("Codex", "codex"),
		).
		Value(&selected)
	field := &clearableSkillMultiSelect{Field: multiSelect}
	field.WithKeyMap(newSkillPromptKeyMap(skillLanguageEN))
	field.Focus()

	var hasAll, hasNone bool
	for _, binding := range field.KeyBinds() {
		hasAll = hasAll || (binding.Enabled() && slices.Contains(binding.Keys(), "a"))
		hasNone = hasNone || (binding.Enabled() && slices.Contains(binding.Keys(), "n"))
	}
	if !hasAll || !hasNone {
		t.Fatalf("help has all=%v none=%v", hasAll, hasNone)
	}
}

func TestSkillPlatformValidatorWaitsUntilSubmission(t *testing.T) {
	t.Parallel()

	validate := newSkillPlatformValidator(skillLanguageEN)
	if err := validate(nil); err != nil {
		t.Fatalf("initial validation = %v, want nil", err)
	}
	if err := validate(nil); err == nil || err.Error() != "Select at least one platform" {
		t.Fatalf("submit validation = %v", err)
	}
}

func TestClearableSkillMultiSelectAppliesAllKeysBeyondActiveFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		initial  []string
		expected []string
	}{
		{
			name:     "select all",
			key:      "a",
			initial:  []string{"codex"},
			expected: []string{"claude", "codex"},
		},
		{
			name:     "clear all",
			key:      "n",
			initial:  []string{"codex"},
			expected: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selected := append([]string(nil), test.initial...)
			multiSelect := huh.NewMultiSelect[string]().
				Options(
					huh.NewOption("Claude Code", "claude"),
					huh.NewOption("Codex", "codex"),
				).
				Value(&selected).
				Filterable(true)
			field := &clearableSkillMultiSelect{Field: multiSelect}
			field.WithKeyMap(newSkillPromptKeyMap(skillLanguageEN))
			field.Focus()

			for _, keyMsg := range []tea.KeyPressMsg{
				skillPromptKeyPress("/"),
				skillPromptKeyPress("c"),
				skillPromptKeyPress("l"),
				skillPromptKeyPress("a"),
				tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}),
			} {
				_, _ = field.Update(keyMsg)
			}
			_, _ = field.Update(skillPromptKeyPress(test.key))

			if !slices.Equal(selected, test.expected) {
				t.Fatalf("selected = %v, want %v", selected, test.expected)
			}
			if field.filterText != "cla" {
				t.Fatalf("filter text = %q, want preserved filter", field.filterText)
			}
		})
	}
}

func TestHuhSkillPromptRunnerSequencesRequestedFields(t *testing.T) {
	original := runSkillHuhForm
	t.Cleanup(func() {
		runSkillHuhForm = original
	})

	var calls int
	runSkillHuhForm = func(*huh.Form) error {
		calls++
		return nil
	}

	result, err := (huhSkillPromptRunner{}).Run(&skillPromptRequest{
		Language:        skillLanguageZH,
		AskLanguage:     true,
		Scope:           skillInstallProject,
		AskScope:        true,
		PlatformOptions: orderedSkillPromptPlatforms([]string{"codex"}),
		AskPlatforms:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("form runs = %d, want 3", calls)
	}
	if result.Language != skillLanguageZH || result.Scope != skillInstallProject {
		t.Fatalf("result = %+v", result)
	}
	if !slices.Equal(result.PlatformIDs, []string{"codex"}) {
		t.Fatalf("platform IDs = %v", result.PlatformIDs)
	}
}

func TestHuhSkillPromptRunnerReturnsCancellation(t *testing.T) {
	original := runSkillHuhForm
	t.Cleanup(func() {
		runSkillHuhForm = original
	})

	runSkillHuhForm = func(*huh.Form) error {
		return huh.ErrUserAborted
	}

	result, err := (huhSkillPromptRunner{}).Run(&skillPromptRequest{
		Language:    skillLanguageEN,
		AskLanguage: true,
	})
	if !errors.Is(err, huh.ErrUserAborted) {
		t.Fatalf("error = %v", err)
	}
	if result.Language != "" || result.Scope != "" || result.PlatformIDs != nil {
		t.Fatalf("result = %+v, want zero value", result)
	}
}
