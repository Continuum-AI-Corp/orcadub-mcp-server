# Orca Logo CLI Banner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the six-line Skill installer banner with a self-contained 77×20 composition containing the supplied Orca logo and an equally tall `ORCADUB` wordmark.

**Architecture:** Import the supplied 40×20 ANSI logo into generated package assets, embed color and monochrome variants with `go:embed`, and compose them row-by-row with a generated 34×20 bitmap wordmark. Keep color selection in the existing `renderSkillBanner(io.Writer, bool)` boundary.

**Tech Stack:** Go 1.25 standard library (`embed`, `io`, `strings`, `unicode/utf8` in tests), Python 3 importer for committed generated assets, ANSI 16-color terminal output.

## Global Constraints

- Use `/Users/a0000/GolandProjects/orcadub-plugin/tools/orca-logo` as the visual source.
- The logo canvas is exactly 40 columns by 20 rows.
- The `ORCADUB` canvas is exactly 34 columns by 20 rows.
- The complete banner is exactly 77 columns by 20 rows.
- Remove `AI DUBBING CLI` and `SKILL INSTALLER / 技能安装器`.
- Color mode preserves the source ANSI 16-color logo and uses blue/cyan for the wordmark.
- Plain mode contains no ANSI escapes and preserves the same geometry.
- Released binaries must not perform runtime file I/O.
- Repository code, comments, docs, scripts, and commit messages remain in English.

---

### Task 1: Import and Embed the Supplied Logo

**Files:**
- Create: `tools/import_orca_logo.py`
- Create: `internal/assets/orca_logo_color.ansi`
- Create: `internal/assets/orca_logo_plain.txt`
- Create: `internal/skill_banner_logo.go`
- Test: `internal/skill_banner_test.go`

**Interfaces:**
- Consumes: a standalone shell resource containing an `ORCA_EOF` heredoc.
- Produces: `skillBannerLogoRows(color bool) []string`, always returning 20 rows of 40 visible columns.

- [ ] **Step 1: Write the failing embedded-logo test**

Replace the old banner-content assertions with helpers and this focused test:

```go
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
```

Add `regexp` and `unicode/utf8` imports to `internal/skill_banner_test.go`.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./internal -run '^TestSkillBannerLogoRows$' -count=1
```

Expected: compilation fails because `skillBannerLogoRows` does not exist.

- [ ] **Step 3: Add the deterministic asset importer**

Create `tools/import_orca_logo.py`:

```python
#!/usr/bin/env python3
"""Import the standalone Orca ANSI logo into embedded CLI assets."""

import argparse
import re
from pathlib import Path

ANSI = re.compile(rb"\x1b\[[0-9;]*m")
START = b"cat <<'ORCA_EOF'\n"
END = b"\nORCA_EOF\n"


def extract_payload(source: bytes) -> bytes:
    if START not in source or END not in source:
        raise ValueError("source does not contain the ORCA_EOF heredoc")
    payload = source.split(START, 1)[1].split(END, 1)[0]
    lines = payload.splitlines()
    if len(lines) != 20:
        raise ValueError(f"logo has {len(lines)} rows, want 20")
    for index, line in enumerate(lines):
        visible = ANSI.sub(b"", line).decode("utf-8")
        if len(visible) != 40:
            raise ValueError(f"logo row {index} has width {len(visible)}, want 40")
    return payload + b"\n"


def plain_payload(color_payload: bytes) -> bytes:
    rows = []
    for line in color_payload.rstrip(b"\n").splitlines():
        visible = ANSI.sub(b"", line).decode("utf-8")
        rows.append(visible.translate(str.maketrans({"▀": "█", "▄": "█"})))
    return ("\n".join(rows) + "\n").encode("utf-8")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("source", type=Path)
    parser.add_argument("output_dir", type=Path)
    args = parser.parse_args()

    color = extract_payload(args.source.read_bytes())
    plain = plain_payload(color)
    args.output_dir.mkdir(parents=True, exist_ok=True)
    (args.output_dir / "orca_logo_color.ansi").write_bytes(color)
    (args.output_dir / "orca_logo_plain.txt").write_bytes(plain)


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Generate the committed assets**

Run:

```bash
python3 tools/import_orca_logo.py \
  /Users/a0000/GolandProjects/orcadub-plugin/tools/orca-logo \
  internal/assets
```

Expected:

```text
internal/assets/orca_logo_color.ansi
internal/assets/orca_logo_plain.txt
```

Both files must contain 20 newline-terminated rows.

- [ ] **Step 5: Embed and validate the logo rows**

Create `internal/skill_banner_logo.go`:

```go
package dub

import (
	_ "embed"
	"strings"
)

const (
	skillBannerLogoWidth  = 40
	skillBannerHeight     = 20
	skillBannerWordWidth  = 34
	skillBannerGapWidth   = 3
	skillBannerTotalWidth = skillBannerLogoWidth + skillBannerGapWidth + skillBannerWordWidth
)

//go:embed assets/orca_logo_color.ansi
var skillBannerLogoColor string

//go:embed assets/orca_logo_plain.txt
var skillBannerLogoPlain string

func skillBannerLogoRows(color bool) []string {
	value := skillBannerLogoPlain
	if color {
		value = skillBannerLogoColor
	}
	rows := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	if len(rows) == skillBannerHeight {
		return rows
	}
	blank := strings.Repeat(" ", skillBannerLogoWidth)
	rows = make([]string, skillBannerHeight)
	for index := range rows {
		rows[index] = blank
	}
	return rows
}
```

- [ ] **Step 6: Run the focused test and verify GREEN**

Run the command from Step 2.

Expected: `TestSkillBannerLogoRows` passes.

- [ ] **Step 7: Commit the embedded logo**

```bash
git add tools/import_orca_logo.py internal/assets internal/skill_banner_logo.go internal/skill_banner_test.go
git commit -m "feat: embed Orca CLI logo"
```

### Task 2: Render the 20-Row ORCADUB Wordmark

**Files:**
- Modify: `internal/skill_banner.go`
- Test: `internal/skill_banner_test.go`

**Interfaces:**
- Consumes: `skillBannerWordmarkGlyphs map[rune][7]string`.
- Produces: `skillBannerWordmarkRows(color bool) []string`, exactly 20 rows of 34 visible columns.

- [ ] **Step 1: Write the failing wordmark geometry test**

Add:

```go
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
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./internal -run '^TestSkillBannerWordmarkRows$' -count=1
```

Expected: compilation fails because `skillBannerWordmarkRows` does not exist.

- [ ] **Step 3: Define the bitmap glyphs**

Replace `skill_banner.go` with a focused renderer using these exact glyphs:

```go
package dub

import (
	"io"
	"strings"
)

const (
	skillBannerWord = "ORCADUB"
	skillBannerBlue = "\x1b[94m"
	skillBannerCyan = "\x1b[96m"
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
```

- [ ] **Step 4: Implement the wordmark renderer**

Add:

```go
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
```

- [ ] **Step 5: Run the focused test and verify GREEN**

Run the command from Step 2.

Expected: `TestSkillBannerWordmarkRows` passes.

- [ ] **Step 6: Commit the wordmark**

```bash
git add internal/skill_banner.go internal/skill_banner_test.go
git commit -m "feat: add ORCADUB terminal wordmark"
```

### Task 3: Compose and Verify the Final Banner

**Files:**
- Modify: `internal/skill_banner.go`
- Modify: `internal/skill_banner_test.go`
- Modify: `internal/cli_test.go`

**Interfaces:**
- Consumes: `skillBannerLogoRows` and `skillBannerWordmarkRows`.
- Produces: the unchanged public package boundary `renderSkillBanner(io.Writer, bool)`.

- [ ] **Step 1: Write failing final-layout tests**

Add:

```go
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
```

Update the CLI TUI assertion from the removed `ORCA//DUB` literal to:

```go
if !strings.Contains(out.out, skillBannerBlue) ||
	!strings.Contains(out.out, skillBannerCyan) {
	t.Fatalf("stdout lacks Orca banner colors:\n%s", out.out)
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./internal -run 'TestRenderSkillBannerLayout|TestRunCLISkillInstallTUI' -count=1
```

Expected: layout test fails because the old six-line renderer does not produce
20 rows of 77 columns and still contains the removed subtitles.

- [ ] **Step 3: Implement the row compositor**

Replace the old renderer with:

```go
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
```

- [ ] **Step 4: Run focused tests and verify GREEN**

Run the command from Step 2.

Expected: all matching banner and TUI tests pass.

- [ ] **Step 5: Run complete repository verification**

Run:

```bash
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  gofmt -w internal/skill_banner.go internal/skill_banner_logo.go internal/skill_banner_test.go internal/cli_test.go
test -z "$(gofmt -l internal cmd)"
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  golangci-lint run
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test ./... -count=1
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go test -race ./internal -count=1
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go vet ./...
env GOROOT=/Users/a0000/.gvm/gos/go1.25.0 \
  PATH=/Users/a0000/.gvm/gos/go1.25.0/bin:$PATH \
  GOTOOLCHAIN=auto \
  go build ./...
node --check npm/install.js
node --check npm/bin/run.js
git diff --check
```

Expected: every command exits with status 0 and golangci-lint reports
`0 issues.`.

- [ ] **Step 6: Run real TTY smoke tests**

Build the binary and run:

```bash
orcadub skill install --lang zh --scope project
```

in a real 80-column TTY. Verify:

- the banner occupies 20 rows and 77 columns;
- the Orca and wordmark canvases are top/bottom aligned;
- no line wraps;
- neither subtitle appears;
- the first platform prompt begins immediately after the banner.

Repeat with `NO_COLOR=1` and verify the same geometry without escape sequences.
Abort both runs before installation.

- [ ] **Step 7: Commit the completed banner**

```bash
git add internal/skill_banner.go internal/skill_banner_test.go internal/cli_test.go
git commit -m "feat: replace CLI banner with Orca logo"
```
