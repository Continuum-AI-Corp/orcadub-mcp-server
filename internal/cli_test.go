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
