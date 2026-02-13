# Sync Lifecycle

## What `dotctl sync` does

1. Acquires sync lock (`sync.lock`) to prevent concurrent runs.
2. Runs `git pull --rebase` on the active repository.
3. Loads and validates `manifest.yaml`.
4. Resolves entries by `os` and `profile` conditions.
5. Runs `pre_sync` hooks.
6. Applies file actions (`symlink` / `copy`, optional `decrypt`).
7. Runs `post_sync` hooks.
8. Stages, commits, and pushes if there are changes.
9. Updates `last_sync` timestamp.
10. Rotates backups according to config retention.

## Failure behavior

- Any action error stops sync.
- If filesystem changes were already applied, dotctl attempts rollback.
- If rollback has errors, those are returned in the final error path.

## Dry-run mode

`dotctl sync --dry-run` shows what would happen without modifying files, hooks, or Git state.

## Watch mode

`dotctl watch` monitors repository changes and triggers sync after debounce/cooldown windows.

- `--debounce`: wait time before running sync.
- `--cooldown`: ignore events briefly after a sync to avoid loops.
