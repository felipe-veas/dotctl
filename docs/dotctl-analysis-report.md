# dotctl Analysis Report

Date: 2026-06-15

## Executive summary

`dotctl` is aligned with its original goal: a focused tool for versioning and synchronizing dotfiles through a private Git repository. Compared with a full Nix-based setup, its main advantage is operational simplicity: explicit files, a readable manifest, backups, rollback, and a CLI that maps well to the mental model of managing personal configuration.

The project has a solid technical base: Cobra-based CLI, declarative manifest, profile and OS filters, backup snapshots, rollback on sync failures, locking, `doctor`, JSON output, secret management with `age`, CI, and meaningful test coverage in several core packages.

The main risk is not the overall architecture. The main risk is that some convenience features currently expand the blast radius too much:

1. Hooks from the repository can execute local shell commands.
2. Manifest targets can point to broad filesystem locations.
3. `sync` can backfill missing repository sources from local targets.
4. `sync` can push changes through `git add -A` without the same secret preflight protection as `dotctl push`.
5. Secret decryption is architecturally split between native `age` support and external `age`/`sops` binaries.

Recommendation: keep the tool and continue investing in it, but prioritize security hardening and core UX simplification before adding more product surface.

## Scope reviewed

- Go CLI implementation under `cmd/`, `internal/`, and `pkg/`.
- Manifest parsing, resolution, sync, linker, backup, secrets, git operations, hooks, and doctor flows.
- Documentation under `README.md` and `docs/`.
- CI/CD workflows and pre-commit configuration.
- macOS and Linux tray-related code at a high level.

## Current strengths

- Clear CLI structure with commands such as `init`, `sync`, `doctor`, `status`, `diff`, `push`, `secrets`, and `repos`.
- Declarative `manifest.yaml` with OS/profile filtering.
- Backup and rollback mechanism around filesystem changes.
- Sync lock to prevent concurrent apply/push flows.
- `doctor` validates repository state, manifest, symlinks, secrets, ignore patterns, and decrypt tooling.
- Native `age` support exists in `internal/secrets`.
- JSON output mode supports automation and tray integrations.
- CI uses explicit `contents: read` permissions in the main CI workflow.
- Tests pass with the race detector.
- Secret-related docs and model are already present.

## Threat model summary

### Critical assets

- `~/.config/dotctl/age-identity.txt`.
- Plaintext secrets deployed by `decrypt: true`.
- Local dotfiles and application configuration.
- SSH keys, GitHub credentials, and shell environment.
- Repository integrity and release artifacts.

### Trust boundaries

- Remote Git repository to local machine during `pull` and `sync`.
- `manifest.yaml` to local filesystem operations.
- Repository-defined hooks to shell execution.
- Local `PATH` to external commands such as `git`, `gh`, `sops`, `age`, and `dotctl`.
- GitHub Actions to release artifacts and Homebrew publishing.
- Tray/autostart integrations to persistent local execution.

### STRIDE summary

- **Spoofing:** malicious binaries earlier in `PATH`, especially for `git`, `gh`, `sops`, `age`, or tray-discovered `dotctl`.
- **Tampering:** compromised manifest can overwrite or remove local files; release artifacts lack documented verification.
- **Repudiation:** local operations are logged, but there is no immutable audit trail.
- **Information disclosure:** secrets can leak through backfill, backups, diffs, logs, or broad `git add -A` behavior.
- **Denial of service:** destructive filesystem operations against broad targets can break user environments.
- **Elevation of privilege:** hooks and external command execution can run arbitrary code as the user.

## Prioritized findings

### 1. High — Repository hooks are local code execution

Evidence:

- `internal/cmd/hooks.go:50` executes hooks through `/bin/sh -c`.
- `internal/cmd/sync.go:109-112` resolves and runs `pre_sync` and `post_sync` hooks.
- `bootstrap` also executes manifest-defined hooks.

Impact:

A compromised repository commit can execute arbitrary commands during `sync`, `watch`, tray-triggered sync, or bootstrap. This can exfiltrate the age identity file, SSH agent access, GitHub credentials, shell history, and other local data.

Recommendation:

- Disable hooks by default or require `--allow-hooks`.
- Add a local, non-versioned hook allowlist based on command path and SHA-256.
- Prompt when hooks change after `git pull`.
- Add execution timeouts.
- Run hooks with a minimal environment.
- Document hooks as trusted code, not configuration.

### 2. High — Manifest targets can write or remove broad filesystem paths

Evidence:

- `internal/manifest/parser.go:109-125` resolves target paths but does not enforce a filesystem policy.
- `internal/linker/linker.go:142`, `150`, `200`, `207`, `449`, and `491` use `os.RemoveAll` on targets.

Impact:

A malicious or mistaken manifest can overwrite or delete any path writable by the user. If the command is ever run with elevated privileges, the impact expands significantly.

Recommendation:

- Fail closed by default.
- Allow targets only under `$HOME`, `$XDG_CONFIG_HOME`, and other explicit XDG directories.
- Reject `/`, `$HOME` itself, empty paths, `..`, and high-risk directories.
- Require an explicit override such as `--allow-target-outside-home` for exceptions.
- Show target risk classification in `sync --dry-run`.

### 3. High — Automatic backfill can copy sensitive local files into the repository

Evidence:

- `internal/cmd/sync.go:80` calls backfill during sync.
- `internal/cmd/managed_sources.go:190-238` copies missing repo sources from local targets.
- `internal/cmd/sync.go:253-255` pushes changes after apply.

Impact:

If a manifest references a missing source and a sensitive local target, `sync` can copy the local file into the repository and later push it. This is a direct data exfiltration path for `.env`, SSH keys, tokens, and credentials.

Recommendation:

- Remove implicit backfill from `sync`.
- Move it to an explicit command such as `dotctl manifest backfill`.
- Require confirmation and show exact source/target pairs.
- Block sensitive names and directories by default.
- Never auto-push backfilled files without explicit user action.

### 4. High — `sync` bypasses the stronger secret preflight used by `dotctl push`

Evidence:

- `internal/cmd/push.go:65-75` runs a preflight secret check.
- `internal/cmd/sync.go:253-255` calls `gitops.Push` directly.
- `internal/gitops/gitops.go:409` stages everything with `git add -A`.

Impact:

Sensitive untracked files can be added and pushed during `sync`. The current preflight also checks tracked files, while `git add -A` can add new untracked files.

Recommendation:

- Move the secret preflight into `gitops.Push` or a shared push wrapper.
- Scan staged, modified, and untracked files before `git add -A`.
- Fail closed if the scan fails.
- Require `--force` only with explicit warnings and clear remediation.

### 5. Medium/high — Decryption architecture is inconsistent

Evidence:

- `internal/secrets/age_backend.go` implements native `age` encryption/decryption.
- `internal/decrypt/decrypt.go` shells out to `age` or `sops` for manifest-driven sync decryption.

Impact:

Users can encrypt with built-in `age`, but syncing encrypted files requires an external binary in `PATH`. This weakens the single-binary experience and increases failure modes and spoofing risk.

Recommendation:

- Use native `age` decryption for files managed by `dotctl secrets`.
- Keep external `sops` only for explicit compatibility.
- Update `doctor` and documentation to distinguish native age support from external `sops` requirements.

### 6. Medium — Source symlinks can escape the repository in copy flows

Evidence:

- `internal/linker/linker.go:49-76` validates source paths lexically.
- `os.Stat` and `os.Open` later follow symlinks.

Impact:

A repository source path could be a symlink pointing outside the repo. In copy mode, dotctl may copy content from outside the intended repository root.

Recommendation:

- Use `Lstat` for source validation.
- Reject source symlinks by default, or resolve with `EvalSymlinks` and verify the real path remains inside the repository root.
- Add tests for symlink escape attempts.

### 7. Medium — Directory copy and backup do not preserve symlinks correctly

Evidence:

- `internal/backup/backup.go:215-237` copies files during directory backup.
- `internal/linker/linker.go:325-349` copies directory entries and follows symlink targets.

Impact:

Symlinks inside directories may become regular files or cause failures if broken. This can silently change the semantics of backed-up or copied configuration directories.

Recommendation:

- Detect symlinks with `Lstat`.
- Recreate symlinks with `Readlink` and `Symlink`.
- Add tests for valid and broken symlinks inside directories.

### 8. Medium — Release supply chain can be hardened

Evidence:

- `.github/workflows/goreleaser.yml` uses `version: latest` for GoReleaser.
- Release workflow has `contents: write` and publishing tokens.
- Installation docs show direct binary download without checksum/signature verification.

Impact:

Release tooling drift or compromise can affect published artifacts. Users install a binary with broad access to local configuration and secrets.

Recommendation:

- Pin GoReleaser to a specific version.
- Use protected GitHub Environments for release publishing.
- Ensure workflow tokens are fine-grained and scoped.
- Publish signed checksums and document verification.
- Consider cosign/minisign/GPG signatures and provenance attestations.

### 9. Medium — Identity and backup permissions should be verified more strictly

Evidence:

- `internal/secrets/age_backend.go` creates identity files with `0600`, which is good.
- `internal/secrets/identity.go` reads identity files without validating owner or permissions.
- `internal/backup/backup.go` creates backup directories with `0755`.

Impact:

If permissions drift, sensitive identity material or plaintext backup content may be exposed to other local users.

Recommendation:

- Warn or fail if identity file permissions are broader than `0600`.
- Warn if parent directories are group/world writable.
- Create backup roots with `0700`.
- Cap sensitive restored/copied files to `0600` where appropriate.

### 10. Medium — `dotctl diff --details` can expose decrypted secrets

Evidence:

- `internal/cmd/diff.go` can decrypt and render detailed diffs.
- `docs/command-reference.md` recommends `dotctl diff --details` generally.

Impact:

Secrets may be printed to terminal, terminal scrollback, captured logs, or CI output.

Recommendation:

- Redact `decrypt: true` diffs by default.
- Require `--show-secrets` for plaintext secret diffs.
- Add a clear warning before printing secret material.

### 11. Low/medium — Secret scanning exists in pre-commit but not CI

Evidence:

- `.pre-commit-config.yaml` includes gitleaks.
- `.github/workflows/ci.yml` does not run gitleaks or pre-commit across the repository.

Impact:

Contributors or alternate machines can bypass local pre-commit hooks.

Recommendation:

- Add a CI job for gitleaks or `pre-commit run --all-files`.
- Keep workflow permissions as `contents: read` for this job.

## UX and usability review

### Main user journeys

1. First machine setup: install, `dotctl init`, `dotctl manifest suggest`, review manifest, commit, `dotctl sync`.
2. Second machine setup: install, import age identity if needed, `dotctl init`, `dotctl doctor`, `dotctl sync`.
3. Daily operation: edit dotfiles, `dotctl push`, or pull/apply with `dotctl sync`.
4. Secret lifecycle: `dotctl secrets init`, encrypt file, add manifest entry with `decrypt: true`, sync.
5. Recovery: rely on automatic backup and rollback, but manual restore flow is not yet first-class.

### UX strengths

- Command names are mostly predictable.
- `doctor` is a strong onboarding and troubleshooting primitive.
- `--dry-run` is available and should be emphasized more.
- `--json` supports automation.
- The manifest format is understandable and portable.

### UX friction points

#### Hidden repository model is not explained early enough

The docs mention `~/.config/dotctl/repo`, but the mental model should be introduced immediately:

> dotctl keeps a local clone of your dotfiles repository under `~/.config/dotctl/repo`. Most managed files in your home directory are symlinks to that clone. You can edit files normally through `$HOME`, then use `dotctl push` to commit and publish changes.

#### `manifest suggest` copies by default

This can leave orphaned files in the repository if the user later removes entries from `manifest.suggested.yaml`.

Recommendation:

- Make suggestion generation non-copying by default, or
- Add an interactive confirm-per-file flow, or
- Introduce a separate `dotctl add` workflow.

#### There is no `dotctl add <path>`

For the original dotfile-versioning use case, this would likely be the highest-impact UX command.

Suggested behavior:

```bash
dotctl add ~/.zshrc
dotctl add ~/.config/nvim --profile laptop
dotctl add ~/.config/ghostty --mode symlink
```

The command should:

1. Copy or move the file into the repo.
2. Add/update the manifest entry.
3. Backup the local target if needed.
4. Create the symlink or copy.
5. Show a dry-run plan by default for risky paths.

#### There is no `dotctl edit`

Recommended commands:

```bash
dotctl edit manifest
dotctl edit repo
dotctl edit config
```

This should use `$VISUAL`, `$EDITOR`, or a platform default.

#### Backup recovery is not first-class

Backups exist, but user-facing recovery is underdocumented.

Recommended commands:

```bash
dotctl backups list
dotctl backups show <snapshot>
dotctl backups restore <snapshot>
```

#### Second-machine secrets onboarding is still manual

Copying `age-identity.txt` is acceptable, but the docs and `doctor` should give exact guidance when encrypted files exist and no identity is present.

## Architecture assessment

### Keep the simple model

The current model is appropriate:

- Git repository as source of truth.
- Manifest as declarative mapping.
- Symlink by default.
- Copy only when symlink is not appropriate.
- Encrypted source in repo, plaintext deployed locally only when necessary.

Avoid turning this into a full package/configuration orchestrator. Nix already covers that domain. `dotctl` should stay focused on dotfiles and local config sync.

### Recommended product boundary

`dotctl` should manage:

- Dotfiles.
- App config directories.
- Encrypted config files.
- Profile-specific machine differences.
- Safe backup/restore around target changes.

`dotctl` should avoid or keep very explicit:

- Installing system packages.
- Running arbitrary setup scripts by default.
- Managing privileged system files.
- Acting as a general provisioning framework.

## Technical quality notes

### Positive technical observations

- `sync` uses a lock to prevent concurrent runs.
- Rollback is implemented and attempted when apply fails after changes.
- Config migration supports legacy single-repo and newer multi-repo shape.
- `manifest.Parse` validates duplicate targets, source escaping, unsupported modes, and decrypt mode constraints.
- `secrets` uses high-level `filippo.io/age` primitives rather than custom cryptography.
- `logging` includes token redaction for GitHub-style tokens.

### Areas to improve

- Centralize filesystem policy for targets instead of distributing checks across parser/linker/sync.
- Centralize push safety in the git layer.
- Reduce reliance on external binaries for age decryption.
- Preserve symlinks consistently across all copy helpers.
- Close logger file descriptors explicitly on exit.
- Increase tests for `cmd`, `gitops`, `decrypt`, and `linker` edge cases.

## Verification performed

The following commands were executed successfully:

```bash
go test ./... -race -count=1
go vet ./...
go test -cover ./...
go build ./cmd/dotctl
```

`golangci-lint` was attempted but is not installed locally:

```text
zsh:1: command not found: golangci-lint
```

Coverage summary from `go test -cover ./...`:

| Package | Coverage |
|---|---:|
| `internal/profile` | 100.0% |
| `internal/manifest` | 90.7% |
| `internal/output` | 86.7% |
| `internal/backup` | 77.7% |
| `internal/config` | 74.8% |
| `internal/logging` | 74.0% |
| `internal/secrets` | 71.8% |
| `internal/lock` | 64.3% |
| `internal/linker` | 62.7% |
| `internal/platform` | 58.8% |
| `internal/gitops` | 57.1% |
| `internal/auth` | 51.2% |
| `internal/decrypt` | 46.7% |
| `internal/cmd` | 45.9% |

## Recommended roadmap

### Phase 1 — Security hardening

1. Protect hooks: disabled by default, allowlist by hash, timeout, prompt on change.
2. Add target path policy: restrict to safe home/XDG directories by default.
3. Remove implicit backfill from `sync`.
4. Centralize secret preflight in `gitops.Push` or a common push wrapper.
5. Validate source symlinks and prevent repo escape.
6. Pin GoReleaser version and document binary verification.
7. Add CI secret scanning.

### Phase 2 — Core UX improvements

1. Add `dotctl add <path>`.
2. Add `dotctl edit manifest|repo|config`.
3. Add `dotctl backups list|show|restore`.
4. Make `manifest suggest` less surprising by separating scan from copy.
5. Improve Getting Started with the repository/symlink mental model.
6. Improve second-machine onboarding for secrets.

### Phase 3 — Technical cleanup and reliability

1. Use native `age` decryption for manifest `decrypt: true` where applicable.
2. Preserve symlinks in all directory copy and backup paths.
3. Add tests for malicious manifests, symlink escapes, untracked secret push, hooks, and rollback failures.
4. Close logging resources explicitly.
5. Add structured error guidance for common Git identity and auth failures.

## Suggested near-term implementation order

1. Fix `sync` push preflight and backfill behavior first, because they directly affect accidental secret exposure.
2. Add target policy validation before expanding the feature set.
3. Harden hooks before recommending tray/watch workflows broadly.
4. Implement `dotctl add` as the main UX improvement after the safety model is stronger.
5. Update documentation and roadmap to reflect these priorities.

## Final recommendation

Continue with `dotctl` as a focused dotfile manager. Do not try to recreate Nix. The tool’s value is its narrow scope, readable state, and operational reversibility.

The best next step is to make unsafe states difficult or impossible: constrain targets, make backfill explicit, centralize push safety, and treat hooks as trusted code requiring approval. After that, invest in `dotctl add`, `dotctl edit`, and backup restore flows to make the simple workflow feel complete.
