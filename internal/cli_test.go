package dub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
