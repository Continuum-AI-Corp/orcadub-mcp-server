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

func TestSkillBannerPlainAssetHasNoTrailingWhitespace(t *testing.T) {
	t.Parallel()

	rows := strings.Split(strings.TrimSuffix(skillBannerLogoPlain, "\n"), "\n")
	for index, row := range rows {
		if strings.HasSuffix(row, " ") || strings.HasSuffix(row, "\t") {
			t.Fatalf("plain asset row %d has trailing whitespace", index)
		}
	}
}

func TestSkillBannerWordmarkRows(t *testing.T) {
	t.Parallel()

	for _, color := range []bool{false, true} {
		rows := skillBannerWordmarkRows(color)
		if len(rows) != skillBannerHeight {
			t.Fatalf("color=%v row count=%d, want %d", color, len(rows), skillBannerHeight)
		}
		visibleRows := make([][]rune, len(rows))
		for index, row := range rows {
			visibleRows[index] = []rune(visibleSkillBannerText(row))
			if len(visibleRows[index]) != skillBannerWordWidth {
				t.Fatalf(
					"color=%v row=%d width=%d, want %d",
					color,
					index,
					len(visibleRows[index]),
					skillBannerWordWidth,
				)
			}
		}
		for glyphIndex := range []rune("ORCADUB") {
			start := glyphIndex * 5
			hasInk := false
			for _, row := range visibleRows {
				for _, cell := range row[start : start+4] {
					hasInk = hasInk || cell != ' '
				}
			}
			if !hasInk {
				t.Fatalf("color=%v glyph %d has no ink", color, glyphIndex)
			}
		}
	}
}

func TestRenderSkillBannerLayout(t *testing.T) {
	t.Parallel()

	for _, color := range []bool{false, true} {
		var output bytes.Buffer
		renderSkillBanner(&output, color)
		value := output.String()
		rows := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
		if len(rows) != skillBannerHeight {
			t.Fatalf("color=%v row count=%d, want %d", color, len(rows), skillBannerHeight)
		}
		for index, row := range rows {
			width := utf8.RuneCountInString(visibleSkillBannerText(row))
			if width != skillBannerTotalWidth {
				t.Fatalf(
					"color=%v row=%d width=%d, want %d",
					color,
					index,
					width,
					skillBannerTotalWidth,
				)
			}
		}
		if strings.Contains(value, "AI DUBBING CLI") ||
			strings.Contains(value, "SKILL INSTALLER") ||
			strings.Contains(value, "技能安装器") {
			t.Fatalf("color=%v banner contains removed subtitle", color)
		}
		if color != strings.Contains(value, "\x1b[") {
			t.Fatalf("color=%v ANSI presence mismatch", color)
		}
	}
}
