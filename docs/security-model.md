# Security Model

## Authentication

- SSH repository URLs do not require `gh`.
- HTTPS URLs rely on `gh` authentication.
- `dotctl` does not persist GitHub tokens itself.

## Secret hygiene

- Keep secrets out of tracked files whenever possible.
- Use `manifest.ignore` patterns to avoid accidental sync of sensitive material.
- Use encrypted files with `mode: copy` + `decrypt: true` for controlled local plaintext deployment.

## Encrypted file flow

- Source file name must contain `.enc.`.
- `sops` or `age` must be available in PATH.
- Decryption happens during apply; output is written to the target file.

## Backups and rollback

- Existing targets are backed up before overwrite by default.
- Sync attempts rollback if a later step fails after changes were applied.

## Logging

- `--verbose` enables detailed runtime and Git trace output.
- Logs are written to dotctl state directory.
- Token-like patterns are redacted in logger output.

## Concurrency control

- Sync uses a lock file to prevent parallel apply/push flows.
