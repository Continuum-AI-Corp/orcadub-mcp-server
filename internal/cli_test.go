package dub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"charm.land/huh/v2"
)

func TestApplyCreateOpts(t *testing.T) {
	var in CreateInput
	err := applyCreateOpts(&in, []string{
		"preserve_bgm=true",
		"watermark=false",
		"resolution=1080p",
		"profile=podcast",
		"glossary.OrcaDub=虎鲸配音",
		"glossary.Foo=Bar",
		"speaker_assignments.SPEAKER_00=char-1",
	})
	if err != nil {
		t.Fatalf("applyCreateOpts: %v", err)
	}
	if in.PreserveBGM == nil || *in.PreserveBGM != true {
		t.Errorf("preserve_bgm = %v, want *true", in.PreserveBGM)
	}
	if in.Watermark == nil || *in.Watermark != false {
		t.Errorf("watermark = %v, want *false", in.Watermark)
	}
	if in.Resolution != "1080p" {
		t.Errorf("resolution = %q", in.Resolution)
	}
	if in.Profile != "podcast" {
		t.Errorf("profile = %q", in.Profile)
	}
	if in.Glossary["OrcaDub"] != "虎鲸配音" || in.Glossary["Foo"] != "Bar" {
		t.Errorf("glossary = %v", in.Glossary)
	}
	if in.SpeakerAssignments["SPEAKER_00"] != "char-1" {
		t.Errorf("speaker_assignments = %v", in.SpeakerAssignments)
	}
}

// TestRunCLIHealthAndGet points RunCLI at a fake gateway by overriding the
// base+origin URLs through the env-driven client, capturing stdout. It
// mirrors the client_test.go httptest pattern.
func TestRunCLIHealthAndGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/videos" && r.Method == http.MethodPost:
			// health probe: dub service validation error
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"invalid_request","message":"source_lang is required"}`))
		case r.URL.Path == "/v1/videos/job-1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"job-1","object":"video","status":"completed","progress":100,"job_id":"uuid-9"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ORCADUB_API_KEY", "sk-test")
	t.Setenv("ORCADUB_BASE_URL", srv.URL)

	out := captureStdout(t, func() int { return RunCLI([]string{"health"}) })
	if out.code != 0 {
		t.Fatalf("health exit = %d, stderr=%s", out.code, out.err)
	}
	var health map[string]any
	if err := json.Unmarshal([]byte(out.out), &health); err != nil {
		t.Fatalf("health stdout not JSON: %q", out.out)
	}
	if health["status"] != "ok" {
		t.Errorf("health status = %v", health["status"])
	}

	out = captureStdout(t, func() int { return RunCLI([]string{"get", "--video-id", "job-1"}) })
	if out.code != 0 {
		t.Fatalf("get exit = %d, stderr=%s", out.code, out.err)
	}
	if !strings.Contains(out.out, `"content_url"`) || !strings.Contains(out.out, "completed") {
		t.Errorf("get stdout = %s", out.out)
	}
}

func TestRunCLIUnknownSubcommand(t *testing.T) {
	out := captureStdout(t, func() int { return RunCLI([]string{"bogus"}) })
	if out.code != 2 {
		t.Errorf("unknown subcommand exit = %d, want 2", out.code)
	}
	if !strings.Contains(out.err, "bogus") {
		t.Errorf("stderr = %q, want to name the bad subcommand", out.err)
	}
}

func TestRunCLIMissingKey(t *testing.T) {
	t.Setenv("ORCADUB_API_KEY", "")
	out := captureStdout(t, func() int { return RunCLI([]string{"health"}) })
	if out.code != 1 {
		t.Errorf("missing-key exit = %d, want 1", out.code)
	}
	if !strings.Contains(out.err, "ORCADUB_API_KEY") {
		t.Errorf("stderr = %q, want ORCADUB_API_KEY guidance", out.err)
	}
}

type cliOut struct {
	out, err string
	code     int
}

// captureStdout redirects os.Stdout and os.Stderr around fn and returns what
// each captured plus fn's return code.
func captureStdout(t *testing.T, fn func() int) cliOut {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr
	var bo, be bytes.Buffer
	_, _ = bo.ReadFrom(rOut)
	_, _ = be.ReadFrom(rErr)
	return cliOut{out: bo.String(), err: be.String(), code: code}
}

func TestApplyCreateOptsErrors(t *testing.T) {
	cases := map[string][]string{
		"unknown --opt key": {"no_such_field=1"},
		"malformed --opt":   {"preserve_bgm"},
		"preserve_bgm":      {"preserve_bgm=maybe"},
	}
	for wantSubstr, opts := range cases {
		var in CreateInput
		err := applyCreateOpts(&in, opts)
		if err == nil || !strings.Contains(err.Error(), wantSubstr) {
			t.Errorf("opts %v: err = %v, want substring %q", opts, err, wantSubstr)
		}
	}
}

func TestRunCLICreate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/videos" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":"job-9","object":"video","status":"queued","progress":0}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ORCADUB_API_KEY", "sk-test")
	t.Setenv("ORCADUB_BASE_URL", srv.URL)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"create",
			"--source-lang", "en", "--target-lang", "ja",
			"--url", "https://youtu.be/x",
			"--opt", "preserve_bgm=true",
			"--opt", "resolution=1080p",
		})
	})
	if out.code != 0 {
		t.Fatalf("create exit = %d, stderr=%s", out.code, out.err)
	}
	if gotBody["model"] != "orca/dub" || gotBody["source_lang"] != "en" || gotBody["target_lang"] != "ja" {
		t.Errorf("body = %v", gotBody)
	}
	if gotBody["preserve_bgm"] != "true" {
		t.Errorf("preserve_bgm on wire = %v, want string \"true\"", gotBody["preserve_bgm"])
	}
	if gotBody["resolution"] != "1080p" {
		t.Errorf("resolution = %v", gotBody["resolution"])
	}
	if !strings.Contains(out.out, `"job-9"`) {
		t.Errorf("stdout = %s", out.out)
	}
}

func TestRunCLICreateValidation(t *testing.T) {
	t.Setenv("ORCADUB_API_KEY", "sk-test")
	// neither url nor file-id
	out := captureStdout(t, func() int {
		return RunCLI([]string{"create", "--source-lang", "en", "--target-lang", "ja"})
	})
	if out.code != 1 || !strings.Contains(out.err, "exactly one") {
		t.Errorf("want XOR error exit 1, got code=%d err=%q", out.code, out.err)
	}
	// unknown --opt
	out = captureStdout(t, func() int {
		return RunCLI([]string{"create", "--source-lang", "en", "--target-lang", "ja",
			"--url", "https://x", "--opt", "nope=1"})
	})
	if out.code != 1 || !strings.Contains(out.err, "unknown --opt key") {
		t.Errorf("want unknown-opt error exit 1, got code=%d err=%q", out.code, out.err)
	}
}

func TestRunCLIUploadAndDownload(t *testing.T) {
	payload := bytes.Repeat([]byte{0xCD}, 2048)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/files" && r.Method == http.MethodPost:
			file, _, _ := r.FormFile("file")
			n, _ := io.Copy(io.Discard, file)
			_ = file.Close()
			_, _ = fmt.Fprintf(w, `{"id":"file-1","object":"file","bytes":%d,"purpose":"user_data","status":"processed"}`, n)
		case r.URL.Path == "/v1/videos/job-1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"job-1","object":"video","status":"completed","progress":100,"job_id":"uuid-9"}`))
		case r.URL.Path == "/v1/videos/uuid-9/content":
			_, _ = w.Write(payload)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ORCADUB_API_KEY", "sk-test")
	t.Setenv("ORCADUB_BASE_URL", srv.URL)

	// upload
	src := writeTempFile(t, 1024)
	out := captureStdout(t, func() int { return RunCLI([]string{"upload", "--path", src}) })
	if out.code != 0 || !strings.Contains(out.out, `"file-1"`) {
		t.Fatalf("upload code=%d out=%s err=%s", out.code, out.out, out.err)
	}

	// download
	dest := filepath.Join(t.TempDir(), "out.mp4")
	out = captureStdout(t, func() int {
		return RunCLI([]string{"download", "--video-id", "job-1", "--dest", dest})
	})
	if out.code != 0 {
		t.Fatalf("download code=%d err=%s", out.code, out.err)
	}
	if !strings.Contains(out.out, `"bytes": 2048`) {
		t.Errorf("download stdout = %s", out.out)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, payload) {
		t.Error("downloaded bytes mismatch")
	}
}

func TestRunCLIUploadDownloadValidation(t *testing.T) {
	t.Setenv("ORCADUB_API_KEY", "sk-test")
	out := captureStdout(t, func() int { return RunCLI([]string{"upload"}) })
	if out.code != 1 || !strings.Contains(out.err, "--path") {
		t.Errorf("upload without --path: code=%d err=%q", out.code, out.err)
	}
	out = captureStdout(t, func() int { return RunCLI([]string{"download", "--video-id", "x"}) })
	if out.code != 1 || !strings.Contains(out.err, "--dest") {
		t.Errorf("download without --dest: code=%d err=%q", out.code, out.err)
	}
}

func setSkillCLIRuntimeForTest(
	installer skillInstaller,
	projectDir string,
	homeDir string,
) func() {
	oldInstaller := skillCLIInstaller
	oldWorkingDir := skillCLIWorkingDir
	oldHomeDir := skillCLIHomeDir
	skillCLIInstaller = func() skillInstaller { return installer }
	skillCLIWorkingDir = func() (string, error) { return projectDir, nil }
	skillCLIHomeDir = func() (string, error) { return homeDir, nil }
	return func() {
		skillCLIInstaller = oldInstaller
		skillCLIWorkingDir = oldWorkingDir
		skillCLIHomeDir = oldHomeDir
	}
}

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

func setSkillPromptRuntimeForTest(
	runner skillPromptRunner,
	isTerminal bool,
	locale string,
) func() {
	oldPromptRunner := skillCLIPromptRunner
	oldIsTerminal := skillCLIIsTerminal
	oldGetenv := skillCLIGetenv
	skillCLIPromptRunner = func() skillPromptRunner { return runner }
	skillCLIIsTerminal = func() bool { return isTerminal }
	skillCLIGetenv = func(string) string { return locale }
	return func() {
		skillCLIPromptRunner = oldPromptRunner
		skillCLIIsTerminal = oldIsTerminal
		skillCLIGetenv = oldGetenv
	}
}

func TestRunCLISkillInstallTUI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		projectDir,
		t.TempDir(),
	)
	t.Cleanup(restoreCLI)
	prompt := &fakeSkillPromptRunner{result: skillPromptResult{
		Language:    skillLanguageZH,
		Scope:       skillInstallProject,
		PlatformIDs: []string{"codex"},
	}}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, true, "zh_CN.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install"})
	})
	if out.code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if prompt.calls != 1 {
		t.Fatalf("prompt calls = %d", prompt.calls)
	}
	if !prompt.request.AskLanguage || !prompt.request.AskScope || !prompt.request.AskPlatforms {
		t.Fatalf("prompt request = %+v", prompt.request)
	}
	if !strings.Contains(out.out, "ORCA//DUB") {
		t.Fatalf("stdout lacks banner:\n%s", out.out)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "orcadub", "SKILL.md")); err != nil {
		t.Fatalf("Codex Skill was not installed: %v", err)
	}
}

func TestRunCLISkillInstallTUIExplicitScopeSkipsScopePrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	homeDir := t.TempDir()
	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		t.TempDir(),
		homeDir,
	)
	t.Cleanup(restoreCLI)
	prompt := &fakeSkillPromptRunner{result: skillPromptResult{
		Language:    skillLanguageEN,
		Scope:       skillInstallGlobal,
		PlatformIDs: []string{"codex"},
	}}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, true, "en_US.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--scope", "global"})
	})
	if out.code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if prompt.calls != 1 || !prompt.request.AskLanguage ||
		prompt.request.AskScope || !prompt.request.AskPlatforms {
		t.Fatalf("prompt request = %+v, calls=%d", prompt.request, prompt.calls)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".agents", "skills", "orcadub", "SKILL.md")); err != nil {
		t.Fatalf("global Codex Skill was not installed: %v", err)
	}
}

func TestRunCLISkillInstallExplicitPlatformSkipsPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	projectDir := t.TempDir()
	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		projectDir,
		t.TempDir(),
	)
	t.Cleanup(restoreCLI)
	prompt := &fakeSkillPromptRunner{}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, false, "en_US.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--platform", "codex"})
	})
	if out.code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if prompt.calls != 0 {
		t.Fatalf("prompt calls = %d", prompt.calls)
	}
	if strings.Contains(out.out, "ORCA//DUB") {
		t.Fatalf("non-interactive stdout contains banner:\n%s", out.out)
	}
}

func TestRunCLISkillInstallYesSkipsPromptAndBanner(t *testing.T) {
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
	prompt := &fakeSkillPromptRunner{}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, true, "en_US.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--platform", "codex", "--yes"})
	})
	if out.code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if prompt.calls != 0 || strings.Contains(out.out, "ORCA//DUB") || strings.Contains(out.out, "\x1b[") {
		t.Fatalf("prompt calls=%d stdout=%q", prompt.calls, out.out)
	}
}

func TestRunCLISkillInstallCancellation(t *testing.T) {
	contacted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacted = true
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		t.TempDir(),
		t.TempDir(),
	)
	t.Cleanup(restoreCLI)
	prompt := &fakeSkillPromptRunner{err: huh.ErrUserAborted}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, true, "en_US.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install"})
	})
	if out.code != 130 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if contacted {
		t.Fatal("installer source was contacted after cancellation")
	}
}

func TestRunCLISkillInstallNonTTY(t *testing.T) {
	restoreCLI := setSkillCLIRuntimeForTest(
		skillInstaller{client: http.DefaultClient, sourceURL: "http://unused.invalid"},
		t.TempDir(),
		t.TempDir(),
	)
	t.Cleanup(restoreCLI)
	prompt := &fakeSkillPromptRunner{}
	restorePrompt := setSkillPromptRuntimeForTest(prompt, false, "zh_CN.UTF-8")
	t.Cleanup(restorePrompt)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install"})
	})
	if out.code != 2 {
		t.Fatalf("code=%d stdout=%q stderr=%q", out.code, out.out, out.err)
	}
	if !strings.Contains(out.err, "--lang") || !strings.Contains(out.err, "--scope") ||
		!strings.Contains(out.err, "--platform") || !strings.Contains(out.err, "--yes") {
		t.Fatalf("stderr lacks flag guidance: %q", out.err)
	}
	if prompt.calls != 0 {
		t.Fatalf("prompt calls = %d", prompt.calls)
	}
}

func TestRenderSkillInstallReportLocalized(t *testing.T) {
	report := skillInstallReport{
		Source: "https://example.test/SKILL.md",
		Scope:  skillInstallProject,
		Results: []skillInstallResult{
			{
				Platforms:     []string{"codex"},
				PlatformNames: []string{"Codex"},
				Path:          "/tmp/.agents/skills/orcadub/SKILL.md",
				Status:        skillInstallStatusConflict,
			},
		},
	}

	tests := []struct {
		name     string
		language skillLanguage
		title    string
		scope    string
	}{
		{
			name:     "English",
			language: skillLanguageEN,
			title:    "OrcaDub Skill installation",
			scope:    "Project — current directory",
		},
		{
			name:     "Simplified Chinese",
			language: skillLanguageZH,
			title:    "OrcaDub Skill 安装结果",
			scope:    "当前项目",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := captureStdout(t, func() int {
				renderSkillInstallReport(report, test.language)
				return 0
			})
			for _, want := range []string{
				test.title,
				test.scope,
				"--force",
				"Codex",
				"/tmp/.agents/skills/orcadub/SKILL.md",
			} {
				if !strings.Contains(out.out, want) {
					t.Fatalf("stdout lacks %q:\n%s", want, out.out)
				}
			}
		})
	}
	if report.Results[0].Status != skillInstallStatusConflict {
		t.Fatalf("machine status changed to %q", report.Results[0].Status)
	}
}

func TestRunCLISkillInstallExplicitPlatformJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	projectDir := t.TempDir()
	homeDir := t.TempDir()
	restore := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		projectDir,
		homeDir,
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
	if out.code != 0 {
		t.Fatalf("skill install exit = %d, stderr=%s", out.code, out.err)
	}
	var report skillInstallReport
	if err := json.Unmarshal([]byte(out.out), &report); err != nil {
		t.Fatalf("stdout is not a skillInstallReport: %v\n%s", err, out.out)
	}
	if len(report.Results) != 1 || report.Results[0].Status != skillInstallStatusInstalled {
		t.Fatalf("report = %#v", report)
	}
	destination := filepath.Join(projectDir, ".agents", "skills", "orcadub", "SKILL.md")
	if report.Results[0].Path != destination {
		t.Fatalf("path = %q, want %q", report.Results[0].Path, destination)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != validDubVideoSkill {
		t.Fatalf("installed Skill = %q, err=%v", got, err)
	}
}

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

func TestRunCLISkillInstallRejectsUnknownPlatform(t *testing.T) {
	restore := setSkillCLIRuntimeForTest(
		skillInstaller{client: http.DefaultClient, sourceURL: "http://unused.invalid"},
		t.TempDir(),
		t.TempDir(),
	)
	t.Cleanup(restore)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--platform", "unknown", "--yes"})
	})
	if out.code != 2 {
		t.Fatalf("exit = %d, want 2; stderr=%s", out.code, out.err)
	}
	if !strings.Contains(out.err, `unknown platform "unknown"`) {
		t.Fatalf("stderr = %q, want unknown platform guidance", out.err)
	}
}

func TestRunCLISkillInstallHelp(t *testing.T) {
	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--help"})
	})
	if out.code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%s", out.code, out.err)
	}
	if !strings.Contains(out.out, "orcadub skill install") ||
		!strings.Contains(out.out, "--platform") ||
		!strings.Contains(out.out, "--scope") {
		t.Fatalf("stdout lacks install help:\n%s", out.out)
	}
	if out.err != "" {
		t.Fatalf("stderr = %q, want empty", out.err)
	}
}

func TestRunCLISkillInstallYesUsesDetectedPlatforms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	projectDir := t.TempDir()
	for _, marker := range []string{".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(projectDir, marker), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	restore := setSkillCLIRuntimeForTest(
		skillInstaller{client: srv.Client(), sourceURL: srv.URL},
		projectDir,
		t.TempDir(),
	)
	t.Cleanup(restore)

	out := captureStdout(t, func() int {
		return RunCLI([]string{"skill", "install", "--yes", "--json"})
	})
	if out.code != 0 {
		t.Fatalf("skill install exit = %d, stderr=%s", out.code, out.err)
	}
	var report skillInstallReport
	if err := json.Unmarshal([]byte(out.out), &report); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.out)
	}
	gotPlatforms := make([]string, 0)
	for _, result := range report.Results {
		gotPlatforms = append(gotPlatforms, result.Platforms...)
	}
	wantPlatforms := []string{"claude", "codex"}
	if !reflect.DeepEqual(gotPlatforms, wantPlatforms) {
		t.Fatalf("installed platforms = %v, want %v", gotPlatforms, wantPlatforms)
	}
}

func TestRunCLISkillInstallOnlyRequiresSelectedScopeBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validDubVideoSkill))
	}))
	t.Cleanup(srv.Close)

	t.Run("project does not require home", func(t *testing.T) {
		projectDir := t.TempDir()
		restore := setSkillCLIRuntimeForTest(
			skillInstaller{client: srv.Client(), sourceURL: srv.URL},
			projectDir,
			t.TempDir(),
		)
		t.Cleanup(restore)
		skillCLIHomeDir = func() (string, error) {
			return "", errors.New("injected home lookup failure")
		}

		out := captureStdout(t, func() int {
			return RunCLI([]string{
				"skill", "install",
				"--platform", "claude",
				"--scope", "project",
				"--json",
			})
		})
		if out.code != 0 {
			t.Fatalf("exit = %d, stderr=%s", out.code, out.err)
		}
	})

	t.Run("explicit global does not require cwd", func(t *testing.T) {
		homeDir := t.TempDir()
		restore := setSkillCLIRuntimeForTest(
			skillInstaller{client: srv.Client(), sourceURL: srv.URL},
			t.TempDir(),
			homeDir,
		)
		t.Cleanup(restore)
		skillCLIWorkingDir = func() (string, error) {
			return "", errors.New("injected cwd lookup failure")
		}

		out := captureStdout(t, func() int {
			return RunCLI([]string{
				"skill", "install",
				"--platform", "claude",
				"--scope", "global",
				"--json",
			})
		})
		if out.code != 0 {
			t.Fatalf("exit = %d, stderr=%s", out.code, out.err)
		}
	})
}

func TestNPMPackageExposesOrcadubCommand(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "npm", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Name string            `json:"name"`
		Bin  map[string]string `json:"bin"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "@orcadub/cli" {
		t.Fatalf("npm package name = %q, want @orcadub/cli", pkg.Name)
	}
	if got := pkg.Bin["orcadub"]; got != "bin/run.js" {
		t.Fatalf("npm bin orcadub = %q, want bin/run.js", got)
	}
	if len(pkg.Bin) != 1 {
		t.Fatalf("npm bin entries = %v, want only orcadub", pkg.Bin)
	}
}
