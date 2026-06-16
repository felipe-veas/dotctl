# Security Model

## Authentication

- SSH repository URLs do not require `gh`.
- HTTPS URLs rely on `gh` authentication.
- `dotctl` does not persist GitHub tokens itself.

## Secret hygiene

- Keep secrets out of tracked files whenever possible.
- Use `manifest.ignore` patterns to avoid accidental sync of sensitive material.
- Use dedicated external secret-management tools for credentials, tokens, and private keys.
- `dotctl` is intended for non-sensitive dotfiles and does not manage encryption or decryption.

## Sensitive file guardrails

- `dotctl init` adds conservative `.gitignore` defaults for common secret names.
- `dotctl doctor` reports potentially sensitive tracked files.
- `dotctl push` blocks sensitive-looking tracked files unless `--force` is used.
- Externally encrypted artifacts may still be tracked if you intentionally manage them outside dotctl.

## Path safety

- Manifest targets must resolve strictly under the user's home directory.
- `dotctl` rejects relative targets, targets outside `$HOME`, `$HOME` itself, and parent-traversal escapes before applying filesystem changes.
- Repository source paths must remain relative to the repository root.
- `dotctl status`, `dotctl doctor`, and `dotctl sync` warn on sensitive-looking manifest targets such as `.ssh`, `.gnupg`, `.kube`, `.aws`, `.config/gh`, `.config/gcloud`, and `.env` files.

### What the guardrails protect

- Accidental commits of obvious sensitive filenames.
- Accidental application of ignored sources.
- Accidental overwrite or removal of paths outside the user's home directory.
- Accidental management of high-risk local credential/config paths without an explicit warning.

### What the guardrails do not protect

- Secrets with non-obvious names.
- Secrets already committed to Git history.
- Local malware or compromised developer machines.
- Encryption, decryption, key rotation, or key recovery.
- Symlink-based attacks from already-compromised local directories outside dotctl's control.

## Backups and rollback

- Existing targets are backed up before overwrite by default.
- Sync attempts rollback if a later step fails after changes were applied.
- `dotctl backups list` exposes local backup snapshots.
- `dotctl backups restore <snapshot>` requires `--force` unless `--dry-run` is used. With `--target`, restores are limited to exact backed-up paths and the flag can be repeated for multiple entries.
- Restore rejects targets outside `$HOME` or equal to `$HOME` before applying filesystem changes.

## Logging

- `--verbose` enables detailed runtime and Git trace output.
- Logs are written to dotctl state directory.
- Token-like patterns are redacted in logger output.

## Concurrency control

- Sync uses a lock file to prevent parallel apply/push flows.

Historical notes for the removed secrets implementation are archived under [docs/archive](./archive/README.md).
