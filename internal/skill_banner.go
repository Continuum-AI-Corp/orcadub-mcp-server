package dub

import (
	"fmt"
	"io"
)

const skillBannerPlain = `          ▄▄▖
     ▄██▀▀███▙
   ▄██▛  ▄████▌   ORCA//DUB
   ▀███▄███▀▀     AI DUBBING CLI
      ▀█▀    ◌━━━━━━━━◌
       SKILL INSTALLER / 技能安装器
`

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
