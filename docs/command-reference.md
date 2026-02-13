# Command Reference

## Core commands

- `dotctl init`: configure profile and clone repo.
- `dotctl sync`: pull, apply manifest, run hooks, push.
- `dotctl status`: show repo/auth/symlink state.
- `dotctl doctor`: run health checks.
- `dotctl diff`: show drift and content differences.
- `dotctl pull`: run `git pull --rebase`.
- `dotctl push`: stage, commit, and push local changes.
- `dotctl watch`: run auto-sync on filesystem changes.
- `dotctl bootstrap`: run bootstrap hooks.
- `dotctl open`: open repository in browser.
- `dotctl repos`: manage multiple configured repositories.
- `dotctl version`: print binary version and OS/arch.

## Multi-repo subcommands

- `dotctl repos list`
- `dotctl repos add --name <name> --url <url> [--path <path>] [--activate]`
- `dotctl repos use <name>`
- `dotctl repos remove <name>`

## Common global flags

- `--config <path>`
- `--profile <name>`
- `--repo-name <name>`
- `--json`
- `--dry-run`
- `--verbose`
- `--force`

## Examples

```bash
dotctl status --json
dotctl diff --details
dotctl push -m "chore: update shell aliases"
dotctl watch --debounce 2s --cooldown 4s
```
