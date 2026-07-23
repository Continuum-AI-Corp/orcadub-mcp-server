package dub

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderSkillBannerPlain(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	renderSkillBanner(&out, false)
	got := out.String()
	if !strings.Contains(got, "ORCA//DUB") ||
		!strings.Contains(got, "SKILL INSTALLER / 技能安装器") {
		t.Fatalf("banner = %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain banner contains ANSI: %q", got)
	}
}

func TestRenderSkillBannerColor(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	renderSkillBanner(&out, true)
	got := out.String()
	if !strings.Contains(got, "\x1b[") || !strings.Contains(got, "ORCA//DUB") {
		t.Fatalf("colored banner = %q", got)
	}
}
