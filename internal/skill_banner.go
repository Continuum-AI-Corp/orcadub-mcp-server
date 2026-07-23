package dub

import (
	"io"
	"strings"
)

const (
	skillBannerWord  = "ORCADUB"
	skillBannerBlue  = "\x1b[94m"
	skillBannerCyan  = "\x1b[96m"
	skillBannerReset = "\x1b[0m"
)

var skillBannerWordmarkGlyphs = map[rune][7]string{
	'O': {" ██ ", "█  █", "█  █", "█  █", "█  █", "█  █", " ██ "},
	'R': {"███ ", "█  █", "█  █", "███ ", "█ █ ", "█  █", "█  █"},
	'C': {" ███", "█   ", "█   ", "█   ", "█   ", "█   ", " ███"},
	'A': {" ██ ", "█  █", "█  █", "████", "█  █", "█  █", "█  █"},
	'D': {"███ ", "█  █", "█  █", "█  █", "█  █", "█  █", "███ "},
	'U': {"█  █", "█  █", "█  █", "█  █", "█  █", "█  █", " ██ "},
	'B': {"███ ", "█  █", "█  █", "███ ", "█  █", "█  █", "███ "},
}

var skillBannerWordmarkRowRepeats = [7]int{3, 3, 3, 3, 3, 3, 2}

func skillBannerWordmarkRows(color bool) []string {
	rows := make([]string, 0, skillBannerHeight)
	for glyphRow, repeats := range skillBannerWordmarkRowRepeats {
		var builder strings.Builder
		for index, letter := range skillBannerWord {
			if index > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(skillBannerWordmarkGlyphs[letter][glyphRow])
		}
		plain := builder.String()
		for range repeats {
			row := plain
			if color {
				tone := skillBannerBlue
				if len(rows) >= skillBannerHeight/2 {
					tone = skillBannerCyan
				}
				row = tone + row + skillBannerReset
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func renderSkillBanner(writer io.Writer, color bool) {
	logoRows := skillBannerLogoRows(color)
	wordmarkRows := skillBannerWordmarkRows(color)
	gap := strings.Repeat(" ", skillBannerGapWidth)
	for index := 0; index < skillBannerHeight; index++ {
		_, _ = io.WriteString(writer, logoRows[index]+gap+wordmarkRows[index])
		if color {
			_, _ = io.WriteString(writer, skillBannerReset)
		}
		_, _ = io.WriteString(writer, "\n")
	}
}
