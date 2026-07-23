package dub

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

type huhSkillPromptRunner struct{}

var runSkillHuhForm = func(form *huh.Form) error {
	return form.Run()
}

func newSkillPromptKeyMap(language skillLanguage) *huh.KeyMap {
	keys := huh.NewDefaultKeyMap()

	keys.Select.Up = key.NewBinding(
		key.WithKeys("up", "k", "ctrl+k", "ctrl+p"),
		key.WithHelp("↑", skillText(language, skillTextHelpMove)),
	)
	keys.Select.Down = key.NewBinding(
		key.WithKeys("down", "j", "ctrl+j", "ctrl+n"),
		key.WithHelp("↓", skillText(language, skillTextHelpMove)),
	)
	keys.Select.Submit = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", skillText(language, skillTextHelpConfirm)),
	)

	keys.MultiSelect.Up = key.NewBinding(
		key.WithKeys("up", "k", "ctrl+p"),
		key.WithHelp("↑", skillText(language, skillTextHelpMove)),
	)
	keys.MultiSelect.Down = key.NewBinding(
		key.WithKeys("down", "j", "ctrl+n"),
		key.WithHelp("↓", skillText(language, skillTextHelpMove)),
	)
	keys.MultiSelect.Toggle = key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", skillText(language, skillTextHelpToggle)),
	)
	keys.MultiSelect.Filter = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", skillText(language, skillTextHelpFilter)),
	)
	keys.MultiSelect.SelectAll = key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", skillText(language, skillTextHelpAll)),
	)
	keys.MultiSelect.SelectNone = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", skillText(language, skillTextHelpNone)),
		key.WithDisabled(),
	)
	keys.MultiSelect.Submit = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", skillText(language, skillTextHelpConfirm)),
	)

	return keys
}

func buildHuhSkillPlatformOptions(
	language skillLanguage,
	platforms []skillPromptPlatform,
) ([]huh.Option[string], []string) {
	options := make([]huh.Option[string], 0, len(platforms))
	defaults := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		label := fmt.Sprintf("%s  (%s)", platform.Name, platform.ID)
		if platform.Detected {
			label += "  ·  " + skillText(language, skillTextDetected)
		}
		options = append(options, huh.NewOption(label, platform.ID).Selected(platform.Selected))
		if platform.Selected {
			defaults = append(defaults, platform.ID)
		}
	}
	return options, defaults
}

func validatePromptPlatforms(language skillLanguage, platformIDs []string) error {
	if len(platformIDs) == 0 {
		return fmt.Errorf("%s", skillText(language, skillTextSelectOnePlatform))
	}
	return nil
}

func newOrcaDubSkillTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		styles := huh.ThemeBase(isDark)
		blue := lipgloss.Color("#007BFF")
		cyan := lipgloss.Color("#53C7FF")
		muted := lipgloss.Color("#7A869A")
		green := lipgloss.Color("#23C483")
		red := lipgloss.Color("#FF5D73")

		styles.Focused.Base = styles.Focused.Base.BorderForeground(blue)
		styles.Focused.Card = styles.Focused.Base
		styles.Focused.Title = styles.Focused.Title.Foreground(cyan).Bold(true)
		styles.Focused.Description = styles.Focused.Description.Foreground(muted)
		styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(cyan).SetString("❯ ")
		styles.Focused.Option = styles.Focused.Option.Foreground(lipgloss.Color("#E7F4FF"))
		styles.Focused.MultiSelectSelector = styles.Focused.MultiSelectSelector.Foreground(cyan).SetString("❯ ")
		styles.Focused.SelectedPrefix = styles.Focused.SelectedPrefix.Foreground(green).SetString("◉ ")
		styles.Focused.UnselectedPrefix = styles.Focused.UnselectedPrefix.Foreground(muted).SetString("○ ")
		styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(green)
		styles.Focused.UnselectedOption = styles.Focused.UnselectedOption.Foreground(lipgloss.Color("#E7F4FF"))
		styles.Focused.ErrorIndicator = styles.Focused.ErrorIndicator.Foreground(red)
		styles.Focused.ErrorMessage = styles.Focused.ErrorMessage.Foreground(red)

		styles.Blurred = styles.Focused
		styles.Blurred.Base = styles.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
		styles.Blurred.Card = styles.Blurred.Base
		styles.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
		styles.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ")

		styles.Group.Title = styles.Focused.Title
		styles.Group.Description = styles.Focused.Description
		return styles
	})
}

func (huhSkillPromptRunner) Run(request skillPromptRequest) (skillPromptResult, error) {
	result := skillPromptResult{
		Language: request.Language,
		Scope:    request.Scope,
	}
	if result.Language == "" {
		result.Language = skillLanguageEN
	}
	if result.Scope == "" {
		result.Scope = skillInstallProject
	}

	if request.AskLanguage {
		field := huh.NewSelect[skillLanguage]().
			Title(skillText(result.Language, skillTextLanguageTitle)).
			Options(
				huh.NewOption("简体中文", skillLanguageZH),
				huh.NewOption("English", skillLanguageEN),
			).
			Value(&result.Language)
		if err := runSkillPromptForm(field, request, result.Language); err != nil {
			return skillPromptResult{}, err
		}
	}

	if request.AskScope {
		field := huh.NewSelect[skillInstallScope]().
			Title(skillText(result.Language, skillTextScopeTitle)).
			Description(skillText(result.Language, skillTextScopeDescription)).
			Options(
				huh.NewOption(skillText(result.Language, skillTextScopeProject), skillInstallProject),
				huh.NewOption(skillText(result.Language, skillTextScopeGlobal), skillInstallGlobal),
			).
			Value(&result.Scope)
		if err := runSkillPromptForm(field, request, result.Language); err != nil {
			return skillPromptResult{}, err
		}
	}

	if request.AskPlatforms {
		options, defaults := buildHuhSkillPlatformOptions(result.Language, request.PlatformOptions)
		result.PlatformIDs = defaults
		field := huh.NewMultiSelect[string]().
			Title(skillText(result.Language, skillTextPlatformTitle)).
			Description(skillText(result.Language, skillTextPlatformDescription)).
			Options(options...).
			Value(&result.PlatformIDs).
			Height(8).
			Filterable(true).
			Validate(func(ids []string) error {
				return validatePromptPlatforms(result.Language, ids)
			})
		if err := runSkillPromptForm(field, request, result.Language); err != nil {
			return skillPromptResult{}, err
		}
	} else {
		for _, platform := range request.PlatformOptions {
			if platform.Selected {
				result.PlatformIDs = append(result.PlatformIDs, platform.ID)
			}
		}
	}

	return result, nil
}

func runSkillPromptForm(
	field huh.Field,
	request skillPromptRequest,
	language skillLanguage,
) error {
	form := huh.NewForm(huh.NewGroup(field)).
		WithTheme(newOrcaDubSkillTheme()).
		WithKeyMap(newSkillPromptKeyMap(language))
	if request.Input != nil {
		form = form.WithInput(request.Input)
	}
	if request.Output != nil {
		form = form.WithOutput(request.Output)
	}
	return runSkillHuhForm(form)
}

func normalizePromptPlatformIDs(ids []string) []string {
	normalized := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	return normalized
}
