package dub

import (
	"errors"
	"slices"
	"strings"
	"testing"

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

	result, err := (huhSkillPromptRunner{}).Run(skillPromptRequest{
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

	result, err := (huhSkillPromptRunner{}).Run(skillPromptRequest{
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
