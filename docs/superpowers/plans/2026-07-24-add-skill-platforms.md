# Additional Skill Platforms Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Hermes, OpenClaw, and Command Code to the OrcaDub guided Skill installer.

**Architecture:** Extend the existing `skillPlatforms` data catalog so every
existing installer path consumes the new platforms automatically. Preserve
conservative detection with explicit markers and executable metadata, then
verify the catalog, resolved paths, and false-positive boundaries in unit
tests.

**Tech Stack:** Go 1.25.8, Bubble Tea/Huh terminal UI, Go standard library tests,
Markdown documentation.

## Global Constraints

- Preserve all 33 existing platform definitions and ordering.
- Append `hermes`, `openclaw`, and `command-code` in that order.
- Install OpenClaw project Skills at the workspace `skills/` root.
- Never detect Command Code through the `cmd` executable.
- Never treat every project as OpenClaw merely because its platform root is `.`.
- Keep repository content in English.

---

### Task 1: Platform Catalog and Install Targets

**Files:**
- Modify: `internal/skill_installer_test.go`
- Modify: `internal/skill_installer.go`

**Interfaces:**
- Consumes: `skillPlatform`, `findSkillPlatform`, and `resolveSkillTargets`.
- Produces: catalog entries for IDs `hermes`, `openclaw`, and `command-code`.

- [ ] **Step 1: Write failing catalog and target tests**

Append the three IDs to the catalog expectation, assert their project/global
roots, and add target cases with these exact paths:

```text
<project>/.hermes/skills/orcadub/SKILL.md
<home>/.hermes/skills/orcadub/SKILL.md
<project>/skills/orcadub/SKILL.md
<home>/.openclaw/skills/orcadub/SKILL.md
<project>/.commandcode/skills/orcadub/SKILL.md
<home>/.commandcode/skills/orcadub/SKILL.md
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
go test ./internal -run 'TestSkillPlatforms|TestResolveSkillTargets' -count=1
```

Expected: failure because the new platform IDs are absent.

- [ ] **Step 3: Add the platform definitions**

Append:

```go
{
	ID:                   "hermes",
	Name:                 "Hermes",
	ProjectRoot:          ".hermes",
	GlobalRoot:           ".hermes",
	GlobalDetectionPaths: []string{".hermes"},
	Executables:          []string{"hermes"},
},
{
	ID:                   "openclaw",
	Name:                 "OpenClaw",
	ProjectRoot:          ".",
	GlobalRoot:           ".openclaw",
	DetectionPaths:       []string{},
	GlobalDetectionPaths: []string{".openclaw"},
	Executables:          []string{"openclaw"},
},
{
	ID:                   "command-code",
	Name:                 "Command Code",
	ProjectRoot:          ".commandcode",
	GlobalRoot:           ".commandcode",
	GlobalDetectionPaths: []string{".commandcode"},
},
```

Update the catalog comment so it describes the Comet-compatible base plus
additional Agent Skills hosts.

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run:

```bash
go test ./internal -run 'TestSkillPlatforms|TestResolveSkillTargets' -count=1
```

Expected: PASS.

### Task 2: Conservative Detection

**Files:**
- Modify: `internal/skill_installer_test.go`

**Interfaces:**
- Consumes: `detectSkillPlatforms(projectDir, homeDir, lookPath) []string`.
- Produces: regression coverage for the three new platform detection policies.

- [ ] **Step 1: Add detection scenarios**

Cover:

```text
.hermes project marker -> hermes
.openclaw global marker -> openclaw
.commandcode project marker -> command-code
hermes executable -> hermes
openclaw executable -> openclaw
cmd executable only -> not command-code
empty ordinary project -> not openclaw
```

- [ ] **Step 2: Run the focused detection tests**

Run:

```bash
go test ./internal -run TestDetectSkillPlatforms -count=1
```

Expected: PASS because the platform metadata added in Task 1 encodes these
behaviors.

### Task 3: Documentation and Full Verification

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: the final 36-platform catalog.
- Produces: accurate installer documentation.

- [ ] **Step 1: Update the README**

Replace the 33-platform statement with:

```text
It supports 36 platforms: the 33-platform Comet-compatible catalog plus
Hermes, OpenClaw, and Command Code.
```

- [ ] **Step 2: Format and run the complete check**

Run:

```bash
goimports -w .
gofmt -w .
make check
```

Expected: lint reports `0 issues`, all Go tests pass, vet and builds succeed,
and both Node syntax checks succeed.

- [ ] **Step 3: Review the final diff**

Run:

```bash
git diff --check
git diff --stat
git status --short
```

Expected: only the design, plan, installer catalog/tests, and README are
changed; `git diff --check` prints no errors.
