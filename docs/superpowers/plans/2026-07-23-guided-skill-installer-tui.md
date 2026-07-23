# Guided Skill Installer TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the line-oriented Skill installer prompts with a branded, bilingual, arrow-key TUI while preserving every non-interactive installation contract.

**Architecture:** Keep installation and filesystem safety in `skill_installer.go`. Add pure prompt data, localization, and banner components, then adapt them to `charm.land/huh/v2`. `cli.go` decides between TUI and non-interactive paths and injects prompt/terminal dependencies for deterministic tests.

**Tech Stack:** Go 1.25, `charm.land/huh/v2@v2.0.3`, `golang.org/x/term@v0.45.0`, Go standard testing, existing Makefile/golangci-lint.

## Global Constraints

- Interactive languages are exactly Simplified Chinese (`zh`) and English (`en`).
- The selected Digital Orca banner appears only when a real TUI is entered.
- Project/Global uses arrow-key single selection.
- Platform selection is filterable, eight rows high, detected-first, and checkbox-based.
- Existing platform paths and the 33-entry catalog remain unchanged.
- `--yes`, explicit targets, `--force`, and `--json` retain existing behavior.
- JSON must contain no ANSI, banner, or localized prose.
- Non-TTY incomplete invocation must fail instead of blocking.
- Everything committed to this public repository is written in English.
- Implementation uses TDD: every behavior test must fail for the intended reason before production code is added.

---

### Task 1: Pure prompt model, localization, and banner

**Files:**
- Create: `internal/skill_prompt.go`
- Create: `internal/skill_prompt_test.go`
- Create: `internal/skill_i18n.go`
- Create: `internal/skill_i18n_test.go`
- Create: `internal/skill_banner.go`
- Create: `internal/skill_banner_test.go`

**Interfaces:**
- Consumes: `skillPlatform`, `skillInstallScope`, and `skillPlatforms` from `internal/skill_installer.go`.
- Produces:
  - `type skillLanguage string`
  - `const skillLanguageZH`, `skillLanguageEN`
  - `parseSkillLanguage(string) (skillLanguage, error)`
  - `defaultSkillLanguage(getenv func(string) string) skillLanguage`
  - `orderedSkillPromptPlatforms([]string) []skillPromptPlatform`
  - `skillText(skillLanguage, skillTextKey) string`
  - `renderSkillBanner(io.Writer, bool)`

- [ ] **Step 1: Add failing language and ordering tests**

Create `internal/skill_prompt_test.go` with these cases:

```go
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
		{"lc-all-wins", map[string]string{"LC_ALL": "zh_CN.UTF-8", "LANG": "en_US.UTF-8"}, skillLanguageZH},
		{"zh-hans", map[string]string{"LANG": "zh-Hans"}, skillLanguageZH},
		{"zh-sg", map[string]string{"LC_MESSAGES": "zh_SG.UTF-8"}, skillLanguageZH},
		{"traditional-falls-back", map[string]string{"LANG": "zh_TW.UTF-8"}, skillLanguageEN},
		{"english-default", nil, skillLanguageEN},
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
```

- [ ] **Step 2: Run the prompt tests and verify RED**

Run:

```bash
go test ./internal -run 'Test(ParseSkillLanguage|DefaultSkillLanguage|OrderedSkillPromptPlatforms)$' -count=1
```

Expected: compilation fails because `skillLanguage`, parsing, locale detection,
and prompt ordering do not exist.

- [ ] **Step 3: Implement the pure prompt model**

Create `internal/skill_prompt.go` with:

```go
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
	"claude", "codex", "cursor", "github-copilot", "gemini", "opencode", "windsurf",
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
	appendUnique := func(id string) {
		if _, ok := findSkillPlatform(id); !ok {
			return
		}
		for _, existing := range order {
			if existing == id {
				return
			}
		}
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
			ID: id, Name: platform.Name, Detected: detectedSet[id], Selected: detectedSet[id],
		})
	}
	return options
}
```

- [ ] **Step 4: Run prompt tests and verify GREEN**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 5: Add failing translation completeness tests**

Create `internal/skill_i18n_test.go`:

```go
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
```

- [ ] **Step 6: Verify translation tests fail**

Run:

```bash
go test ./internal -run '^TestSkillTranslation' -count=1
```

Expected: compilation fails because the keys and dictionary do not exist.

- [ ] **Step 7: Implement the typed translation dictionary**

Create `internal/skill_i18n.go`. Define contiguous `skillTextKey` constants for:
language title, scope title/description/options, platform title/description,
detected suffix, platform-empty validation, move/toggle/filter/all/none/confirm
help, result title/scope/source, five statuses, conflict guidance, non-TTY
guidance, download/install failure prefixes.

Use:

```go
type skillTextKey int

const (
	skillTextLanguageTitle skillTextKey = iota
	skillTextScopeTitle
	skillTextScopeDescription
	skillTextScopeProject
	skillTextScopeGlobal
	skillTextPlatformTitle
	skillTextPlatformDescription
	skillTextDetected
	skillTextSelectOnePlatform
	skillTextHelpMove
	skillTextHelpToggle
	skillTextHelpFilter
	skillTextHelpAll
	skillTextHelpNone
	skillTextHelpConfirm
	skillTextResultTitle
	skillTextResultScope
	skillTextResultSource
	skillTextStatusInstalled
	skillTextStatusUpdated
	skillTextStatusUnchanged
	skillTextStatusConflict
	skillTextStatusError
	skillTextConflictGuidance
	skillTextNonTTYGuidance
)

var skillTranslations = map[skillLanguage]map[skillTextKey]string{
	skillLanguageEN: {
		skillTextLanguageTitle: "Language / 语言",
		skillTextScopeTitle: "Install scope",
		skillTextScopeDescription: "Choose where OrcaDub should install the Skill",
		skillTextScopeProject: "Project — current directory",
		skillTextScopeGlobal: "Global — home directory",
		skillTextPlatformTitle: "Select platforms",
		skillTextPlatformDescription: "Detected platforms are preselected",
		skillTextDetected: "detected",
		skillTextSelectOnePlatform: "Select at least one platform",
		skillTextHelpMove: "move",
		skillTextHelpToggle: "toggle",
		skillTextHelpFilter: "filter",
		skillTextHelpAll: "all",
		skillTextHelpNone: "none",
		skillTextHelpConfirm: "confirm",
		skillTextResultTitle: "OrcaDub Skill installation",
		skillTextResultScope: "Scope",
		skillTextResultSource: "Source",
		skillTextStatusInstalled: "installed",
		skillTextStatusUpdated: "updated",
		skillTextStatusUnchanged: "unchanged",
		skillTextStatusConflict: "conflict",
		skillTextStatusError: "error",
		skillTextConflictGuidance: "kept existing file; rerun with --force",
		skillTextNonTTYGuidance: "interactive input requires a terminal; use --lang, --scope, --platform, or --yes",
	},
	skillLanguageZH: {
		skillTextLanguageTitle: "Language / 语言",
		skillTextScopeTitle: "安装范围",
		skillTextScopeDescription: "选择 OrcaDub Skill 的安装位置",
		skillTextScopeProject: "当前项目",
		skillTextScopeGlobal: "全局安装",
		skillTextPlatformTitle: "选择安装平台",
		skillTextPlatformDescription: "已检测到的平台已自动勾选",
		skillTextDetected: "已检测",
		skillTextSelectOnePlatform: "请至少选择一个平台",
		skillTextHelpMove: "移动",
		skillTextHelpToggle: "勾选",
		skillTextHelpFilter: "搜索",
		skillTextHelpAll: "全选",
		skillTextHelpNone: "清空",
		skillTextHelpConfirm: "确认",
		skillTextResultTitle: "OrcaDub Skill 安装结果",
		skillTextResultScope: "安装范围",
		skillTextResultSource: "来源",
		skillTextStatusInstalled: "已安装",
		skillTextStatusUpdated: "已更新",
		skillTextStatusUnchanged: "无需更新",
		skillTextStatusConflict: "冲突",
		skillTextStatusError: "错误",
		skillTextConflictGuidance: "已保留现有文件；使用 --force 重新运行可覆盖",
		skillTextNonTTYGuidance: "交互式安装需要终端；请使用 --lang、--scope、--platform 或 --yes",
	},
}

func skillText(language skillLanguage, key skillTextKey) string {
	if text := skillTranslations[language][key]; text != "" {
		return text
	}
	return skillTranslations[skillLanguageEN][key]
}
```

- [ ] **Step 8: Run translation tests and verify GREEN**

Run the command from Step 6.

Expected: PASS.

- [ ] **Step 9: Add failing banner tests**

Create `internal/skill_banner_test.go`:

```go
func TestRenderSkillBannerPlain(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	renderSkillBanner(&out, false)
	got := out.String()
	if !strings.Contains(got, "ORCA//DUB") ||
		!strings.Contains(got, "SKILL INSTALLER / 技能安装器") {
		t.Fatalf("banner = %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain banner contains ANSI: %q", got)
	}
}

func TestRenderSkillBannerColor(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	renderSkillBanner(&out, true)
	if got := out.String(); !strings.Contains(got, "\x1b[") || !strings.Contains(got, "ORCA//DUB") {
		t.Fatalf("colored banner = %q", got)
	}
}
```

- [ ] **Step 10: Verify banner tests fail**

Run:

```bash
go test ./internal -run '^TestRenderSkillBanner' -count=1
```

Expected: compilation fails because `renderSkillBanner` does not exist.

- [ ] **Step 11: Implement plain and colored Digital Orca output**

Create `internal/skill_banner.go` with one raw banner constant and
`renderSkillBanner`. Color only the orca/wordmark segments with explicit ANSI
blue `38;2;0;123;255` and cyan `38;2;83;199;255`, always ending with reset
`\x1b[0m`. The plain branch writes the same five-line glyph plus bilingual
subtitle with no escape sequences.

- [ ] **Step 12: Run all Task 1 tests**

Run:

```bash
go test ./internal -run 'Test(ParseSkillLanguage|DefaultSkillLanguage|OrderedSkillPromptPlatforms|SkillTranslation|RenderSkillBanner)' -count=1
```

Expected: PASS.

- [ ] **Step 13: Commit Task 1**

```bash
git add internal/skill_prompt.go internal/skill_prompt_test.go internal/skill_i18n.go internal/skill_i18n_test.go internal/skill_banner.go internal/skill_banner_test.go
git commit -m "feat: add Skill installer prompt foundations"
```

---

### Task 2: Huh-based scope and platform selectors

**Files:**
- Create: `internal/skill_prompt_huh.go`
- Create: `internal/skill_prompt_huh_test.go`

**Interfaces:**
- Consumes: language, translations, ordered platform options from Task 1.
- Produces:
  - `skillPromptRequest`
  - `skillPromptResult`
  - `skillPromptRunner`
  - `huhSkillPromptRunner`
  - `newOrcaDubSkillTheme`
  - `newSkillPromptKeyMap`

- [ ] **Step 1: Add the failing prompt-adapter tests**

Create `internal/skill_prompt_huh_test.go` with tests that assert:

```go
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
	if err := validatePromptPlatforms(skillLanguageEN, nil); err == nil ||
		err.Error() != "Select at least one platform" {
		t.Fatalf("error = %v", err)
	}
}
```

- [ ] **Step 2: Verify prompt-adapter tests fail**

Run:

```bash
go test ./internal -run '^Test(NewSkillPrompt|SkillPrompt)' -count=1
```

Expected: compilation fails because the adapter functions do not exist.

- [ ] **Step 3: Add the TUI dependency**

Run:

```bash
go get charm.land/huh/v2@v2.0.3
go mod tidy
```

Expected: `go.mod` directly requires `charm.land/huh/v2 v2.0.3`; `go.sum`
contains its verified module hashes.

- [ ] **Step 4: Implement request/result and runner boundaries**

Add these types to `internal/skill_prompt.go`:

```go
type skillPromptRequest struct {
	Language        skillLanguage
	AskLanguage     bool
	Scope           skillInstallScope
	AskScope        bool
	PlatformOptions []skillPromptPlatform
	AskPlatforms    bool
	Input           io.Reader
	Output          io.Writer
}

type skillPromptResult struct {
	Language    skillLanguage
	Scope       skillInstallScope
	PlatformIDs []string
}

type skillPromptRunner interface {
	Run(skillPromptRequest) (skillPromptResult, error)
}
```

- [ ] **Step 5: Implement the Huh adapter**

Create `internal/skill_prompt_huh.go`:

- Build one `huh.Select[skillLanguage]` for language when `AskLanguage`.
- Build one localized `huh.Select[skillInstallScope]` for scope when
  `AskScope`.
- Build one `huh.MultiSelect[string]` for platforms when `AskPlatforms`.
- Run each field in sequence through a one-field `huh.Form` so language choice
  localizes every later field.
- Apply `.Height(8)`, `.Filterable(true)`, `.Validate(...)`, `.Value(...)`.
- Supply request input/output with `.WithInput` and `.WithOutput`.
- Return `huh.ErrUserAborted` unchanged for CLI cancellation mapping.

Construct labels as `"<Name>  ·  <detected text>"` only for detected options.
Use `huh.NewOption(label, id).Selected(option.Selected)`.

Create `newSkillPromptKeyMap` from `huh.NewDefaultKeyMap()` and replace:

```go
keys.MultiSelect.SelectAll = key.NewBinding(
	key.WithKeys("a"),
	key.WithHelp("a", skillText(language, skillTextHelpAll)),
)
keys.MultiSelect.SelectNone = key.NewBinding(
	key.WithKeys("n"),
	key.WithHelp("n", skillText(language, skillTextHelpNone)),
)
keys.MultiSelect.Toggle = key.NewBinding(
	key.WithKeys("space"),
	key.WithHelp("space", skillText(language, skillTextHelpToggle)),
)
```

Localize the help text on Up, Down, Filter, ClearFilter, and Submit bindings.

Create `newOrcaDubSkillTheme` with `huh.ThemeBase`. Set:

- Focused title/selector to `#53C7FF`.
- Selected prefix to `[✓] ` in `#35D3A7`.
- Unselected prefix to `[ ] ` in muted gray.
- Error message/indicator to `#FF5C7A`.
- Description/help to muted blue-gray.
- Focused left border to OrcaDub blue.

- [ ] **Step 6: Run prompt-adapter tests and verify GREEN**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 7: Add cancellation and form-sequencing tests**

Introduce a package-level form-run function inside the adapter:

```go
var runSkillHuhForm = func(form *huh.Form) error { return form.Run() }
```

Add these non-parallel tests:

```go
func TestHuhSkillPromptRunnerRunsRequestedFieldsInOrder(t *testing.T) {
	oldRun := runSkillHuhForm
	t.Cleanup(func() { runSkillHuhForm = oldRun })
	calls := 0
	runSkillHuhForm = func(_ *huh.Form) error {
		calls++
		return nil
	}
	runner := huhSkillPromptRunner{}
	result, err := runner.Run(skillPromptRequest{
		Language: skillLanguageEN,
		AskLanguage: true,
		Scope: skillInstallProject,
		AskScope: true,
		PlatformOptions: []skillPromptPlatform{
			{ID: "codex", Name: "Codex", Detected: true, Selected: true},
		},
		AskPlatforms: true,
		Input: strings.NewReader(""),
		Output: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("form calls = %d, want language + scope + platforms", calls)
	}
	if result.Language != skillLanguageEN ||
		result.Scope != skillInstallProject ||
		!slices.Equal(result.PlatformIDs, []string{"codex"}) {
		t.Fatalf("result = %#v", result)
	}
}

func TestHuhSkillPromptRunnerReturnsCancellation(t *testing.T) {
	oldRun := runSkillHuhForm
	t.Cleanup(func() { runSkillHuhForm = oldRun })
	runSkillHuhForm = func(_ *huh.Form) error { return huh.ErrUserAborted }
	result, err := (huhSkillPromptRunner{}).Run(skillPromptRequest{
		Language: skillLanguageEN,
		AskLanguage: true,
		Input: strings.NewReader(""),
		Output: io.Discard,
	})
	if !errors.Is(err, huh.ErrUserAborted) {
		t.Fatalf("error = %v", err)
	}
	if result != (skillPromptResult{}) {
		t.Fatalf("result after cancellation = %#v", result)
	}
}
```

- [ ] **Step 8: Verify sequencing tests fail, implement the injection, and verify GREEN**

Run:

```bash
go test ./internal -run '^TestHuhSkillPromptRunner' -count=1
```

Expected RED: runner lacks injectable form execution.

After adding the injection seam and minimal logic, expected GREEN: PASS.

- [ ] **Step 9: Commit Task 2**

```bash
git add go.mod go.sum internal/skill_prompt.go internal/skill_prompt_huh.go internal/skill_prompt_huh_test.go
git commit -m "feat: add interactive Skill installer selectors"
```

---

### Task 3: Integrate TUI, language flag, terminal detection, and localized output

**Files:**
- Modify: `internal/cli.go`
- Modify: `internal/cli_test.go`

**Interfaces:**
- Consumes: `skillPromptRunner`, language helpers, banner, translations, and
  the existing installer.
- Produces:
  - `--lang zh|en`
  - TTY mode decision
  - cancellation exit code `130`
  - localized human report

- [ ] **Step 1: Add failing CLI option and JSON-isolation tests**

Extend `internal/cli_test.go`:

```go
func TestParseSkillCLIOptionsLanguage(t *testing.T) {
	t.Parallel()
	options, err := parseSkillCLIOptions([]string{"--lang", "zh"})
	if err != nil || options.languageValue != "zh" {
		t.Fatalf("options=%#v err=%v", options, err)
	}
	if _, err := parseSkillCLIOptions([]string{"--lang", "fr"}); err == nil {
		t.Fatal("expected invalid language error")
	}
}

func TestRunCLISkillInstallJSONHasNoPresentation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)
	restore := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		t.TempDir(),
		t.TempDir(),
	)
	t.Cleanup(restore)
	out := captureStdout(t, func() int {
		return RunCLI([]string{
			"skill", "install",
			"--platform", "codex",
			"--scope", "project",
			"--json",
		})
	})
	if out.code != 0 || !json.Valid([]byte(out.out)) {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if strings.Contains(out.out, "ORCA//DUB") || strings.Contains(out.out, "\x1b[") {
		t.Fatalf("JSON contains presentation output: %q", out.out)
	}
}
```

- [ ] **Step 2: Verify CLI option tests fail**

Run:

```bash
go test ./internal -run 'Test(ParseSkillCLIOptionsLanguage|RunCLISkillInstallJSONHasNoPresentation)$' -count=1
```

Expected: compile/assertion failure because `--lang` is not parsed.

- [ ] **Step 3: Parse and validate `--lang`**

Add `languageValue string` to `skillCLIOptions`. Register:

```go
flags.StringVar(&options.languageValue, "lang", "", "guidance language: zh or en")
```

After parsing, call `parseSkillLanguage` when non-empty and return its error.
Update `skillInstallUsage` and the top-level usage string.

- [ ] **Step 4: Run option tests and verify GREEN**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 5: Add failing orchestration tests with a fake prompt**

Define:

```go
type fakeSkillPromptRunner struct {
	request skillPromptRequest
	result  skillPromptResult
	err     error
	calls   int
}

func (f *fakeSkillPromptRunner) Run(request skillPromptRequest) (skillPromptResult, error) {
	f.calls++
	f.request = request
	return f.result, f.err
}
```

Add tests:

- A no-argument terminal invocation calls the prompt once, asks language,
  scope, and platforms, emits the banner, and installs the fake-selected
  Codex target.
- `--scope global` skips scope but still asks language/platform.
- Explicit `--platform codex` does not call the prompt and defaults Project.
- `--yes` and `--json` do not call the prompt or emit the banner.
- `huh.ErrUserAborted` returns `130` and never contacts the Skill HTTP server.
- Non-TTY no-argument invocation returns `2` and contains flag guidance.

Inject:

```go
var (
	skillCLIPromptRunner = func() skillPromptRunner { return huhSkillPromptRunner{} }
	skillCLIIsTerminal   = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}
	skillCLIGetenv = os.Getenv
)
```

Extend `setSkillCLIRuntimeForTest` to save and restore these globals.

- [ ] **Step 6: Verify orchestration tests fail**

Run:

```bash
go test ./internal -run 'TestRunCLISkillInstall(TUI|Cancellation|NonTTY|Explicit|Yes)' -count=1
```

Expected: failures because `runSkillCLI` still reads line-oriented input.

- [ ] **Step 7: Add terminal detection dependency**

Run:

```bash
go get golang.org/x/term@v0.45.0
go mod tidy
```

- [ ] **Step 8: Replace line prompts with prompt orchestration**

In `runSkillCLI`:

1. Determine language from `--lang` or `defaultSkillLanguage(skillCLIGetenv)`.
2. Preserve the existing rule that explicit platforms make the command
   non-interactive and default missing scope to Project.
3. Treat `--yes` and `--json` as fully non-interactive.
4. Resolve the current directory before platform prompting so detection can
   populate defaults.
5. If interactive fields remain and `skillCLIIsTerminal()` is false, print
   localized non-TTY guidance and return `2`.
6. Render the banner once.
7. Call the prompt runner with flags determining `AskLanguage`, `AskScope`, and
   `AskPlatforms`.
8. Map `errors.Is(err, huh.ErrUserAborted)` to `130`.
9. Pass confirmed scope/platform IDs into the unchanged installer.

Delete:

- `skillCLIInput`
- `selectSkillInstallScope`
- `selectSkillInstallPlatforms`
- `promptSkillInstallPlatforms`
- `parseSkillPlatformSelection`
- the now-unused `bufio` import

Retain:

- `allSkillPlatformIDs`
- `validateSkillPlatformIDs`

- [ ] **Step 9: Run orchestration tests and verify GREEN**

Run the command from Step 6.

Expected: PASS.

- [ ] **Step 10: Add failing localized report tests**

Create a table test around a refactored:

```go
func renderSkillInstallReport(report skillInstallReport, language skillLanguage)
```

Assert:

- English contains `OrcaDub Skill installation`.
- Simplified Chinese contains `OrcaDub Skill 安装结果`.
- Conflict output in both languages contains `--force`.
- Paths and platform product names are unchanged.
- Machine status constants in `skillInstallReport` remain unchanged.

- [ ] **Step 11: Verify localized report tests fail, implement, and verify GREEN**

Run:

```bash
go test ./internal -run '^TestRenderSkillInstallReportLocalized$' -count=1
```

Expected RED: renderer has no language parameter.

Implement language-specific labels through `skillText`, then rerun.

Expected GREEN: PASS.

- [ ] **Step 12: Run all internal tests and race tests**

Run:

```bash
go test ./internal -count=1
go test -race ./internal -count=1
```

Expected: PASS with zero race reports.

- [ ] **Step 13: Commit Task 3**

```bash
git add go.mod go.sum internal/cli.go internal/cli_test.go
git commit -m "feat: integrate branded bilingual Skill installer"
```

---

### Task 4: Documentation, full verification, and release readiness

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`

**Interfaces:**
- Consumes: all CLI behavior from Tasks 1–3.
- Produces: public usage documentation and verified release artifacts.

- [ ] **Step 1: Add the new interactive usage documentation**

Update the README Skill installation section to document:

```bash
npx -y @orcadub/cli skill install
```

Describe:

- Digital Orca banner.
- Simplified Chinese/English selection.
- Arrow-key Project/Global selection.
- Space-toggle platform checkboxes.
- `/` filter, `a` select all, `n` clear, Enter confirm.
- `--lang zh|en`.
- Non-interactive examples remain valid.

Update `AGENTS.md` layout/development notes to mention the Huh prompt adapter,
installer translations, and the requirement that JSON mode remain decoration
free.

- [ ] **Step 2: Run formatting and complete repository checks**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  make check
```

Expected:

- `golangci-lint`: `0 issues`.
- Go tests: PASS.
- `go vet`: exit `0`.
- `go build`: exit `0`.
- Node syntax checks: exit `0`.

- [ ] **Step 3: Run race, prompt-specific, and diff checks**

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  go test -race ./... -count=1

go test ./internal -run 'Skill(Prompt|Translation|Banner|Install)' -count=1
git diff --check
```

Expected: all commands exit `0`.

- [ ] **Step 4: Cross-build all published targets**

Use a temporary output directory and run:

```bash
GOOS=darwin  GOARCH=amd64 go build -o /tmp/orcadub-darwin-amd64 ./cmd
GOOS=darwin  GOARCH=arm64 go build -o /tmp/orcadub-darwin-arm64 ./cmd
GOOS=linux   GOARCH=amd64 go build -o /tmp/orcadub-linux-amd64 ./cmd
GOOS=linux   GOARCH=arm64 go build -o /tmp/orcadub-linux-arm64 ./cmd
GOOS=windows GOARCH=amd64 go build -o /tmp/orcadub-windows-amd64.exe ./cmd
GOOS=windows GOARCH=arm64 go build -o /tmp/orcadub-windows-arm64.exe ./cmd
```

Expected: six exit codes of `0`.

- [ ] **Step 5: Run a real TTY smoke test**

Build:

```bash
go build -o /tmp/orcadub-tui ./cmd
/tmp/orcadub-tui skill install
```

In a real terminal verify:

1. Digital Orca appears.
2. Language arrow selection works.
3. Scope arrow selection works.
4. Detected platforms are checked and first.
5. Space, `/`, `a`, `n`, and Enter work.
6. `Ctrl+C` exits `130` before installation.

Use a temporary project directory so the smoke test does not modify the
repository.

- [ ] **Step 6: Commit documentation and final verification state**

```bash
git add README.md AGENTS.md
git commit -m "docs: explain the interactive Skill installer"
```

- [ ] **Step 7: Final review**

Run:

```bash
git status --short --branch
git log --oneline -5
```

Expected: clean feature worktree with the design commit plus four
single-purpose implementation commits.
