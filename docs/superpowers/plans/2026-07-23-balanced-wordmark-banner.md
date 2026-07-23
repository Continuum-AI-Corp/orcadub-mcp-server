# Balanced ORCADUB Wordmark Banner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the narrow 34×15 installer wordmark with the approved balanced 68×8 `ORCADUB` wordmark.

**Architecture:** Keep the existing seven 4×7 bitmap glyphs, expand every glyph cell to two terminal columns, join letters with two-column gaps, and repeat only the middle source row. Preserve the existing renderer and ANSI/plain color boundary.

**Tech Stack:** Go 1.25, `strings.Builder`, ANSI 16-color output, Lip Gloss display-width assertions.

## Global Constraints

- The visible banner is exactly 68 columns by 8 rows.
- The first four rows are bright blue and the final four rows are bright cyan.
- Plain output has identical geometry and no ANSI escapes.
- No image, subtitle, runtime asset, or new dependency is introduced.
- Installer behavior outside the banner remains unchanged.
- Repository code, docs, and commit messages remain in English.
- Per the user's instruction, update implementation and coverage directly rather than using a test-first cycle.

---

### Task 1: Render and Verify the Balanced Wide Wordmark

**Files:**
- Modify: `internal/skill_banner.go`
- Modify: `internal/skill_banner_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `skillBannerWordmarkGlyphs map[rune][7]string`.
- Produces: `skillBannerWordmarkRows(color bool) []string`, exactly eight rows of 68 terminal columns.

- [ ] **Step 1: Expand the wordmark horizontally and reduce its height**

In `internal/skill_banner.go`, set the geometry and repeat pattern:

```go
const (
	skillBannerWord       = "ORCADUB"
	skillBannerHeight     = 8
	skillBannerWordWidth  = 68
	skillBannerGlyphScale = 2
	skillBannerLetterGap  = 2
)

var skillBannerWordmarkRowRepeats = [7]int{1, 1, 1, 2, 1, 1, 1}
```

Replace the per-letter write inside `skillBannerWordmarkRows` with scaled cell
output and a two-column gap:

```go
for index, letter := range skillBannerWord {
	if index > 0 {
		builder.WriteString(strings.Repeat(" ", skillBannerLetterGap))
	}
	for _, cell := range skillBannerWordmarkGlyphs[letter][glyphRow] {
		builder.WriteString(strings.Repeat(string(cell), skillBannerGlyphScale))
	}
}
```

- [ ] **Step 2: Update geometry coverage**

In `internal/skill_banner_test.go`, retain the existing row-count,
display-width, ANSI, and subtitle assertions. Update glyph-region indexing:

```go
glyphWidth := 4 * skillBannerGlyphScale
glyphStride := glyphWidth + skillBannerLetterGap
for glyphIndex := range []rune(skillBannerWord) {
	start := glyphIndex * glyphStride
	hasInk := false
	for _, row := range visibleRows {
		for _, cell := range row[start : start+glyphWidth] {
			hasInk = hasInk || cell != ' '
		}
	}
	if !hasInk {
		t.Fatalf("color=%v glyph %d has no ink", color, glyphIndex)
	}
}
```

The existing constant-based assertions then verify 68×8 in color and plain
modes.

- [ ] **Step 3: Update public documentation**

Change the installer description in `README.md` to:

```markdown
The guided installer opens with a balanced 8-row `ORCADUB` wordmark, then lets you:
```

- [ ] **Step 4: Format and run focused verification**

Run:

```bash
goimports -w internal/skill_banner.go internal/skill_banner_test.go
gofmt -w internal/skill_banner.go internal/skill_banner_test.go
go test ./internal -run 'Test(SkillBannerWordmarkRows|RenderSkillBannerLayout)$' -count=1
```

Expected: both focused banner tests pass.

- [ ] **Step 5: Run full verification**

Run:

```bash
golangci-lint run
go test ./... -count=1
go test -race ./internal -count=1
go vet ./...
go build ./...
node --check npm/install.js
node --check npm/bin/run.js
git diff --check
```

Expected: zero lint issues, all tests pass, and every remaining command exits
with status zero.

- [ ] **Step 6: Run a real terminal smoke test**

Build the binary and open the installer in an 80-column TTY:

```bash
go build -o /tmp/orcadub-balanced-wordmark ./cmd
/tmp/orcadub-balanced-wordmark skill install
```

Expected: the banner is 68 columns by 8 rows, does not wrap, contains no image
or subtitle, and the language prompt begins immediately below it.

- [ ] **Step 7: Commit**

```bash
git add README.md internal/skill_banner.go internal/skill_banner_test.go
git commit -m "fix: balance installer wordmark"
```
