package dub

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolLayer binds the MCP tool handlers to one Client.
type toolLayer struct {
	client *Client
}

// jsonResult renders v as pretty JSON text content — MCP clients display
// text; agents parse it back.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

// boolStr converts an MCP boolean input to the dub-server's *string
// "true"/"false" wire convention (see CreateVideoRequest docs).
func boolStr(b *bool) *string {
	if b == nil {
		return nil
	}
	s := strconv.FormatBool(*b)
	return &s
}

// HealthInput has no fields (empty object schema).
type HealthInput struct{}

// UploadInput selects a local file to push through the gateway.
type UploadInput struct {
	Path    string `json:"path" jsonschema:"absolute path of the local video file to upload"`
	Purpose string `json:"purpose,omitempty" jsonschema:"OpenAI file purpose; default user_data"`
}

// CreateInput is the dub_create tool surface — the FULL per-job parameter
// set of POST /v1/videos. The model=orca/dub gateway routing field is
// attached automatically. Required fields (source_lang, target_lang, the
// file_id/url source, video_name with file_id) must come from the USER —
// the calling agent is instructed (skill + schema descriptions) to ask
// rather than guess. Only multi-request orchestration fields (batch_id/
// batch_total/group_id) and the admin-only bench_run_id stay unexposed.
type CreateInput struct {
	SourceLang string `json:"source_lang" jsonschema:"REQUIRED — ask the user if not stated. Source language code (en zh ja ko fr de es pt ru ar it hi tr th vi id bn pl nl uk fil el cs sv da no fi sk)"`
	TargetLang string `json:"target_lang" jsonschema:"REQUIRED — ask the user if not stated. Target language code; auto is not allowed"`
	FileID     string `json:"file_id,omitempty" jsonschema:"file id returned by dub_upload; exactly one of file_id or url is REQUIRED — ask the user which source video to dub"`
	URL        string `json:"url,omitempty" jsonschema:"http(s) source video URL (YouTube etc., fetched server-side); exactly one of file_id or url is REQUIRED"`
	VideoName  string `json:"video_name,omitempty" jsonschema:"user-visible job title; REQUIRED when file_id is used (ask the user), optional for url"`
	// Content / translation knobs.
	Profile          string            `json:"profile,omitempty" jsonschema:"content preset: movie | podcast | lecture | music_video | short_drama | ad_creative; empty = generic"`
	TranslationStyle string            `json:"translation_style,omitempty" jsonschema:"formal | casual | literary | news | drama | humorous | business | cute; empty = neutral"`
	Glossary         map[string]string `json:"glossary,omitempty" jsonschema:"pinned source→target term renderings, max 64 entries"`
	AdaptIdioms      *bool             `json:"adapt_idioms,omitempty" jsonschema:"render idioms as natural target-language equivalents; empty = deploy default"`
	CometEnabled     *bool             `json:"comet_enabled,omitempty" jsonschema:"COMET translation-quality gate; empty = deploy default"`
	// Voice / TTS knobs.
	TTSBackend         string            `json:"tts_backend,omitempty" jsonschema:"TTS backend id (qwen3 | higgs); empty = deploy default"`
	SongTranslation    *bool             `json:"song_translation,omitempty" jsonschema:"dub sung segments instead of passing original audio through; empty = false"`
	ProjectID          string            `json:"project_id,omitempty" jsonschema:"dubbing project id for cross-job character voice memory"`
	SpeakerAssignments map[string]string `json:"speaker_assignments,omitempty" jsonschema:"ASR diarization label → character id map; requires project_id"`
	VoiceCloneConsent  bool              `json:"voice_clone_consent,omitempty" jsonschema:"attest the caller holds rights/consent to clone voices in the source"`
	// Audio bed / mix knobs.
	PreserveBGM     *bool   `json:"preserve_bgm,omitempty" jsonschema:"keep background music via source separation; empty = deploy default"`
	BedLevelMatch   *bool   `json:"bed_level_match,omitempty" jsonschema:"match bed loudness to broadcast level; empty = deploy default"`
	BedDuck         *bool   `json:"bed_duck,omitempty" jsonschema:"duck the bed under dialog; empty = deploy default"`
	BedReverbPreset *string `json:"bed_reverb_preset,omitempty" jsonschema:"bed reverb: none | small_room | hall | outdoor; empty = deploy default"`
	LoudnessEnabled *bool   `json:"loudness_enabled,omitempty" jsonschema:"EBU R128 loudness-match gate on the final mux; empty = deploy default"`
	// Alignment / video output knobs.
	AlignPerWord         *bool  `json:"align_per_word,omitempty" jsonschema:"per-word forced-alignment atempo; empty = deploy default"`
	Lipsync              *bool  `json:"lipsync,omitempty" jsonschema:"enable lipsync; empty = deploy default"`
	LipsyncVisemes       *bool  `json:"lipsync_visemes,omitempty" jsonschema:"viseme-aware lipsync plan; empty = deploy default"`
	LipsyncIdentityGuard *bool  `json:"lipsync_identity_guard,omitempty" jsonschema:"post-lipsync face-identity guard; empty = deploy default"`
	Watermark            *bool  `json:"watermark,omitempty" jsonschema:"burn watermark; empty = deploy default"`
	RemoveWatermark      *bool  `json:"remove_watermark,omitempty" jsonschema:"paid MPS add-on removing source logo/subtitles before dubbing; empty = OFF (extra per-minute charge)"`
	Resolution           string `json:"resolution,omitempty" jsonschema:"output height: source | 720p | 1080p | 2k; empty = 720p"`
	Ratio                string `json:"ratio,omitempty" jsonschema:"output canvas: source | 16:9 | 9:16 | 1:1; empty = source"`
	CompactOutput        *bool  `json:"compact_output,omitempty" jsonschema:"re-encode final mp4 smaller (720p cap, crf 28); empty = false"`
}

// GetInput identifies one gateway-submitted job.
type GetInput struct {
	VideoID string `json:"video_id" jsonschema:"the job id returned by dub_create (gateway task ids are gateway-scoped)"`
}

// DownloadInput saves a completed job's MP4 locally.
type DownloadInput struct {
	VideoID string `json:"video_id" jsonschema:"the completed job/video id"`
	Dest    string `json:"dest" jsonschema:"absolute local path to write the MP4 (must not already exist)"`
}

func (t *toolLayer) dubHealth(ctx context.Context, _ *mcp.CallToolRequest, _ HealthInput) (*mcp.CallToolResult, any, error) {
	h, err := t.client.Health(ctx)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(h)
}

func (t *toolLayer) dubUpload(ctx context.Context, _ *mcp.CallToolRequest, in UploadInput) (*mcp.CallToolResult, any, error) {
	fo, err := t.client.UploadFile(ctx, in.Path, in.Purpose)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(fo)
}

//nolint:gocritic // in must be a value type: mcp.AddTool infers the input schema from the In type parameter.
func (t *toolLayer) dubCreate(ctx context.Context, _ *mcp.CallToolRequest, in CreateInput) (*mcp.CallToolResult, any, error) {
	hasFile, hasURL := in.FileID != "", in.URL != ""
	if hasFile == hasURL {
		return nil, nil, fmt.Errorf("provide exactly one of file_id or url")
	}
	if hasFile && in.VideoName == "" {
		return nil, nil, fmt.Errorf("video_name is required when file_id is used (uploaded files carry no title metadata)")
	}
	req := CreateVideoRequest{
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
	}
	v, err := t.client.CreateVideo(ctx, &req)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(v)
}

func (t *toolLayer) dubGet(ctx context.Context, _ *mcp.CallToolRequest, in GetInput) (*mcp.CallToolResult, any, error) {
	v, err := t.client.GetVideo(ctx, in.VideoID)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(v)
}

func (t *toolLayer) dubDownload(ctx context.Context, _ *mcp.CallToolRequest, in DownloadInput) (*mcp.CallToolResult, any, error) {
	n, err := t.client.DownloadContent(ctx, in.VideoID, in.Dest)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(map[string]any{"video_id": in.VideoID, "dest": in.Dest, "bytes": n})
}

// RegisterTools wires every tool onto the server. The surface is exactly the
// OrcaRouter gateway's documented dub lifecycle (upload → create → poll →
// download); list/cancel/delete/native-detail are not exposed by the gateway.
// Annotations follow the MCP spec so hosts can pick approval policies:
// health/get are read-only; upload/create/download are additive
// (non-destructive) and non-idempotent; everything talks to an external
// service (open world).
func RegisterTools(s *mcp.Server, c *Client) {
	t := &toolLayer{client: c}
	fptr := func(b bool) *bool { return &b }
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	additive := &mcp.ToolAnnotations{DestructiveHint: fptr(false)}
	mcp.AddTool(s, &mcp.Tool{Name: "dub_health", Annotations: readOnly, Description: "Probe the OrcaRouter gateway → orca/dub route end to end (no job is created). Returns ok when the gateway is reachable and routes to the dubbing service."}, t.dubHealth)
	mcp.AddTool(s, &mcp.Tool{Name: "dub_upload", Annotations: additive, Description: "Upload a local video file through the OrcaRouter gateway; returns a file object whose id feeds dub_create's file_id. Large files are chunked automatically (up to 8 GiB)."}, t.dubUpload)
	mcp.AddTool(s, &mcp.Tool{Name: "dub_create", Annotations: additive, Description: "Submit a dubbing job (model orca/dub) that translates and re-voices a video from source_lang into target_lang. Source is either an uploaded file_id or a remote url (exactly one). REQUIRED fields (source_lang, target_lang, file_id/url, video_name with file_id) must be confirmed with the user — ask instead of guessing; submission is billed per minute. Returns the queued job; poll with dub_get."}, t.dubCreate)
	mcp.AddTool(s, &mcp.Tool{Name: "dub_get", Annotations: readOnly, Description: "Get a dub job's status/progress (status: queued | in_progress | completed | failed). On completed the response carries content_url (Bearer-authenticated delivery address of the mp4) and job_id; on failed an error message."}, t.dubGet)
	mcp.AddTool(s, &mcp.Tool{Name: "dub_download", Annotations: additive, Description: "Download a completed dub job's MP4 to a local path (refuses to overwrite). Delivery step: after dub_get reports completed, confirm with the user (default destination: current directory, <video_name>-<target_lang>.mp4) and then call this."}, t.dubDownload)
}
