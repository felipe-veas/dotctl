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
- `dotctl secrets`: manage encrypted secrets in the repository.
- `dotctl version`: print binary version and OS/arch.

## Secrets subcommands

- `dotctl secrets init [--identity <path>] [--import <path>]`: generate or import an age identity.
- `dotctl secrets encrypt <file> [file...] [--recipient <key>] [--keep]`: encrypt files for the repo.
- `dotctl secrets decrypt <file> [file...] [--identity <path>] [--keep] [--stdout]`: decrypt files.
- `dotctl secrets status`: show secrets protection status.
- `dotctl secrets rotate [--identity <path>]`: generate new key and re-encrypt all files.

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

# Secrets workflow
dotctl secrets init
dotctl secrets encrypt configs/env/.env
dotctl secrets decrypt configs/env/.env.enc --stdout
dotctl secrets status
dotctl secrets rotate
```
