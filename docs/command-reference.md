# Command Reference

Use this page for the daily command surface. For first-time setup, see
[Getting Started](./getting-started.md). For manifest rules, see
[Manifest Specification](./manifest-spec.md).

## Core commands

| Command | Purpose |
|---|---|
| `dotctl init` | Configure and clone the active repo. |
| `dotctl add <path>` | Copy one local dotfile into the repo, update `manifest.yaml`, back up the original target, and replace it with a symlink. |
| `dotctl remove <path>` | Remove matching manifest entries without deleting local files or repo sources. |
| `dotctl sync` | Pull, apply the manifest, and push if there are changes. |
| `dotctl status` | Show repo, auth, and symlink state. |
| `dotctl doctor` | Run health checks. |
| `dotctl diff` | Show drift and content differences. |
| `dotctl pull` | Run `git pull --rebase`. |
| `dotctl push` | Stage, commit, and push local changes. |
| `dotctl edit repo` / `dotctl edit manifest` | Open the repo directory or manifest with `$VISUAL` or `$EDITOR`. |
| `dotctl backups list` | List local backup snapshots. |
| `dotctl backups restore <snapshot>` | Restore a local backup snapshot. Use repeatable `--target` for exact entries, or omit it for a full restore. |
| `dotctl open` | Open the repository in a browser. |
| `dotctl version` | Print the binary version and OS/arch. |
| `dotctl manifest suggest` | Scan common paths and write `manifest.suggested.yaml`. |

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

## Notes

- `dotctl add` accepts one path at a time. It rejects paths outside `$HOME`, paths overlapping the dotctl repo, and sensitive-looking targets unless `--force` is used.
- `dotctl remove` only untracks the path. It does not delete the local target or the repo source.
- `dotctl edit` requires `$VISUAL` or `$EDITOR`.
- `dotctl backups restore` overwrites local targets and therefore requires `--force` for real restores. Run `--dry-run` first and review the planned targets.
- `--target` matches backed-up target paths exactly, can be repeated, and restores the directory entry when the snapshot has one.
