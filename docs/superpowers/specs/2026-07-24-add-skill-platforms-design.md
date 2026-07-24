# Additional Skill Platforms Design

## Goal

Extend the guided OrcaDub Skill installer from 33 to 36 platforms by adding
Hermes, OpenClaw, and Command Code without changing the existing platform
entries or installer interaction.

## Platform Layouts

The installer will append the platforms in this order:

| ID | Display name | Project root | Global root |
| --- | --- | --- | --- |
| `hermes` | Hermes | `.hermes` | `.hermes` |
| `openclaw` | OpenClaw | `.` | `.openclaw` |
| `command-code` | Command Code | `.commandcode` | `.commandcode` |

The existing target resolver appends `skills/orcadub/SKILL.md` to each root.
The resulting targets are:

- Hermes project: `<project>/.hermes/skills/orcadub/SKILL.md`
- Hermes global: `~/.hermes/skills/orcadub/SKILL.md`
- OpenClaw project: `<workspace>/skills/orcadub/SKILL.md`
- OpenClaw global: `~/.openclaw/skills/orcadub/SKILL.md`
- Command Code project:
  `<project>/.commandcode/skills/orcadub/SKILL.md`
- Command Code global: `~/.commandcode/skills/orcadub/SKILL.md`

## Detection

Detection remains conservative so the installer does not preselect platforms
that are not installed:

- Hermes uses `.hermes` as both its project and global marker, plus the
  `hermes` executable.
- OpenClaw disables project marker detection because its project Skill root is
  the generic workspace `skills/` directory. It uses the global `.openclaw`
  marker and the `openclaw` executable instead.
- Command Code uses `.commandcode` as both its project and global marker. It
  does not use its `cmd` executable for detection because `cmd` is a standard
  Windows executable and would create false positives.

## User Interface

No new screens or controls are needed. The existing multi-select prompt is
generated from the platform catalog, so the three platforms appear at the end
of the list and participate in filtering, select-all, detected-first sorting,
JSON output, and unattended installation automatically.

## Documentation

The README platform count changes from 33 to 36 and names the three additional
platforms while retaining the Comet compatibility description for the
original catalog.

## Verification

Automated tests will cover:

- Exact platform order and roots.
- Project and global target resolution for each new platform.
- Positive marker and executable detection.
- The absence of false OpenClaw project detection in an ordinary project.
- The absence of `cmd`-based Command Code detection.

The complete repository check remains `make check`.
