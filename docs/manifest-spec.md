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

  - source: configs/app/config.enc.yaml
    target: "{{ .config_home }}/app/config.yaml"
    mode: copy
    decrypt: true

ignore:
  - ".env"
  - "*.pem"

hooks:
  pre_sync:
    - command: ./scripts/pre-sync.sh
  post_sync:
    - command: ./scripts/post-sync.sh
  bootstrap:
    - command: ./scripts/bootstrap.sh
      when:
        os: darwin
```

## Top-level keys

- `version`: currently `1`.
- `vars`: reusable variables for templated targets.
- `files`: file or directory rules.
- `ignore`: source patterns that should not be applied.
- `hooks`: lifecycle hooks (`pre_sync`, `post_sync`, `bootstrap`).

## `files[]` fields

- `source` (required): relative path inside repo.
- `target` (required): destination path in local machine.
- `mode`: `symlink` (default) or `copy`.
- `when.os`: `darwin`, `linux`, or list.
- `when.profile`: profile name(s) to include.
- `decrypt`: valid only with `mode: copy`; source name must contain `.enc.`.
- `backup`: `true` by default.

## Template variables in `target`

Built-in:

- `home`
- `os`
- `arch`
- `profile`
- `hostname`

User-defined:

- any key declared under `vars`.

## Hook execution

Hooks run with `/bin/sh -c` in the repository directory.

Environment variables:

- `DOTCTL_HOOK_PHASE`
- `DOTCTL_HOOK_REPO`
