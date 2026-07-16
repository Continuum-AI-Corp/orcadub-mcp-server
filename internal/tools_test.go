package dub

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDubCreateToolValidatesInput(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"job-1","status":"queued"}`))
	})
	tl := &toolLayer{client: c}
	// both file_id and url -> reject before any HTTP call
	_, _, err := tl.dubCreate(context.Background(), nil, CreateInput{
		SourceLang: "en", TargetLang: "zh", FileID: "file-1", URL: "https://x/v.mp4",
	})
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("want XOR validation error, got %v", err)
	}
	// file_id without video_name -> reject (server would 400 later anyway)
	_, _, err = tl.dubCreate(context.Background(), nil, CreateInput{
		SourceLang: "en", TargetLang: "zh", FileID: "file-1",
	})
	if err == nil || !strings.Contains(err.Error(), "video_name") {
		t.Fatalf("want video_name error, got %v", err)
	}
}

func TestDubCreateToolBoolKnobs(t *testing.T) {
	var got map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"id":"job-1","status":"queued"}`))
	})
	tl := &toolLayer{client: c}
	tr := true
	res, _, err := tl.dubCreate(context.Background(), nil, CreateInput{
		SourceLang: "en", TargetLang: "zh", URL: "https://x/v.mp4",
		PreserveBGM: &tr,
	})
	if err != nil {
		t.Fatalf("dubCreate: %v", err)
	}
	if got["preserve_bgm"] != "true" {
		t.Errorf("preserve_bgm on wire = %v, want string \"true\"", got["preserve_bgm"])
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("want non-empty result content")
	}
}

func TestDubGetTool(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"job-1","status":"completed","progress":100,"output_url":"https://cos/x.mp4"}`))
	})
	tl := &toolLayer{client: c}
	res, _, err := tl.dubGet(context.Background(), nil, GetInput{VideoID: "job-1"})
	if err != nil {
		t.Fatalf("dubGet: %v", err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "completed") || !strings.Contains(text, "output_url") {
		t.Errorf("result text = %s", text)
	}
}
