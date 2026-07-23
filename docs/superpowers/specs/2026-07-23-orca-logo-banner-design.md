# Orca Logo CLI Banner Design

## Goal

Replace the current six-line Skill installer banner with the supplied Orca
logo and a large `ORCADUB` wordmark. The logo and wordmark must have the same
20-row visual height and align at their top and bottom edges.

The banner contains no subtitle. The existing `AI DUBBING CLI` and
`SKILL INSTALLER / 技能安装器` lines are removed.

## Source Asset

The visual source is:

```text
/Users/a0000/GolandProjects/orcadub-plugin/tools/orca-logo
```

It is a self-contained shell resource generated from the 256×256 transparent
Orca PNG. Executing it produces a 40-column by 20-row ANSI 16-color rendering.
The implementation extracts the rendered ANSI rows and derives a monochrome
40×20 silhouette from the same cells. Both assets are embedded into the Go
binary. The released npm package and standalone binary do not read the source
file at runtime.

## Layout

The banner is a fixed 77-column by 20-row terminal composition:

```text
40-column Orca logo + 3-column gap + 34-column ORCADUB wordmark
```

The wordmark uses seven custom 4×7 bitmap glyphs:

```text
O R C A D U B
```

Each glyph is four columns wide. Adjacent glyphs have one blank column, giving
`7 × 4 + 6 × 1 = 34` columns. The seven logical glyph rows expand vertically
to exactly 20 terminal rows. This makes the wordmark fill the same line height
as the supplied logo while keeping the complete banner within a standard
80-column terminal.

The logo is on the left and the wordmark is on the right. Both start on row 1
and end on row 20. No text is rendered above or below them.

## Rendering

### Color terminals

- Preserve the supplied logo's ANSI 16-color rendering.
- Render `ORCADUB` with the existing Orca blue/cyan palette.
- Reset terminal colors at every row boundary and after the final row.
- Do not depend on truecolor support.

### Plain terminals

When color is disabled by `NO_COLOR` or a dumb terminal:

- Render a monochrome silhouette derived from the same 40×20 logo cells.
- Render the same 34×20 `ORCADUB` geometry without ANSI escapes.
- Preserve the identical 77×20 layout.

## Components

### Embedded logo asset

Store the extracted 20 ANSI rows and derived 20 plain rows as package assets
under `internal/assets/`. Use `go:embed` so every release artifact is
self-contained.

### Wordmark renderer

Keep the 4×7 glyph definitions and vertical expansion logic in
`internal/skill_banner.go`. The renderer returns exactly 20 wordmark rows in
either color or plain mode.

### Banner compositor

`renderSkillBanner` selects the color or plain logo rows, builds the matching
wordmark rows, joins each pair with three spaces, and writes 20 newline-
terminated rows.

The prompt flow and all installer behavior remain unchanged.

## Error Handling

The embedded asset is validated by tests and has no runtime I/O. If the
compiled asset has an unexpected row count, the renderer falls back to a
plain, deterministic banner with 40 blank logo columns and the wordmark rather
than panicking.

## Testing

Automated tests verify:

- Color output contains ANSI escapes and the visible `ORCADUB` geometry.
- Plain output contains no ANSI escapes.
- Both modes contain exactly 20 newline-terminated rows.
- Every plain row has exactly 77 terminal columns.
- The logo and wordmark canvases both span the same 20 output rows.
- Neither removed subtitle appears.
- The existing `NO_COLOR` terminal path still selects plain rendering.

Manual verification runs the built CLI in a real TTY at 80 columns and checks
the top/bottom alignment, color resets, absence of wrapping, and transition
from the banner into the first installer prompt.
