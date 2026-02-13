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
- Decrypted files are written with `0600` permissions regardless of source permissions.

## Secrets management (`dotctl secrets`)

- `dotctl secrets init` generates an age key pair (X25519 + ChaCha20-Poly1305).
- Private key stored at `~/.config/dotctl/age-identity.txt` with `0600` permissions.
- Public key stored at `.age-recipient.txt` in the repo root (safe to commit).
- The identity file is automatically added to `.gitignore`.
- `dotctl secrets encrypt` encrypts files using the native age format.
- `dotctl secrets decrypt --stdout` outputs to stdout without touching disk.
- `dotctl secrets rotate` generates a new key and re-encrypts all protected files.
- `dotctl push` includes a preflight check that blocks unencrypted sensitive files (override with `--force`).
- `dotctl doctor` reports secrets health: identity presence, encrypted files, and unprotected sensitive files.

### What the scheme protects

- Secrets at rest in the repository (ciphertext only in git history).
- Accidental plaintext commits (preflight check + `.gitignore`).

### What it does NOT protect

- Local malware with process access (can read the key file or plaintext in memory).
- Compromised private key (if leaked with the repo, all secrets are exposed).
- Plaintext at destination after sync (target files exist on disk).

## Backups and rollback

- Existing targets are backed up before overwrite by default.
- Sync attempts rollback if a later step fails after changes were applied.

## Logging

- `--verbose` enables detailed runtime and Git trace output.
- Logs are written to dotctl state directory.
- Token-like patterns are redacted in logger output.

## Concurrency control

- Sync uses a lock file to prevent parallel apply/push flows.

## Secrets Design Docs

- [Secrets design](./secrets-design.md)
- [Secrets MVP implementation plan (historical)](./archive/2026-foundation/v1-secrets-mvp-implementation-plan.md)
