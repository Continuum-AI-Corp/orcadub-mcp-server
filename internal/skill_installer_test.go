package dub

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
)

const validDubVideoSkill = `---
name: dub-video
description: Test OrcaDub Skill.
---

# Dub video
`

func TestSkillPlatformsMatchCometCatalog(t *testing.T) {
	t.Parallel()

	wantIDs := []string{
		"claude",
		"cursor",
		"codex",
		"opencode",
		"windsurf",
		"cline",
		"roocode",
		"continue",
		"github-copilot",
		"gemini",
		"amazon-q",
		"qwen",
		"kilocode",
		"auggie",
		"kimicode",
		"kiro",
		"lingma",
		"junie",
		"codebuddy",
		"costrict",
		"crush",
		"factory",
		"iflow",
		"pi",
		"qoder",
		"antigravity",
		"antigravity2",
		"bob",
		"forgecode",
		"trae",
		"trae-cn",
		"zcode",
		"mimocode",
	}
	gotIDs := make([]string, 0, len(skillPlatforms))
	for _, platform := range skillPlatforms {
		gotIDs = append(gotIDs, platform.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("platform IDs:\n got: %v\nwant: %v", gotIDs, wantIDs)
	}

	cases := []struct {
		id          string
		projectRoot string
		globalRoot  string
	}{
		{id: "claude", projectRoot: ".claude", globalRoot: ".claude"},
		{id: "codex", projectRoot: ".agents", globalRoot: ".agents"},
		{id: "opencode", projectRoot: ".opencode", globalRoot: ".config/opencode"},
		{id: "pi", projectRoot: ".pi", globalRoot: ".pi/agent"},
		{id: "antigravity", projectRoot: ".agents", globalRoot: ".gemini/antigravity"},
		{id: "antigravity2", projectRoot: ".agents", globalRoot: ".gemini/config"},
		{id: "mimocode", projectRoot: ".mimocode", globalRoot: ".config/mimocode"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			platform, ok := findSkillPlatform(tc.id)
			if !ok {
				t.Fatalf("platform %q not found", tc.id)
			}
			if platform.ProjectRoot != tc.projectRoot || platform.GlobalRoot != tc.globalRoot {
				t.Fatalf(
					"%s roots = (%q, %q), want (%q, %q)",
					tc.id,
					platform.ProjectRoot,
					platform.GlobalRoot,
					tc.projectRoot,
					tc.globalRoot,
				)
			}
		})
	}
}

func TestCanonicalSkillSourceURL(t *testing.T) {
	t.Parallel()

	const want = "https://raw.githubusercontent.com/Continuum-AI-Corp/orcadub-plugin/main/skills/dub-video/SKILL.md"
	if defaultSkillSourceURL != want {
		t.Fatalf("canonical Skill URL = %q, want %q", defaultSkillSourceURL, want)
	}
}

func TestDetectSkillPlatformsUsesPlatformMarkers(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	for _, marker := range []string{".claude", ".codex", ".github/copilot-instructions.md"} {
		path := filepath.Join(projectDir, filepath.FromSlash(marker))
		if filepath.Ext(path) != "" {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("marker"), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := detectSkillPlatforms(projectDir, t.TempDir(), func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	want := []string{"claude", "codex", "github-copilot"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("detected platforms = %v, want %v", got, want)
	}
}

func TestDetectSkillPlatformsUsesGlobalMarkers(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestDetectSkillPlatformsIgnoresSharedProjectAgentsDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := detectSkillPlatforms(projectDir, t.TempDir(), func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	for _, id := range got {
		if id == "codex" || id == "antigravity" || id == "antigravity2" {
			t.Fatalf("shared project .agents directory falsely detected %q", id)
		}
	}
}

func TestDetectSkillPlatformsDeduplicatesSignals(t *testing.T) {
	t.Parallel()

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

func TestResolveSkillTargetsDeduplicatesSharedPaths(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	homeDir := t.TempDir()
	targets, err := resolveSkillTargets(
		[]string{"codex", "antigravity"},
		skillInstallProject,
		projectDir,
		homeDir,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("target count = %d, want 1: %#v", len(targets), targets)
	}
	wantPath := filepath.Join(projectDir, ".agents", "skills", "orcadub", "SKILL.md")
	if targets[0].Path != wantPath {
		t.Fatalf("target path = %q, want %q", targets[0].Path, wantPath)
	}
	if want := []string{"codex", "antigravity"}; !reflect.DeepEqual(targets[0].PlatformIDs, want) {
		t.Fatalf("platform IDs = %v, want %v", targets[0].PlatformIDs, want)
	}
}

func TestResolveSkillTargetsUsesGlobalRoot(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	homeDir := t.TempDir()
	targets, err := resolveSkillTargets(
		[]string{"opencode"},
		skillInstallGlobal,
		projectDir,
		homeDir,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(homeDir, ".config", "opencode", "skills", "orcadub", "SKILL.md")
	if len(targets) != 1 || targets[0].Path != wantPath {
		t.Fatalf("targets = %#v, want one target at %q", targets, wantPath)
	}
}

func TestResolveSkillTargetsRejectsUnknownPlatform(t *testing.T) {
	t.Parallel()

	_, err := resolveSkillTargets(
		[]string{"not-a-platform"},
		skillInstallProject,
		t.TempDir(),
		t.TempDir(),
	)
	if err == nil {
		t.Fatal("expected unknown platform error")
	}
}

func TestResolveSkillTargetsRejectsRelativeBase(t *testing.T) {
	t.Parallel()

	_, err := resolveSkillTargets(
		[]string{"claude"},
		skillInstallGlobal,
		t.TempDir(),
		"relative-home",
	)
	if err == nil {
		t.Fatal("expected relative global base error")
	}
}

func TestSkillInstallerDownloadsOnceAndInstallsSelectedTargets(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(server.Close)

	projectDir := t.TempDir()
	installer := skillInstaller{client: server.Client(), sourceURL: server.URL}
	report, err := installer.install(
		context.Background(),
		[]string{"claude", "codex"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, want 1", requests.Load())
	}
	if len(report.Results) != 2 {
		t.Fatalf("results = %#v, want 2", report.Results)
	}
	for _, root := range []string{".claude", ".agents"} {
		path := filepath.Join(projectDir, root, "skills", "orcadub", "SKILL.md")
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		if string(got) != validDubVideoSkill {
			t.Fatalf("%s content = %q, want canonical Skill", path, got)
		}
	}
	for _, result := range report.Results {
		if result.Status != skillInstallStatusInstalled {
			t.Fatalf("result = %#v, want installed", result)
		}
	}
}

func TestSkillInstallerRejectsInvalidDocumentBeforeWriting(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not a Skill</html>"))
	}))
	t.Cleanup(server.Close)

	projectDir := t.TempDir()
	installer := skillInstaller{client: server.Client(), sourceURL: server.URL}
	_, err := installer.install(
		context.Background(),
		[]string{"claude"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		false,
	)
	if err == nil {
		t.Fatal("expected invalid Skill error")
	}
	destination := filepath.Join(projectDir, ".claude", "skills", "orcadub", "SKILL.md")
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination must not exist after validation failure; stat err = %v", statErr)
	}
}

func TestValidateSkillDocumentRejectsWrongNameAndOversize(t *testing.T) {
	t.Parallel()

	wrongName := []byte("---\nname: another-skill\n---\n")
	if err := validateSkillDocument(wrongName); err == nil {
		t.Fatal("expected wrong Skill name error")
	}
	oversize := make([]byte, maxSkillDocumentSize+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(oversize)
	}))
	t.Cleanup(server.Close)
	installer := skillInstaller{client: server.Client(), sourceURL: server.URL}
	if _, err := installer.downloadSkill(context.Background()); err == nil {
		t.Fatal("expected oversized Skill error")
	}
}

func TestSkillInstallerReportsUnchangedAndProtectsConflicts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(server.Close)

	projectDir := t.TempDir()
	destination := filepath.Join(projectDir, ".claude", "skills", "orcadub", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	installer := skillInstaller{client: server.Client(), sourceURL: server.URL}

	if err := os.WriteFile(destination, []byte(validDubVideoSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := installer.install(
		context.Background(),
		[]string{"claude"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Results[0].Status; got != skillInstallStatusUnchanged {
		t.Fatalf("identical status = %q, want %q", got, skillInstallStatusUnchanged)
	}

	const localSkill = "local customization\n"
	if err := os.WriteFile(destination, []byte(localSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = installer.install(
		context.Background(),
		[]string{"claude"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Results[0].Status; got != skillInstallStatusConflict {
		t.Fatalf("protected status = %q, want %q", got, skillInstallStatusConflict)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != localSkill {
		t.Fatalf("conflict overwrote existing content: %q", got)
	}

	report, err = installer.install(
		context.Background(),
		[]string{"claude"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Results[0].Status; got != skillInstallStatusUpdated {
		t.Fatalf("forced status = %q, want %q", got, skillInstallStatusUpdated)
	}
	got, err = os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != validDubVideoSkill {
		t.Fatalf("forced content = %q, want canonical Skill", got)
	}
}

func TestSkillInstallerRejectsHTTPFailureBeforeWriting(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	projectDir := t.TempDir()
	installer := skillInstaller{client: server.Client(), sourceURL: server.URL}
	_, err := installer.install(
		context.Background(),
		[]string{"claude"},
		skillInstallProject,
		projectDir,
		t.TempDir(),
		false,
	)
	if err == nil {
		t.Fatal("expected HTTP status error")
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, ".claude")); !os.IsNotExist(statErr) {
		t.Fatalf("install root must not exist after download failure; stat err = %v", statErr)
	}
}

func TestInstallSkillDocumentDoesNotClobberConcurrentCreate(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "skills", "orcadub", "SKILL.md")
	const concurrentContent = "created by another process\n"

	oldLink := skillLinkFile
	skillLinkFile = func(oldPath, newPath string) error {
		if err := os.WriteFile(newPath, []byte(concurrentContent), 0o644); err != nil {
			return err
		}
		return os.Link(oldPath, newPath)
	}
	t.Cleanup(func() { skillLinkFile = oldLink })

	status, err := installSkillDocument(destination, []byte(validDubVideoSkill), false)
	if err != nil {
		t.Fatal(err)
	}
	if status != skillInstallStatusConflict {
		t.Fatalf("status = %q, want conflict", status)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != concurrentContent {
		t.Fatalf("concurrent file was overwritten: %q", got)
	}
}

func TestInstallSkillDocumentPreservesExistingWhenReplaceFails(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "skills", "orcadub", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	const existingContent = "existing customized Skill\n"
	if err := os.WriteFile(destination, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	oldRename := skillRenameFile
	skillRenameFile = func(_, _ string) error { return errors.New("injected rename failure") }
	t.Cleanup(func() { skillRenameFile = oldRename })

	status, err := installSkillDocument(destination, []byte(validDubVideoSkill), true)
	if err == nil {
		t.Fatal("expected replacement failure")
	}
	if status != skillInstallStatusError {
		t.Fatalf("status = %q, want error", status)
	}
	got, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != existingContent {
		t.Fatalf("existing file was lost after replacement failure: %q", got)
	}
}
