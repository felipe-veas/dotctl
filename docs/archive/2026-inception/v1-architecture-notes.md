# V1 Architecture Notes

## Component model

- `cmd/`: command entry points.
- `internal/manifest`: parses and resolves manifest entries.
- `internal/linker`: applies symlink/copy actions.
- `internal/gitops`: clone, pull, push, and repo inspection.
- `internal/auth`: `gh` auth checks for HTTPS workflows.
- `internal/backup`: backup snapshots before overwrite.
- `internal/config`: local machine config handling.
- `internal/platform`: OS-specific behavior abstractions.

## Data flow

1. Load config and resolve active profile/repo.
2. Load `manifest.yaml` and filter by context.
3. Apply actions with safety checks.
4. Commit/push changes when needed.
5. Surface human and JSON output.

## UI architecture

Tray apps were designed as lightweight remotes over CLI commands using JSON output contracts.
