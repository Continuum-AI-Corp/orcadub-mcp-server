package dub

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

type huhSkillPromptRunner struct{}

type clearableSkillMultiSelect struct {
	huh.Field
	filtering  bool
	filterText string
}

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
		lightDark := lipgloss.LightDark(isDark)
		blue := lipgloss.Color("#007BFF")
		cyan := lipgloss.Color("#53C7FF")
		normal := lightDark(lipgloss.Color("#E7F4FF"), lipgloss.Color("#18324A"))
		muted := lightDark(lipgloss.Color("#7A869A"), lipgloss.Color("#617187"))
		green := lipgloss.Color("#23C483")
		red := lipgloss.Color("#FF5D73")

		styles.Focused.Base = styles.Focused.Base.BorderForeground(blue)
		styles.Focused.Card = styles.Focused.Base
		styles.Focused.Title = styles.Focused.Title.Foreground(cyan).Bold(true)
		styles.Focused.Description = styles.Focused.Description.Foreground(muted)
		styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(cyan).SetString("❯ ")
		styles.Focused.Option = styles.Focused.Option.Foreground(normal)
		styles.Focused.MultiSelectSelector = styles.Focused.MultiSelectSelector.Foreground(cyan).SetString("❯ ")
		styles.Focused.SelectedPrefix = styles.Focused.SelectedPrefix.Foreground(green).SetString("◉ ")
		styles.Focused.UnselectedPrefix = styles.Focused.UnselectedPrefix.Foreground(muted).SetString("○ ")
		styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(green)
		styles.Focused.UnselectedOption = styles.Focused.UnselectedOption.Foreground(normal)
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
		multiSelect := huh.NewMultiSelect[string]().
			Title(skillText(result.Language, skillTextPlatformTitle)).
			Description(skillText(result.Language, skillTextPlatformDescription)).
			Options(options...).
			Value(&result.PlatformIDs).
			Height(8).
			Filterable(true).
			Validate(func(ids []string) error {
				return validatePromptPlatforms(result.Language, ids)
			})
		field := &clearableSkillMultiSelect{Field: multiSelect}
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

func (field *clearableSkillMultiSelect) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		return field.updateEmbedded(msg)
	}

	keyName := keyMsg.Keystroke()
	if field.filtering {
		model, cmd := field.updateFilterText(msg)
		switch keyName {
		case "enter", "esc":
			field.filtering = false
		case "backspace":
			runes := []rune(field.filterText)
			if len(runes) > 0 {
				field.filterText = string(runes[:len(runes)-1])
			}
		default:
			if keyMsg.Text != "" {
				field.filterText += keyMsg.Text
			}
		}
		return model, cmd
	}
	if keyName == "/" {
		model, cmd := field.updateEmbedded(msg)
		field.filtering = true
		return model, cmd
	}
	if keyName == "esc" {
		field.filterText = ""
		return field.updateEmbedded(msg)
	}
	switch keyName {
	case "a":
		return field.applySelectionToAll(false)
	case "n":
		return field.applySelectionToAll(true)
	default:
		return field.updateEmbedded(msg)
	}
}

func (field *clearableSkillMultiSelect) updateEmbedded(msg tea.Msg) (huh.Model, tea.Cmd) {
	model, cmd := field.Field.Update(msg)
	if updated, ok := model.(huh.Field); ok {
		field.Field = updated
	}
	return field, cmd
}

func (field *clearableSkillMultiSelect) updateFilterText(msg tea.Msg) (huh.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok &&
		(keyMsg.Keystroke() == "a" || keyMsg.Keystroke() == "n") {
		msg = tea.PasteMsg{Content: keyMsg.Text}
	}
	return field.updateEmbedded(msg)
}

func skillPromptKeyPress(key string) tea.KeyPressMsg {
	r := []rune(key)[0]
	return tea.KeyPressMsg(tea.Key{Text: key, Code: r})
}

func (field *clearableSkillMultiSelect) applySelectionToAll(clear bool) (huh.Model, tea.Cmd) {
	var commands []tea.Cmd
	update := func(msg tea.Msg) {
		_, cmd := field.updateEmbedded(msg)
		commands = append(commands, cmd)
	}

	activeFilter := field.filterText
	if activeFilter != "" {
		update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	}

	field.Field.KeyBinds()
	update(skillPromptKeyPress("a"))
	if clear {
		update(skillPromptKeyPress("n"))
	}

	if activeFilter != "" {
		update(skillPromptKeyPress("/"))
		for _, r := range activeFilter {
			_, cmd := field.updateFilterText(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
			commands = append(commands, cmd)
		}
		update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	return field, tea.Batch(commands...)
}
