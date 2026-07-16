package orcadub

// Wire types for the OrcaDub surface behind the OrcaRouter gateway. These
// mirror OpenAI's Files / Uploads / Videos objects plus the dub-specific
// extra_body fields, and are kept in lockstep with the server's
// internal/quality/openaicompat package (the source of truth).

// FileObject mirrors OpenAI's FileObject (POST /files).
type FileObject struct {
	ID        string `json:"id"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
	Filename  string `json:"filename"`
	Object    string `json:"object"`
	Purpose   string `json:"purpose"`
	Status    string `json:"status"`
	ExpiresAt *int64 `json:"expires_at,omitempty"`
}

// Video mirrors OpenAI's Video object (POST /videos / GET /videos/{id})
// with the OrcaDub extension fields the gateway serves.
type Video struct {
	ID              string      `json:"id"`
	Name            string      `json:"name,omitempty"`
	Object          string      `json:"object"`
	Status          string      `json:"status"` // queued|in_progress|completed|failed
	Progress        int         `json:"progress"`
	CreatedAt       int64       `json:"created_at"`
	CompletedAt     *int64      `json:"completed_at,omitempty"`
	StartedAt       *int64      `json:"started_at,omitempty"`
	EndedAt         *int64      `json:"ended_at,omitempty"`
	ExpiresAt       *int64      `json:"expires_at,omitempty"`
	Model           string      `json:"model"`
	Prompt          string      `json:"prompt"`
	Seconds         string      `json:"seconds"`
	Size            string      `json:"size"`
	SourceLang      string      `json:"source_lang,omitempty"`
	TargetLang      string      `json:"target_lang,omitempty"`
	DurationSeconds *int32      `json:"duration_seconds,omitempty"`
	OutputURL       string      `json:"output_url,omitempty"`
	Error           *VideoError `json:"error,omitempty"`
}

// VideoError matches OpenAI's VideoCreateError envelope.
type VideoError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateVideoRequest is the JSON body of POST /videos. Dub-specific knobs
// ride along as extra_body fields; boolean knobs are *string "true"/"false"
// on the wire (the server treats nil/empty as "use deploy default").
type CreateVideoRequest struct {
	Prompt                 string            `json:"prompt,omitempty"`
	VideoPath              *VideoPath        `json:"video_path,omitempty"`
	Model                  string            `json:"model,omitempty"`
	Seconds                string            `json:"seconds,omitempty"`
	Size                   string            `json:"size,omitempty"`
	SourceLang             string            `json:"source_lang,omitempty"`
	TargetLang             string            `json:"target_lang,omitempty"`
	Profile                string            `json:"profile,omitempty"`
	Lipsync                *string           `json:"lipsync,omitempty"`
	LipsyncVisemes         *string           `json:"lipsync_visemes,omitempty"`
	LipsyncIdentityGuard   *string           `json:"lipsync_identity_guard,omitempty"`
	PreserveBGM            *string           `json:"preserve_bgm,omitempty"`
	Watermark              *string           `json:"watermark,omitempty"`
	RemoveWatermark        *string           `json:"remove_watermark,omitempty"`
	BedLevelMatch          *string           `json:"bed_level_match,omitempty"`
	BedDuck                *string           `json:"bed_duck,omitempty"`
	BedReverbPreset        *string           `json:"bed_reverb_preset,omitempty"`
	AlignPerWord           *string           `json:"align_per_word,omitempty"`
	CometEnabled           *string           `json:"comet_enabled,omitempty"`
	LoudnessEnabled        *string           `json:"loudness_enabled,omitempty"`
	AdaptIdiomsEnabled     *string           `json:"adapt_idioms_enabled,omitempty"`
	SongTranslationEnabled *string           `json:"song_translation_enabled,omitempty"`
	CompactOutput          *string           `json:"compact_output,omitempty"`
	Resolution             string            `json:"resolution,omitempty"`
	Ratio                  string            `json:"ratio,omitempty"`
	ProjectID              string            `json:"project_id,omitempty"`
	SpeakerAssignments     map[string]string `json:"speaker_assignments,omitempty"`
	Glossary               map[string]string `json:"glossary,omitempty"`
	VoiceCloneConsent      bool              `json:"voice_clone_consent,omitempty"`
	TTSBackend             string            `json:"tts_backend,omitempty"`
	TranslationStyle       string            `json:"translation_style,omitempty"`
	VideoName              string            `json:"video_name,omitempty"`
}

// VideoPath carries the source video reference: exactly one of file_id
// (a previously uploaded file) or url (http(s) remote video, fetched
// server-side).
type VideoPath struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

// Upload mirrors OpenAI's Upload object (POST /v1/uploads chain). Once
// status="completed", File carries the materialised FileObject whose id
// feeds video_path.file_id.
type Upload struct {
	ID        string      `json:"id"`
	Object    string      `json:"object"`
	Bytes     int64       `json:"bytes"`
	CreatedAt int64       `json:"created_at"`
	ExpiresAt int64       `json:"expires_at"`
	Filename  string      `json:"filename"`
	Purpose   string      `json:"purpose"`
	Status    string      `json:"status"`
	File      *FileObject `json:"file,omitempty"`
}

// UploadPart mirrors OpenAI's UploadPart object (POST /v1/uploads/{id}/parts).
type UploadPart struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	UploadID  string `json:"upload_id"`
	CreatedAt int64  `json:"created_at"`
}

// CompleteUploadRequest is the JSON body of POST /v1/uploads/{id}/complete.
type CompleteUploadRequest struct {
	PartIDs []string `json:"part_ids"`
	MD5     string   `json:"md5,omitempty"`
}
