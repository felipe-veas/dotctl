# dotctl

`dotctl` is a CLI (plus optional tray apps) to sync dotfiles across machines using a private GitHub repository as the source of truth.

It is designed for:

- profile-aware dotfiles (`laptop`, `workstation`, `server`, etc.)
- safe sync with backups and rollback
- reproducible setup via `manifest.yaml`

## Features

- Declarative sync from `manifest.yaml`
- `symlink` and `copy` file modes
- Optional encrypted file deployment (`decrypt: true` with `sops` or `age`)
- Suggested manifest generation from common local config paths (`dotctl manifest suggest`)
- Pre/post sync hooks plus bootstrap hooks
- Multi-repo support (`dotctl repos ...`)
- Health checks (`dotctl doctor`)
- JSON output mode for scripting (`--json`)
- Commit/push using your Git identity and signing settings
- Optional tray apps:
  - macOS status bar app
  - Linux system tray app

## Supported platforms

| OS | Architectures | CLI | Tray |
|---|---|---|---|
| macOS | arm64, amd64 | Yes | Yes (Swift status bar app) |
| Linux | arm64, amd64 | Yes | Yes (AppIndicator tray app) |

## Requirements

- `git` (required)
- `gh` CLI only if you use HTTPS GitHub repo URLs (not needed for SSH URLs)
- `sops` or `age` only if your manifest uses `decrypt: true`

## Installation

### Option 1: Homebrew (macOS / Linux)

```bash
brew tap felipe-veas/homebrew-tap
brew install dotctl
```

### Option 2: Download release binary

Download the archive for your OS/arch from [GitHub Releases](https://github.com/felipe-veas/dotctl/releases), then extract and move `dotctl` into your PATH.

Example (macOS arm64):

```bash
curl -L -o dotctl.tar.gz https://github.com/felipe-veas/dotctl/releases/latest/download/dotctl_Darwin_arm64.tar.gz
tar -xzf dotctl.tar.gz
chmod +x dotctl
sudo mv dotctl /usr/local/bin/dotctl
```

### Option 3: Linux packages (`.deb` / `.rpm`)

Releases also include `.deb` and `.rpm` packages.

Debian/Ubuntu:

```bash
sudo dpkg -i dotctl_<version>_linux_amd64.deb
```

Fedora/RHEL:

```bash
sudo rpm -i dotctl_<version>_linux_amd64.rpm
```

### Option 4: Build from source

```bash
git clone https://github.com/felipe-veas/dotctl.git
cd dotctl
make build
./bin/dotctl version
```

## Documentation

Current documentation:

- [Getting Started](./docs/getting-started.md)
- [Installation](./docs/installation.md)
- [Manifest Specification](./docs/manifest-spec.md)
- [Command Reference](./docs/command-reference.md)
- [Sync Lifecycle](./docs/sync-lifecycle.md)
- [Security Model](./docs/security-model.md)
- [Troubleshooting](./docs/troubleshooting.md)
- [Roadmap](./docs/roadmap.md)

Historical planning archive:

- [Archive index](./docs/archive/README.md)

## Quickstart

### 1. Prepare your private dotfiles repo

Your repo should contain a `manifest.yaml` at the root and the files you want to manage.

Example:

```text
dotfiles/
  manifest.yaml
  configs/
    zsh/.zshrc
    git/.gitconfig
    nvim/
```

### 2. Create a `manifest.yaml`

```yaml
version: 1

vars:
  config_home: "~/.config"

files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc

  - source: configs/git/.gitconfig
    target: ~/.gitconfig

  - source: configs/nvim
    target: "{{ .config_home }}/nvim"
    mode: copy
```

### 3. Initialize dotctl on a machine

Using SSH URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git --profile laptop
```

Using HTTPS URL (requires `gh auth login`):

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git --profile laptop
```

You can also set a custom clone location:

```bash
dotctl init --repo <repo-url> --profile laptop --path /custom/path
```

Generate a suggested manifest by scanning common config paths (asks for confirmation first):

```bash
dotctl manifest suggest
```

### 4. Validate and sync

```bash
dotctl doctor
dotctl sync
dotctl status
```

`dotctl sync` flow is:

1. `git pull --rebase`
2. apply manifest actions
3. run hooks
4. commit and push (if there are changes)

## Daily commands

| Command | Purpose |
|---|---|
| `dotctl sync` | Pull, apply manifest, push |
| `dotctl status` | Current state (repo/auth/symlinks) |
| `dotctl doctor` | Health checks (git/auth/manifest/symlinks/security) |
| `dotctl diff` | Show current drift/changes |
| `dotctl diff --details` | Include unified diff for changed files |
| `dotctl pull` | Pull latest changes only |
| `dotctl push` | Commit and push local repo changes |
| `dotctl push -m "msg"` | Push with custom commit message |
| `dotctl watch` | Auto-sync on repo file changes |
| `dotctl bootstrap` | Run `bootstrap` hooks |
| `dotctl open` | Open repo in browser |
| `dotctl repos list` | List configured repos |
| `dotctl repos add --name work --url ...` | Add another repo |
| `dotctl repos use work` | Switch active repo |
| `dotctl manifest suggest` | Scan common paths and write `manifest.suggested.yaml` |

Useful global flags:

- `--dry-run`: show planned actions only
- `--json`: machine-readable output
- `--verbose`: enable detailed logs + git tracing
- `--config <path>`: use a specific config file
- `--profile <name>`: override active profile for this run
- `--repo-name <name>`: pick active repo for this run

## Suggested manifest scan (`dotctl manifest suggest`)

`dotctl manifest suggest` scans common configuration paths from your machine and writes a reviewable draft file:

- default output: `<active-repo>/manifest.suggested.yaml`
- before scanning, dotctl asks for explicit confirmation (`[y/N]`)
- use `--force` to skip confirmation (useful for automation)
- use `--dry-run` to preview without writing files
- use `--output <path>` to customize output file location

Example flow:

```bash
dotctl manifest suggest
# review manifest.suggested.yaml
# merge selected entries into manifest.yaml
```

JSON mode note:

- `dotctl manifest suggest --json` requires `--force` because confirmation is interactive

Security note:

- the scan skips sensitive candidates such as `.env`, SSH key paths, and key/cert suffix patterns

## Commit identity for `dotctl push`

`dotctl push` uses your Git configuration for author/signing instead of forcing a `dotctl` author.

Recommended setup:

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

Or per-repository:

```bash
git -C /path/to/repo config user.name "Your Name"
git -C /path/to/repo config user.email "you@example.com"
```

## Multi-repo workflow

Initialize default repo first:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git --profile laptop
```

Add a second repo:

```bash
dotctl repos add --name work --url git@github.com:<you>/work-dotfiles.git --activate
```

Switch when needed:

```bash
dotctl repos use work
dotctl sync
```

## Manifest reference

Top-level keys:

- `version`: currently `1`
- `vars`: custom variables used in templated targets
- `files`: list of managed entries
- `ignore`: source patterns to skip
- `hooks`: `pre_sync`, `post_sync`, `bootstrap`

Per-file fields:

- `source` (required): path relative to repo root
- `target` (required): absolute path or template
- `mode`: `symlink` (default) or `copy`
- `when.os`: `darwin`, `linux`, or list
- `when.profile`: profile name(s)
- `decrypt`: only valid with `mode: copy`, source filename must contain `.enc.`
- `backup`: `true` (default) or `false`

Available template vars in `target`:

- `home`
- `os`
- `arch`
- `profile`
- `hostname`
- plus your custom `vars`

## Hooks

Hook commands run with `/bin/sh -c` from the repo directory.

Environment variables exposed to hooks:

- `DOTCTL_HOOK_PHASE`
- `DOTCTL_HOOK_REPO`

Example:

```yaml
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

## Encrypted files (`decrypt: true`)

For encrypted sources in the repo:

- use `mode: copy`
- set `decrypt: true`
- ensure source filename includes `.enc.` (for validation)
- install `sops` or `age` in PATH

Example:

```yaml
files:
  - source: configs/secrets/api.enc.yaml
    target: ~/.config/secrets/api.yaml
    mode: copy
    decrypt: true
```

## Paths used by dotctl

Defaults (when XDG vars are not set):

- Config file: `~/.config/dotctl/config.yaml`
- Cloned default repo: `~/.config/dotctl/repo`
- Backups: `~/.config/dotctl/backups`
- Logs:
  - Linux: `~/.local/state/dotctl/dotctl.log`
  - macOS: `~/.config/dotctl/dotctl.log`
- Sync lock: same state dir as log (`sync.lock`)

## Optional tray apps

- macOS status app instructions: [mac/StatusApp/README.md](./mac/StatusApp/README.md)
- Linux tray instructions: [linux/tray/README.md](./linux/tray/README.md)

## Troubleshooting

- `dotctl not initialized`: run `dotctl init --repo <url> --profile <name>`
- `gh not authenticated`: run `gh auth login --web`
- `repository has uncommitted changes`: commit/stash inside dotctl repo, then run `dotctl sync` again
- `configure git identity (user.name and user.email)`: set Git identity in repo or globally, then retry `dotctl push`
- decrypt tool errors: install `sops` or `age` and confirm it is in PATH
- inspect detailed logs with `--verbose` and the log file path above

## License

[MIT](./LICENSE)
