# Sync Lifecycle

## What `dotctl sync` does

1. Acquires sync lock (`sync.lock`) to prevent concurrent runs.
2. Runs `git pull --rebase` on the active repository.
3. Loads and validates `manifest.yaml`.
4. Resolves entries by `os` conditions.
5. Applies file actions (`symlink` / `copy`).
6. Stages, commits, and pushes if there are changes.
7. Updates `last_sync` timestamp.
8. Rotates backups according to config retention.

## Failure behavior

- Any action error stops sync.
- If filesystem changes were already applied, dotctl attempts rollback.
- If rollback has errors, those are returned in the final error path.

## Dry-run mode

`dotctl sync --dry-run` shows what would happen without modifying files or Git state.
