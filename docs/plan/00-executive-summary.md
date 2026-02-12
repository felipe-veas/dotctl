# 0. Resumen Ejecutivo — dotctl MVP

## Qué es dotctl

CLI en Go + tray app (macOS menubar / Linux system tray) que sincroniza dotfiles/configs entre dispositivos usando un repo privado de GitHub como source of truth. Declarativo, idempotente y seguro.

## Plataformas Soportadas

| Plataforma | Arquitecturas | CLI | Tray App |
|---|---|---|---|
| **macOS** | arm64 (Apple Silicon) + amd64 (Intel) | Full | Swift NSStatusBar |
| **Linux** | amd64 + arm64 | Full | GTK AppIndicator (libayatana) |

El binario Go es 100% cross-platform. La tray app es nativa por OS.

## Qué HACE el MVP

| Capacidad | Detalle |
|---|---|
| **Init** | Clona el repo privado, genera config local (`~/.config/dotctl/config.yaml`) con perfil de máquina |
| **Sync bidireccional** | Crea symlinks desde el repo hacia `$HOME` (y otros targets). Idempotente: detecta drift, hace backup antes de sobreescribir |
| **Pull / Push** | `git pull --rebase` / `git add + commit + push` contra el repo privado |
| **Manifest declarativo** | `manifest.yaml` define mapeos source→target, filtros por OS/perfil, hooks pre/post |
| **Múltiples perfiles** | Un mismo repo sirve a N máquinas (ej: `macstudio`, `laptop`, `devserver`) con reglas condicionales |
| **Auth segura** | Reutiliza credenciales de `gh` CLI (OAuth device flow). Tokens nunca en disco plano — macOS Keychain / Linux libsecret vía `gh` |
| **Doctor** | Valida estado: repo limpio, symlinks correctos, auth OK, manifest válido |
| **Status** | Output humano + `--json` para consumo programático (tray app) |
| **Tray App** | Ícono en menubar/system tray: estado rápido, acciones (sync/pull/push/doctor), abrir repo en browser |
| **Seguridad de secretos** | `.gitignore` para archivos sensibles + integración opcional con `age`/`sops` para cifrado at-rest |

## Qué NO hace el MVP

- **No es un package manager** — no instala brew, apt, etc. (el `bootstrap` solo ejecuta hooks del manifest).
- **No sincroniza en tiempo real** — es pull-based, no hay daemon/watcher (se puede agregar post-MVP).
- **No hace merge de contenido** — si hay conflicto Git, lo reporta y el usuario resuelve manualmente.
- **No gestiona secretos end-to-end** — ofrece hooks y gitignore, no un vault completo.
- **No multi-repo** — un solo repo privado como source of truth.
- **No auto-update del binario** — el usuario actualiza manualmente o vía package manager.
- **No soporta Windows** — solo macOS y Linux.

## Stack

| Componente | Tecnología |
|---|---|
| CLI | Go 1.22+, `cobra` (commands), `viper` (config) |
| Git operations | shell-out a `git` (no libgit2 — simplicidad) |
| Auth | `gh auth token` — reutiliza OAuth flow de GitHub CLI |
| Tray macOS | Swift (NSStatusBar) invocando el binario Go |
| Tray Linux | Go + `getlantern/systray` (libayatana-appindicator) |
| Manifest | YAML (`gopkg.in/yaml.v3`) |
| Empaquetado CLI | `goreleaser` (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64) |
| Empaquetado macOS | Xcode project → `.app` bundle con binario Go universal embebido |
| Empaquetado Linux | `.deb`/`.rpm` vía goreleaser + `.desktop` entry para tray |

## Targets de Build

```
goreleaser targets:
  - dotctl_darwin_arm64      (macOS Apple Silicon)
  - dotctl_darwin_amd64      (macOS Intel)
  - dotctl_darwin_universal   (macOS fat binary via lipo)
  - dotctl_linux_amd64       (Linux x86_64)
  - dotctl_linux_arm64       (Linux aarch64)
```
