# Getting Started

This guide covers the recommended first-time setup flow for `dotctl`.

## Prerequisites

- `git` is required.
- `gh` CLI is required only for HTTPS GitHub repository URLs.
- `sops` or `age` is required only if your manifest uses `decrypt: true`.

## 1. Install dotctl

See [Installation](./installation.md) for all available methods.

## 2. Initialize dotctl

SSH URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git --profile laptop
```

HTTPS URL:

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git --profile laptop
```

Custom clone path:

```bash
dotctl init --repo <repo-url> --profile laptop --path /custom/path
```

This step clones the dotfiles repository automatically. The remote repository may start empty.

## 3. Generate a suggested manifest

```bash
dotctl manifest suggest
```

This creates `manifest.suggested.yaml` for review.

## 4. Turn the suggested manifest into your real manifest

Use this workflow:

1. Review `manifest.suggested.yaml`.
2. Keep only the files you want managed across machines.
3. Add `when.profile` and `when.os` filters where behavior should differ by machine or OS.
4. Use `mode: copy` only when symlinks are not appropriate.
5. Use `decrypt: true` for sensitive files and keep encrypted sources as `.enc.*`.
6. Confirm detected files exist in the repository under the suggested `source` paths.
7. If this is your first manifest and the suggested file is already correct, rename it:

```bash
mv manifest.suggested.yaml manifest.yaml
```

1. If you already have a `manifest.yaml`, merge selected entries from `manifest.suggested.yaml` into it.

Typical repository structure after this step:

```text
dotfiles/
  manifest.yaml
  configs/
    zsh/.zshrc
    git/.gitconfig
    nvim/
```

## 5. Commit and push the initial repository state

If the remote repository started empty, publish the initial manifest and copied config files:

```bash
git -C ~/.config/dotctl/repo add .
git -C ~/.config/dotctl/repo commit -m "chore: bootstrap dotfiles manifest and config files"
git -C ~/.config/dotctl/repo push -u origin main
```

Or use:

```bash
dotctl push -m "chore: bootstrap dotfiles manifest and config files"
```

## 6. Validate and run the first sync

```bash
dotctl doctor
dotctl sync --dry-run
dotctl sync
dotctl status
```

## 7. Optional bootstrap hooks

```bash
dotctl bootstrap
```

## Next steps

- [Manifest Specification](./manifest-spec.md)
- [Command Reference](./command-reference.md)
- [Troubleshooting](./troubleshooting.md)
