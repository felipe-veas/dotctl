# Design: `dotctl secrets`

## Executive Summary

This document describes the design of the `dotctl secrets` module.

Goal:

- keep sensitive files encrypted in the private dotfiles repository,
- decrypt only at apply time during `dotctl sync`,
- provide a clear and safe setup for new machines.

The implementation targets single-user, multi-machine workflows.

## 1. Threat Model

### 1.1 Actors

| Actor | Description |
| --- | --- |
| Repository disclosure attacker | Gains read access to the private GitHub repository (credential leak, token theft, backup exposure). |
| Stolen GitHub token | Equivalent read impact to repository disclosure if token scopes allow repository access. |
| Local malware | Malicious software running on a user machine with filesystem/process access. |
| Operational leakage | Secrets exposed via logs, shell history, temporary files, or local backups. |

### 1.2 What this model protects

- Secrets at rest in the repository (`.enc.*` files are unreadable without the private key).
- Secret exposure from repository clones, mirrors, and backups.
- Accidental plaintext commits (preflight and policy checks).
- Direct token leakage in dotctl logs through redaction and command output discipline.

### 1.3 What this model does not protect

- Compromised local host with arbitrary code execution.
- Leaked private age identity file.
- Plaintext destination files after successful decryption and apply.
- Side-channel analysis outside normal repository/file controls.

## 2. Technical Design

### 2.1 Key management options (evaluation)

| Criteria | age (direct) | sops + age | passphrase + KDF | password manager backend |
| --- | :---: | :---: | :---: | :---: |
| External dependencies | `age` library/cli | `sops` + `age` | none | provider CLI/runtime |
| Setup complexity | low | medium | medium | high |
| Multi-machine UX | copy identity file | copy identity + policy | memorize/passphrase flow | provider login per machine |
| Structured partial encryption | no | yes | no | n/a |
| Rotation model | re-encrypt files | update keys / re-encrypt | re-encrypt | backend-specific |
| Fit for dotfiles single-user model | excellent | good | acceptable | often overkill |

### 2.2 Recommendation

Use **age direct** as the MVP backend, with a forward-compatible path to `sops + age`.

Reasons:

- minimal operational friction,
- strong and audited primitives,
- direct compatibility with existing `decrypt: true` flow,
- straightforward machine bootstrap.

### 2.3 Encrypted file format and naming

- Native age file format (no custom crypto format).
- Naming convention inserts `.enc` before final extension.

Examples:

| Plaintext | Encrypted |
| --- | --- |
| `config.yaml` | `config.enc.yaml` |
| `.env` | `.env.enc` |
| `.env.local` | `.env.enc.local` |
| `api.key` | `api.enc.key` |

### 2.4 Private key location

Default local identity path:

```text
~/.config/dotctl/age-identity.txt
```

Repository recipient file:

```text
<repo-root>/.age-recipient.txt
```

Expected permission model:

| File | Permissions | Rationale |
| --- | --- | --- |
| `age-identity.txt` | `0600` | private key material |
| Decrypted target files | `0600` (or stricter) | plaintext secrets |
| `.age-recipient.txt` | repo defaults (`0644`) | public key only |

## 3. Integration with Existing Dotctl Flows

### 3.1 `manifest.yaml`

No schema change required. Existing `decrypt: true` remains the control plane.

```yaml
files:
  - source: configs/app/config.enc.yaml
    target: "{{ .config_home }}/app/config.yaml"
    mode: copy
    decrypt: true
```

### 3.2 Package boundaries

- `internal/secrets/`: key lifecycle + encrypt/decrypt helper commands.
- `internal/decrypt/`: sync-time decryption used by linker apply flow.

This keeps runtime sync behavior stable while adding explicit secrets lifecycle commands.

### 3.3 `push`, `doctor`, and security checks

- `push`: preflight checks block unencrypted sensitive files (unless explicit override).
- `doctor`: reports secrets health (identity presence, recipient presence, encrypted/unprotected files).
- `security_checks`: sensitive path detection should ignore `.enc.*` artifacts.

### 3.4 Linker behavior

For decrypted outputs, permissions should remain restrictive (`0600` cap when source mode is broader).

## 4. Command UX

### 4.1 Command surface

```text
dotctl secrets
  init
  encrypt
  decrypt
  status
  rotate
```

### 4.2 `dotctl secrets init`

Purpose:

- create or import age identity,
- write/update `.age-recipient.txt`,
- enforce local key safety defaults.

Key flags:

- `--identity`
- `--import`
- `--force`

### 4.3 `dotctl secrets encrypt`

Purpose:

- encrypt one or multiple files,
- output `.enc.*` artifacts,
- optionally remove plaintext originals.

Key flags:

- `--manifest`
- `--recipient`
- `--keep`
- `--stdout`

### 4.4 `dotctl secrets decrypt`

Purpose:

- decrypt for controlled local use or inspection.

Key flags:

- `--manifest`
- `--identity`
- `--stdout`
- `--keep`

Security note:

- `--stdout` avoids creating plaintext files on disk.

### 4.5 `dotctl secrets status`

Purpose:

- report repository secrets posture.

Expected output areas:

- identity status,
- recipient status,
- encrypted protected files,
- unprotected sensitive findings.

### 4.6 `dotctl secrets rotate`

Purpose:

- generate new key material,
- re-encrypt protected files,
- back up previous identity safely.

## 5. Security Rules

### 5.1 Output handling

- No secret content in logs.
- No plaintext output unless explicit (`--stdout`).
- Verbose mode must never dump decrypted payload data.

### 5.2 File handling

- Restrictive permissions for private key and plaintext outputs.
- Avoid plaintext temporary files whenever possible.

### 5.3 Plaintext commit prevention

Defense-in-depth:

- `.gitignore` defaults,
- push preflight blocking,
- doctor visibility and remediation hints.

### 5.4 Encrypted file conflicts

`*.enc.*` files are opaque blobs. Merge conflicts are expected to require manual resolve/decrypt/re-encrypt workflows.

### 5.5 Key loss model

If the private identity is lost and no backup exists, encrypted repository data is unrecoverable.

Operational guidance:

- keep secure backup copy of identity,
- verify decryption on all active machines,
- rotate keys on compromise suspicion.

## 6. When to Encrypt vs Not Encrypt

### Encrypt in repo (good fit)

- Personal tokens and API keys needed across trusted personal machines.
- Low-rotation secret files tied to developer tooling.

### Prefer alternatives

- CI/CD secret delivery (use platform secrets/vault).
- Shared team-owned secrets without multi-recipient governance.
- Highly sensitive production keys that should remain in dedicated keystores.

## 7. MVP Scope and Evolution

### 7.1 MVP scope

- `init`, `encrypt`, `decrypt`, `status`, `rotate` commands.
- Push preflight protections.
- Doctor secrets checks.
- Restrictive decrypted file permission handling.
- Unit/integration coverage for secrets workflows.

### 7.2 Future extensions

- `sops + age` backend for structured merge-friendly encrypted files.
- `dotctl secrets edit` secure edit workflow.
- Optional OS keychain integrations.
- Multi-recipient support.
- Optional pre-commit integration.

## Final Recommendation

Keep age direct as the default and stable path for dotctl secrets.

This provides a practical, low-friction security layer for private dotfiles repositories while preserving compatibility with existing sync architecture.
