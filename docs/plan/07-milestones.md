# 7. Plan de Implementación por Hitos

## Timeline Overview

```
Semana 1                                Semana 2
├── M0 (2d) ──┤├── M1 (2d) ──┤├── M2 (2d) ──┤├── M3 (3d) ─────┤├── M4 (2d) ──┤
  Bootstrap     Sync+Symlinks   Git+Auth       Tray (mac+linux)   Hardening
```

---

## M0: Bootstrap — CLI scaffold + init/status (2 días)

### Entregables
- [x] `go mod init github.com/felipe-veas/dotctl`
- [x] Estructura de directorios (`cmd/`, `internal/`, `pkg/`, `testdata/`)
- [x] `cmd/dotctl/main.go` con cobra root command
- [x] `dotctl version` — imprime versión + OS/arch (build-time `-ldflags`)
- [x] `dotctl init --repo <url> --profile <name>` — crea `~/.config/dotctl/config.yaml`
- [x] `dotctl status` — lee config, muestra info básica (no symlinks aún)
- [x] `dotctl status --json` — output JSON con os/arch
- [x] `internal/config/` — read/write config YAML (respeta `$XDG_CONFIG_HOME`)
- [x] `internal/platform/` — `OpenURL()`, `ConfigDir()`, OS detection
- [x] `internal/output/` — printer (human + JSON)
- [x] `Makefile` con `build`, `build-linux-amd64`, `build-linux-arm64`, `test`, `lint`
- [x] `.golangci.yml`
- [x] Tests: config parsing, output formatting, platform detection

### Criterio de éxito
```
# En macOS
$ dotctl version
dotctl v0.1.0-dev (darwin/arm64)

# En Linux
$ dotctl version
dotctl v0.1.0-dev (linux/amd64)

$ dotctl init --repo github.com/felipe-veas/dotfiles --profile macstudio
✓ Config saved to ~/.config/dotctl/config.yaml

$ dotctl status --json | jq '{profile, os, arch}'
{ "profile": "macstudio", "os": "darwin", "arch": "arm64" }
```

---

## M1: Sync — Symlinks idempotentes + backups (2 días)

### Entregables
- [x] `internal/manifest/` — parser de `manifest.yaml` + resolver de condiciones
- [x] `internal/linker/` — crear symlinks, detectar drift, modo dry-run
- [x] `internal/backup/` — backup de archivos antes de reemplazar
- [x] `internal/profile/` — resolver OS + arch + hostname + perfil
- [x] `dotctl sync --dry-run` — muestra plan sin ejecutar
- [x] `dotctl sync` — aplica symlinks (sin git aún, asume repo ya clonado manualmente)
- [x] Template resolution en targets (`{{ .config_home }}`)
- [x] Filtros `when.os: darwin` y `when.os: linux` funcionando
- [x] Tests: manifest parsing, condition evaluation, symlink creation (en tmpdir)
- [x] Tests ejecutados en CI matrix (macOS + Linux)
- [x] `testdata/` con manifests de prueba incluyendo reglas por OS

### Criterio de éxito
```
# En macOS
$ dotctl sync --dry-run
Would create symlink: configs/zsh/.zshrc → ~/.zshrc
Would create symlink: configs/nvim → ~/.config/nvim
Skipped: configs/apt/packages.txt (os: linux, current: darwin)

# En Linux
$ dotctl sync --dry-run
Would create symlink: configs/zsh/.zshrc → ~/.zshrc
Would create symlink: configs/nvim → ~/.config/nvim
Skipped: configs/brew/Brewfile (os: darwin, current: linux)

$ dotctl sync && dotctl sync  # segunda vez idempotente
✓ All symlinks up to date
```

---

## M2: Git Integration + Auth (2 días)

### Entregables
- [x] `internal/auth/` — verificar `gh` CLI, auth status (mensajes de error por OS)
- [x] `internal/gitops/` — clone, pull, push, status, dirty check
- [x] `dotctl init` ahora clona el repo (no solo guarda config)
- [x] `dotctl pull` — `git pull --rebase`
- [x] `dotctl push` — `git add -A && git commit && git push`
- [x] `dotctl push --message "custom message"`
- [x] `dotctl sync` ahora hace pull → apply → push (flujo completo)
- [x] `dotctl open` — `platform.OpenURL(url)` (open/xdg-open)
- [x] `dotctl doctor` — checks de auth + git + symlinks + manifest + OS info
- [x] Soporte SSH URL (detección automática, no requiere `gh` si es SSH)
- [x] Tests: git operations en CI matrix (macOS + Linux)

### Criterio de éxito
```
$ dotctl init --repo github.com/felipe-veas/dotfiles --profile devserver
✓ gh authenticated as felipe-veas
✓ Cloned to ~/.config/dotctl/repo
✓ Profile: devserver

$ dotctl doctor
✓ os: linux/amd64
✓ git 2.43.0
✓ gh authenticated
✓ repo clean
✓ manifest valid (14 entries, 8 for this profile)
✓ 8/8 symlinks ok
Overall: HEALTHY

$ dotctl open  # abre browser con xdg-open
```

---

## M3: Tray Apps — macOS menubar + Linux system tray (3 días)

### Entregables macOS (1.5 días)
- [x] `mac/StatusApp/` — scaffold Swift + estructura `StatusApp.xcodeproj/` + README
- [x] `AppDelegate.swift` — launch as menubar-only app (no dock icon)
- [x] `StatusBarController.swift` — menú con status, acciones, open repo
- [x] `DotctlBridge.swift` — ejecuta binario Go, parsea JSON
- [x] Estados de ícono en menubar: synced, drift, error, syncing (SF Symbols)
- [x] `scripts/build-app-macos.sh` — build universal binary (arm64+amd64) + embed en app
- [x] LaunchAgent plist (opt-in)

### Entregables Linux (1.5 días)
- [x] `linux/tray/` — app Go con getlantern/systray
- [x] `main.go` — setup de menú (misma estructura que macOS)
- [x] `bridge.go` — ejecuta dotctl binary, parsea JSON
- [x] PNG icons para tray (22x22: ok, warn, error, sync)
- [x] `.desktop` file para autostart
- [x] `scripts/build-tray-linux.sh` — build con CGO
- [x] systemd user service (alternativa a .desktop)

### Entregables compartidos
- [x] `dotctl bootstrap` — ejecuta hooks `bootstrap` del manifest (autostart vía hooks/scripts)
- [x] Polling cada 60s en ambas plataformas

### Estado actual
- Implementación de M3 completada en código y scripts.
- Validación automática: tests Go en verde (`go test ./...`).
- Validación manual pendiente por entorno: arranque visual de tray en GNOME/KDE y menubar/LaunchAgent en macOS.

### Criterio de éxito
- **macOS**: app aparece en menubar, muestra estado, ejecuta acciones, abre repo
- **Linux**: app aparece en system tray (GNOME/KDE), misma funcionalidad
- Ambas: "Sync Now" ejecuta sync y actualiza ícono
- Ambas: "Open Repo" abre browser

---

## M4: Hardening (2 días)

### Entregables
- [ ] Logging estructurado (`slog` stdlib): file log location por OS:
  - macOS: `~/.config/dotctl/dotctl.log`
  - Linux: `$XDG_STATE_HOME/dotctl/dotctl.log`
- [ ] File locking: `flock` durante sync (funciona en macOS y Linux)
- [ ] Rollback: si sync falla a mitad, revertir symlinks creados en esta ejecución
- [ ] Hooks pre/post sync ejecutándose correctamente
- [ ] Manejo de secretos: `.gitignore` patterns + validación en doctor
- [ ] `--verbose` flag funcional (muestra comandos git, paths completos, OS info)
- [ ] Error messages claros para todos los failure modes en ambos OS
- [ ] CI: GitHub Actions matrix (macOS-latest + ubuntu-latest)
- [ ] Manifest `ignore:` patterns aplicados
- [ ] Edge cases: symlinks circulares, permisos incorrectos, disco lleno
- [ ] `.goreleaser.yaml` con targets: darwin/{arm64,amd64}, linux/{amd64,arm64}

### Criterio de éxito
- Tests verdes en CI matrix (macOS + Linux)
- File lock funciona en ambos OS
- `goreleaser --snapshot` genera binarios para las 4 plataformas
- `dotctl doctor` reporta OS/arch correctamente en cada plataforma

---

## Post-MVP (backlog priorizado)

| Prioridad | Feature | Esfuerzo |
|---|---|---|
| P1 | Cifrado con `age`/`sops` (decrypt: true en manifest) | 2d |
| P1 | `goreleaser` release pipeline + Homebrew tap + .deb/.rpm | 1d |
| P1 | Notificaciones nativas (macOS: UserNotifications, Linux: libnotify) | 1d |
| P2 | File watcher (fsnotify) para sync automático | 2d |
| P2 | Backup rotation (mantener últimos N backups) | 0.5d |
| P2 | `dotctl diff` — muestra diff entre repo y estado actual | 1d |
| P3 | Snap package para Linux | 1d |
| P3 | AUR package para Arch Linux | 0.5d |
| P3 | Multi-repo support | 3d |
