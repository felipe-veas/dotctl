# Getting Started

This guide covers the recommended first-time setup flow for `dotctl`.

## Prerequisites

- `git` is required.
- `gh` CLI is required only for HTTPS GitHub repository URLs.

## 1. Install dotctl

See [Installation](./installation.md) for all available methods.

## 2. Initialize dotctl

SSH URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git
```

HTTPS URL:

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git
```

Custom clone path:

```bash
dotctl init --repo <repo-url> --path /custom/path
```

This step clones the dotfiles repository automatically. The remote repository may start empty.

## 3. Add dotfiles

Add the paths you want dotctl to manage:

```bash
dotctl add ~/.zshrc --dry-run
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
```

This copies each path into your repo, updates `manifest.yaml`, backs up the original target, and replaces it with a symlink.

To stop managing a path later without deleting local files or repo sources:

```bash
dotctl remove ~/.zshrc
```

## 4. Or generate a suggested manifest

```bash
dotctl manifest suggest
```

This creates `manifest.suggested.yaml` for review.

## 5. Turn the suggested manifest into your real manifest

Use this workflow:

1. Review `manifest.suggested.yaml`.
2. Keep only the files you want managed across machines.
3. Add `when.os` filters where behavior should differ by OS.
4. Use `mode: copy` only when symlinks are not appropriate.
5. Keep secrets out of the repository; dotctl is intended for non-sensitive dotfiles.
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

## 6. Commit and push the initial repository state

If the remote repository started empty, publish the initial manifest and copied config files:

```bash
git -C ~/.config/dotctl/repo add .
git -C ~/.config/dotctl/repo commit -m "chore: initialize dotfiles manifest and config files"
git -C ~/.config/dotctl/repo push -u origin main
```

Or use:

```bash
dotctl push -m "chore: initialize dotfiles manifest and config files"
```

## 7. Validate and run the first sync

```bash
dotctl doctor
dotctl sync --dry-run
dotctl sync
dotctl status
```

To edit the repo or manifest directly:

```bash
dotctl edit
dotctl edit manifest
```

## Next steps

- [Manifest Specification](./manifest-spec.md)
- [Command Reference](./command-reference.md)
- [Troubleshooting](./troubleshooting.md)
