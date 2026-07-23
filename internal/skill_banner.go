package dub

import (
	"fmt"
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

const skillBannerPlain = `          ▄▄▖
     ▄██▀▀███▙
   ▄██▛  ▄████▌   ORCA//DUB
   ▀███▄███▀▀     AI DUBBING CLI
      ▀█▀    ◌━━━━━━━━◌
       SKILL INSTALLER / 技能安装器
`

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

func renderSkillBanner(w io.Writer, color bool) {
	if !color {
		_, _ = io.WriteString(w, skillBannerPlain)
		return
	}
	const (
		blue  = "\x1b[38;2;0;123;255m"
		cyan  = "\x1b[38;2;83;199;255m"
		reset = "\x1b[0m"
	)
	_, _ = fmt.Fprintf(
		w,
		"%s          ▄▄▖\n"+
			"     ▄██▀▀███▙\n"+
			"   ▄██▛  ▄████▌   %sORCA//DUB%s\n"+
			"%s   ▀███▄███▀▀     %sAI DUBBING CLI%s\n"+
			"      ▀█▀    ◌━━━━━━━━◌\n"+
			"       %sSKILL INSTALLER / 技能安装器%s\n",
		blue,
		cyan,
		blue,
		blue,
		cyan,
		blue,
		cyan,
		reset,
	)
}
