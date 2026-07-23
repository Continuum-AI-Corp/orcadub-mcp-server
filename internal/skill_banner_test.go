package dub

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var skillBannerANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleSkillBannerText(value string) string {
	return skillBannerANSI.ReplaceAllString(value, "")
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
			if width := lipgloss.Width(row); width != skillBannerWordWidth {
				t.Fatalf(
					"color=%v row=%d width=%d, want %d",
					color,
					index,
					width,
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
			width := lipgloss.Width(row)
			if width != skillBannerWordWidth {
				t.Fatalf(
					"color=%v row=%d width=%d, want %d",
					color,
					index,
					width,
					skillBannerWordWidth,
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
