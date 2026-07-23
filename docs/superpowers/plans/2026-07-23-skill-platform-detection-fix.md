# Skill Platform Detection Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detect installed Claude Code and Codex environments outside platform-marked projects and preselect them during Skill installation.

**Architecture:** Add explicit global-marker and executable metadata to the platform catalog. Pass project, home, and `LookPath` inputs into one best-effort detector shared by interactive and non-interactive CLI flows.

**Tech Stack:** Go 1.25, standard library `os`, `os/exec`, and `path/filepath`; existing Go test suite and fake prompt runner.

## Global Constraints

- Preserve all existing project-marker detection behavior.
- Claude Code detection signals are `.claude` and `claude`.
- Codex detection signals are `.codex` and `codex`.
- Never use `.agents` as a Codex detection marker.
- Detection failures are non-fatal and detected IDs retain catalog order.
- Repository code, tests, documentation, and commits remain in English.

---

### Task 1: Expand the Detector Without False Positives

**Files:**
- Modify: `internal/skill_installer.go`
- Test: `internal/skill_installer_test.go`

**Interfaces:**
- Consumes: project path, optional home path, and a `func(string) (string, error)` executable lookup.
- Produces: `detectSkillPlatforms(projectDir, homeDir string, lookPath func(string) (string, error)) []string`.

- [ ] **Step 1: Write failing detector regression tests**

Add focused tests that call the desired detector signature:

```go
func TestDetectSkillPlatformsUsesGlobalMarkers(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	for _, marker := range []string{".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(homeDir, marker), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := detectSkillPlatforms(projectDir, homeDir, func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	if want := []string{"claude", "codex"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("detected platforms = %v, want %v", got, want)
	}
}

func TestDetectSkillPlatformsUsesExecutables(t *testing.T) {
	installed := map[string]bool{"claude": true, "codex": true}
	got := detectSkillPlatforms(t.TempDir(), t.TempDir(), func(name string) (string, error) {
		if installed[name] {
			return filepath.Join("/bin", name), nil
		}
		return "", exec.ErrNotFound
	})
	if want := []string{"claude", "codex"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("detected platforms = %v, want %v", got, want)
	}
}

func TestDetectSkillPlatformsIgnoresSharedAgentsDirectory(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(homeDir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectSkillPlatforms(projectDir, homeDir, func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	for _, id := range got {
		if id == "codex" || id == "antigravity" {
			t.Fatalf("shared .agents directory falsely detected %q", id)
		}
	}
}

func TestDetectSkillPlatformsDeduplicatesSignals(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectSkillPlatforms(projectDir, t.TempDir(), func(name string) (string, error) {
		if name == "codex" {
			return "/bin/codex", nil
		}
		return "", exec.ErrNotFound
	})
	if want := []string{"codex"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("detected platforms = %v, want %v", got, want)
	}
}
```

Update the existing project-marker test to pass an empty home directory and an
always-missing executable lookup.

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./internal -run 'TestDetectSkillPlatforms' -count=1
```

Expected: compilation fails because `detectSkillPlatforms` still accepts only
the project directory.

- [ ] **Step 3: Implement minimal detector metadata and logic**

Add the fields:

```go
type skillPlatform struct {
	ID                   string
	Name                 string
	ProjectRoot          string
	GlobalRoot           string
	DetectionPaths       []string
	GlobalDetectionPaths []string
	Executables          []string
}
```

Configure only the confirmed signals:

```go
{
	ID:                   "claude",
	Name:                 "Claude Code",
	ProjectRoot:          ".claude",
	GlobalRoot:           ".claude",
	GlobalDetectionPaths: []string{".claude"},
	Executables:          []string{"claude"},
},
{
	ID:                   "codex",
	Name:                 "Codex",
	ProjectRoot:          ".agents",
	GlobalRoot:           ".agents",
	DetectionPaths:       []string{".codex"},
	GlobalDetectionPaths: []string{".codex"},
	Executables:          []string{"codex"},
},
```

Replace the detector with:

```go
func detectSkillPlatforms(
	projectDir string,
	homeDir string,
	lookPath func(string) (string, error),
) []string {
	detected := make([]string, 0)
platformLoop:
	for _, platform := range skillPlatforms {
		projectPaths := platform.DetectionPaths
		if len(projectPaths) == 0 {
			projectPaths = []string{platform.ProjectRoot}
		}
		if projectDir != "" {
			for _, marker := range projectPaths {
				if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(marker))); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
		if homeDir != "" {
			for _, marker := range platform.GlobalDetectionPaths {
				if _, err := os.Stat(filepath.Join(homeDir, filepath.FromSlash(marker))); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
		if lookPath != nil {
			for _, executable := range platform.Executables {
				if _, err := lookPath(executable); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
	}
	return detected
}
```

- [ ] **Step 4: Run focused tests and verify GREEN**

Run the command from Step 2.

Expected: all `TestDetectSkillPlatforms*` tests pass.

- [ ] **Step 5: Commit the detector**

```bash
git add internal/skill_installer.go internal/skill_installer_test.go
git commit -m "fix: detect installed Skill platforms"
```

### Task 2: Wire Environment Detection Into Both CLI Flows

**Files:**
- Modify: `internal/cli.go`
- Test: `internal/cli_test.go`

**Interfaces:**
- Consumes: existing `skillCLIWorkingDir`, `skillCLIHomeDir`, and new injectable `skillCLILookPath`.
- Produces: `detectSkillCLIPlatforms(projectDir string) []string`, used by interactive and non-interactive selection.

- [ ] **Step 1: Write failing orchestration regression tests**

Extend `setSkillCLIRuntimeForTest` to save and restore the intended
`skillCLILookPath` dependency, then add:

```go
func TestRunCLISkillInstallTUIDetectsInstalledCLIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		t.TempDir(),
		t.TempDir(),
	)
	t.Cleanup(restoreCLI)
	skillCLILookPath = func(name string) (string, error) {
		if name == "claude" || name == "codex" {
			return filepath.Join("/bin", name), nil
		}
		return "", exec.ErrNotFound
	}

	prompt := &fakeSkillPromptRunner{result: skillPromptResult{
		Language:    skillLanguageEN,
		Scope:       skillInstallProject,
		PlatformIDs: []string{"claude", "codex"},
	}}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, true, "en_US.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install"})
	})
	if out.code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}

	detected := map[string]bool{}
	for _, option := range prompt.request.PlatformOptions {
		if option.Detected && option.Selected {
			detected[option.ID] = true
		}
	}
	if !detected["claude"] || !detected["codex"] {
		t.Fatalf("detected options = %v, want claude and codex", detected)
	}
}

func TestResolveNonInteractiveSkillSelectionUsesGlobalDetection(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	for _, marker := range []string{".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(homeDir, marker), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	oldWorkingDir := skillCLIWorkingDir
	oldHomeDir := skillCLIHomeDir
	oldLookPath := skillCLILookPath
	t.Cleanup(func() {
		skillCLIWorkingDir = oldWorkingDir
		skillCLIHomeDir = oldHomeDir
		skillCLILookPath = oldLookPath
	})
	skillCLIWorkingDir = func() (string, error) { return projectDir, nil }
	skillCLIHomeDir = func() (string, error) { return homeDir, nil }
	skillCLILookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	result, exitCode, err := resolveNonInteractiveSkillSelection(
		skillLanguageEN,
		skillInstallProject,
	)
	if err != nil || exitCode != 0 {
		t.Fatalf("exit=%d err=%v", exitCode, err)
	}
	if want := []string{"claude", "codex"}; !reflect.DeepEqual(result.PlatformIDs, want) {
		t.Fatalf("selected platforms = %v, want %v", result.PlatformIDs, want)
	}
}

func TestSkillCLIDetectionIgnoresHomeLookupFailure(t *testing.T) {
	oldWorkingDir := skillCLIWorkingDir
	oldHomeDir := skillCLIHomeDir
	oldLookPath := skillCLILookPath
	t.Cleanup(func() {
		skillCLIWorkingDir = oldWorkingDir
		skillCLIHomeDir = oldHomeDir
		skillCLILookPath = oldLookPath
	})
	skillCLIWorkingDir = func() (string, error) { return t.TempDir(), nil }
	skillCLIHomeDir = func() (string, error) {
		return "", errors.New("injected home lookup failure")
	}
	skillCLILookPath = func(name string) (string, error) {
		if name == "codex" {
			return "/bin/codex", nil
		}
		return "", exec.ErrNotFound
	}

	result, exitCode, err := resolveNonInteractiveSkillSelection(
		skillLanguageEN,
		skillInstallProject,
	)
	if err != nil || exitCode != 0 {
		t.Fatalf("exit=%d err=%v", exitCode, err)
	}
	if want := []string{"codex"}; !reflect.DeepEqual(result.PlatformIDs, want) {
		t.Fatalf("selected platforms = %v, want %v", result.PlatformIDs, want)
	}
}
```

The assertions inspect real prompt request values and installer reports rather
than testing only the injected lookup function.

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./internal -run 'TestRunCLISkillInstallTUIDetectsInstalledCLIs|TestResolveNonInteractiveSkillSelectionUsesGlobalDetection|TestSkillCLIDetectionIgnoresHomeLookupFailure' -count=1
```

Expected: tests fail because CLI selection still passes only the current
project directory to the detector.

- [ ] **Step 3: Implement shared CLI detection**

Add:

```go
skillCLILookPath = exec.LookPath
```

and:

```go
func detectSkillCLIPlatforms(projectDir string) []string {
	homeDir, _ := skillCLIHomeDir()
	return detectSkillPlatforms(projectDir, homeDir, skillCLILookPath)
}
```

Use `detectSkillCLIPlatforms` from both
`resolveNonInteractiveSkillSelection` and
`resolveInteractiveSkillSelection`. Keep global installation target resolution
unchanged so a required home-directory error is still reported.

- [ ] **Step 4: Run focused tests and verify GREEN**

Run the command from Step 2.

Expected: all three orchestration regression tests pass.

- [ ] **Step 5: Run complete verification**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  gofmt -w internal/skill_installer.go internal/skill_installer_test.go internal/cli.go internal/cli_test.go
test -z "$(gofmt -l internal cmd)"
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./... -count=1
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go vet ./...
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go build ./...
node --check npm/install.js
node --check npm/bin/orcadub.js
git diff --check
```

Expected: every command exits with status 0 and `gofmt -l` reports no changed
Go files.

- [ ] **Step 6: Run a real terminal smoke test**

Build the binary, enter a temporary directory without `.claude` or `.codex`,
start `orcadub skill install`, and verify the platform page labels both Claude
Code and Codex as detected and preselects both. Abort before writing Skills.

- [ ] **Step 7: Commit CLI integration**

```bash
git add internal/cli.go internal/cli_test.go
git commit -m "fix: preselect installed Skill platforms"
```
