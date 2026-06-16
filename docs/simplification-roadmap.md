# dotctl Simplification Roadmap

Date: 2026-06-15

## Purpose

This roadmap defines the path to refocus `dotctl` as a small, predictable dotfile manager.

The original purpose of the project is the source of truth for this roadmap:

> `dotctl` exists to version, install, and synchronize non-sensitive dotfiles from a private Git repository.

The project should optimize for simplicity, safety, and daily usability. Features that turn `dotctl` into a secrets manager, background agent, package manager, system provisioner, or multi-configuration orchestrator should be removed from the core.

## Product thesis

`dotctl` should be a CLI-only tool that helps one user manage one dotfiles repository and one personal configuration set.

The desired workflow should be easy to understand:

```bash
dotctl init git@github.com:<user>/dotfiles.git
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
dotctl sync --dry-run
dotctl sync
dotctl push -m "update dotfiles"
```

## Target scope

### In scope

- One Git repository as the source of truth.
- One configuration set.
- Dotfiles and application configuration directories.
- Symlink-based installation by default.
- Copy mode only for files that cannot safely be symlinked.
- Safe backups before replacing existing local files.
- Restore from backup.
- Drift detection.
- Dry-run planning.
- Basic secret hygiene guardrails to prevent accidental commits.
- Clear documentation and actionable errors.

### Out of scope

- Multiple repositories.
- Multiple profiles.
- Secret encryption/decryption management.
- `age` or `sops` integration.
- macOS status bar app.
- Linux tray app.
- Background daemon behavior.
- Automatic hooks during sync.
- Package installation.
- General system provisioning.
- Nix replacement behavior.

## Final command surface

The long-term CLI should be intentionally small.

### Keep

```bash
dotctl init <repo-url>
dotctl add <path>
dotctl remove <path>
dotctl sync
dotctl status
dotctl doctor
dotctl diff
dotctl push
dotctl pull
dotctl backups list
dotctl backups restore <snapshot>
dotctl edit
dotctl version
```

### Remove

```bash
dotctl secrets ...
dotctl repos ...
dotctl bootstrap
dotctl watch
```

### Reconsider later as separate projects

If these capabilities are needed again, they should not be part of the core CLI:

- `dotctl-agent` for background sync.
- `dotctl-tray` for desktop integrations.
- External secret managers such as 1Password, Bitwarden, `pass`, `sops`, or `age`, documented as compatible external tools rather than built-in features.

## Final manifest shape

The manifest should not include profiles, repo selection, or secret decryption.

Recommended minimal manifest:

```yaml
version: 1

files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc

  - source: configs/nvim
    target: ~/.config/nvim

  - source: configs/git/.gitconfig
    target: ~/.gitconfig
    mode: copy
```

Supported fields:

```yaml
version: 1

vars:
  config_home: ~/.config

files:
  - source: configs/example
    target: "{{ .config_home }}/example"
    mode: symlink
    backup: true
```

Fields to remove:

- `when.profile`
- profile resolution
- multi-profile examples
- `decrypt`
- hook definitions

Potentially keep:

- `when.os`, only if it remains useful for macOS/Linux differences.

If cross-OS support becomes unnecessary, remove `when.os` too and keep the manifest fully linear.

## Architecture direction

The codebase should move toward a small modular CLI with strict boundaries:

```text
cmd/dotctl
internal/cmd       CLI command wiring
internal/config    local config file
internal/manifest  manifest parsing and validation
internal/linker    symlink/copy apply logic
internal/backup    backup and restore
internal/gitops    git pull/push helpers
internal/doctor    health checks
internal/safety    path and secret hygiene policies
internal/output    human and JSON output
```

The domain language should be simple:

- repository
- manifest
- file entry
- source
- target
- backup
- restore
- drift
- sync plan

Avoid terms that imply broader configuration management:

- profile
- repo group
- secret lifecycle
- bootstrap phase
- hook phase
- tray bridge

## Migration principles

1. Remove behavior in small, verifiable steps.
2. Do not mix removals with new feature work unless required for compatibility.
3. Preserve user data and existing dotfile repositories where possible.
4. Provide clear migration errors when old manifest fields are detected.
5. Keep deprecated documentation briefly under `docs/archive/` if historical context is useful.
6. Run tests after each removal phase.

## Phase 0 — Decision record and communication

Objective: make the scope change explicit before touching code.

Tasks:

1. Add an ADR documenting the refocus decision.
2. Update `README.md` with a short statement of what `dotctl` is and is not.
3. Update `docs/roadmap.md` to point to this simplification roadmap.
4. Mark secrets, tray, multirepo, profiles, hooks, and watch mode as deprecated in docs before removal.

Acceptance criteria:

- There is a documented decision explaining why the project is returning to a smaller scope.
- The intended final product can be understood from the README in less than five minutes.

Suggested ADR title:

```text
docs/adr/0001-refocus-dotctl-as-simple-dotfile-manager.md
```

## Phase 1 — Remove tray and desktop app surface

Status: completed in the initial pruning pass.

Objective: remove desktop integration from the core project.

Remove or archive:

- `mac/DotCtl/`
- `linux/tray/`
- tray assets
- tray install scripts
- tray build scripts
- tray packaging references
- tray documentation

Files likely affected:

- `mac/DotCtl/**`
- `linux/tray/**`
- `scripts/build-app-macos.sh`
- `scripts/build-tray-linux.sh`
- `scripts/install-launchagent-macos.sh`
- `scripts/install-tray-autostart-linux.sh`
- `Makefile`
- `.github/workflows/ci.yml`
- `.goreleaser.yaml`
- `README.md`
- `docs/installation.md`

Acceptance criteria:

- The project builds only the CLI.
- CI no longer builds tray artifacts.
- Installation docs no longer mention tray apps.
- No runtime CLI behavior depends on tray-specific packages.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

## Phase 2 — Remove secrets and decryption from core

Status: completed in the initial pruning pass.

Objective: stop treating `dotctl` as a secret manager.

Remove:

- `dotctl secrets` command.
- `internal/secrets` package.
- `internal/decrypt` package.
- `decrypt: true` manifest field.
- `age` dependency if no longer used.
- `sops` and `age` runtime requirements from docs.
- secrets design docs from active documentation.

Replace with guardrails:

- Keep sensitive filename detection.
- Keep `.gitignore` recommendations.
- Keep push/doctor warnings for obvious secret names.
- Document that secrets should be managed outside `dotctl`.

Manifest migration behavior:

- If `decrypt` is present, return a clear error:

```text
manifest uses deprecated field "decrypt". dotctl no longer manages secret decryption. Store only non-sensitive dotfiles or decrypt secrets outside dotctl.
```

Files likely affected:

- `internal/cmd/secrets.go`
- `internal/secrets/**`
- `internal/decrypt/**`
- `internal/linker/linker.go`
- `internal/manifest/types.go`
- `internal/manifest/parser.go`
- `internal/cmd/doctor.go`
- `internal/cmd/diff.go`
- `go.mod`
- `go.sum`
- `README.md`
- `docs/security-model.md`
- `docs/secrets-design.md`
- `docs/manifest-spec.md`
- `docs/command-reference.md`

Acceptance criteria:

- `dotctl secrets` no longer exists.
- Manifest parsing rejects `decrypt` with an actionable error.
- `sync`, `diff`, and `doctor` do not shell out to `age` or `sops`.
- Secret hygiene remains as warning/preflight behavior only.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

## Phase 3 — Remove multirepo support

Status: completed in the initial pruning pass.

Objective: return to one repository per local dotctl installation.

Remove:

- `dotctl repos` command.
- `--repo-name` global flag.
- `repos` list from config.
- `active_repo` from config.
- repo name normalization and switching behavior.
- multi-repo docs and examples.

Target local config:

```yaml
repo:
  url: git@github.com:<user>/dotfiles.git
  path: ~/.config/dotctl/repo
backup:
  keep: 20
last_sync: null
```

Migration behavior:

- If old config has one repo, migrate automatically to the simplified shape.
- If old config has multiple repos, fail with a clear message asking the user to select one manually before upgrading.

Files likely affected:

- `internal/cmd/repos.go`
- `internal/cmd/root.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/cmd/init.go`
- `internal/cmd/status.go`
- `internal/cmd/doctor.go`
- `README.md`
- `docs/command-reference.md`
- `docs/getting-started.md`

Acceptance criteria:

- No `repos` command exists.
- No `--repo-name` flag exists.
- Config has one repository.
- All commands operate against the single configured repository.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
go run ./cmd/dotctl --help
```

## Phase 4 — Remove profiles

Status: completed in the initial pruning pass.

Objective: simplify the model to one configuration set.

Remove:

- `--profile` global flag.
- `profile` from local config.
- `when.profile` from manifest.
- profile package and tests if no longer needed.
- profile-specific docs and examples.
- profile in commit message generation.

Keep only if needed:

- OS detection for optional `when.os` support.

Target manifest behavior:

- All entries apply unless filtered by OS.
- If `when.profile` is present, fail with a clear deprecation error.

Suggested error:

```text
manifest uses deprecated field "when.profile". dotctl now manages a single configuration set. Remove profile filters from manifest.yaml.
```

Files likely affected:

- `internal/profile/**`
- `internal/manifest/resolver.go`
- `internal/manifest/types.go`
- `internal/manifest/parser.go`
- `internal/cmd/root.go`
- `internal/cmd/init.go`
- `internal/cmd/sync.go`
- `internal/cmd/status.go`
- `internal/cmd/doctor.go`
- `internal/gitops/gitops.go`
- `README.md`
- `docs/manifest-spec.md`
- `docs/getting-started.md`

Acceptance criteria:

- No CLI flag or config field mentions profile.
- `manifest.yaml` has no profile concept.
- Existing profile-specific tests are removed or rewritten for single-config behavior.
- Commit messages do not mention profile.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

## Phase 5 — Remove automatic hooks and bootstrap

Status: completed in the pruning pass.

Objective: eliminate arbitrary command execution from the normal dotfile sync path.

Remove:

- `dotctl bootstrap` command.
- `hooks` section from manifest.
- `pre_sync` and `post_sync` execution.
- hook result JSON from sync output.
- hook docs and tests.

Migration behavior:

- If `hooks` are present in the manifest, fail with a clear message:

```text
manifest uses deprecated field "hooks". dotctl no longer executes commands from manifest.yaml. Run setup commands manually outside dotctl.
```

Files likely affected:

- `internal/cmd/hooks.go`
- `internal/cmd/bootstrap.go`
- `internal/cmd/sync.go`
- `internal/manifest/types.go`
- `internal/manifest/resolver.go`
- `docs/manifest-spec.md`
- `docs/sync-lifecycle.md`
- `README.md`

Acceptance criteria:

- `sync` only performs Git pull, manifest resolution, file apply, optional push, backup rotation, and status updates.
- No shell command execution is driven by manifest content.
- Manifest hooks are rejected with an actionable message.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

## Phase 6 — Reconsider watch mode

Status: completed in the pruning pass.

Objective: remove background-like behavior from the core unless it has a strong single-CLI use case.

Recommended decision: remove `dotctl watch` from core.

Rationale:

- Watch mode encourages implicit background sync behavior.
- It increases failure modes around loops, race conditions, and surprising pushes.
- Manual `sync` is easier to reason about for a personal dotfile manager.

Remove:

- `dotctl watch` command.
- `fsnotify` dependency if no longer used.
- watch docs and tests.

Acceptance criteria:

- No long-running command remains in the core CLI.
- `dotctl` is fully explicit and user-triggered.

Verification:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

## Phase 7 — Strengthen the focused core

Objective: after removing scope creep, improve the features that directly serve dotfile management.

### Add `dotctl add <path>`

Status: completed for the current MVP.

Behavior:

1. Validate the path is under the user's home directory.
2. Reject sensitive-looking files by default.
3. Choose a deterministic repo source path.
4. Copy the file or directory into the repo.
5. Add or update `manifest.yaml`.
6. Replace the original target with a symlink after backup.
7. Support `--dry-run`.

Example:

```bash
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
```

### Add `dotctl remove <path>`

Status: completed for the current untrack-only MVP. Restore behavior remains a future enhancement.

Behavior:

1. Remove the manifest entry.
2. Optionally restore the target from the repo copy.
3. Avoid deleting user data by default.

### Add backup restore commands

Status: completed for the current MVP. Restore requires `--force` unless `--dry-run`; snapshot metadata now preserves logical directory entries for exact restore.

Commands:

```bash
dotctl backups list
dotctl backups restore <snapshot>
```

### Add `dotctl edit`

Status: completed for the current MVP. Platform-default editor fallback remains a future enhancement.

Open the repository or manifest with `$VISUAL`, `$EDITOR`, or a platform default:

```bash
dotctl edit
dotctl edit manifest
dotctl edit repo
```

### Strengthen path safety

Status: completed for the current pruning pass.

Rules:

- Targets must resolve under `$HOME` by default.
- Reject home root itself.
- Reject parent traversal.
- Warn on sensitive paths such as `.ssh`, `.gnupg`, `.kube`, `.aws`, `.config/gh`, `.config/gcloud`, and `.env` files.

Acceptance criteria:

- The core workflow can be completed without manually editing YAML for common use cases.
- Destructive operations always show a dry-run plan or require explicit confirmation.

## Phase 8 — Rewrite documentation around the smaller product

Objective: make the project understandable and attractive again.

Recommended README structure:

1. What `dotctl` is.
2. What `dotctl` is not.
3. Quickstart.
4. Daily workflow.
5. Manifest format.
6. Adding/removing dotfiles.
7. Sync and drift detection.
8. Backups and restore.
9. Safety model.
10. Troubleshooting.

Docs to keep active:

- `docs/getting-started.md`
- `docs/installation.md`
- `docs/manifest-spec.md`
- `docs/command-reference.md`
- `docs/sync-lifecycle.md`
- `docs/troubleshooting.md`
- `docs/security-model.md`
- `docs/simplification-roadmap.md`

Docs to archive or remove:

- secrets design docs
- tray docs
- multirepo docs
- profile-heavy examples
- historical scope docs that no longer reflect the product

Acceptance criteria:

- A new user can understand the complete product in one README pass.
- The docs no longer describe removed features as active capabilities.

## Release strategy

This simplification is a breaking change. Treat it as a major release.

Recommended versioning:

- Current feature-rich line: final `v1.x` release.
- Simplified focused line: `v2.0.0`.

Migration guidance:

- Explain removed commands.
- Explain removed manifest fields.
- Provide before/after examples.
- Provide a one-time migration checklist.

Suggested migration checklist:

1. Remove `decrypt` entries from manifest.
2. Remove `hooks` from manifest.
3. Remove `when.profile` blocks.
4. Select one repository if using multirepo.
5. Run `dotctl doctor`.
6. Run `dotctl sync --dry-run`.
7. Run `dotctl sync`.

## Testing strategy

Prioritize tests around the simplified core:

- Manifest parsing rejects deprecated fields.
- Single-repo config loads and saves correctly.
- `dotctl add` creates deterministic source paths.
- `sync --dry-run` does not mutate filesystem or Git state.
- Existing targets are backed up before replacement.
- Restore recovers files and symlinks correctly.
- Sensitive-looking files are blocked or warned before add/push.
- Targets outside `$HOME` are rejected by default.
- Symlinks inside copied directories are preserved.

Minimum verification after each phase:

```bash
go test ./... -race -count=1
go vet ./...
go build ./cmd/dotctl
```

If available:

```bash
golangci-lint run ./...
pre-commit run --all-files
```

## Definition of done

The simplification effort is complete when:

- The binary exposes only the focused CLI commands.
- The codebase has no active tray, secrets, multirepo, profile, hook, or watch implementation.
- The manifest is small and dotfile-specific.
- The docs match the actual behavior.
- The main workflow works without manual YAML editing for common dotfiles.
- Tests pass with race detector.
- The README clearly explains what `dotctl` does and intentionally does not do.

## Final target

The final product should feel boring, explicit, and reliable:

```bash
dotctl init git@github.com:<user>/dotfiles.git
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
dotctl status
dotctl sync --dry-run
dotctl sync
dotctl push -m "update dotfiles"
```

That workflow is the product. Everything else should exist only if it makes that workflow safer, clearer, or faster.
