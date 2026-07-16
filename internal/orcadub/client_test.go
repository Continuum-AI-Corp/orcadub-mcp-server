package orcadub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL, APIKey: "sk-test"})
	c.originURL = srv.URL // deliver from the same fake server in tests
	return c
}

// The gateway origin is fixed in code; the only env input is the key.
func TestLoadConfigDefaultsToGateway(t *testing.T) {
	t.Setenv("ORCADUB_API_KEY", "sk-orca-test")
	cfg := LoadConfig()
	if cfg.BaseURL != "https://api.orcarouter.ai" {
		t.Errorf("BaseURL = %q, want hardcoded gateway origin", cfg.BaseURL)
	}
	if cfg.APIKey != "sk-orca-test" {
		t.Errorf("APIKey = %q", cfg.APIKey)
	}
}

// OrcaRouter authorization is mandatory, but enforced at CALL time: without
// a key every call short-circuits (no HTTP) into a sign-up redirect naming
// the OrcaRouter console, so the agent can send the user there to register.
func TestUnauthorizedCallsRedirectToConsole(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL}) // no APIKey
	ctx := context.Background()
	if _, err := c.Health(ctx); err == nil || !strings.Contains(err.Error(), "orcarouter.ai/console") {
		t.Fatalf("Health err = %v, want console redirect", err)
	}
	if _, err := c.GetVideo(ctx, "job-1"); err == nil || !strings.Contains(err.Error(), "ORCADUB_API_KEY") {
		t.Fatalf("GetVideo err = %v, want ORCADUB_API_KEY guidance", err)
	}
	if _, err := c.DownloadContent(ctx, "job-1", filepath.Join(t.TempDir(), "x.mp4")); err == nil || !strings.Contains(err.Error(), "console") {
		t.Fatalf("DownloadContent err = %v, want console redirect", err)
	}
	if hit {
		t.Error("unauthorized calls must not reach the gateway")
	}
}

// Auth is the Bearer key alone — the gateway resolves workspace + billing
// from it, so no X-Workspace-Id header is ever sent.
func TestAuthHeadersSent(t *testing.T) {
	var gotAuth string
	var hasWS bool
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, hasWS = r.Header["X-Workspace-Id"]
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	var out map[string]bool
	if err := c.doJSON(context.Background(), http.MethodGet, "/ping", nil, &out); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if hasWS {
		t.Error("X-Workspace-Id must never be sent on the gateway path")
	}
}

func TestErrorEnvelopeOpenAI(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Model name not specified","type":"orcarouter_api_error"}}`))
	})
	err := c.doJSON(context.Background(), http.MethodPost, "/v1/videos", map[string]string{}, nil)
	if err == nil || !strings.Contains(err.Error(), "Model name not specified") || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v, want gateway message + status", err)
	}
}

func TestErrorEnvelopeTask(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"task_not_exist","message":"task_not_exist","data":null}`))
	})
	err := c.doJSON(context.Background(), http.MethodGet, "/v1/videos/nope", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "task_not_exist") {
		t.Fatalf("err = %v, want task envelope message", err)
	}
}

// The health probe is an intentionally-invalid submit: healthy means the
// gateway routed it to the dub service, whose validation error names
// source_lang. Anything else is surfaced as an error.
func TestHealthProbe(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "orca/dub" {
			t.Errorf("health probe model = %q", body["model"])
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"invalid_request","message":"source_lang is required","data":null}`))
	})
	h, err := c.Health(context.Background())
	if err != nil || h.Status != "ok" {
		t.Fatalf("Health = %+v, %v", h, err)
	}
}

func TestHealthProbeGatewayBroken(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Model name not specified","type":"orcarouter_api_error"}}`))
	})
	if _, err := c.Health(context.Background()); err == nil || !strings.Contains(err.Error(), "Model name") {
		t.Fatalf("want gateway error surfaced, got %v", err)
	}
}

func TestGetVideo(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/videos/job-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"job-1","object":"video","status":"completed","progress":100,"job_id":"uuid-9","output_url":"https://cos/x.mp4"}`))
	})
	v, err := c.GetVideo(context.Background(), "job-1")
	if err != nil || v.Status != "completed" || v.OutputURL != "https://cos/x.mp4" {
		t.Fatalf("GetVideo = %+v, %v", v, err)
	}
	// completed → content_url derives from the ORIGIN route keyed by the
	// underlying job_id (Bearer-authenticated, freshly signed per request)
	if !strings.HasSuffix(v.ContentURL, "/v1/videos/uuid-9/content") {
		t.Errorf("ContentURL = %q, want origin content route for job_id", v.ContentURL)
	}
}

func TestGetVideoNoContentURLWhileRunning(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"job-2","object":"video","status":"in_progress","progress":40}`))
	})
	v, err := c.GetVideo(context.Background(), "job-2")
	if err != nil || v.ContentURL != "" {
		t.Fatalf("running job must carry no content_url: %+v, %v", v, err)
	}
}

// Every create must carry the gateway's model routing field.
func TestCreateVideoForcesModel(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/videos" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["model"] != "orca/dub" {
			t.Errorf("model = %v, want orca/dub", body["model"])
		}
		if body["source_lang"] != "en" || body["target_lang"] != "zh" {
			t.Errorf("langs missing: %v", body)
		}
		vp, _ := body["video_path"].(map[string]any)
		if vp["url"] != "https://example.com/v.mp4" {
			t.Errorf("video_path = %v", vp)
		}
		_, _ = w.Write([]byte(`{"id":"job-9","object":"video","status":"queued","progress":0}`))
	})
	req := CreateVideoRequest{
		SourceLang: "en",
		TargetLang: "zh",
		VideoPath:  &VideoPath{URL: "https://example.com/v.mp4"},
	}
	v, err := c.CreateVideo(context.Background(), &req)
	if err != nil || v.ID != "job-9" {
		t.Fatalf("CreateVideo = %+v, %v", v, err)
	}
}

func writeTempFile(t *testing.T, size int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "vid-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(bytes.Repeat([]byte{0xAB}, size)); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return f.Name()
}

func TestUploadSmallGoesDirect(t *testing.T) {
	var hits []string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.Method+" "+r.URL.Path)
		if r.URL.Path != "/v1/files" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()
		if got := r.FormValue("purpose"); got != "user_data" {
			t.Errorf("purpose = %q", got)
		}
		if got := r.FormValue("model"); got != "orca/dub" {
			t.Errorf("model form field = %q, want orca/dub", got)
		}
		n, _ := io.Copy(io.Discard, file)
		_, _ = fmt.Fprintf(w, `{"id":"file-1","object":"file","bytes":%d,"filename":%q,"purpose":"user_data","status":"processed"}`, n, hdr.Filename)
	})
	p := writeTempFile(t, 1024)
	fo, err := c.UploadFile(context.Background(), p, "")
	if err != nil || fo.ID != "file-1" || fo.Bytes != 1024 {
		t.Fatalf("UploadFile = %+v, %v (hits %v)", fo, err, hits)
	}
}

//nolint:gocyclo // inline fake of the three-endpoint upload chain; splitting it would obscure the flow.
func TestUploadLargeGoesMultipart(t *testing.T) {
	old := uploadPartSize
	uploadPartSize = 2 << 20 // 2 MiB parts so a 5 MiB file makes 3 parts
	t.Cleanup(func() { uploadPartSize = old })

	var partSizes []int64
	var completedParts []string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/uploads":
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["model"] != "orca/dub" {
				t.Errorf("create upload model = %v, want orca/dub", req["model"])
			}
			if req["filename"] == "" || req["bytes"] == float64(0) {
				t.Errorf("create upload body: %v", req)
			}
			_, _ = w.Write([]byte(`{"id":"upload-1","object":"upload","status":"pending"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/uploads/upload-1/parts":
			file, _, err := r.FormFile("data")
			if err != nil {
				t.Fatalf("part FormFile: %v", err)
			}
			n, _ := io.Copy(io.Discard, file)
			_ = file.Close()
			partSizes = append(partSizes, n)
			_, _ = fmt.Fprintf(w, `{"id":"part-%d","object":"upload.part","upload_id":"upload-1"}`, len(partSizes))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/uploads/upload-1/complete":
			var req struct {
				PartIDs []string `json:"part_ids"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			completedParts = req.PartIDs
			_, _ = w.Write([]byte(`{"id":"upload-1","object":"upload","status":"completed","file":{"id":"file-9","object":"file","bytes":5242880}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	c.directUploadLimit = 1 << 20 // 1 MiB threshold so a 5 MiB file chunks
	p := writeTempFile(t, 5<<20)
	fo, err := c.UploadFile(context.Background(), p, "")
	if err != nil || fo.ID != "file-9" {
		t.Fatalf("UploadFile = %+v, %v", fo, err)
	}
	// 5 MiB at 2 MiB parts = 2+2+1
	if len(partSizes) != 3 || partSizes[0] != 2<<20 || partSizes[2] != 1<<20 {
		t.Errorf("partSizes = %v", partSizes)
	}
	if len(completedParts) != 3 || completedParts[0] != "part-1" {
		t.Errorf("completedParts = %v", completedParts)
	}
}

func TestDownloadContent(t *testing.T) {
	payload := bytes.Repeat([]byte{0xCD}, 2048)
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		// download resolves the task first, then streams the origin content
		// route for the underlying job — always with the Bearer key attached.
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing Bearer on %s", r.URL.Path)
		}
		switch r.URL.Path {
		case "/v1/videos/job-1":
			_, _ = w.Write([]byte(`{"id":"job-1","object":"video","status":"completed","progress":100,"job_id":"uuid-9"}`))
		case "/v1/videos/uuid-9/content":
			_, _ = w.Write(payload)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	dest := filepath.Join(t.TempDir(), "out.mp4")
	n, err := c.DownloadContent(context.Background(), "job-1", dest)
	if err != nil || n != int64(len(payload)) {
		t.Fatalf("DownloadContent = %d, %v", n, err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, payload) {
		t.Error("downloaded bytes mismatch")
	}
	// second download to the same dest must refuse to overwrite
	if _, err := c.DownloadContent(context.Background(), "job-1", dest); err == nil {
		t.Error("want error when dest exists")
	}
}
