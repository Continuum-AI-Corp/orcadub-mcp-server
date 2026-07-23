package dub

import (
	"testing"
)

func TestParseSkillLanguage(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]skillLanguage{
		"zh": skillLanguageZH,
		"en": skillLanguageEN,
	} {
		got, err := parseSkillLanguage(raw)
		if err != nil || got != want {
			t.Fatalf("parseSkillLanguage(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	if _, err := parseSkillLanguage("zh-TW"); err == nil {
		t.Fatal("expected unsupported language error")
	}
}

func TestDefaultSkillLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want skillLanguage
	}{
		{
			name: "lc-all-wins",
			env:  map[string]string{"LC_ALL": "zh_CN.UTF-8", "LANG": "en_US.UTF-8"},
			want: skillLanguageZH,
		},
		{
			name: "zh-hans",
			env:  map[string]string{"LANG": "zh-Hans"},
			want: skillLanguageZH,
		},
		{
			name: "zh-sg",
			env:  map[string]string{"LC_MESSAGES": "zh_SG.UTF-8"},
			want: skillLanguageZH,
		},
		{
			name: "traditional-falls-back",
			env:  map[string]string{"LANG": "zh_TW.UTF-8"},
			want: skillLanguageEN,
		},
		{name: "english-default", want: skillLanguageEN},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			if got := defaultSkillLanguage(getenv); got != tc.want {
				t.Fatalf("defaultSkillLanguage = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOrderedSkillPromptPlatforms(t *testing.T) {
	t.Parallel()

	got := orderedSkillPromptPlatforms([]string{"codex", "claude"})
	if got[0].ID != "claude" || got[1].ID != "codex" {
		t.Fatalf("detected prefix = %q, %q", got[0].ID, got[1].ID)
	}
	if !got[0].Detected || !got[0].Selected || !got[1].Detected || !got[1].Selected {
		t.Fatalf("detected options were not marked and selected: %#v", got[:2])
	}
	seen := map[string]bool{}
	for _, option := range got {
		if seen[option.ID] {
			t.Fatalf("duplicate option %q", option.ID)
		}
		seen[option.ID] = true
	}
	if len(got) != len(skillPlatforms) {
		t.Fatalf("options = %d, want %d", len(got), len(skillPlatforms))
	}
}
