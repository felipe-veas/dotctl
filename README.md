# dotctl

`dotctl` is a CLI for managing one personal dotfile set in a private Git repository.
It uses an explicit `manifest.yaml`, keeps sync behavior predictable, and makes
backups and rollback part of the normal flow.

## Why dotctl?

I wanted something narrower than a full environment manager. Tools like Nix are
valid, but they solve a broader problem than I needed here. `dotctl` keeps the
scope tight: one repo, one manifest, explicit commands.

## What it does

- add, remove, and edit tracked dotfiles
- apply `manifest.yaml` entries as symlinks or copies
- suggest a starting manifest from common local config paths
- run status, diff, doctor, pull, push, and sync
- keep local backups and restore them when needed
- print JSON output for scripting

## Supported platforms

| OS | Architectures | CLI |
|---|---|---|
| macOS | arm64, amd64 | Yes |
| Linux | arm64, amd64 | Yes |

## Requirements

- `git`
- `gh` only if you use HTTPS GitHub repository URLs

## Install

See [Installation](./docs/installation.md) for the full set of options.
Short version:

- Homebrew: `brew tap felipe-veas/homebrew-tap && brew install dotctl`
- Release binaries or Linux packages: download from GitHub Releases
- Source build: clone the repo and run `make build`

## Quickstart

1. Initialize the repo:

   ```bash
   dotctl init --repo git@github.com:<you>/dotfiles.git
   ```

2. Add a dotfile:

   ```bash
   dotctl add ~/.zshrc
   ```

3. Optionally generate a starter manifest:

   ```bash
   dotctl manifest suggest
   ```

4. Check the state and apply it:

   ```bash
   dotctl doctor
   dotctl sync --dry-run
   dotctl sync
   dotctl status
   ```

If you want the detailed onboarding flow, use [Getting Started](./docs/getting-started.md).

## Documentation

- [Documentation index](./docs/README.md)
- [Getting Started](./docs/getting-started.md)
- [Installation](./docs/installation.md)
- [Command Reference](./docs/command-reference.md)
- [Manifest Specification](./docs/manifest-spec.md)
- [Sync Workflow](./docs/sync-workflow.md)
- [Security Model](./docs/security-model.md)
- [Troubleshooting](./docs/troubleshooting.md)
- [Roadmap](./docs/roadmap.md)

## Security note

`dotctl` is intended for non-sensitive dotfiles. It does not manage encryption
or decryption. Keep secrets, tokens, credentials, and private keys out of the
repository.

## License

[MIT](./LICENSE)
