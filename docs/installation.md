# Installation

## Requirements

- `git`
- Optional: `gh` (for HTTPS GitHub URLs)
- Optional: `sops` or `age` (for `decrypt: true` entries)

## Option 1: Homebrew (macOS / Linux)

```bash
brew tap felipe-veas/homebrew-tap
brew install dotctl
```

## Option 2: Release binaries

Download from [GitHub Releases](https://github.com/felipe-veas/dotctl/releases).

Example (macOS arm64):

```bash
curl -L -o dotctl.tar.gz https://github.com/felipe-veas/dotctl/releases/latest/download/dotctl_Darwin_arm64.tar.gz
tar -xzf dotctl.tar.gz
chmod +x dotctl
sudo mv dotctl /usr/local/bin/dotctl
```

## Option 3: Linux packages

Debian/Ubuntu:

```bash
sudo dpkg -i dotctl_<version>_linux_amd64.deb
```

Fedora/RHEL:

```bash
sudo rpm -i dotctl_<version>_linux_amd64.rpm
```

## Option 4: Build from source

```bash
git clone https://github.com/felipe-veas/dotctl.git
cd dotctl
make build
./bin/dotctl version
```

## Verify installation

```bash
dotctl version
```
