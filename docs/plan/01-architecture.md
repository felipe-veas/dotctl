# 1. Arquitectura Propuesta

## Diagrama de Componentes

```
┌─────────────────────────────────────────────────────────────────┐
│                        Usuario / Shell                          │
└──────────────┬──────────────────────┬───────────────────────────┘
               │                      │
               ▼                      ▼
┌──────────────────────┐  ┌──────────────────────────────────────┐
│     dotctl CLI       │  │        Tray App (por OS)             │
│     (Go binary)      │  │                                      │
│                      │  │  macOS: Swift NSStatusBar             │
│  ┌────────────────┐  │  │  Linux: Go + libayatana-appindicator │
│  │ cmd/            │  │  │                                      │
│  │  init           │  │  │  Ejecuta: dotctl --json              │
│  │  status         │  │  │  Parsea: JSON stdout                 │
│  │  sync           │  │  │  Muestra: estado + menú              │
│  │  pull / push    │  │  └──────────────────────────────────────┘
│  │  doctor         │  │             │
│  │  open           │  │             │ (exec binario)
│  │  bootstrap      │  │             ▼
│  └────────────────┘  │  ┌──────────────────────────────────────┐
│                      │  │    dotctl binary                      │
│  ┌────────────────┐  │  │    (mismo binario CLI)                │
│  │ internal/       │  │  └──────────────────────────────────────┘
│  │  manifest/      │──┼──→ Lee y valida manifest.yaml
│  │  linker/        │──┼──→ Crea/verifica symlinks
│  │  gitops/        │──┼──→ git pull/push/status
│  │  auth/          │──┼──→ gh auth token / credential store
│  │  backup/        │──┼──→ Backup antes de overwrite
│  │  profile/       │──┼──→ Filtro por OS/hostname
│  │  doctor/        │──┼──→ Validaciones de salud
│  │  config/        │──┼──→ Config local (~/.config/dotctl/)
│  │  platform/      │──┼──→ Abstracciones OS-specific (open, paths)
│  └────────────────┘  │
└──────────────────────┘
          │
          ▼
┌──────────────────────┐
│  Repo Privado GitHub │
│  (source of truth)   │
│                      │
│  manifest.yaml       │
│  configs/            │
│    zsh/              │
│    git/              │
│    nvim/             │
│    ...               │
└──────────────────────┘
```

## Componentes y Responsabilidades

### CLI (`cmd/`)
- **Responsabilidad**: punto de entrada, parsing de flags, orquestación de comandos.
- **Límite**: NO contiene lógica de negocio. Delega a `internal/`.
- **Output**: texto humano por defecto, JSON con `--json`.

### Manifest Engine (`internal/manifest/`)
- **Responsabilidad**: parsear `manifest.yaml`, resolver condicionales (OS, perfil), generar lista de acciones.
- **Límite**: no ejecuta nada, solo produce un plan de ejecución.

### Linker (`internal/linker/`)
- **Responsabilidad**: crear symlinks, detectar drift (archivo existente != symlink esperado), hacer backup.
- **Límite**: solo opera en filesystem local. No toca Git. Usa `os.Symlink` (funciona idéntico en macOS y Linux).

### GitOps (`internal/gitops/`)
- **Responsabilidad**: `git clone`, `pull --rebase`, `add + commit + push`, detección de dirty state.
- **Límite**: shell-out a `git`. No libgit2. Requiere `git` en PATH.

### Auth (`internal/auth/`)
- **Responsabilidad**: obtener token de GitHub vía `gh auth token`, validar que auth es funcional.
- **Límite**: no almacena tokens — delega a `gh` que usa macOS Keychain / Linux libsecret.

### Backup (`internal/backup/`)
- **Responsabilidad**: copiar archivo existente a `~/.config/dotctl/backups/<timestamp>/` antes de reemplazar.
- **Límite**: no gestiona retención (MVP: backups ilimitados, post-MVP: rotación).

### Profile (`internal/profile/`)
- **Responsabilidad**: determinar perfil activo (flag `--profile` o config local), resolver `runtime.GOOS`, `runtime.GOARCH`, hostname.
- **Límite**: solo lectura de estado del sistema.

### Platform (`internal/platform/`)
- **Responsabilidad**: abstraer diferencias entre macOS y Linux.
- **Funciones**:
  - `OpenURL(url)` → `open` (macOS) / `xdg-open` (Linux)
  - `OpenFileManager(path)` → `open` (Finder) / `xdg-open` (file manager)
  - `ConfigDir()` → `~/.config/dotctl` (ambos OS, respeta `$XDG_CONFIG_HOME` en Linux)
  - `LockFile()` → `flock` (ambos OS)
- **Límite**: solo diferencias de plataforma. No lógica de negocio.

### Config (`internal/config/`)
- **Responsabilidad**: leer/escribir `~/.config/dotctl/config.yaml` (repo URL, perfil activo, paths).
- **Límite**: config local, no se sincroniza con el repo. Respeta `$XDG_CONFIG_HOME`.

### Doctor (`internal/doctor/`)
- **Responsabilidad**: ejecutar checks de salud (auth, git, symlinks, manifest).
- **Límite**: read-only, nunca modifica estado.

### Tray App macOS (Swift)
- **Responsabilidad**: ícono en menubar, mostrar estado, ejecutar acciones invocando el binario.
- **Límite**: no contiene lógica de sync. Es solo UI que llama a `dotctl`.
- **Comunicación**: ejecuta `dotctl status --json`, `dotctl sync`, etc. y parsea stdout.

### Tray App Linux (Go)
- **Responsabilidad**: ícono en system tray (GNOME, KDE, etc.), misma funcionalidad que macOS.
- **Límite**: misma filosofía — es un "remote control" del binario CLI.
- **Comunicación**: idéntica al macOS (exec + JSON stdout).
- **Implementación**: `getlantern/systray` con libayatana-appindicator.

## Flujo de Datos Principal (sync)

```
1. Usuario ejecuta: dotctl sync --profile macstudio
2. CLI resuelve perfil activo
3. manifest/ parsea manifest.yaml → genera Plan{actions: [...]}
4. gitops/ ejecuta git pull --rebase
5. Para cada acción en el plan:
   a. linker/ verifica estado actual del target
   b. Si existe y no es symlink correcto → backup/ crea backup
   c. linker/ crea symlink (o copia si está configurado así)
6. Si hubo cambios locales → gitops/ ejecuta add + commit + push
7. CLI imprime resumen (o JSON si --json)
```

## Decisión: Shell-out vs Libraries

| Operación | Decisión | Rationale |
|---|---|---|
| Git | Shell-out a `git` | Evita CGO (libgit2). `git` siempre está en PATH en máquinas dev. Mismo comportamiento en macOS y Linux |
| Auth | Shell-out a `gh` | Reutiliza auth existente. `gh` usa Keychain (macOS) o libsecret (Linux) automáticamente |
| Symlinks | `os.Symlink` stdlib | Comportamiento idéntico en macOS y Linux. No necesita dependencia externa |
| YAML | `gopkg.in/yaml.v3` | Estándar de facto en Go |
| CLI framework | `cobra` | Estándar para CLIs Go. Autocompletado gratis (bash, zsh, fish) |
| Open URL | `internal/platform/` | `open` (macOS) / `xdg-open` (Linux) |

## Paths por OS

| Path | macOS | Linux |
|---|---|---|
| Config | `~/.config/dotctl/config.yaml` | `$XDG_CONFIG_HOME/dotctl/config.yaml` (default: `~/.config/`) |
| Repo clone | `~/.config/dotctl/repo/` | `$XDG_CONFIG_HOME/dotctl/repo/` |
| Backups | `~/.config/dotctl/backups/` | `$XDG_CONFIG_HOME/dotctl/backups/` |
| Logs | `~/.config/dotctl/dotctl.log` | `$XDG_STATE_HOME/dotctl/dotctl.log` (default: `~/.local/state/`) |
| Lock | `~/.config/dotctl/lock` | `$XDG_RUNTIME_DIR/dotctl.lock` (fallback: `/tmp/`) |

La abstracción vive en `internal/platform/` y respeta las convenciones XDG en Linux.
