# Command Reference

## Core commands

- `dotctl init`: configure and clone repo.
- `dotctl add <path>`: copy one local dotfile into the repo, update `manifest.yaml`, back up the original, and replace it with a symlink.
- `dotctl remove <path>`: remove matching manifest entries and managed-source metadata without deleting local files or repo sources.
- `dotctl sync`: pull, apply manifest, push.
- `dotctl status`: show repo/auth/symlink state.
- `dotctl doctor`: run health checks.
- `dotctl diff`: show drift and content differences.
- `dotctl pull`: run `git pull --rebase`.
- `dotctl push`: stage, commit, and push local changes.
- `dotctl edit [repo|manifest]`: open the repo directory or manifest with `$VISUAL` or `$EDITOR`.
- `dotctl backups list`: list local backup snapshots.
- `dotctl backups restore <snapshot>`: restore a local backup snapshot; use repeatable `--target` to restore exact entries, or omit it for a full restore. Requires `--force` unless `--dry-run`.
- `dotctl open`: open repository in browser.
- `dotctl version`: print binary version and OS/arch.

## Common global flags

- `--config <path>`
- `--json`
- `--dry-run`
- `--verbose`
- `--force`

## Examples

```bash
dotctl status --json
dotctl add ~/.zshrc --dry-run
dotctl add ~/.zshrc
dotctl remove ~/.zshrc --dry-run
dotctl edit manifest
dotctl backups list
dotctl backups restore 20260101-010101.000001 --dry-run
dotctl backups restore 20260101-010101.000001 --target ~/.zshrc --dry-run
dotctl backups restore 20260101-010101.000001 --target ~/.zshrc --force
dotctl diff --details
dotctl push -m "chore: update shell aliases"
```

`dotctl add` accepts one path at a time. It rejects paths outside `$HOME`, paths overlapping the dotctl repo, and sensitive-looking targets unless `--force` is used.

`dotctl remove` also accepts one path at a time. It untracks the path only; it does not delete the local target or the repo source.

`dotctl edit` requires `$VISUAL` or `$EDITOR`. Use `dotctl edit` or `dotctl edit repo` for the repository directory and `dotctl edit manifest` for `manifest.yaml`.

`dotctl backups restore` overwrites local targets and therefore requires `--force` for real restores. Run `--dry-run` first and review the planned targets. `--target` matches backed-up target paths exactly, can be repeated for multiple entries, and restores the directory entry when the snapshot has one.
