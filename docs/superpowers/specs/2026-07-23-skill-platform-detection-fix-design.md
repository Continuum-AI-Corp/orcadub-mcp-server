# Skill Platform Detection Fix Design

## Problem

The Skill installer currently detects platforms only by checking marker paths
under the current working directory. As a result, an installed Claude Code or
Codex CLI is not preselected when the installer runs outside a project that
already contains `.claude` or `.codex`.

The interactive and non-interactive installation flows both call the same
project-only detector, so the defect affects the checkbox prompt and
`skill install --yes`.

## Desired Behavior

A platform is detected when any supported signal is present:

1. A platform-specific marker exists in the current project.
2. A platform-specific marker exists in the user's home directory.
3. A known CLI executable is available on `PATH`.

For this fix:

- Claude Code uses `.claude` and the `claude` executable.
- Codex uses `.codex` and the `codex` executable.
- Existing project-marker behavior remains unchanged for every platform.
- Detection remains best effort. Failure to resolve the home directory or
  search `PATH` must not prevent installation.

Detected platforms are deduplicated and returned in the existing catalog
order. They are preselected in the interactive checkbox prompt and used by the
existing non-interactive detected-platform behavior.

## False-Positive Protection

Installation destinations and detection markers remain separate concepts.
Codex installs Skills under `.agents`, but `.agents` is a shared destination
also used by other platforms. Therefore `.agents` must not be treated as a
Codex detection marker.

Only explicit global detection paths are checked. A platform may also
explicitly disable the project-root fallback when its project root is shared.
This prevents a generic or shared Skill directory from selecting unrelated
platforms.

## Design

Extend `skillPlatform` with explicit detection metadata:

- `GlobalDetectionPaths []string`
- `Executables []string`

Keep `DetectionPaths` as the project marker override. A nil value uses the
project root, preserving current behavior, while an explicit empty slice
disables project detection for shared roots such as Antigravity's `.agents`.

Change the detector to accept project directory, home directory, and a
`LookPath` dependency. For each platform, it checks project markers, then
explicit global markers, then known executables. The first successful signal
adds the platform once.

CLI orchestration resolves the home directory for detection on a best-effort
basis even for project-scoped installs. It injects `exec.LookPath` through the
same package-level dependency pattern already used for working-directory and
home-directory resolution, keeping tests deterministic.

## Error Handling

- Missing marker paths are normal and ignored.
- Home-directory lookup errors are ignored only by detection; global target
  resolution still reports its existing error when the home directory is
  required for installation.
- `LookPath` misses and errors are treated as an absent executable.
- Existing explicit platform validation and install errors are unchanged.

## Testing

Regression tests will prove:

- Claude Code and Codex are detected from explicit home markers when the
  project has no markers.
- Claude Code and Codex are detected from `PATH` when neither project nor home
  markers exist.
- A shared project or home `.agents` directory alone does not detect Codex,
  Antigravity, or Antigravity 2.0.
- Multiple signals for one platform do not create duplicates.
- Existing project-marker detection still works.
- The interactive prompt request preselects Claude Code and Codex using the
  injected environment.
- Home-directory lookup failure does not break a project-scoped install.

Verification includes focused tests, the complete Go test suite, formatting,
vet, build, Node syntax checks, and a terminal smoke test from a project with
no local platform markers.
