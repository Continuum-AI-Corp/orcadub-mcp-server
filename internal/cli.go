package dub

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/term"
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
//
//nolint:gocyclo // flat key→field dispatch; splitting the switch would obscure the 1:1 option mapping.
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
  orcadub skill install [--platform <id> ...] [--scope project|global] [--lang zh|en] [--yes] [--force] [--json]

Auth: set ORCADUB_API_KEY (sk-orca-... from https://www.orcarouter.ai/console).
Skill installation does not require an API key.
With no subcommand the binary runs as an MCP stdio server.`

const skillInstallUsage = `Usage:
  orcadub skill install [options]

Options:
  --platform <id>          target platform (repeatable)
  --scope project|global   install under the current project or home directory
  --lang zh|en             guidance language: Simplified Chinese or English
  --yes                    accept defaults; detected platforms, or all when none are detected
  --force                  replace an existing different OrcaDub Skill
  --json                   print a structured report and do not prompt`

var (
	skillCLIInstaller    = newSkillInstaller
	skillCLIWorkingDir   = os.Getwd
	skillCLIHomeDir      = os.UserHomeDir
	skillCLILookPath     = exec.LookPath
	skillCLIPromptRunner = func() skillPromptRunner { return huhSkillPromptRunner{} }
	skillCLIIsTerminal   = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}
	skillCLIGetenv    = os.Getenv
	skillCLILookupEnv = os.LookupEnv
)

type skillCLIOptions struct {
	platformIDs   []string
	scopeValue    string
	languageValue string
	yes           bool
	force         bool
	jsonOutput    bool
}

// RunCLI executes one CLI subcommand. args is os.Args[1:] (args[0] is the
// subcommand). Success prints result JSON to stdout and returns 0; failures
// print to stderr and return 1; an unknown subcommand returns 2.
//
//nolint:gocyclo // subcommand dispatch; one case per CLI verb is clearer flat than split.
func RunCLI(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, cliUsage)
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	if cmd == "skill" {
		return runSkillCLI(rest)
	}
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
		req, err := buildCreateRequest(&in)
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
		_, _ = fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s\n", cmd, cliUsage)
		return 2
	}
}

func runSkillCLI(args []string) int {
	if handled, code := handleSkillCLICommandPreamble(args); handled {
		return code
	}

	options, err := parseSkillCLIOptions(args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	selection, selectionCode, err := resolveSkillCLISelection(options)
	if selectionCode != 0 {
		if err != nil {
			if selectionCode == 2 {
				_, _ = fmt.Fprintln(os.Stderr, err.Error())
			} else {
				return fail(err)
			}
		}
		return selectionCode
	}

	projectDir, homeDir, err := resolveSkillCLIBaseDirs(selection.Scope, false)
	if err != nil {
		return fail(err)
	}

	report, err := skillCLIInstaller().install(
		context.Background(),
		selection.PlatformIDs,
		selection.Scope,
		projectDir,
		homeDir,
		options.force,
	)
	if err != nil {
		if !options.jsonOutput {
			err = fmt.Errorf("%s: %w", skillText(selection.Language, skillTextInstallFailed), err)
		}
		return fail(err)
	}
	if options.jsonOutput {
		data, marshalErr := jsonResultBytes(report)
		if marshalErr != nil {
			return fail(marshalErr)
		}
		_, _ = fmt.Fprintln(os.Stdout, string(data))
	} else {
		renderSkillInstallReport(report, selection.Language)
	}
	if skillInstallReportFailed(report) {
		return 1
	}
	return 0
}

func resolveSkillCLISelection(options skillCLIOptions) (skillPromptResult, int, error) {
	language := defaultSkillLanguage(skillCLIGetenv)
	var err error
	if options.languageValue != "" {
		language, err = parseSkillLanguage(options.languageValue)
		if err != nil {
			return skillPromptResult{}, 2, err
		}
	}

	scope, err := parseSkillInstallScope(options.scopeValue)
	if err != nil {
		return skillPromptResult{}, 2, err
	}

	switch {
	case len(options.platformIDs) > 0:
		selected, validationErr := validateSkillPlatformIDs(options.platformIDs)
		return skillPromptResult{
			Language:    language,
			Scope:       scope,
			PlatformIDs: selected,
		}, errorExitCode(validationErr, 2), validationErr
	case options.yes || options.jsonOutput:
		return resolveNonInteractiveSkillSelection(language, scope)
	default:
		return resolveInteractiveSkillSelection(options, language, scope)
	}
}

func resolveNonInteractiveSkillSelection(
	language skillLanguage,
	scope skillInstallScope,
) (skillPromptResult, int, error) {
	projectDir, _, err := resolveSkillCLIBaseDirs(scope, true)
	if err != nil {
		return skillPromptResult{}, 1, err
	}
	selected := detectSkillCLIPlatforms(projectDir)
	if len(selected) == 0 {
		selected = allSkillPlatformIDs()
	}
	return skillPromptResult{
		Language:    language,
		Scope:       scope,
		PlatformIDs: selected,
	}, 0, nil
}

func resolveInteractiveSkillSelection(
	options skillCLIOptions,
	language skillLanguage,
	scope skillInstallScope,
) (skillPromptResult, int, error) {
	if !skillCLIIsTerminal() {
		return skillPromptResult{}, 2, fmt.Errorf(
			"%s",
			skillText(language, skillTextNonTTYGuidance),
		)
	}
	detectionDir, err := skillCLIWorkingDir()
	if err != nil {
		return skillPromptResult{}, 1, fmt.Errorf("resolve current directory: %w", err)
	}

	detected := detectSkillCLIPlatforms(detectionDir)
	colorEnabled := skillCLIColorEnabled(skillCLILookupEnv)
	renderSkillBanner(os.Stdout, colorEnabled)
	result, err := skillCLIPromptRunner().Run(&skillPromptRequest{
		Language:        language,
		AskLanguage:     options.languageValue == "",
		Scope:           scope,
		AskScope:        options.scopeValue == "",
		PlatformOptions: orderedSkillPromptPlatforms(detected),
		AskPlatforms:    true,
		ColorEnabled:    colorEnabled,
		Input:           os.Stdin,
		Output:          os.Stdout,
	})
	if errors.Is(err, huh.ErrUserAborted) {
		return skillPromptResult{}, 130, nil
	}
	if err != nil {
		return skillPromptResult{}, 1, err
	}

	result.PlatformIDs, err = validateSkillPlatformIDs(result.PlatformIDs)
	return result, errorExitCode(err, 2), err
}

func detectSkillCLIPlatforms(projectDir string) []string {
	homeDir, _ := skillCLIHomeDir()
	return detectSkillPlatforms(projectDir, homeDir, skillCLILookPath)
}

func skillCLIColorEnabled(lookupEnv func(string) (string, bool)) bool {
	if _, disabled := lookupEnv("NO_COLOR"); disabled {
		return false
	}
	termName, _ := lookupEnv("TERM")
	return !strings.EqualFold(strings.TrimSpace(termName), "dumb")
}

func errorExitCode(err error, code int) int {
	if err != nil {
		return code
	}
	return 0
}

func parseSkillInstallScope(raw string) (skillInstallScope, error) {
	if raw == "" {
		return skillInstallProject, nil
	}
	scope := skillInstallScope(raw)
	if scope != skillInstallProject && scope != skillInstallGlobal {
		return "", fmt.Errorf("unknown install scope %q (use project or global)", raw)
	}
	return scope, nil
}

func resolveSkillCLIBaseDirs(
	scope skillInstallScope,
	needDetection bool,
) (projectDir, homeDir string, err error) {
	if scope == skillInstallProject {
		projectDir, err = skillCLIWorkingDir()
		if err != nil {
			return "", "", fmt.Errorf("resolve current directory: %w", err)
		}
		return projectDir, "", nil
	}

	homeDir, err = skillCLIHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve home directory: %w", err)
	}
	if needDetection {
		// Detection is only a convenience for global installs. If cwd is
		// unavailable, the platform selector can still use explicit/all choices.
		projectDir, _ = skillCLIWorkingDir()
	}
	return projectDir, homeDir, nil
}

func handleSkillCLICommandPreamble(args []string) (handled bool, exitCode int) {
	if len(args) == 0 || args[0] != "install" {
		if len(args) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, "skill: missing command\n\n"+skillInstallUsage)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "skill: unknown command %q\n\n%s\n", args[0], skillInstallUsage)
		}
		return true, 2
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h" || args[1] == "help") {
		_, _ = fmt.Fprintln(os.Stdout, skillInstallUsage)
		return true, 0
	}
	return false, 0
}

func parseSkillCLIOptions(args []string) (skillCLIOptions, error) {
	flags := flag.NewFlagSet("skill install", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := skillCLIOptions{}
	var platformIDs stringSlice
	flags.Var(&platformIDs, "platform", "target platform ID (repeatable)")
	flags.StringVar(&options.scopeValue, "scope", "", "install scope: project or global")
	flags.StringVar(&options.languageValue, "lang", "", "guidance language: zh or en")
	flags.BoolVar(&options.yes, "yes", false, "accept detected/default targets without prompting")
	flags.BoolVar(&options.force, "force", false, "replace an existing different Skill")
	flags.BoolVar(&options.jsonOutput, "json", false, "print a structured JSON report")
	if err := flags.Parse(args); err != nil {
		return skillCLIOptions{}, err
	}
	if flags.NArg() != 0 {
		return skillCLIOptions{}, fmt.Errorf(
			"skill install: unexpected arguments: %s",
			strings.Join(flags.Args(), " "),
		)
	}
	if options.languageValue != "" {
		if _, err := parseSkillLanguage(options.languageValue); err != nil {
			return skillCLIOptions{}, err
		}
	}
	options.platformIDs = append([]string(nil), platformIDs...)
	return options, nil
}

func allSkillPlatformIDs() []string {
	all := make([]string, 0, len(skillPlatforms))
	for index := range skillPlatforms {
		all = append(all, skillPlatforms[index].ID)
	}
	return all
}

func validateSkillPlatformIDs(ids []string) ([]string, error) {
	selected := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if _, ok := findSkillPlatform(id); !ok {
			return nil, fmt.Errorf("unknown platform %q", id)
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		selected = append(selected, id)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("select at least one platform")
	}
	return selected, nil
}

func renderSkillInstallReport(report skillInstallReport, language skillLanguage) {
	_, _ = fmt.Fprintln(os.Stdout, skillText(language, skillTextResultTitle))
	_, _ = fmt.Fprintf(
		os.Stdout,
		"%s: %s\n%s: %s\n",
		skillText(language, skillTextResultScope),
		skillInstallScopeText(language, report.Scope),
		skillText(language, skillTextResultSource),
		report.Source,
	)
	for _, result := range report.Results {
		platforms := strings.Join(result.PlatformNames, ", ")
		switch result.Status {
		case skillInstallStatusConflict:
			_, _ = fmt.Fprintf(
				os.Stdout,
				"%-9s %s -> %s (%s)\n",
				skillText(language, skillTextStatusConflict),
				platforms,
				result.Path,
				skillText(language, skillTextConflictGuidance),
			)
		case skillInstallStatusError:
			_, _ = fmt.Fprintf(
				os.Stdout,
				"%-9s %s -> %s: %s\n",
				skillText(language, skillTextStatusError),
				platforms,
				result.Path,
				result.Error,
			)
		default:
			_, _ = fmt.Fprintf(
				os.Stdout,
				"%-9s %s -> %s\n",
				skillInstallStatusText(language, result.Status),
				platforms,
				result.Path,
			)
		}
	}
}

func skillInstallScopeText(language skillLanguage, scope skillInstallScope) string {
	switch scope {
	case skillInstallProject:
		return skillText(language, skillTextScopeProject)
	case skillInstallGlobal:
		return skillText(language, skillTextScopeGlobal)
	default:
		return string(scope)
	}
}

func skillInstallStatusText(language skillLanguage, status skillInstallStatus) string {
	switch status {
	case skillInstallStatusInstalled:
		return skillText(language, skillTextStatusInstalled)
	case skillInstallStatusUpdated:
		return skillText(language, skillTextStatusUpdated)
	case skillInstallStatusUnchanged:
		return skillText(language, skillTextStatusUnchanged)
	default:
		return string(status)
	}
}

func skillInstallReportFailed(report skillInstallReport) bool {
	for _, result := range report.Results {
		if result.Status == skillInstallStatusConflict || result.Status == skillInstallStatusError {
			return true
		}
	}
	return false
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
	_, _ = fmt.Fprintln(os.Stdout, string(b))
	return 0
}

// fail prints err to stderr and returns exit code 1.
func fail(err error) int {
	_, _ = fmt.Fprintln(os.Stderr, err.Error())
	return 1
}

// stringSlice collects a repeatable string flag (--opt a=b --opt c=d).
type stringSlice []string

// String renders the collected values (flag.Value interface).
func (s *stringSlice) String() string { return strings.Join(*s, ",") }

// Set appends one occurrence of the flag (flag.Value interface).
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
