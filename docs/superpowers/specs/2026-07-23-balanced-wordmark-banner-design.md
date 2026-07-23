# Balanced ORCADUB Wordmark Banner Design

## Context

The current installer banner renders `ORCADUB` in a 34-column by 15-row
canvas. Each source glyph column occupies one terminal cell while source rows
are repeated vertically. Because terminal cells are taller than they are wide,
the result appears compressed, narrow, and visually inconsistent with the
installer controls.

## Goal

Render a wider, shorter `ORCADUB` wordmark that looks balanced in a standard
80-column terminal while preserving the existing blue-to-cyan brand treatment.

## Selected Direction

Use the approved **Balanced Wide** layout:

- Visible canvas: exactly 68 columns by 8 rows.
- Expand every source glyph column to two terminal columns.
- Separate adjacent letters with two spaces.
- Preserve the existing seven 4-column bitmap glyph definitions.
- Repeat only the middle glyph row, using row repeats
  `[1, 1, 1, 2, 1, 1, 1]`.
- Render the first four output rows in bright blue and the final four rows in
  bright cyan.
- Keep color-free output geometrically identical and free of ANSI sequences.

The width calculation is:

```text
7 letters × (4 source columns × 2) + 6 gaps × 2 columns = 68 columns
```

The height calculation is:

```text
7 source rows + 1 repeated middle row = 8 rows
```

An 80-column terminal therefore retains 12 columns of horizontal space and
does not wrap the banner.

## Rendering

`skillBannerWordmarkRows` remains the single wordmark generator. It expands
each rune in a glyph row horizontally before joining letters. The banner keeps
no image assets, subtitles, or runtime file dependencies.

`renderSkillBanner` continues writing the generated rows followed by newlines.
The language and platform prompts start immediately after the eighth row.

## Compatibility and Fallbacks

- `NO_COLOR` and `TERM=dumb` use the same 68×8 layout without ANSI escapes.
- JSON and unattended installer modes remain decoration-free.
- Dubbing commands, MCP mode, platform detection, installation paths, and
  overwrite safety are unchanged.
- No responsive alternate wordmark is introduced; the approved layout targets
  standard terminals of at least 68 columns.

## Verification

- Assert eight generated and rendered rows.
- Assert every color and plain row has a terminal display width of 68.
- Assert color output contains ANSI sequences and plain output does not.
- Assert all seven glyph regions contain visible ink at their expanded
  positions.
- Run formatting, lint, full tests, race tests, vet, build, and Node launcher
  syntax checks.
- Run a real 80-column TTY smoke test and confirm the installer prompt appears
  immediately below the banner without wrapping.

## Non-Goals

- Reintroducing the Orca image.
- Adding subtitles or taglines.
- Changing installer interaction, language selection, or platform detection.
- Changing release packaging or the dubbing API surface.
