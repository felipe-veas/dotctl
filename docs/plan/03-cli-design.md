# 3. Diseño de Comandos CLI

## Plataformas

La CLI funciona idéntico en macOS (arm64+amd64) y Linux (amd64+arm64). Las diferencias de plataforma se abstraen en `internal/platform/`.

## Estructura de Comandos

```
dotctl
├── init          Inicializa dotctl: clona repo, configura perfil
├── status        Muestra estado actual (symlinks, git, auth)
├── sync          Sincroniza: pull + aplica manifest + push cambios locales
├── pull          Solo git pull (no aplica symlinks)
├── push          Solo git add + commit + push
├── doctor        Diagnóstico completo del sistema
├── open          Abre el repo privado en el navegador
├── bootstrap     Ejecuta hooks de bootstrap del manifest
└── version       Versión del binario (incluye OS/arch)
```

## Flags Globales

```
--profile <name>   Perfil activo (override de config). Ej: macstudio, laptop, devserver
--json             Output en JSON (para tray app y scripting)
--dry-run          Muestra qué haría sin ejecutar
--verbose          Output detallado (debug)
--force            Omitir confirmaciones y sobreescribir sin backup
--config <path>    Path a config alternativa (default: ~/.config/dotctl/config.yaml)
```

## Detalle por Comando

### `dotctl init`

```
dotctl init --repo <github-url> --profile <name>

Flags:
  --repo      URL del repo (HTTPS o SSH). Requerido en primera ejecución
  --profile   Nombre del perfil para esta máquina. Requerido
  --path      Path donde clonar (default: ~/.config/dotctl/repo)

Ejemplo:
  $ dotctl init --repo github.com/felipe-veas/dotfiles --profile macstudio

Output:
  ✓ gh authenticated as felipe-veas
  ✓ Cloned to ~/.config/dotctl/repo
  ✓ Profile: macstudio
  ✓ Config saved to ~/.config/dotctl/config.yaml
```

### `dotctl status`

```
$ dotctl status

Profile: macstudio
OS:      darwin/arm64
Repo:    github.com/felipe-veas/dotfiles (clean)
Last sync: 2025-01-15 10:30:00

Symlinks (12 total):
  ✓ 10 ok
  ✗ 1 broken: ~/.config/nvim → repo/configs/nvim (target missing)
  ~ 1 drift:  ~/.zshrc exists but is not a symlink

Auth: ✓ gh authenticated
```

```
$ dotctl status --json
```
```json
{
  "profile": "macstudio",
  "os": "darwin",
  "arch": "arm64",
  "repo": {
    "url": "github.com/felipe-veas/dotfiles",
    "status": "clean",
    "branch": "main",
    "last_commit": "abc1234",
    "last_sync": "2025-01-15T10:30:00Z"
  },
  "symlinks": {
    "total": 12,
    "ok": 10,
    "broken": 1,
    "drift": 1,
    "details": [
      {
        "source": "configs/nvim",
        "target": "~/.config/nvim",
        "status": "broken",
        "error": "target missing in repo"
      },
      {
        "source": "configs/zsh/.zshrc",
        "target": "~/.zshrc",
        "status": "drift",
        "error": "regular file, not symlink"
      }
    ]
  },
  "auth": {
    "method": "gh",
    "user": "felipe-veas",
    "ok": true
  },
  "errors": []
}
```

### `dotctl sync`

```
$ dotctl sync

Pulling latest...
  ✓ Already up to date

Applying manifest (profile: macstudio, os: darwin)...
  ✓ configs/zsh/.zshrc → ~/.zshrc (symlink created)
  ✓ configs/git/.gitconfig → ~/.gitconfig (already linked)
  ✗ configs/nvim → ~/.config/nvim (backed up existing → ~/.config/dotctl/backups/20250115-103000/nvim)
  ✓ configs/nvim → ~/.config/nvim (symlink created)

  Ran hook: post_sync → brew bundle --file=configs/brew/Brewfile (exit 0)

Pushing changes...
  ✓ Nothing to push

Summary: 3 ok, 1 backup created, 0 errors
```

```
$ dotctl sync --dry-run
Would create symlink: configs/zsh/.zshrc → ~/.zshrc
Would backup: ~/.config/nvim (regular dir exists)
Would create symlink: configs/nvim → ~/.config/nvim
Would run hook: post_sync → brew bundle --file=configs/brew/Brewfile
Skipped: configs/apt/packages.txt (os: linux, current: darwin)
```

### `dotctl pull`

```
$ dotctl pull
Pulling from origin/main...
✓ Updated: abc1234 → def5678 (3 files changed)
```

### `dotctl push`

```
$ dotctl push
Staging changes...
  + configs/zsh/.zshrc (modified)
  + configs/starship.toml (new file)

Commit message [auto]: dotctl push from macstudio @ 2025-01-15
✓ Pushed to origin/main
```

```
$ dotctl push --message "update zsh config"
```

### `dotctl doctor`

```
$ dotctl doctor

System:
  ✓ os: darwin/arm64
  ✓ git installed (2.43.0)
  ✓ gh installed (2.42.0)
  ✓ gh authenticated as felipe-veas

Repo:
  ✓ repo cloned at ~/.config/dotctl/repo
  ✓ repo clean (no uncommitted changes)
  ✓ manifest.yaml valid (14 entries, 12 for this profile)

Symlinks:
  ✓ all symlinks intact (12/12)

Warnings:
  ! backup dir has 2.1GB — consider cleanup

Overall: HEALTHY (1 warning)
```

### `dotctl open`

```
$ dotctl open
Opening https://github.com/felipe-veas/dotfiles in browser...
```

Implementación: lee `repo.url` de config → construye URL GitHub → `platform.OpenURL(url)`:
- macOS: `exec open <url>`
- Linux: `exec xdg-open <url>`

### `dotctl bootstrap`

```
$ dotctl bootstrap

Running bootstrap hooks for profile: macstudio (os: darwin)
  → brew bundle --file=configs/brew/Brewfile ... ✓ (42s)
  → configs/scripts/setup-defaults.sh ... ✓ (3s)
  Skipped: apt install (os: linux)

Bootstrap complete.
```

Ejecuta los hooks marcados como `bootstrap` en el manifest, filtrados por OS y perfil.

### `dotctl version`

```
$ dotctl version
dotctl v0.1.0 (darwin/arm64) built 2025-01-15
```

## Interface Go (Cobra)

```go
// cmd/root.go
package cmd

import "github.com/spf13/cobra"

var (
    profile string
    jsonOut bool
    dryRun  bool
    verbose bool
    force   bool
)

func NewRootCmd() *cobra.Command {
    root := &cobra.Command{
        Use:   "dotctl",
        Short: "Sync dotfiles across machines (macOS + Linux)",
    }

    root.PersistentFlags().StringVar(&profile, "profile", "", "active profile name")
    root.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output")
    root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show plan without executing")
    root.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
    root.PersistentFlags().BoolVar(&force, "force", false, "skip confirmations")

    root.AddCommand(
        newInitCmd(),
        newStatusCmd(),
        newSyncCmd(),
        newPullCmd(),
        newPushCmd(),
        newDoctorCmd(),
        newOpenCmd(),
        newBootstrapCmd(),
    )

    return root
}
```

```go
// internal/platform/open.go
package platform

import (
    "fmt"
    "os/exec"
    "runtime"
)

func OpenURL(url string) error {
    switch runtime.GOOS {
    case "darwin":
        return exec.Command("open", url).Run()
    case "linux":
        return exec.Command("xdg-open", url).Run()
    default:
        return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }
}

func OpenFileManager(path string) error {
    return OpenURL(path) // open/xdg-open work with file paths too
}
```

## Exit Codes

| Code | Significado |
|---|---|
| 0 | OK |
| 1 | Error general |
| 2 | Auth error (gh not logged in) |
| 3 | Manifest error (invalid YAML, missing fields) |
| 4 | Git error (conflict, network) |
| 5 | Drift detected (con `--dry-run`, indica que hay cambios pendientes) |
