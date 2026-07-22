# OrcaDub CLI Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a one-shot CLI mode to the existing OrcaDub Go binary (reusing `internal.Client` verbatim) so the `dub-video` skill can call it instead of driving a resident MCP server, and rename the npm package `@orcadub/mcp` → `@orcadub/cli`.

**Architecture:** The binary keeps its MCP server mode (no args). A new `internal/cli.go` parses the subcommands `health|upload|create|get|download`, builds a `Client` from `LoadConfig()`, calls the matching client method, and prints result JSON to stdout (errors to stderr, non-zero exit). The `CreateInput → CreateVideoRequest` mapping is extracted into a shared `buildCreateRequest` helper used by both the MCP tool and the CLI so the two paths produce identical gateway requests.

**Tech Stack:** Go 1.25 (stdlib `flag`, `encoding/json`), Node ≥18 npm shim (unchanged forwarding), goreleaser (unchanged).

## Global Constraints

- Module path: `github.com/Continuum-AI-Corp/orcadub-mcp-server`; internal package is `dub` (`internal/*.go`).
- Do NOT modify `internal/client.go`'s request behavior (HTTP, upload chunking, atomic download, `errNotAuthorized`, presigned-URL blanking). CLI reuses it as-is. The ONE permitted change is `LoadConfig` gaining an optional `ORCADUB_BASE_URL` override (test-only seam; defaults to the fixed gateway when unset) — this changes configuration reading, not request behavior.
- API key comes ONLY from the `ORCADUB_API_KEY` environment variable via `LoadConfig()`. No `--api-key` flag.
- No-args invocation MUST still start the MCP stdio server (backward compatibility).
- CLI output contract: success → pretty JSON on **stdout**; failure → message on **stderr** + non-zero process exit.
- `create` optional parameters are expressed via repeatable `--opt key=val`; `key` is the wire/JSON field name. Booleans are tri-state (`true`/`false`/absent=deploy default) via the existing `boolStr` convention. Unknown `--opt` keys are a hard error.
- Binary artifact name (`orcadub-mcp-server`), goreleaser config, release URLs, and the npm `bin` name stay UNCHANGED — only the npm package `name`/description/keywords and human-facing docs change to `@orcadub/cli`.
- Every code change follows TDD: failing test → run (see it fail) → minimal impl → run (see it pass) → commit.
- Work happens on the `cli-mode` branch (already created; design doc already committed there).

---

## Repo A: `orcadub-mcp-server` (the binary + npm packaging)

### Task 1: Extract shared `buildCreateRequest` helper

Refactor the `CreateInput → CreateVideoRequest` construction out of `dubCreate` (currently inline in `internal/tools.go:125-154`) into a package-level helper both MCP and CLI can call. Behavior must not change — existing MCP tests must stay green.

**Files:**
- Modify: `internal/tools.go` (extract helper from `dubCreate`)
- Test: `internal/tools_test.go` (add one test asserting the helper's mapping)

**Interfaces:**
- Produces: `func buildCreateRequest(in CreateInput) (CreateVideoRequest, error)` — performs the file_id/url XOR check, the `video_name` required-with-file_id check, and the full field mapping (returns the same error strings `dubCreate` returns today: `"provide exactly one of file_id or url"`, `"video_name is required when file_id is used ..."`).
- Consumes (Task 4, Task 5): `CreateInput` struct (already defined in `internal/tools.go`), `buildCreateRequest`.

- [ ] **Step 1: Write the failing test**

Add to `internal/tools_test.go`:

```go
func TestBuildCreateRequest(t *testing.T) {
	tr := true
	req, err := buildCreateRequest(CreateInput{
		SourceLang: "en", TargetLang: "zh", URL: "https://x/v.mp4",
		PreserveBGM: &tr,
	})
	if err != nil {
		t.Fatalf("buildCreateRequest: %v", err)
	}
	if req.SourceLang != "en" || req.TargetLang != "zh" {
		t.Errorf("langs = %q/%q", req.SourceLang, req.TargetLang)
	}
	if req.VideoPath == nil || req.VideoPath.URL != "https://x/v.mp4" {
		t.Errorf("video_path = %+v", req.VideoPath)
	}
	if req.PreserveBGM == nil || *req.PreserveBGM != "true" {
		t.Errorf("preserve_bgm = %v, want *\"true\"", req.PreserveBGM)
	}
	// XOR + video_name validation lives in the helper now.
	if _, err := buildCreateRequest(CreateInput{SourceLang: "en", TargetLang: "zh", FileID: "f", URL: "u"}); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("want XOR error, got %v", err)
	}
	if _, err := buildCreateRequest(CreateInput{SourceLang: "en", TargetLang: "zh", FileID: "f"}); err == nil || !strings.Contains(err.Error(), "video_name") {
		t.Errorf("want video_name error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestBuildCreateRequest -v`
Expected: FAIL — `undefined: buildCreateRequest`.

- [ ] **Step 3: Extract the helper**

In `internal/tools.go`, add the helper (place it just above `dubCreate`):

```go
// buildCreateRequest performs dub_create's input validation and maps
// CreateInput onto the wire CreateVideoRequest. Shared by the MCP tool
// (dubCreate) and the CLI create handler so both build an identical body.
func buildCreateRequest(in CreateInput) (CreateVideoRequest, error) {
	hasFile, hasURL := in.FileID != "", in.URL != ""
	if hasFile == hasURL {
		return CreateVideoRequest{}, fmt.Errorf("provide exactly one of file_id or url")
	}
	if hasFile && in.VideoName == "" {
		return CreateVideoRequest{}, fmt.Errorf("video_name is required when file_id is used (uploaded files carry no title metadata)")
	}
	return CreateVideoRequest{
		SourceLang:             in.SourceLang,
		TargetLang:             in.TargetLang,
		VideoPath:              &VideoPath{FileID: in.FileID, URL: in.URL},
		VideoName:              in.VideoName,
		Profile:                in.Profile,
		TranslationStyle:       in.TranslationStyle,
		Glossary:               in.Glossary,
		AdaptIdiomsEnabled:     boolStr(in.AdaptIdioms),
		CometEnabled:           boolStr(in.CometEnabled),
		TTSBackend:             in.TTSBackend,
		SongTranslationEnabled: boolStr(in.SongTranslation),
		ProjectID:              in.ProjectID,
		SpeakerAssignments:     in.SpeakerAssignments,
		VoiceCloneConsent:      in.VoiceCloneConsent,
		PreserveBGM:            boolStr(in.PreserveBGM),
		BedLevelMatch:          boolStr(in.BedLevelMatch),
		BedDuck:                boolStr(in.BedDuck),
		BedReverbPreset:        in.BedReverbPreset,
		LoudnessEnabled:        boolStr(in.LoudnessEnabled),
		AlignPerWord:           boolStr(in.AlignPerWord),
		Lipsync:                boolStr(in.Lipsync),
		LipsyncVisemes:         boolStr(in.LipsyncVisemes),
		LipsyncIdentityGuard:   boolStr(in.LipsyncIdentityGuard),
		Watermark:              boolStr(in.Watermark),
		RemoveWatermark:        boolStr(in.RemoveWatermark),
		Resolution:             in.Resolution,
		Ratio:                  in.Ratio,
		CompactOutput:          boolStr(in.CompactOutput),
	}, nil
}
```

Then replace the body of `dubCreate` (from the `hasFile, hasURL := ...` check through the `req := CreateVideoRequest{...}` literal) with:

```go
func (t *toolLayer) dubCreate(ctx context.Context, _ *mcp.CallToolRequest, in CreateInput) (*mcp.CallToolResult, any, error) {
	req, err := buildCreateRequest(in)
	if err != nil {
		return nil, nil, err
	}
	v, err := t.client.CreateVideo(ctx, &req)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(v)
}
```

Keep the `//nolint:gocritic` comment above `dubCreate`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run 'TestBuildCreateRequest|TestDubCreateTool' -v`
Expected: PASS (`TestBuildCreateRequest`, `TestDubCreateToolValidatesInput`, `TestDubCreateToolBoolKnobs` all pass — the extraction preserved behavior).

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add internal/tools.go internal/tools_test.go
git commit -m "refactor: extract buildCreateRequest shared by MCP and CLI create paths"
```

---

### Task 2: `--opt key=val` parser for create options

Add a parser that turns a list of `key=val` strings into a `CreateInput`'s optional fields. This isolates the trickiest CLI logic (tri-state bools, string fields, dotted map keys, unknown-key rejection) behind one tested function before wiring the CLI.

**Files:**
- Create: `internal/cli.go` (parser only for now)
- Test: `internal/cli_test.go`

**Interfaces:**
- Produces: `func applyCreateOpts(in *CreateInput, opts []string) error` — mutates `in` from `key=val` entries. Bool keys set the matching `*bool` field; string keys set the matching string field; `glossary.<k>`/`speaker_assignments.<k>` accumulate into the maps (allocating the map on first use); unknown keys return `fmt.Errorf("unknown --opt key %q", key)`; malformed entries (no `=`) return `fmt.Errorf("malformed --opt %q (want key=value)", raw)`; a bool key with a non-bool value returns an error naming the key.
- Consumes (Task 4): `applyCreateOpts`.

- [ ] **Step 1: Write the failing test**

Create `internal/cli_test.go`:

```go
package dub

import (
	"strings"
	"testing"
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

func TestApplyCreateOptsErrors(t *testing.T) {
	cases := map[string][]string{
		"unknown --opt key":  {"no_such_field=1"},
		"malformed --opt":    {"preserve_bgm"},
		"preserve_bgm":       {"preserve_bgm=maybe"},
	}
	for wantSubstr, opts := range cases {
		var in CreateInput
		err := applyCreateOpts(&in, opts)
		if err == nil || !strings.Contains(err.Error(), wantSubstr) {
			t.Errorf("opts %v: err = %v, want substring %q", opts, err, wantSubstr)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestApplyCreateOpts -v`
Expected: FAIL — `undefined: applyCreateOpts`.

- [ ] **Step 3: Write the parser**

Create `internal/cli.go`:

```go
package dub

import (
	"fmt"
	"strconv"
	"strings"
)

// parseBoolOpt converts an --opt value to a *bool, erroring (naming key) on
// anything strconv.ParseBool rejects.
func parseBoolOpt(key, val string) (*bool, error) {
	b, err := strconv.ParseBool(val)
	if err != nil {
		return nil, fmt.Errorf("--opt %s: %q is not a boolean (use true or false)", key, val)
	}
	return &b, nil
}

// applyCreateOpts folds repeatable --opt key=value entries into in. key is the
// wire/JSON field name. Bool keys are tri-state (absent leaves the field nil =
// deploy default). Dotted keys (glossary.TERM, speaker_assignments.LABEL)
// accumulate into maps. Unknown keys are a hard error so a mistyped paid-job
// parameter never gets silently dropped.
func applyCreateOpts(in *CreateInput, opts []string) error {
	for _, raw := range opts {
		key, val, ok := strings.Cut(raw, "=")
		if !ok {
			return fmt.Errorf("malformed --opt %q (want key=value)", raw)
		}
		if mapKey, sub, isMap := strings.Cut(key, "."); isMap {
			switch mapKey {
			case "glossary":
				if in.Glossary == nil {
					in.Glossary = map[string]string{}
				}
				in.Glossary[sub] = val
			case "speaker_assignments":
				if in.SpeakerAssignments == nil {
					in.SpeakerAssignments = map[string]string{}
				}
				in.SpeakerAssignments[sub] = val
			default:
				return fmt.Errorf("unknown --opt key %q", key)
			}
			continue
		}
		var err error
		switch key {
		// tri-state booleans
		case "adapt_idioms":
			in.AdaptIdioms, err = parseBoolOpt(key, val)
		case "comet_enabled":
			in.CometEnabled, err = parseBoolOpt(key, val)
		case "song_translation":
			in.SongTranslation, err = parseBoolOpt(key, val)
		case "preserve_bgm":
			in.PreserveBGM, err = parseBoolOpt(key, val)
		case "bed_level_match":
			in.BedLevelMatch, err = parseBoolOpt(key, val)
		case "bed_duck":
			in.BedDuck, err = parseBoolOpt(key, val)
		case "loudness_enabled":
			in.LoudnessEnabled, err = parseBoolOpt(key, val)
		case "align_per_word":
			in.AlignPerWord, err = parseBoolOpt(key, val)
		case "lipsync":
			in.Lipsync, err = parseBoolOpt(key, val)
		case "lipsync_visemes":
			in.LipsyncVisemes, err = parseBoolOpt(key, val)
		case "lipsync_identity_guard":
			in.LipsyncIdentityGuard, err = parseBoolOpt(key, val)
		case "watermark":
			in.Watermark, err = parseBoolOpt(key, val)
		case "remove_watermark":
			in.RemoveWatermark, err = parseBoolOpt(key, val)
		case "compact_output":
			in.CompactOutput, err = parseBoolOpt(key, val)
		// bed_reverb_preset is a *string on CreateInput
		case "bed_reverb_preset":
			v := val
			in.BedReverbPreset = &v
		// plain string fields
		case "profile":
			in.Profile = val
		case "translation_style":
			in.TranslationStyle = val
		case "tts_backend":
			in.TTSBackend = val
		case "project_id":
			in.ProjectID = val
		case "resolution":
			in.Resolution = val
		case "ratio":
			in.Ratio = val
		// voice_clone_consent is a plain bool on CreateInput (not tri-state)
		case "voice_clone_consent":
			var b *bool
			b, err = parseBoolOpt(key, val)
			if err == nil {
				in.VoiceCloneConsent = *b
			}
		default:
			return fmt.Errorf("unknown --opt key %q", key)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestApplyCreateOpts -v`
Expected: PASS (`TestApplyCreateOpts`, `TestApplyCreateOptsErrors`).

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add internal/cli.go internal/cli_test.go
git commit -m "feat: --opt key=val parser for CLI create options (tri-state bools, maps, unknown-key error)"
```

---

### Task 3: `RunCLI` dispatch — health & get subcommands

Add the entry point `RunCLI` with output plumbing and the two simplest subcommands, so the dispatch skeleton is tested before create/upload/download hang off it.

**Files:**
- Modify: `internal/cli.go` (add `RunCLI`, output helpers, health & get handlers)
- Test: `internal/cli_test.go`

**Interfaces:**
- Produces:
  - `func RunCLI(args []string) int` — `args` is `os.Args[1:]` (element 0 is the subcommand). Returns process exit code (0 success, 1 failure). Prints result JSON to `stdout`, errors to `stderr`. On unknown subcommand prints usage to stderr and returns 2.
  - Internal: uses `context.Background()`, `LoadConfig()`, `NewClient`.
- Consumes: `Client.Health`, `Client.GetVideo` (from `internal/client.go`); `jsonResultBytes` (below).

- [ ] **Step 1: Write the failing test**

Add to `internal/cli_test.go` (add imports `bytes`, `encoding/json`, `net/http`, `net/http/httptest`, `os` to the file's import block):

```go
// runCLIWith points RunCLI at a fake gateway by overriding the base+origin
// URLs through the env-driven client, capturing stdout. It mirrors the
// client_test.go httptest pattern.
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
```

Also add this stdout/stderr capture helper at the bottom of `internal/cli_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestRunCLI -v`
Expected: FAIL — `undefined: RunCLI` (and `ORCADUB_BASE_URL` not yet honored).

- [ ] **Step 3: Add a test-only base-URL override to LoadConfig, then implement RunCLI**

First, teach `LoadConfig` to honor an optional `ORCADUB_BASE_URL` override (needed so the CLI is testable against httptest without touching the fixed production origin). In `internal/client.go`, change `LoadConfig`:

```go
func LoadConfig() Config {
	base := gatewayBaseURL
	if v := os.Getenv("ORCADUB_BASE_URL"); v != "" {
		base = v
	}
	return Config{
		BaseURL: base,
		APIKey:  os.Getenv("ORCADUB_API_KEY"),
	}
}
```

Note: this override also affects the origin/content URL only through the gateway base — for CLI download tests we set both via the same fake server; `NewClient` still sets `originURL = originBaseURL`, so add an origin override too, right after `NewClient(cfg)` is called inside `RunCLI` (see below), guarded by the same env var. (Keeping `NewClient` itself unchanged honors the "don't change client behavior" constraint — the override lives in the CLI layer.)

Now append to `internal/cli.go` (add imports `context`, `encoding/json`, `flag`, `os` to its import block):

```go
// jsonResultBytes renders v as the same pretty JSON the MCP tools emit.
func jsonResultBytes(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

const cliUsage = `orcadub CLI — OrcaDub video dubbing (OrcaRouter model orca/dub).

Usage:
  orcadub health
  orcadub upload   --path <file> [--purpose <p>]
  orcadub create   --source-lang <c> --target-lang <c> (--url <u> | --file-id <id> --video-name <name>) [--opt key=val ...]
  orcadub get      --video-id <id>
  orcadub download --video-id <id> --dest <path>

Auth: set ORCADUB_API_KEY (sk-orca-... from https://www.orcarouter.ai/console).
With no subcommand the binary runs as an MCP stdio server.`

// RunCLI executes one CLI subcommand. args is os.Args[1:] (args[0] is the
// subcommand). Success prints result JSON to stdout and returns 0; failures
// print to stderr and return 1; an unknown subcommand returns 2.
func RunCLI(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, cliUsage)
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	c := newCLIClient()
	ctx := context.Background()

	switch cmd {
	case "health":
		return emit(c.Health(ctx))
	case "get":
		fs := flag.NewFlagSet("get", flag.ContinueOnError)
		id := fs.String("video-id", "", "job id returned by create")
		if err := fs.Parse(rest); err != nil {
			return fail(err)
		}
		if *id == "" {
			return fail(fmt.Errorf("get: --video-id is required"))
		}
		return emit(c.GetVideo(ctx, *id))
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s\n", cmd, cliUsage)
		return 2
	}
}

// newCLIClient builds the client from env config, applying the test-only
// ORCADUB_BASE_URL override to the origin URL too so downloads in tests hit
// the same fake server.
func newCLIClient() *Client {
	cfg := LoadConfig()
	c := NewClient(cfg)
	if v := os.Getenv("ORCADUB_BASE_URL"); v != "" {
		c.originURL = v
	}
	return c
}

// emit prints a successful result as JSON to stdout (returns 0) or routes the
// error through fail. Generic over the (value, error) pairs client methods
// return.
func emit[T any](v T, err error) int {
	if err != nil {
		return fail(err)
	}
	b, mErr := jsonResultBytes(v)
	if mErr != nil {
		return fail(mErr)
	}
	fmt.Fprintln(os.Stdout, string(b))
	return 0
}

// fail prints err to stderr and returns exit code 1.
func fail(err error) int {
	fmt.Fprintln(os.Stderr, err.Error())
	return 1
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run 'TestRunCLI|TestLoadConfig' -v`
Expected: PASS (`TestRunCLIHealthAndGet`, `TestRunCLIUnknownSubcommand`, `TestRunCLIMissingKey`, and existing `TestLoadConfigDefaultsToGateway` — the `ORCADUB_BASE_URL` override defaults to the gateway when unset, so that test stays green).

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add internal/cli.go internal/cli_test.go internal/client.go
git commit -m "feat: RunCLI dispatch with health & get subcommands (JSON to stdout, errors to stderr)"
```

---

### Task 4: `create` subcommand

Wire the `create` subcommand onto `RunCLI` using `buildCreateRequest` (Task 1) and `applyCreateOpts` (Task 2).

**Files:**
- Modify: `internal/cli.go` (add the `create` case)
- Test: `internal/cli_test.go`

**Interfaces:**
- Consumes: `buildCreateRequest`, `applyCreateOpts`, `Client.CreateVideo`.
- Produces: the `create` case in `RunCLI`'s switch. Flags: `--source-lang`, `--target-lang`, `--url`, `--file-id`, `--video-name`, and a repeatable `--opt` (collected via a custom `flag.Value`).

- [ ] **Step 1: Write the failing test**

Add to `internal/cli_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestRunCLICreate -v`
Expected: FAIL — `create` returns exit 2 (unknown subcommand) so assertions fail.

- [ ] **Step 3: Add the create case**

In `internal/cli.go`, add a repeatable-flag type at the end of the file:

```go
// stringSlice collects a repeatable string flag (--opt a=b --opt c=d).
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
```

Add the `create` case to the switch in `RunCLI`, before `default`:

```go
	case "create":
		fs := flag.NewFlagSet("create", flag.ContinueOnError)
		in := CreateInput{}
		fs.StringVar(&in.SourceLang, "source-lang", "", "source language code (REQUIRED)")
		fs.StringVar(&in.TargetLang, "target-lang", "", "target language code (REQUIRED)")
		fs.StringVar(&in.URL, "url", "", "remote source video URL")
		fs.StringVar(&in.FileID, "file-id", "", "uploaded file id from `upload`")
		fs.StringVar(&in.VideoName, "video-name", "", "job title (REQUIRED with --file-id)")
		var opts stringSlice
		fs.Var(&opts, "opt", "optional parameter as key=value (repeatable)")
		if err := fs.Parse(rest); err != nil {
			return fail(err)
		}
		if in.SourceLang == "" || in.TargetLang == "" {
			return fail(fmt.Errorf("create: --source-lang and --target-lang are required"))
		}
		if err := applyCreateOpts(&in, opts); err != nil {
			return fail(err)
		}
		req, err := buildCreateRequest(in)
		if err != nil {
			return fail(err)
		}
		return emit(c.CreateVideo(ctx, &req))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestRunCLICreate -v`
Expected: PASS (`TestRunCLICreate`, `TestRunCLICreateValidation`).

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add internal/cli.go internal/cli_test.go
git commit -m "feat: CLI create subcommand (--opt options, shared buildCreateRequest)"
```

---

### Task 5: `upload` & `download` subcommands

Wire the two file-streaming subcommands.

**Files:**
- Modify: `internal/cli.go` (add `upload` and `download` cases)
- Test: `internal/cli_test.go`

**Interfaces:**
- Consumes: `Client.UploadFile`, `Client.DownloadContent`.
- Produces: `upload` case (flags `--path` required, `--purpose` optional) and `download` case (flags `--video-id`, `--dest` both required). Download prints `map[string]any{"video_id":..., "dest":..., "bytes":...}` (same shape as `dubDownload`).

- [ ] **Step 1: Write the failing test**

Add to `internal/cli_test.go` (add `io`, `os`, `path/filepath`, `bytes` to imports if not present — `bytes`/`os` already added in Task 3):

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestRunCLIUpload -v`
Expected: FAIL — `upload`/`download` return exit 2 (unknown subcommand).

- [ ] **Step 3: Add the cases**

In `internal/cli.go`, add these cases to the switch before `default`:

```go
	case "upload":
		fs := flag.NewFlagSet("upload", flag.ContinueOnError)
		path := fs.String("path", "", "absolute path of the local video file (REQUIRED)")
		purpose := fs.String("purpose", "", "OpenAI file purpose; default user_data")
		if err := fs.Parse(rest); err != nil {
			return fail(err)
		}
		if *path == "" {
			return fail(fmt.Errorf("upload: --path is required"))
		}
		return emit(c.UploadFile(ctx, *path, *purpose))
	case "download":
		fs := flag.NewFlagSet("download", flag.ContinueOnError)
		id := fs.String("video-id", "", "the completed job/video id (REQUIRED)")
		dest := fs.String("dest", "", "local path to write the MP4 (REQUIRED, must not exist)")
		if err := fs.Parse(rest); err != nil {
			return fail(err)
		}
		if *id == "" {
			return fail(fmt.Errorf("download: --video-id is required"))
		}
		if *dest == "" {
			return fail(fmt.Errorf("download: --dest is required"))
		}
		n, err := c.DownloadContent(ctx, *id, *dest)
		if err != nil {
			return fail(err)
		}
		return emit[any](map[string]any{"video_id": *id, "dest": *dest, "bytes": n}, nil)
```

Note: `emit[any]` is used for download because the map literal has no inferrable named type; the explicit type parameter keeps `emit` generic.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./internal/ -run TestRunCLIUpload -v`
Expected: PASS (`TestRunCLIUploadAndDownload`, `TestRunCLIUploadDownloadValidation`).

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add internal/cli.go internal/cli_test.go
git commit -m "feat: CLI upload & download subcommands"
```

---

### Task 6: Wire `RunCLI` into `main.go`

Route known CLI subcommands from `main` into `RunCLI` while keeping no-args = MCP server.

**Files:**
- Modify: `cmd/main.go`
- Test: none (main wiring; covered by `RunCLI` tests + a manual build/run smoke check in Step 4)

**Interfaces:**
- Consumes: `dub.RunCLI`.

- [ ] **Step 1: Add the dispatch branch**

In `cmd/main.go`, replace the existing `if len(os.Args) > 1 { switch os.Args[1] { ... } }` block so that after the version/help cases, known subcommands delegate to the CLI. The full new block:

```go
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("orcadub-mcp-server " + serverVersion)
			return
		case "--help", "-h", "help":
			fmt.Println("orcadub — OrcaDub video dubbing.")
			fmt.Println("With no subcommand: runs as an MCP stdio server.")
			fmt.Println("CLI subcommands: health | upload | create | get | download (see `orcadub <cmd> -h`).")
			fmt.Println("Configuration: ORCADUB_API_KEY environment variable (https://www.orcarouter.ai/console).")
			fmt.Println("Docs: https://github.com/Continuum-AI-Corp/orcadub-mcp-server")
			return
		case "health", "upload", "create", "get", "download":
			os.Exit(dub.RunCLI(os.Args[1:]))
		}
	}
```

(The MCP server startup below the block is unchanged.)

- [ ] **Step 2: Build**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go build -o bin/orcadub ./cmd`
Expected: builds with no errors.

- [ ] **Step 3: Smoke-test the CLI dispatch (no key → sign-up redirect on stderr)**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && ORCADUB_API_KEY= ./bin/orcadub health; echo "exit=$?"`
Expected: prints the `ORCADUB_API_KEY is not set ... orcarouter.ai/console` message to stderr and `exit=1`.

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && ./bin/orcadub --help`
Expected: prints the updated help naming the CLI subcommands, `exit=0`.

- [ ] **Step 4: Run the full test suite**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && go test ./... -count=1`
Expected: all packages PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add cmd/main.go
git commit -m "feat: dispatch CLI subcommands from main (no args still starts MCP server)"
```

---

### Task 7: Rename npm package to `@orcadub/cli` + docs

Rename the npm package and update the server repo's human-facing docs. The binary artifact name, `bin` name, release URLs, and `install.js` asset names stay unchanged (out of scope per the design).

**Files:**
- Modify: `npm/package.json` (name, description, keywords)
- Modify: `README.md` (add CLI usage; update install line to `@orcadub/cli`)
- Test: `node --check` on the npm scripts + `make check`

**Interfaces:** none (packaging/docs).

- [ ] **Step 1: Update `npm/package.json`**

Change these fields (leave `bin`, `files`, `scripts`, `mcpName`, `engines`, `os`, `cpu`, `repository`, `homepage`, `version` as-is):

```json
  "name": "@orcadub/cli",
  "description": "OrcaDub CLI — AI video dubbing via OrcaRouter (model orca/dub): upload, submit, poll, download. Also runs as an MCP server with no subcommand.",
  "keywords": [
    "orcadub",
    "orcarouter",
    "dubbing",
    "video",
    "translation",
    "cli",
    "mcp"
  ],
```

- [ ] **Step 2: Update `README.md` CLI usage**

In `README.md`, in the section that shows the npm/MCP usage, update the package name to `@orcadub/cli` and add a CLI usage block. Add after the existing usage description:

```markdown
## CLI usage

Every operation is a one-shot subcommand (no resident server needed):

```bash
export ORCADUB_API_KEY=sk-orca-...   # from https://www.orcarouter.ai/console

npx -y @orcadub/cli health
npx -y @orcadub/cli upload --path ./clip.mp4
npx -y @orcadub/cli create --source-lang en --target-lang ja \
  --url https://youtu.be/... --opt preserve_bgm=true
npx -y @orcadub/cli get --video-id <id>
npx -y @orcadub/cli download --video-id <id> --dest ./out.mp4
```

Optional `create` parameters use repeatable `--opt key=val` (e.g.
`--opt watermark=false --opt resolution=1080p --opt glossary.OrcaDub=虎鲸配音`).
Results print as JSON on stdout; errors go to stderr with a non-zero exit.

With no subcommand the same binary runs as an MCP stdio server (`npx -y @orcadub/cli`).
```

Replace any remaining `@orcadub/mcp` occurrences in `README.md` with `@orcadub/cli`.

- [ ] **Step 3: Verify npm scripts still parse**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && node --check npm/install.js && node --check npm/bin/run.js && echo OK`
Expected: `OK` (these files are unchanged; this guards against accidental edits).

- [ ] **Step 4: Run `make check`**

Run: `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && make check`
Expected: fmt/lint/test/vet/build all pass, plus `node --check` on both npm scripts.

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-mcp-server-1
git add npm/package.json README.md
git commit -m "chore: rename npm package @orcadub/mcp -> @orcadub/cli; document CLI usage"
```

---

## Repo B: `orcadub-plugin` (the skill + plugin metadata)

All Repo B tasks run in `/Users/a0000/GolandProjects/orcadub-plugin`. Create a branch first:

```bash
cd /Users/a0000/GolandProjects/orcadub-plugin && git checkout -b cli-mode
```

### Task 8: Rewrite `SKILL.md` to call the CLI

Replace every `dub_*` MCP tool reference with the `@orcadub/cli` subcommand, keeping the workflow, required-param discipline, poll etiquette, and download confirmation.

**Files:**
- Modify: `skills/dub-video/SKILL.md`
- Test: manual review against the checklist in Step 2

**Interfaces:** none (skill prose).

- [ ] **Step 1: Rewrite the skill**

Overwrite `skills/dub-video/SKILL.md` with the CLI-based workflow. Key changes from the current version:

- **Frontmatter `description`**: replace "via the orcadub MCP tools (dub_upload / dub_create / dub_get / dub_download)" and "Requires the orcadub MCP server with an OrcaRouter API key." with CLI wording: "via the `@orcadub/cli` command-line tool (`npx -y @orcadub/cli <subcommand>`). Requires an OrcaRouter API key in `ORCADUB_API_KEY` and Node (for `npx`) or the orcadub binary on PATH."
- **Prerequisites** section: replace "This skill drives the orcadub MCP server ..." with:
  > This skill calls the **`@orcadub/cli`** command-line tool — one short-lived
  > process per operation, no resident server. Invoke it as
  > `npx -y @orcadub/cli <subcommand>` (needs Node) or, if a prebuilt binary is
  > on PATH, `orcadub <subcommand>`. Auth comes from the `ORCADUB_API_KEY`
  > environment variable (an OrcaRouter `sk-orca-...` key from
  > https://www.orcarouter.ai/console). If it is unset/invalid, every command
  > exits non-zero with a "not authorized" message and the sign-up link on
  > stderr — have the user `export ORCADUB_API_KEY=sk-orca-...`, then retry.
  > If `npx`/`node` itself is missing, install Node or use the prebuilt binary.
- **Required parameters** table: unchanged, but note "when calling `create`" instead of "`dub_create`".
- **Options** table: unchanged labels/meanings, but note they are passed as `--opt <parameter>=true` (e.g. `--opt preserve_bgm=true`).
- **Standard flow**: rewrite each numbered step's tool call to a CLI command:
  1. `npx -y @orcadub/cli health`
  2. `npx -y @orcadub/cli upload --path <path>` → note the `id` in the JSON.
  3. `npx -y @orcadub/cli create --source-lang <c> --target-lang <c> (--url <u> | --file-id <id> --video-name <name>) [--opt k=v ...]`
  4. `npx -y @orcadub/cli get --video-id <id>` — same status/progress/poll etiquette.
  5. Download: same confirm-then-download UX; call `npx -y @orcadub/cli download --video-id <id> --dest <path>`. Keep the `content_url` re-download note and the curl example.
- **Every command prints JSON on stdout; a non-zero exit + stderr is the error path** (add this once near the flow).
- **Debugging** section: change "If the `dub_*` tools are missing entirely, the MCP server isn't connected." to "If the command fails to launch (command not found / npx cannot fetch the package), Node/npx is unavailable or the network blocked the download — install Node or drop in the prebuilt binary." Keep the gateway/402/429/`task_not_exist` guidance verbatim.
- **Example** block: update to CLI form:
  ```
  npx -y @orcadub/cli create --source-lang en --target-lang ja \
    --url "https://www.youtube.com/watch?v=..." --opt preserve_bgm=true
  -> {"id":"<job>","status":"queued"}
  npx -y @orcadub/cli get --video-id <job>   # until completed
  npx -y @orcadub/cli download --video-id <job> --dest ./out.mp4
  ```

- [ ] **Step 2: Review against checklist**

Confirm the rewritten `SKILL.md`:
- [ ] contains no `dub_health`/`dub_upload`/`dub_create`/`dub_get`/`dub_download` MCP tool references (grep: `grep -nE 'dub_(health|upload|create|get|download)' skills/dub-video/SKILL.md` returns nothing)
- [ ] contains no "MCP server" as a runtime dependency (grep `grep -ni 'mcp server' skills/dub-video/SKILL.md` — only allowed if describing the optional no-args mode, otherwise none)
- [ ] keeps the "ASK, never guess" required-params section
- [ ] keeps the download confirmation UX and `content_url` note

Run: `cd /Users/a0000/GolandProjects/orcadub-plugin && grep -nE 'dub_(health|upload|create|get|download)' skills/dub-video/SKILL.md; echo "grep exit=$?"`
Expected: no matches, `grep exit=1`.

- [ ] **Step 3: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-plugin
git add skills/dub-video/SKILL.md
git commit -m "feat: dub-video skill calls @orcadub/cli instead of MCP tools"
```

---

### Task 9: Update `plugin.json`, `marketplace.json`, `README.md`

Remove the MCP server wiring and userConfig; update descriptions to the CLI model.

**Files:**
- Modify: `.claude-plugin/plugin.json`
- Modify: `.claude-plugin/marketplace.json`
- Modify: `README.md`

**Interfaces:** none.

- [ ] **Step 1: Edit `.claude-plugin/plugin.json`**

- Remove the entire `mcpServers` block.
- Remove the entire `userConfig` block.
- Change `description` to drop the MCP-bundling clause. New value:

```json
  "description": "Dub videos into another language from your agent — upload or pass a URL, submit to the orca/dub model via OrcaRouter, poll, and download the result. Ships the dub-video skill, which drives the @orcadub/cli command-line tool (no resident MCP server).",
```

Keep `name`, `displayName`, `author`, `homepage`, `repository`, `license`, `keywords` (optionally drop `"mcp"` from keywords — leave the rest).

- [ ] **Step 2: Edit `.claude-plugin/marketplace.json`**

Update the plugin `description` to match the CLI model:

```json
      "description": "Dub videos into another language from your agent (upload/URL → orca/dub via OrcaRouter → poll → download). Ships the dub-video skill driving the @orcadub/cli command-line tool.",
```

- [ ] **Step 3: Rewrite `README.md`**

- Replace the tagline/intro references to "`@orcadub/mcp` server" with the CLI tool.
- **Install** section: the Claude Code / Codex plugin install commands stay (the plugin still installs the skill), but drop the "MCP + skill" framing → "skill that calls the OrcaDub CLI". Update the "Any other MCP client" section to a generic "Any agent" section: install the skill (copy `skills/dub-video`) and ensure Node is available for `npx -y @orcadub/cli`, with `ORCADUB_API_KEY` exported. Remove the "add a stdio server" instructions.
- **What's inside** table: change "MCP server `@orcadub/mcp` — 5 tools: dub_health, ..." row to "CLI `@orcadub/cli` — 5 subcommands: `health`, `upload`, `create`, `get`, `download`". Keep the skill row (update tool names → subcommand names).
- **Dubbing options** table: change parameter references to `--opt <param>=true` form; the labels stay.
- **MCP server details** section: retitle to "CLI / binary details" and note the same binary also runs as an MCP server with no subcommand, linking the server repo. Update the `@orcadub/mcp` reference.
- Replace any remaining `@orcadub/mcp` with `@orcadub/cli`.

- [ ] **Step 4: Verify no stale MCP wiring remains**

Run: `cd /Users/a0000/GolandProjects/orcadub-plugin && grep -rn 'mcpServers\|@orcadub/mcp\|user_config' .claude-plugin README.md; echo "grep exit=$?"`
Expected: no matches, `grep exit=1`.

- [ ] **Step 5: Commit**

```bash
cd /Users/a0000/GolandProjects/orcadub-plugin
git add .claude-plugin/plugin.json .claude-plugin/marketplace.json README.md
git commit -m "chore: drop MCP server wiring; plugin ships the CLI-driven dub-video skill"
```

---

## Final verification

- [ ] **Server repo:** `cd /Users/a0000/GolandProjects/orcadub-mcp-server-1 && make check` → all green.
- [ ] **Server repo smoke:** `go build -o bin/orcadub ./cmd && ./bin/orcadub --help` shows CLI subcommands; `ORCADUB_API_KEY= ./bin/orcadub create --source-lang en --target-lang ja --url x` exits 1 with the sign-up redirect on stderr.
- [ ] **Plugin repo:** the two grep checks (Tasks 8 & 9) return no matches.
- [ ] Review both branches' diffs before any merge/PR (per user's git workflow — do not push or open PRs without asking).
