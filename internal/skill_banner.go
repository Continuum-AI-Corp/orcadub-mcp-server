package dub

import (
	"io"
	"strings"
)

const (
	skillBannerWord       = "ORCADUB"
	skillBannerHeight     = 8
	skillBannerWordWidth  = 68
	skillBannerGlyphScale = 2
	skillBannerLetterGap  = 2
	skillBannerBlue       = "\x1b[94m"
	skillBannerCyan       = "\x1b[96m"
	skillBannerReset      = "\x1b[0m"
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

var skillBannerWordmarkRowRepeats = [7]int{1, 1, 1, 2, 1, 1, 1}

func skillBannerWordmarkRows(color bool) []string {
	rows := make([]string, 0, skillBannerHeight)
	for glyphRow, repeats := range skillBannerWordmarkRowRepeats {
		var builder strings.Builder
		for index, letter := range skillBannerWord {
			if index > 0 {
				builder.WriteString(strings.Repeat(" ", skillBannerLetterGap))
			}
			for _, cell := range skillBannerWordmarkGlyphs[letter][glyphRow] {
				builder.WriteString(strings.Repeat(string(cell), skillBannerGlyphScale))
			}
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
	for _, row := range skillBannerWordmarkRows(color) {
		_, _ = io.WriteString(writer, row)
		_, _ = io.WriteString(writer, "\n")
	}
}
