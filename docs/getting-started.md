# Getting Started

This guide covers the fastest path to set up `dotctl` on a new machine.

## Prerequisites

- `git` is required.
- `gh` CLI is required only for HTTPS GitHub repo URLs.
- `sops` or `age` is required only if your manifest uses `decrypt: true`.

## 1. Install dotctl

See [Installation](./installation.md) for all methods.

## 2. Prepare your dotfiles repository

Your dotfiles repository must include `manifest.yaml` at the root.

Example:

```text
dotfiles/
  manifest.yaml
  configs/
    zsh/.zshrc
    git/config
```

## 3. Initialize dotctl

SSH URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git --profile laptop
```

HTTPS URL:

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git --profile laptop
```

## 4. Validate and run first sync

```bash
dotctl doctor
dotctl sync --dry-run
dotctl sync
dotctl status
```

## 5. Optional bootstrap hooks

```bash
dotctl bootstrap
```

## Next steps

- [Manifest Specification](./manifest-spec.md)
- [Command Reference](./command-reference.md)
- [Troubleshooting](./troubleshooting.md)
