package dub

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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

// stringSlice collects a repeatable string flag (--opt a=b --opt c=d).
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
