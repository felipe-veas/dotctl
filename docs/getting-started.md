# Getting Started

This guide covers the first-time setup flow for `dotctl`.

## Prerequisites

- `git`
- `gh` only for HTTPS GitHub repository URLs

## 1. Install dotctl

See [Installation](./installation.md) for all supported methods.

## 2. Initialize the repo

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git
```

For HTTPS URLs:

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git
```

For a custom clone path:

```bash
dotctl init --repo <repo-url> --path /custom/path
```

`dotctl init` clones the repo for you. The remote can start empty.

## 3. Add the first dotfiles

```bash
dotctl add ~/.zshrc --dry-run
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
```

`dotctl add` copies each path into the repo, updates `manifest.yaml`, backs up
the original target, and replaces it with a symlink.

To stop managing a path later:

```bash
dotctl remove ~/.zshrc
```

## 4. Or generate a suggested manifest

```bash
dotctl manifest suggest
```

This writes `manifest.suggested.yaml` for review.

Use the suggested file as a starting point:

1. Review `manifest.suggested.yaml`.
2. Keep only the files you want managed across machines.
3. Add `when.os` filters where behavior differs by OS.
4. Use `mode: copy` only when symlinks are not appropriate.
5. Keep secrets out of the repository.
6. Confirm the suggested `source` paths exist in the repo.
7. If this is a new manifest and the draft looks right, rename it:

   ```bash
   mv manifest.suggested.yaml manifest.yaml
   ```

If you already have a `manifest.yaml`, merge selected entries instead.

Typical repository layout:

```text
dotfiles/
  manifest.yaml
  configs/
    zsh/.zshrc
    git/.gitconfig
    nvim/
```

## 5. Commit and push the initial state

```bash
git -C ~/.config/dotctl/repo add .
git -C ~/.config/dotctl/repo commit -m "chore: initialize dotfiles manifest and config files"
git -C ~/.config/dotctl/repo push -u origin main
```

You can also use:

```bash
dotctl push -m "chore: initialize dotfiles manifest and config files"
```

## 6. Validate and sync

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

- [Command Reference](./command-reference.md)
- [Manifest Specification](./manifest-spec.md)
- [Troubleshooting](./troubleshooting.md)
