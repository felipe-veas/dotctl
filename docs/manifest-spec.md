# Manifest Specification

`manifest.yaml` defines what `dotctl` manages and how each target is applied.

## Location

`manifest.yaml` must be in the root of your dotfiles repository.

## Schema

```yaml
version: 1

vars:
  config_home: "~/.config"

files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc

  - source: configs/git/config
    target: "{{ .config_home }}/git/config"

  - source: configs/app/config.yaml
    target: "{{ .config_home }}/app/config.yaml"
    mode: copy

ignore:
  - ".env"
  - "*.pem"
```

## Top-level keys

- `version`: currently `1`.
- `vars`: reusable variables for templated targets.
- `files`: file or directory rules.
- `ignore`: source patterns that should not be applied.

## `files[]` fields

- `source` (required): relative path inside repo.
- `target` (required): destination path on the local machine. It may use `~`, an absolute path, or a template, but it must resolve strictly under the user's home directory.
- `mode`: `symlink` (default) or `copy`.
- `when.os`: `darwin`, `linux`, or list.
- `backup`: `true` by default.

## Template variables in `target`

Built-in:

- `home`
- `os`
- `arch`
- `hostname`

User-defined:

- any key declared under `vars`.

## Target path safety

`dotctl` rejects manifest targets that are relative, outside `$HOME`, equal to `$HOME`, or escape `$HOME` through parent traversal such as `~/../outside`.

Sensitive-looking targets under credential-heavy paths such as `.ssh`, `.gnupg`, `.kube`, `.aws`, `.config/gh`, `.config/gcloud`, and `.env` files are allowed but reported as warnings.
