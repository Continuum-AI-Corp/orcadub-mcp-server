package dub

import (
	"strings"
	"testing"
)

func TestSkillTranslationsAreComplete(t *testing.T) {
	t.Parallel()

	for _, language := range []skillLanguage{skillLanguageZH, skillLanguageEN} {
		for key := skillTextLanguageTitle; key <= skillTextNonTTYGuidance; key++ {
			if got := skillText(language, key); strings.TrimSpace(got) == "" {
				t.Errorf("missing translation language=%q key=%d", language, key)
			}
		}
	}
}

func TestSkillTranslationFallbackIsEnglish(t *testing.T) {
	t.Parallel()

	if got := skillText("bad", skillTextScopeTitle); got != "Install scope" {
		t.Fatalf("fallback = %q", got)
	}
}
