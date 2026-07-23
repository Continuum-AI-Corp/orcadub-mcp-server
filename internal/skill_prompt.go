package dub

import (
	"fmt"
	"strings"
)

type skillLanguage string

const (
	skillLanguageZH skillLanguage = "zh"
	skillLanguageEN skillLanguage = "en"
)

type skillPromptPlatform struct {
	ID       string
	Name     string
	Detected bool
	Selected bool
}

var popularSkillPlatformIDs = []string{
	"claude",
	"codex",
	"cursor",
	"github-copilot",
	"gemini",
	"opencode",
	"windsurf",
}

func parseSkillLanguage(raw string) (skillLanguage, error) {
	switch skillLanguage(strings.ToLower(strings.TrimSpace(raw))) {
	case skillLanguageZH:
		return skillLanguageZH, nil
	case skillLanguageEN:
		return skillLanguageEN, nil
	default:
		return "", fmt.Errorf("unknown guidance language %q (use zh or en)", raw)
	}
}

func defaultSkillLanguage(getenv func(string) string) skillLanguage {
	locale := ""
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale = strings.TrimSpace(getenv(key)); locale != "" {
			break
		}
	}
	locale = strings.ToLower(strings.ReplaceAll(strings.Split(locale, ".")[0], "_", "-"))
	switch locale {
	case "zh", "zh-cn", "zh-sg", "zh-hans":
		return skillLanguageZH
	default:
		return skillLanguageEN
	}
}

func orderedSkillPromptPlatforms(detected []string) []skillPromptPlatform {
	detectedSet := make(map[string]bool, len(detected))
	for _, id := range detected {
		detectedSet[id] = true
	}

	order := make([]string, 0, len(skillPlatforms))
	seen := make(map[string]bool, len(skillPlatforms))
	appendUnique := func(id string) {
		if seen[id] {
			return
		}
		if _, ok := findSkillPlatform(id); !ok {
			return
		}
		seen[id] = true
		order = append(order, id)
	}
	for _, platform := range skillPlatforms {
		if detectedSet[platform.ID] {
			appendUnique(platform.ID)
		}
	}
	for _, id := range popularSkillPlatformIDs {
		appendUnique(id)
	}
	for _, platform := range skillPlatforms {
		appendUnique(platform.ID)
	}

	options := make([]skillPromptPlatform, 0, len(order))
	for _, id := range order {
		platform, _ := findSkillPlatform(id)
		options = append(options, skillPromptPlatform{
			ID:       id,
			Name:     platform.Name,
			Detected: detectedSet[id],
			Selected: detectedSet[id],
		})
	}
	return options
}
