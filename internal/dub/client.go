// Package main implements orcadub-mcp, an MCP stdio server that exposes the
// OrcaDub dubbing service as MCP tools. ALL HTTP requests go through the
// OrcaRouter gateway (https://api.orcarouter.ai) — the officially documented
// integration surface (see /api-docs): the gateway routes requests to the
// orca/dub model by the `model` field, wraps billing, and scopes task ids.
// Direct dub-server routes (native /api/v1/dub/*, GET /v1/videos list,
// DELETE, cancel) are NOT exposed by the gateway and therefore not offered
// here. Wire shapes reuse internal/quality/openai
package dub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dubModel is the OrcaRouter model id the gateway uses to route requests to
// the dubbing service. Attached automatically to every create/upload call.
const dubModel = "orca/dub"

// gatewayBaseURL is the OrcaRouter gateway origin. Fixed in code by design —
// the only runtime configuration is the ORCADUB_API_KEY credential.
const gatewayBaseURL = "https://api.orcarouter.ai"

// consoleURL is where users register and mint sk-orca API keys. Unauthorized
// tool calls return it so the agent can send the user there to sign up.
const consoleURL = "https://www.orcarouter.ai/console"

// originBaseURL is the dub-server origin. Used ONLY for delivering the
// finished mp4: the gateway's own /content proxy is SSRF-blocked from its
// network and its stored presigned URL expires, while the origin's
// GET /v1/videos/{job_id}/content re-fetches the object freshly on every
// request with the same Bearer key. All submit/query traffic stays on the
// gateway; job_id comes from the gateway's task retrieve response.
const originBaseURL = "https://orcadub.orcarouter.ai"

// Config carries the MCP server's runtime configuration. The gateway origin
// is fixed in code (gatewayBaseURL); the ONLY environment input is the
// ORCADUB_API_KEY credential. BaseURL stays a struct field solely so tests
// can point the client at an httptest server.
type Config struct {
	BaseURL string // OrcaRouter gateway origin; gatewayBaseURL outside tests
	APIKey  string // REQUIRED Authorization: Bearer credential (OrcaRouter sk-orca key)
}

// LoadConfig reads ORCADUB_API_KEY. A missing key does NOT fail startup —
// the server comes up and every tool call returns errNotAuthorized instead,
// so the calling agent can redirect the user to the OrcaRouter console to
// register; once the key is set (and the session restarted) authentication
// passes automatically.
func LoadConfig() Config {
	return Config{
		BaseURL: gatewayBaseURL,
		APIKey:  os.Getenv("ORCADUB_API_KEY"),
	}
}

// Client issues authenticated HTTP calls to the OrcaRouter gateway.
type Client struct {
	cfg Config
	hc  *http.Client
	// originURL is the dub-server origin used for content delivery
	// (originBaseURL outside tests).
	originURL string
	// directUploadLimit is the size threshold (bytes) above which UploadFile
	// switches from single-request POST /v1/files to the multipart
	// /v1/uploads chain. Variable (not const) so tests can lower it.
	directUploadLimit int64
}

// NewClient builds a Client. Timeout is generous because uploads and
// downloads stream video bytes through this process.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:               cfg,
		hc:                &http.Client{Timeout: 30 * time.Minute},
		originURL:         originBaseURL,
		directUploadLimit: 64 << 20, // above 64MiB use the chunked path (gateway UploadPartMaxSize)
	}
}

// errNotAuthorized guides the user through OrcaRouter sign-up. Returned by
// every tool call while ORCADUB_API_KEY is unset.
var errNotAuthorized = fmt.Errorf("not authorized: ORCADUB_API_KEY is not set. "+
	"Send the user to %s to register and create an sk-orca-... API key, "+
	"then set ORCADUB_API_KEY in the environment and restart the session — "+
	"authentication passes automatically once the key is present", consoleURL)

// ensureAuth gates every outbound call on OrcaRouter authorization.
func (c *Client) ensureAuth() error {
	if c.cfg.APIKey == "" {
		return errNotAuthorized
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	// The gateway resolves workspace + billing from the Bearer alone.
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
}

// apiError decodes the error envelopes seen on the gateway path —
// OpenAI-style {"error":{"message":...}} (gateway routing errors) and the
// OrcaRouter task envelope {"code":..,"message":..,"data":..} (dub task
// errors) — into a readable Go error carrying the HTTP status.
func apiError(status int, raw []byte) error {
	var oa struct {
		Error struct {
			Message string `json:"message"`
			Param   string `json:"param"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &oa) == nil && oa.Error.Message != "" {
		if oa.Error.Param != "" {
			return fmt.Errorf("orcarouter HTTP %d (param %s): %s", status, oa.Error.Param, oa.Error.Message)
		}
		return fmt.Errorf("orcarouter HTTP %d: %s", status, oa.Error.Message)
	}
	var task struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &task) == nil && task.Message != "" {
		return fmt.Errorf("orcarouter HTTP %d: %s", status, task.Message)
	}
	body := string(raw)
	if len(body) > 300 {
		body = body[:300]
	}
	return fmt.Errorf("orcarouter HTTP %d: %s", status, body)
}

// do sends one request and returns the raw body. Non-2xx becomes an error via
// apiError. body may be nil.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build %s %s: %w", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.setAuth(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s %s response: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp.StatusCode, raw)
	}
	return raw, nil
}

// doJSON marshals in (when non-nil) as the JSON request body and unmarshals
// the response into out (when non-nil).
func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	contentType := ""
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal %s %s body: %w", method, path, err)
		}
		body = bytes.NewReader(b)
		contentType = "application/json"
	}
	raw, err := c.do(ctx, method, path, body, contentType)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

// HealthResult is dub_health's report.
type HealthResult struct {
	Status  string `json:"status"` // ok | degraded
	Gateway string `json:"gateway"`
	Detail  string `json:"detail"`
}

// Health probes the gateway → orca/dub route end to end without creating a
// job: POST /v1/videos with only the model field must come back as the dub
// service's own "source_lang is required" validation error. Any other reply
// (gateway "Model name not specified", auth failure, network error) means
// the chain is broken and is surfaced verbatim.
func (c *Client) Health(ctx context.Context) (HealthResult, error) {
	err := c.doJSON(ctx, http.MethodPost, "/v1/videos", map[string]string{"model": dubModel}, nil)
	if err == nil {
		// A 2xx on an empty submit would mean a job got created — treat as
		// unexpected rather than healthy so nobody gets silently billed.
		return HealthResult{}, fmt.Errorf("health probe unexpectedly succeeded — gateway accepted an empty submit")
	}
	if strings.Contains(err.Error(), "source_lang") {
		return HealthResult{
			Status:  "ok",
			Gateway: c.cfg.BaseURL,
			Detail:  "gateway reachable, request routed to " + dubModel + " (validation probe)",
		}, nil
	}
	return HealthResult{}, err
}

// GatewayVideo is Video plus gateway-specific fields: job_id (the
// underlying dub job uuid the gateway maps the task to) and content_url
// (the delivery address of the finished mp4 — the gateway retrieve strips
// output_url by design, so the client derives the origin content route
// from job_id; the origin re-signs the object freshly on every request).
type GatewayVideo struct {
	Video
	JobID string `json:"job_id,omitempty"`
	// ContentURL is the delivery address of the finished video. It requires
	// the same Authorization: Bearer header as every other call
	// (curl -H "Authorization: Bearer sk-orca-..." <content_url>), or use
	// dub_download which sends it automatically.
	ContentURL string `json:"content_url,omitempty"`
}

// GetVideo calls GET /v1/videos/{id} on the gateway. The id must come from a
// job submitted through the gateway (gateway task ids are its own scope).
// On completed jobs the response gains content_url pointing at the origin's
// content route for the underlying job.
func (c *Client) GetVideo(ctx context.Context, id string) (GatewayVideo, error) {
	var v GatewayVideo
	if err := c.doJSON(ctx, http.MethodGet, "/v1/videos/"+url.PathEscape(id), nil, &v); err != nil {
		return v, err
	}
	if v.Status == "completed" && v.JobID != "" {
		v.ContentURL = c.originURL + "/v1/videos/" + url.PathEscape(v.JobID) + "/content"
	}
	return v, nil
}

// CreateVideo calls POST /v1/videos through the gateway, forcing the
// model=orca/dub routing field.
func (c *Client) CreateVideo(ctx context.Context, req *CreateVideoRequest) (Video, error) {
	req.Model = dubModel
	var v Video
	err := c.doJSON(ctx, http.MethodPost, "/v1/videos", req, &v)
	return v, err
}

// uploadPartSize is the chunk size for the /v1/uploads path. 32 MiB sits
// comfortably under the gateway's 64 MiB part cap and above the 1 MiB
// non-last-part minimum; the 250-part cap * 32 MiB covers the 8 GiB total
// limit. Variable so tests can shrink it.
var uploadPartSize int64 = 32 << 20

// createUploadRequest is the gateway variant of CreateUploadRequest:
// same fields plus the model routing discriminator the gateway requires on
// every entry-point request.
type createUploadRequest struct {
	Model    string `json:"model"`
	Filename string `json:"filename"`
	Purpose  string `json:"purpose"`
	Bytes    int64  `json:"bytes"`
	MimeType string `json:"mime_type"`
}

// UploadFile pushes a local file through the gateway and returns the
// resulting FileObject whose ID feeds dub_create's file_id. Files at or
// under directUploadLimit go through single-request POST /v1/files; larger
// files walk the OpenAI Uploads chain (create → parts → complete).
func (c *Client) UploadFile(ctx context.Context, path, purpose string) (FileObject, error) {
	if purpose == "" {
		purpose = "user_data"
	}
	f, err := os.Open(path)
	if err != nil {
		return FileObject{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return FileObject{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if st.Size() <= c.directUploadLimit {
		return c.uploadDirect(ctx, f, purpose)
	}
	return c.uploadChunked(ctx, f, st.Size(), purpose)
}

// uploadDirect streams the whole file as one POST /v1/files multipart
// request (form fields: file, purpose, model) via io.Pipe so the body never
// buffers in RAM.
func (c *Client) uploadDirect(ctx context.Context, f *os.File, purpose string) (FileObject, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		var werr error
		defer func() { _ = pw.CloseWithError(werr) }()
		part, err := mw.CreateFormFile("file", filepath.Base(f.Name()))
		if err != nil {
			werr = err
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			werr = err
			return
		}
		if err := mw.WriteField("purpose", purpose); err != nil {
			werr = err
			return
		}
		if err := mw.WriteField("model", dubModel); err != nil {
			werr = err
			return
		}
		werr = mw.Close()
	}()
	raw, err := c.do(ctx, http.MethodPost, "/v1/files", pr, mw.FormDataContentType())
	if err != nil {
		return FileObject{}, err
	}
	var fo FileObject
	if err := json.Unmarshal(raw, &fo); err != nil {
		return FileObject{}, fmt.Errorf("decode /v1/files response: %w", err)
	}
	return fo, nil
}

// uploadChunked walks POST /v1/uploads → N× parts (form field "data") →
// complete. Sequential on purpose — MCP tool latency is dominated by the dub
// pipeline, not upload concurrency.
func (c *Client) uploadChunked(ctx context.Context, f *os.File, size int64, purpose string) (FileObject, error) {
	createBody := createUploadRequest{
		Model:    dubModel,
		Filename: filepath.Base(f.Name()),
		Purpose:  purpose,
		Bytes:    size,
		MimeType: mimeTypeByExt(f.Name()),
	}
	var up Upload
	if err := c.doJSON(ctx, http.MethodPost, "/v1/uploads", createBody, &up); err != nil {
		return FileObject{}, fmt.Errorf("create upload: %w", err)
	}
	var partIDs []string
	buf := make([]byte, uploadPartSize)
	for {
		n, rerr := io.ReadFull(f, buf)
		if rerr == io.EOF {
			break
		}
		if rerr != nil && rerr != io.ErrUnexpectedEOF {
			return FileObject{}, fmt.Errorf("read part: %w", rerr)
		}
		part, err := c.uploadOnePart(ctx, up.ID, buf[:n])
		if err != nil {
			return FileObject{}, fmt.Errorf("upload part %d: %w", len(partIDs)+1, err)
		}
		partIDs = append(partIDs, part.ID)
		if rerr == io.ErrUnexpectedEOF {
			break
		}
	}
	var done Upload
	completeBody := CompleteUploadRequest{PartIDs: partIDs}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/uploads/"+url.PathEscape(up.ID)+"/complete", completeBody, &done); err != nil {
		return FileObject{}, fmt.Errorf("complete upload: %w", err)
	}
	if done.File == nil {
		return FileObject{}, fmt.Errorf("upload completed but no file object returned")
	}
	return *done.File, nil
}

// uploadOnePart posts one chunk as multipart form field "data".
func (c *Client) uploadOnePart(ctx context.Context, uploadID string, chunk []byte) (UploadPart, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("data", "part")
	if err != nil {
		return UploadPart{}, err
	}
	if _, err := part.Write(chunk); err != nil {
		return UploadPart{}, err
	}
	if err := mw.Close(); err != nil {
		return UploadPart{}, err
	}
	raw, err := c.do(ctx, http.MethodPost, "/v1/uploads/"+url.PathEscape(uploadID)+"/parts", &body, mw.FormDataContentType())
	if err != nil {
		return UploadPart{}, err
	}
	var p UploadPart
	if err := json.Unmarshal(raw, &p); err != nil {
		return UploadPart{}, fmt.Errorf("decode part response: %w", err)
	}
	return p, nil
}

// mimeTypeByExt maps common video extensions; the gateway only uses this for
// bookkeeping, so a generic fallback is fine.
func mimeTypeByExt(name string) string {
	if mt := mime.TypeByExtension(filepath.Ext(name)); mt != "" {
		return mt
	}
	return "application/octet-stream"
}

// DownloadContent streams the finished mp4 into dest and returns the byte
// count. It resolves the gateway task id to the underlying job first (the
// origin content route is the reliable delivery path), then streams from
// the URL GetVideo derived. Refuses to overwrite an existing file.
func (c *Client) DownloadContent(ctx context.Context, id, dest string) (int64, error) {
	if err := c.ensureAuth(); err != nil {
		return 0, err
	}
	v, err := c.GetVideo(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("resolve task %s: %w", id, err)
	}
	if v.ContentURL == "" {
		return 0, fmt.Errorf("task %s has no downloadable content yet (status %s)", id, v.Status)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.ContentURL, http.NoBody)
	if err != nil {
		return 0, err
	}
	c.setAuth(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return 0, apiError(resp.StatusCode, raw)
	}
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", dest, err)
	}
	n, cerr := io.Copy(out, resp.Body)
	if err := out.Close(); cerr == nil {
		cerr = err
	}
	if cerr != nil {
		return n, fmt.Errorf("write %s: %w", dest, cerr)
	}
	return n, nil
}
