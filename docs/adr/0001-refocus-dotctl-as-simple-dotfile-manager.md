# ADR 0001: Refocus dotctl as a Simple Dotfile Manager

Date: 2026-06-15

Status: Accepted

## Context

`dotctl` started as a small tool to version and synchronize dotfiles through a private Git repository. Over time, the project expanded beyond that original purpose and added capabilities such as:

- encrypted secret management;
- decrypt-on-sync behavior;
- macOS status bar integration;
- Linux tray integration;
- background/watch-style workflows;
- multi-repository configuration;
- profile-based configuration;
- manifest-driven hooks and bootstrap commands.

These features are individually useful in some contexts, but they shift the product away from its core job. They also increase maintenance cost, security exposure, documentation burden, and cognitive load for the user.

The intended use case is now clarified:

> `dotctl` should help one user manage one set of non-sensitive dotfiles from one private Git repository.

Secrets do not need to be managed by `dotctl` because dotfiles are expected to be non-sensitive. Sensitive material should be excluded from the repository or handled by dedicated secret-management tools outside `dotctl`.

Desktop tray apps and background agents are also outside the desired product shape. The preferred operational model is explicit CLI usage.

Profiles and multirepo support are no longer required because the project should manage a single configuration set.

## Decision

Refocus `dotctl` as a CLI-only, single-repository, single-configuration dotfile manager.

The core product will be:

- one Git repository as source of truth;
- one configuration set;
- manifest-driven dotfile mapping;
- symlink-first installation;
- optional copy mode for special cases;
- safe backups before replacement;
- restore from backup;
- drift/status checks;
- explicit pull, sync, and push commands;
- safety guardrails to prevent accidental tracking of sensitive files.

The following capabilities will be removed from the core:

- built-in secrets management;
- `age` and `sops` encryption/decryption integration;
- `decrypt` manifest behavior;
- macOS status bar app;
- Linux tray app;
- background/watch mode;
- multi-repository support;
- profile support;
- manifest-driven hooks;
- bootstrap command execution.

The target CLI surface is intentionally small:

```bash
dotctl init <repo-url>
dotctl add <path>
dotctl remove <path>
dotctl sync
dotctl status
dotctl doctor
dotctl diff
dotctl pull
dotctl push
dotctl backups list
dotctl backups restore <snapshot>
dotctl edit
dotctl version
```

The target manifest is also intentionally small:

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

## Rationale

### Simplicity is the product advantage

The value of `dotctl` is not replacing Nix, secret managers, package managers, or desktop agents. Its value is being easy to understand and safe to run:

1. files live in a Git repository;
2. local paths point to those files;
3. `sync` applies the manifest;
4. backups make mistakes recoverable.

Keeping this model small improves usability and maintainability.

### Secret management is a different problem

Secret management introduces key lifecycle, encryption formats, rotation, plaintext exposure, backup handling, and recovery concerns. These are not core dotfile-management responsibilities.

`dotctl` should still help prevent accidental secret commits through warnings and preflight checks, but it should not own encryption or decryption.

### Tray and background behavior make sync less explicit

A personal dotfile manager should be predictable. Tray apps and watch mode encourage implicit background changes and surprising pushes. Manual CLI execution is easier to reason about, easier to test, and easier to document.

### Profiles and multirepo support are unnecessary for the clarified use case

The user wants one configuration set. Supporting profiles and multiple repositories adds branching logic, configuration migration complexity, and documentation overhead without serving the current product goal.

## Consequences

### Positive consequences

- Smaller codebase.
- Lower security exposure.
- Simpler documentation.
- Easier onboarding.
- Fewer runtime dependencies.
- Clearer mental model.
- Easier test strategy.
- More focused roadmap.
- Stronger alignment with the original dotfile-versioning goal.

### Negative consequences

- Existing users of secrets, profiles, multirepo, tray apps, hooks, or watch mode will need to migrate.
- This is a breaking change and should be released as a major version.
- Some convenience workflows will be removed.
- Cross-machine or OS-specific branching becomes less flexible if profile support is removed.
- Users needing secret management must use external tools.

### Neutral trade-offs

- `dotctl` becomes less general-purpose but more coherent.
- The project gives up some feature breadth to improve trust and usability.
- Some removed features could return later as separate projects or documented integrations, but not as core responsibilities.

## Migration strategy

This decision should be implemented through a phased simplification, not a large unverified rewrite.

The migration roadmap is documented in:

```text
docs/simplification-roadmap.md
```

High-level phases:

1. Document the scope decision.
2. Remove tray and desktop app surface.
3. Remove secrets and decryption from core.
4. Remove multirepo support.
5. Remove profile support.
6. Remove automatic hooks and bootstrap.
7. Remove or reconsider watch mode.
8. Strengthen the focused core with `add`, `remove`, `edit`, and backup restore.
9. Rewrite documentation around the smaller product.

## Compatibility policy

Because this is a product-level scope reduction, it should be treated as a breaking change.

Recommended release approach:

- keep the current feature-rich line as the final `v1.x` series;
- release the simplified product as `v2.0.0`;
- provide explicit migration notes for removed commands and manifest fields.

Deprecated manifest fields should fail with actionable errors rather than being silently ignored.

Examples:

```text
manifest uses deprecated field "decrypt". dotctl no longer manages secret decryption. Store only non-sensitive dotfiles or decrypt secrets outside dotctl.
```

```text
manifest uses deprecated field "when.profile". dotctl now manages a single configuration set. Remove profile filters from manifest.yaml.
```

```text
manifest uses deprecated field "hooks". dotctl no longer executes commands from manifest.yaml. Run setup commands manually outside dotctl.
```

## Security posture after this decision

The simplified product should still include basic safety controls:

- block or warn on sensitive-looking filenames;
- validate target paths before filesystem mutation;
- default to symlinks;
- backup before replacement;
- support dry-run planning;
- avoid manifest-driven shell execution;
- keep sync explicit and user-triggered.

The goal is not to become a high-security secret system. The goal is to make a dotfile manager that avoids obvious footguns.

## Decision outcome

Accepted.

`dotctl` will be simplified into a focused, explicit, single-repository, single-configuration dotfile manager. Features outside that scope will be removed from the core and, if needed later, considered as separate tools or external integrations.
