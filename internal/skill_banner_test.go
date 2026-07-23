package dub

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

var skillBannerANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleSkillBannerText(value string) string {
	return skillBannerANSI.ReplaceAllString(value, "")
}

func TestSkillBannerLogoRows(t *testing.T) {
	t.Parallel()

	for _, color := range []bool{false, true} {
		rows := skillBannerLogoRows(color)
		if len(rows) != 20 {
			t.Fatalf("color=%v row count=%d, want 20", color, len(rows))
		}
		for index, row := range rows {
			if width := utf8.RuneCountInString(visibleSkillBannerText(row)); width != 40 {
				t.Fatalf("color=%v row=%d width=%d, want 40", color, index, width)
			}
		}
		if color && !strings.Contains(strings.Join(rows, "\n"), "\x1b[") {
			t.Fatal("color logo lacks ANSI")
		}
		if !color && strings.Contains(strings.Join(rows, "\n"), "\x1b[") {
			t.Fatal("plain logo contains ANSI")
		}
	}
}

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
