# Guided Skill Installer TUI Design

**Status:** Ready for user review

**Date:** 2026-07-23

**Repository:** `Continuum-AI-Corp/orcadub-mcp-server`

## Summary

Replace the current line-oriented OrcaDub Skill installer prompts with a
branded terminal interface. The installer will show a digital orca banner,
offer Simplified Chinese or English guidance, use an arrow-key selector for
installation scope, and provide a filterable checkbox list for the 33 supported
agent platforms.

The change applies only to interactive Skill installation. Explicit and
machine-readable invocations retain their current behavior and output
contracts.

## Goals

- Make `orcadub skill install` feel like a polished OrcaDub installer.
- Show a recognizable digital-orca terminal mark during interactive installs.
- Let users select Project or Global scope with arrow keys.
- Let users select one or more platforms with checkboxes.
- Put detected platforms first and preselect them.
- Support Simplified Chinese and English guidance.
- Preserve all existing non-interactive flags and JSON output.
- Keep the existing safe file-installation logic unchanged.
- Continue supporting macOS, Linux, and Windows on amd64 and arm64.

## Non-goals

- Adding Traditional Chinese.
- Changing the 33-platform catalog or any installation path.
- Changing Skill contents or dubbing behavior.
- Adding API-key collection to the installer.
- Replacing the existing npm launcher or release mechanism.
- Building a general-purpose localization framework for other CLI commands.

## Confirmed Visual Direction

The interactive installer uses the selected “Digital Orca” direction:

```text
          ▄▄▖
     ▄██▀▀███▙
   ▄██▛  ▄████▌   ORCA//DUB
   ▀███▄███▀▀     AI DUBBING CLI
      ▀█▀    ◌━━━━━━━━◌
```

The banner uses OrcaDub blue and cyan when color is available. It renders as
plain text when color is disabled or unsupported. Before language selection,
its subtitle is bilingual:

```text
SKILL INSTALLER / 技能安装器
```

After the user chooses a language, all installer-owned guidance uses only that
language.

## User Experience

### Full interactive flow

Running the installer without explicit targets starts the complete wizard:

```text
orcadub skill install

Digital Orca banner
        ↓
Language / 语言
  › 简体中文
    English
        ↓
Install scope / 安装范围
  › Project
    Global
        ↓
Platform checkbox list
  › [✓] Detected platform
    [ ] Other platform
        ↓
Download and install
        ↓
Localized result summary
```

### Language selection

The language field contains exactly two choices:

- `简体中文`
- `English`

The initial selection is derived in this order:

1. `--lang zh|en`, when supplied.
2. System locale.
3. English.

System locale lookup checks `LC_ALL`, `LC_MESSAGES`, then `LANG`. The following
locale families default to Simplified Chinese:

- `zh`
- `zh_CN`
- `zh_SG`
- `zh-Hans`

Matching is case-insensitive and treats `_` and `-` as equivalent separators.
Traditional Chinese locales such as `zh_TW`, `zh_HK`, and `zh-Hant` default to
English because Traditional Chinese is not supported. Users can still select
Simplified Chinese manually.

When `--lang` is present, the language screen is skipped. An unsupported value
returns exit code `2` and an English usage error because a valid output
language has not yet been selected.

### Scope selection

The scope page is a single-select field:

```text
安装范围

  › 当前项目
    全局安装

  ↑/↓ 移动 • Enter 确认
```

English labels are:

- `Project — current directory`
- `Global — home directory`

Project remains the default to preserve existing behavior. If `--scope` is
supplied, this page is skipped.

### Platform selection

The platform page is a fixed-height, filterable multi-select:

```text
选择安装平台
已检测到的平台已自动勾选

  › [✓] Codex                  已检测
    [✓] Claude Code            已检测
    [ ] Cursor
    [ ] Gemini CLI
    [ ] GitHub Copilot
    [ ] OpenCode
    [ ] Windsurf
    ↓ 还有 26 项 • / 搜索

  ↑/↓ 移动 • Space 勾选 • a 全选 • n 清空 • Enter 确认
```

The visible list height is eight rows. The list scrolls when more rows exist.
The exact ordering is:

1. Detected platforms, preserving their platform-catalog order.
2. Undetected commonly used platforms in this order:
   `claude`, `codex`, `cursor`, `github-copilot`, `gemini`, `opencode`,
   `windsurf`.
3. Every remaining platform in existing platform-catalog order.

Detected platforms are selected by default. If no platforms are detected, the
list starts with no selection. The user must select at least one platform.

Key bindings:

- `↑` / `↓` or `j` / `k`: move.
- `Space`: toggle the highlighted platform.
- `/`: enter filter mode.
- `Esc`: clear the active filter before cancelling the form.
- `a`: select every catalog option, regardless of the active filter.
- `n`: clear all selections.
- `Enter`: confirm.
- `Ctrl+C`: cancel the entire installer.

Filtering searches both the stable platform ID and display name,
case-insensitively. Selections remain checked while hidden by a filter.

Submitting an empty selection keeps the user on the platform page and displays
a localized validation message.

### Installation and result summary

No filesystem mutation occurs until all interactive fields have been
confirmed. After confirmation, the existing installer downloads and validates
the canonical Skill document and writes it using the current atomic,
no-clobber behavior.

The interactive summary is localized and shows:

- Selected scope.
- Platform display names.
- Destination paths.
- Status for every destination: installed, updated, unchanged, conflict, or
  error.
- `--force` guidance for conflicts.

Low-level errors retain their original detail after a localized contextual
prefix. The installer never asks for or prints `ORCADUB_API_KEY`; Skill
installation remains keyless.

## Terminal Behavior

### Interactive detection

The TUI is used only when both input and output are terminals and at least one
interactive field remains.

If input or output is not a terminal and required choices are missing, the
command returns exit code `2` with actionable guidance to provide
`--lang`, `--scope`, `--platform`, or `--yes`. It must never block waiting for
input in CI, an agent subprocess, or a pipe.

### Color and terminal capability

Color is disabled when any of these conditions is true:

- `NO_COLOR` is present.
- `TERM=dumb`.
- Standard output is not a terminal.

Disabling color removes ANSI sequences only; the banner and layout remain
readable. Terminal resize events are delegated to the TUI library. The platform
field retains an eight-row target height but may shrink when the terminal is
short.

### Cancellation

`Ctrl+C` returns exit code `130`. Cancellation before confirmation performs no
download and no file write. Once the form has been confirmed, installation
keeps the existing process-signal behavior and atomic-write guarantees.

## CLI Compatibility

Add one flag:

```text
--lang zh|en    guidance and human-readable output language
```

Existing flags retain their meanings:

```text
--platform <id>          repeatable explicit platform
--scope project|global   explicit destination scope
--yes                    accept detected/default targets
--force                  replace a different existing OrcaDub Skill
--json                   emit a structured report
```

### Mode matrix

| Invocation shape | Behavior |
| --- | --- |
| No selection flags | Full language, scope, and platform TUI |
| `--lang` only | Skip language; prompt for scope and platforms |
| `--scope` only | Prompt for language and platforms |
| Explicit `--platform` with no scope | Preserve current Project default; no TUI |
| Explicit `--platform` and `--scope` | No TUI |
| `--yes` | No TUI; detected platforms or all when none are detected |
| `--json` | No TUI, banner, localized prose, or ANSI |

`--json` remains byte-for-byte valid JSON framing. It does not include the
banner, progress text, or localized status labels. Stable JSON status values
remain English identifiers.

The banner is displayed only when the TUI is actually entered. It is not shown
for `--yes`, `--json`, or fully explicit installation.

## Architecture

Use `charm.land/huh/v2` version `v2.0.3` for terminal fields. It supplies
single-select, multi-select, filtering, fixed height, validation, key maps, and
theme hooks. The OrcaDub layer owns ordering, defaults, localization, and
colors.

### Components

#### `internal/skill_prompt.go`

Contains library-independent prompt data:

- `skillLanguage` with `zh` and `en`.
- Locale normalization and default-language selection.
- Platform option ordering and detected/default-selected state.
- Prompt request/result structs.
- The prompt-runner interface used by CLI orchestration.

This file must not import `huh` or terminal styling packages.

#### `internal/skill_prompt_huh.go`

Implements the prompt-runner interface using `huh`:

- Builds only fields not already answered by flags.
- Configures arrow-key scope selection.
- Configures the eight-row filterable platform multi-select.
- Applies detected defaults.
- Applies the custom `a` and `n` bindings.
- Maps cancellation to the internal cancellation sentinel.
- Applies the OrcaDub theme.

The implementation receives input/output handles and terminal capability data
instead of reading global standard streams directly.

#### `internal/skill_i18n.go`

Owns installer-specific Simplified Chinese and English strings. Messages are
addressed by typed keys rather than scattered conditionals. Product and
platform names remain untranslated.

The dictionary covers:

- Field titles and descriptions.
- Key-help labels.
- Validation messages.
- TTY guidance errors.
- Result summary headings and status labels.
- Conflict and force-overwrite guidance.

Tests require every key to exist in both languages.

#### `internal/skill_banner.go`

Owns the Digital Orca text and presentation:

- Bilingual pre-language subtitle.
- ANSI blue/cyan version.
- Plain version.
- Width-safe rendering.
- Output writer injection for tests.

#### `internal/cli.go`

Retains command parsing and installation orchestration. The current
line-oriented scope/platform prompt functions are removed after their behavior
is covered by new tests.

The flow becomes:

1. Parse and validate flags, including `--lang`.
2. Determine which values are explicit.
3. Choose TUI or non-interactive mode.
4. In TUI mode, render banner and run the prompt adapter.
5. Resolve base directories and detected platforms.
6. Call the existing `skillInstaller.install`.
7. Render JSON or localized human output.

#### `internal/skill_installer.go`

No behavior change. Platform paths, download validation, conflict handling,
atomic writes, forced replacement, and report structs remain the source of
truth.

## Data Flow

```text
CLI flags + environment locale + terminal capability
                         │
                         ▼
             prompt request preparation
                         │
             ┌───────────┴───────────┐
             │                       │
             ▼                       ▼
       interactive TUI        non-interactive defaults
             │                       │
             └───────────┬───────────┘
                         ▼
            language + scope + platform IDs
                         │
                         ▼
             existing target resolution
                         │
                         ▼
              existing safe installer
                         │
                         ▼
        JSON report or localized human summary
```

## Error Handling

- Invalid `--lang`, `--scope`, or `--platform`: exit `2`; no TUI or write.
- Missing TTY with unanswered choices: exit `2`; print flag guidance.
- Empty platform selection: remain in form with localized validation.
- User cancellation: exit `130`; no install starts.
- Skill download or validation failure: exit `1`; localized prefix plus source
  error.
- Destination conflict: retain the existing file, report conflict, exit `1`.
- Per-target write error: report all results and exit `1`.
- JSON-mode failures continue to avoid decorative output.

## Testing Strategy

Implementation follows test-driven development.

### Pure unit tests

- Locale normalization and precedence.
- Simplified Chinese locale recognition.
- Traditional Chinese fallback to English.
- Complete translation-key coverage.
- Detected-first and popular-platform ordering.
- Detected default selections.
- Banner plain output.
- Banner ANSI output and `NO_COLOR` behavior.

### Prompt model tests

Drive the `huh` form model with terminal key messages rather than a real
terminal:

- Language arrow selection.
- Scope arrow selection.
- Platform Space toggle.
- Select-all and clear-all keys.
- Filtering and selection persistence.
- Empty-selection validation.
- Enter confirmation.
- `Ctrl+C` cancellation.

### CLI integration tests

- Full interactive orchestration through an injected fake prompt runner.
- Explicit-platform behavior remains non-interactive.
- `--yes` detected/all behavior remains unchanged.
- `--json` contains no ANSI, banner, or prose.
- `--lang zh` and `--lang en` localized summaries.
- Invalid language exits before installer construction.
- Non-TTY incomplete invocation exits instead of blocking.
- Cancellation does not contact the Skill source server or create files.

### Existing safety and release tests

Continue running:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `golangci-lint run ./...`
- `go build ./...`
- Node launcher syntax checks.
- Cross-compilation for darwin/linux/windows on amd64/arm64.
- npm package dry-pack and public-package smoke tests at release time.

## Dependency and Distribution Impact

`charm.land/huh/v2@v2.0.3` and its transitive Charmbracelet dependencies are
compiled into the existing Go binary. Users do not install a separate UI
package. The npm package continues downloading the matching native binary in
`postinstall`.

The additional binary size is accepted in exchange for cross-platform terminal
input, filtering, theming, resize handling, and maintained key behavior.

## Acceptance Criteria

The feature is complete when:

1. `orcadub skill install` shows the Digital Orca banner in a real terminal.
2. The user can choose Simplified Chinese or English.
3. Scope uses arrow-key single selection.
4. Platforms use a filterable checkbox list with detected defaults.
5. The confirmed key bindings behave as documented.
6. Cancelling before confirmation writes nothing.
7. Non-TTY incomplete commands fail fast with flag guidance.
8. `--yes`, explicit flags, and `--json` remain compatible.
9. JSON output contains no ANSI or human guidance.
10. Existing installation safety tests still pass.
11. The complete verification suite and six target cross-builds pass.
