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
