# dotctl

`dotctl` is a CLI for managing dotfiles and other configuration files across machines.

It keeps one personal config set under Git, applies an explicit `manifest.yaml`, and handles sync with backups and rollback.

## Why dotctl?

I wanted a simple way to keep personal dotfiles under control without bringing in a full environment manager.
Tools like Nix are legitimate and powerful, but they can be more than I needed for this use case.
`dotctl` is intentionally narrower: predictable dotfile management with Git and a manifest, nothing more.

It is designed for:

- one personal dotfile configuration
- safe sync with backups and rollback
- reproducible setup via `manifest.yaml`

## Features

- Declarative sync from `manifest.yaml`
- `symlink` and `copy` file modes
- Suggested manifest generation from common local config paths (`dotctl manifest suggest`)
- Health checks (`dotctl doctor`)
- JSON output mode for scripting (`--json`)
- Commit/push using your Git identity and signing settings

## Supported platforms

| OS | Architectures | CLI |
|---|---|---|
| macOS | arm64, amd64 | Yes |
| Linux | arm64, amd64 | Yes |

## Requirements

- `git` (required)
- `gh` CLI only if you use HTTPS GitHub repo URLs (not needed for SSH URLs)

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

Recommended onboarding: initialize first, add the dotfiles you want managed, then sync.

### 1. Initialize dotctl on the machine

Using SSH URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git
```

Using HTTPS URL (requires `gh auth login`):

```bash
dotctl init --repo https://github.com/<you>/dotfiles.git
```

You can also set a custom clone location:

```bash
dotctl init --repo <repo-url> --path /custom/path
```

This step clones your dotfiles repository automatically. The repository can be empty for a first-time setup.

If the remote repository started empty, commit and push your initial content from the local clone after generating your manifest/files:

```bash
git -C ~/.config/dotctl/repo add .
git -C ~/.config/dotctl/repo commit -m "chore: initialize dotfiles manifest and config files"
git -C ~/.config/dotctl/repo push -u origin main
```

You can also use:

```bash
dotctl push -m "chore: initialize dotfiles manifest and config files"
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
configs/tmux/plugins/
```

You can refine this list in your repo if your workflow needs different rules.

### 2. Add dotfiles

Add individual paths from your home directory:

```bash
dotctl add ~/.zshrc
dotctl add ~/.config/nvim
```

`dotctl add` copies the path into your repo, updates `manifest.yaml`, backs up the original target, and replaces it with a symlink. Use `--dry-run` to preview the plan first.

To stop managing a path without deleting local files or repo sources:

```bash
dotctl remove ~/.zshrc
```

### 3. Generate a suggested manifest instead

Scan common config paths (asks for confirmation first):

```bash
dotctl manifest suggest
```

### 4. Turn the suggested manifest into a production manifest

After running `dotctl manifest suggest`, use this workflow:

1. Review `manifest.suggested.yaml`.
1. Keep only files you want to manage across machines.
1. Add `when.os` filters where behavior should differ by OS.
1. Use `mode: copy` only when symlink is not appropriate.
1. Keep secrets out of the repository; dotctl is intended for non-sensitive dotfiles.
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

### 5. Manual manifest path (optional)

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
3. commit and push (if there are changes)

## Use the same repo on another machine

If you already have a working dotfiles repo on machine A and want the same setup on machine B:

1. On machine A, ensure everything is pushed:

```bash
dotctl status
dotctl push -m "sync latest dotfiles before onboarding machine B"
```

1. On machine B, install `dotctl` and run init with the same repository URL:

```bash
dotctl init --repo git@github.com:<you>/dotfiles.git
```

1. On machine B, apply the repo state:

```bash
dotctl doctor
dotctl sync
```

Notes:

- You do not need to manually clone the repo first; `dotctl init` clones it automatically.
- `dotctl manifest suggest` is mainly for creating a new manifest, not required when reusing an existing one.

## Daily commands

| Command | Purpose |
|---|---|
| `dotctl sync` | Pull, apply manifest, push |
| `dotctl add <path>` | Copy one local dotfile into the repo and symlink it |
| `dotctl remove <path>` | Untrack one dotfile without deleting local files or repo sources |
| `dotctl status` | Current state (repo/auth/symlinks) |
| `dotctl doctor` | Health checks (git/auth/manifest/symlinks/security) |
| `dotctl diff` | Show current drift/changes |
| `dotctl diff --details` | Include unified diff for changed files |
| `dotctl pull` | Pull latest changes only |
| `dotctl push` | Commit and push local repo changes |
| `dotctl push -m "msg"` | Push with custom commit message |
| `dotctl edit` | Open the repo with `$VISUAL` or `$EDITOR` |
| `dotctl edit manifest` | Open `manifest.yaml` with `$VISUAL` or `$EDITOR` |
| `dotctl backups list` | List local backup snapshots |
| `dotctl backups restore <snapshot>` | Restore a backup snapshot; use repeatable `--target` to restore exact entries, or omit it for a full restore. Requires `--force` unless `--dry-run` |
| `dotctl open` | Open repo in browser |
| `dotctl manifest suggest` | Scan common paths and write `manifest.suggested.yaml` |

Useful global flags:

- `--dry-run`: show planned actions only
- `--json`: machine-readable output
- `--verbose`: enable detailed logs + git tracing
- `--config <path>`: use a specific config file

`dotctl add` rejects sensitive-looking paths by default. Use `--force` only when you intentionally want to manage a path such as `.ssh`, `.gnupg`, `.kube`, `.aws`, `.config/gh`, `.config/gcloud`, or an `.env` file.

Always preview restores first:

```bash
dotctl backups list
dotctl backups restore <snapshot> --dry-run
dotctl backups restore <snapshot> --target ~/.zshrc --dry-run
dotctl backups restore <snapshot> --target ~/.zshrc --force
dotctl backups restore <snapshot> --force
```

Restore targets must resolve strictly under your home directory. `--target` matches the backed-up target path exactly, and you can repeat it to restore multiple entries. Directory targets restore the directory entry when the snapshot has one.

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

## Manifest reference

Top-level keys:

- `version`: currently `1`
- `vars`: custom variables used in templated targets
- `files`: list of managed entries
- `ignore`: source patterns to skip

Per-file fields:

- `source` (required): path relative to repo root
- `target` (required): `~` path, absolute path, or template that resolves under your home directory
- `mode`: `symlink` (default) or `copy`
- `when.os`: `darwin`, `linux`, or list
- `backup`: `true` (default) or `false`

Available template vars in `target`:

- `home`
- `os`
- `arch`
- `hostname`
- plus your custom `vars`

Targets must resolve strictly under your home directory. `dotctl` rejects targets that are relative, outside `$HOME`, equal to `$HOME`, or escape through parent traversal.
Sensitive-looking targets such as `.ssh`, `.gnupg`, `.kube`, `.aws`, `.config/gh`, `.config/gcloud`, and `.env` files produce warnings.

## Secret hygiene

`dotctl` is intended for non-sensitive dotfiles. It does not manage encryption or decryption.

- Keep secrets such as `.env`, private keys, tokens, and credentials out of the repo.
- Use `.gitignore` and `manifest.ignore` to exclude sensitive material.
- `dotctl push` warns when tracked files look sensitive. Use `--force` only if you have reviewed the files intentionally.
- If you need encrypted secrets, manage them with a dedicated external tool outside dotctl.

## Paths used by dotctl

Defaults (when XDG vars are not set):

- Config file: `~/.config/dotctl/config.yaml`
- Cloned default repo: `~/.config/dotctl/repo`
- Backups: `~/.config/dotctl/backups`
  - Snapshot layout: `~/.config/dotctl/backups/<timestamp>/targets/<target-path>`
- Logs:
  - Linux: `~/.local/state/dotctl/dotctl.log`
  - macOS: `~/.config/dotctl/dotctl.log`
- Sync lock: same state dir as log (`sync.lock`)

## Troubleshooting

- `dotctl not initialized`: run `dotctl init --repo <url>`
- `gh not authenticated`: run `gh auth login --web`
- `repository has uncommitted changes`: commit/stash inside dotctl repo, then run `dotctl sync` again
- `configure git identity (user.name and user.email)`: set Git identity in repo or globally, then retry `dotctl push`
- inspect detailed logs with `--verbose` and the log file path above

## License

[MIT](./LICENSE)
