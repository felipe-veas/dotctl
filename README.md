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
- Built-in secrets management (`dotctl secrets` with age encryption)
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

## Quickstart

Recommended onboarding: initialize first, generate suggested manifest second, refine it, then sync.

### 1. Initialize dotctl on the machine

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

This step clones your dotfiles repository automatically. The repository can be empty for a first-time setup.

If the remote repository started empty, commit and push your initial content from the local clone after generating your manifest/files:

```bash
git -C ~/.config/dotctl/repo add .
git -C ~/.config/dotctl/repo commit -m "chore: bootstrap dotfiles manifest and config files"
git -C ~/.config/dotctl/repo push -u origin main
```

You can also use:

```bash
dotctl push -m "chore: bootstrap dotfiles manifest and config files"
```

to stage/commit/push from the active dotctl repository.

During init, dotctl also ensures recommended default ignore patterns in repo `.gitignore`:

```gitignore
.DS_Store
Thumbs.db
.env
.env.*
*.pem
*.key
*.p12
*.pfx
*.token
*credentials*
*secret*
!configs/secrets/
!configs/secrets/**
!configs/credentials/
!configs/credentials/**
configs/tmux/plugins/
```

You can refine this list in your repo if your workflow needs different rules.

### 2. Generate a suggested manifest

Scan common config paths (asks for confirmation first):

```bash
dotctl manifest suggest
```

### 3. Turn the suggested manifest into a production manifest

After running `dotctl manifest suggest`, use this workflow:

1. Review `manifest.suggested.yaml`.
1. Keep only files you want to manage across machines.
1. Add `when.profile` and `when.os` filters where behavior should differ by machine/OS.
1. Use `mode: copy` only when symlink is not appropriate.
1. Use `decrypt: true` for sensitive files and keep encrypted sources as `.enc.*`.
1. Confirm detected files exist in the repo under the suggested `source` paths.
1. If this is your first manifest and the suggested file looks good as-is, rename it:

   ```bash
   mv manifest.suggested.yaml manifest.yaml
   ```

1. If you already have a `manifest.yaml`, merge selected entries from `manifest.suggested.yaml` into the existing file.
1. Commit and push those changes.

Typical repository structure after this step:

```text
dotfiles/
  manifest.yaml
  configs/
    zsh/.zshrc
    git/.gitconfig
    nvim/
```

### 4. Manual manifest path (optional)

If you prefer full manual control, create `manifest.yaml` directly:

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

### 5. Validate and sync

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

## Use the same repo on another machine

If you already have a working dotfiles repo on machine A and want the same setup on machine B:

1. On machine A, ensure everything is pushed:

```bash
dotctl status
dotctl push -m "sync latest dotfiles before onboarding machine B"
```

1. On machine B, install `dotctl` and run init with the same repository URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git --profile laptop
```

1. On machine B, apply the repo state:

```bash
dotctl doctor
dotctl sync
```

Notes:

- You do not need to manually clone the repo first; `dotctl init` clones it automatically.
- If both machines should use identical rules, keep the same `--profile`.
- If a machine needs different rules, use another profile and `when.profile` entries in `manifest.yaml`.
- `dotctl manifest suggest` is mainly for bootstrapping a new manifest, not required when reusing an existing one.
- If you use `dotctl secrets`, copy `~/.config/dotctl/age-identity.txt` to machine B and run `dotctl secrets init --import <path>`.

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
| `dotctl secrets init` | Generate or import age encryption keys |
| `dotctl secrets encrypt <file>` | Encrypt a file for safe repo storage |
| `dotctl secrets decrypt <file>` | Decrypt a file (or `--stdout` to inspect) |
| `dotctl secrets status` | Show secrets protection status |
| `dotctl secrets rotate` | Rotate keys and re-encrypt all files |

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
- by default, it also copies detected local config files/directories into repo `source` paths
- on `dotctl sync`, if a `manifest.yaml` `source` is missing in the repo but its local `target` exists, dotctl backfills the repo source from that local target
- on later `dotctl sync`, sources previously managed by this flow are pruned from the repo if their `source` entries were removed from `manifest.yaml`
- use `--force` to skip confirmation (useful for automation)
- use `--dry-run` to preview without writing files
- use `--output <path>` to customize output file location
- use `--no-copy-sources` to only generate the suggestion without copying files

Current scan candidates include:

- Home files: `.zshrc`, `.zprofile`, `.bashrc`, `.bash_profile`, `.profile`, `.gitconfig`, `.gitignore`, `.tmux.conf`, `.vimrc`
- `~/.config` entries: `nvim`, `wezterm`, `kitty`, `alacritty`, `starship.toml`, `fish`, `gh`, `bat`, `tmux`, `helix`, `lazygit`, `ghostty`

Example flow:

```bash
dotctl manifest suggest
# review manifest.suggested.yaml
# merge selected entries into manifest.yaml
```

Other common usage:

```bash
# non-interactive
dotctl manifest suggest --force

# preview only
dotctl manifest suggest --dry-run --force

# generate suggestion only (no source copy)
dotctl manifest suggest --no-copy-sources --force

# custom output filename/path
dotctl manifest suggest --output manifest.suggested.work.yaml --force
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

## Secrets management (`dotctl secrets`)

`dotctl secrets` provides built-in key generation, encryption, and rotation using [age](https://github.com/FiloSottile/age) (X25519 + ChaCha20-Poly1305).

### Setup

```bash
# Generate an age key pair
dotctl secrets init

# Encrypt a sensitive file
dotctl secrets encrypt configs/env/.env

# Add to manifest with decrypt: true
```

### Multi-machine

Copy `~/.config/dotctl/age-identity.txt` to each machine, then import:

```bash
dotctl secrets init --import ~/path/to/age-identity.txt
dotctl sync
```

### Other operations

```bash
# Inspect encrypted file without writing to disk
dotctl secrets decrypt configs/env/.env.enc --stdout

# Check what is protected and what is not
dotctl secrets status

# Rotate keys and re-encrypt everything
dotctl secrets rotate
```

`dotctl push` will block if unencrypted sensitive files (`.env`, `*.key`, etc.) are tracked. Use `--force` to override, or encrypt first.

## Paths used by dotctl

Defaults (when XDG vars are not set):

- Config file: `~/.config/dotctl/config.yaml`
- Cloned default repo: `~/.config/dotctl/repo`
- Backups: `~/.config/dotctl/backups`
  - Snapshot layout: `~/.config/dotctl/backups/<timestamp>/targets/<target-path>`
- Age identity (secrets): `~/.config/dotctl/age-identity.txt`
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
