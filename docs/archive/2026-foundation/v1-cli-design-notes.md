# V1 CLI Design Notes

## Initial command surface

- `init`
- `status`
- `sync`
- `pull`
- `push`
- `doctor`
- `open`
- `bootstrap`
- `version`

## Core principles

- CLI-first orchestration.
- Human-readable output by default.
- Optional `--json` output for tray integration and scripting.
- Global flags for config overrides, dry-run, and verbosity.

## UX goals

- Explicit errors with actionable next steps.
- Deterministic command behavior across macOS and Linux.
