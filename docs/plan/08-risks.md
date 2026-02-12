# 8. Riesgos y Decisiones (Tradeoffs)

## R1: Shell-out a `git` vs `go-git` (library pura)

| Opción | Pro | Con |
|---|---|---|
| **Shell-out a `git`** | Sin CGO, debuggeable (`--verbose` muestra comandos), comportamiento idéntico en macOS y Linux | Requiere `git` en PATH. Parsing de stdout frágil |
| `go-git` | Sin dependencia de binario externo. API Go nativa | No soporta todas las features de git. Bugs edge-case. Más código |
| `libgit2` (git2go) | API completa | Requiere CGO. Cross-compile doloroso (4 targets) |

**Decisión: Shell-out a `git`.**
- **Rationale**: en máquinas dev, `git` siempre está instalado (macOS y Linux). Es la opción con menos sorpresas. Si `git` falla, el error es el mismo que el usuario vería en terminal. Cross-compile trivial (CGO_ENABLED=0).
- **Mitigación**: wrapper `internal/gitops/` que parsea exit codes (no stdout) para determinar estado. Tests con `git init` en tmpdir, ejecutados en CI matrix (macOS + Linux).

## R2: Shell-out a `gh` para auth vs OAuth propio

| Opción | Pro | Con |
|---|---|---|
| **Shell-out a `gh`** | Reutiliza login existente. Credential store nativo por OS (Keychain/libsecret). Zero code de OAuth | Requiere `gh` instalado |
| OAuth device flow propio | Sin dependencia de `gh` | Reimplementar OAuth, token storage por OS, refresh. Mucho código para poco valor |
| PAT manual | Simple | Inseguro si el usuario lo guarda en texto plano |

**Decisión: Shell-out a `gh`, con fallback implícito a SSH/GCM.**
- **Rationale**: `gh` es estándar en workflows de GitHub. Disponible vía brew (macOS), apt/dnf (Linux). Si no está, SSH keys funcionan sin código adicional.
- **Mitigación**: `dotctl doctor` verifica auth y sugiere instalar `gh` con instrucciones por OS.

## R3: Symlinks vs Copias

| Opción | Pro | Con |
|---|---|---|
| **Symlinks (default)** | Cambios reflejados inmediatamente. Detección de drift trivial. Comportamiento idéntico en macOS y Linux | Algunos programas no siguen symlinks. Rompe si el repo se mueve |
| Copias | Funciona con cualquier programa | Cambios locales no se propagan automáticamente. Drift más complejo |

**Decisión: Symlinks por defecto, con `mode: copy` opcional en manifest.**
- **Rationale**: `os.Symlink` de Go funciona idéntico en macOS y Linux. Symlinks son el estándar en herramientas de dotfiles (stow, yadm).
- **Mitigación**: `mode: copy` para los edge cases.

## R4: Tray App — Swift (macOS) + Go/systray (Linux) vs solución única

| Opción | Pro | Con |
|---|---|---|
| **Swift macOS + Go/systray Linux** | UX nativa por OS. Cada app es simple (~200 líneas) | Dos implementaciones de tray que mantener |
| Go/systray para ambos | Un solo código | CGO en macOS (no necesario). UX inferior en macOS (sin SF Symbols) |
| Electron/Tauri wrapper | Cross-platform UI | Pesado (>100MB). Runtime dependency. Over-engineering para un menú |

**Decisión: Nativo por OS.**
- **Rationale**: la tray app es un "remote control" trivial. Misma interfaz JSON, misma funcionalidad. La diferencia es solo UI layer. Swift en macOS da UX premium. Go/systray en Linux es pragmático y funciona en GNOME/KDE/XFCE.
- **Mitigación**: ambas apps comparten el contrato JSON y la misma lógica de exec+parse. Cambios en la CLI se reflejan automáticamente.

## R5: Conflictos Git

| Escenario | Impacto | Mitigación |
|---|---|---|
| Dos máquinas pushean cambios al mismo file | `git pull --rebase` puede fallar con conflicto | `dotctl` detecta conflicto, muestra mensaje claro, **no intenta resolver automáticamente**. El usuario resuelve en el repo |
| Push rechazado (non-fast-forward) | Sync falla | `dotctl sync` reintenta un pull --rebase. Si falla de nuevo, muestra instrucciones para resolver manualmente |

**Decisión: No auto-merge. Reportar y delegar al usuario.**
- **Rationale**: auto-merge de dotfiles es peligroso. Mejor ser explícito.

## R6: Seguridad de Secretos

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Usuario commitea `.env` o key privada | Media | Alto | `manifest.yaml` → `ignore:` patterns. `doctor` valida. Pre-commit hook sugerido |
| Token de `gh` expuesto en logs | Baja | Alto | Nunca ejecutamos `gh auth token`. Solo `gh auth status`. Logger redacta patterns de tokens |
| Archivo cifrado pusheado sin cifrar | Baja | Alto | Si `decrypt: true`, el source DEBE terminar en `.enc.*`. Validación en parser |

**Decisión: Defense in depth — múltiples capas.** Igual en ambos OS.

## R7: Idempotencia y Estado Inconsistente

| Escenario | Mitigación |
|---|---|
| Sync interrumpido (SIGINT, crash) | Rollback: guardar lista de acciones ejecutadas en esta sesión. En cleanup, revertir |
| Dos `dotctl sync` simultáneos | File lock (`flock`) en lock file. `flock` funciona en macOS y Linux. Segundo proceso falla con error claro |
| Symlink apunta a archivo que ya no existe en repo | `doctor` detecta symlinks rotos. `sync` los recrea si el source vuelve a existir |

## R8: Cross-platform Build Complexity

| Riesgo | Mitigación |
|---|---|
| 4+ targets de build (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64) | `goreleaser` automatiza cross-compilation. CLI es CGO_ENABLED=0 (trivial) |
| Linux tray requiere CGO (libayatana) | Tray Linux es binario separado. Solo se compila en Linux. No bloquea el CLI |
| macOS universal binary (lipo) | `scripts/build-app-macos.sh` automatiza. goreleaser tiene `universal_binaries` built-in |
| Tests deben pasar en ambos OS | CI matrix: `[macos-latest, ubuntu-latest]`. `internal/platform/` aísla diferencias |

## R9: Linux Desktop Environment Fragmentation

| DE | Tray Support | Status |
|---|---|---|
| GNOME 44+ | AppIndicator via extensión | Funciona con `libayatana-appindicator` |
| KDE Plasma | Nativo (StatusNotifierItem) | Funciona |
| XFCE | Nativo (system tray) | Funciona |
| i3/Sway | Requiere `trayer` o `waybar` | Funciona con config adicional |
| GNOME sin extensión | No hay tray | CLI funciona. Tray no se muestra |

**Decisión**: soportar AppIndicator (libayatana). Cubren ~90% de los setups Linux con DE. Para tiling WMs, documentar cómo configurar. La CLI siempre funciona independiente del tray.

## Matriz de Decisiones Resumida

| Decisión | Opción Elegida | Alternativa Principal | Razón Clave |
|---|---|---|---|
| Git operations | Shell-out | go-git | Simplicidad, cross-platform, debuggeabilidad |
| Auth | gh CLI | SSH keys | UX, credential store nativo por OS |
| File linking | Symlinks (default) | Copias | Estándar de facto, idéntico en macOS/Linux |
| Tray macOS | Swift nativo | Go systray | UX nativa premium |
| Tray Linux | Go systray (libayatana) | Python/GTK | Mismo lenguaje que CLI, sin runtime extra |
| Conflicts | No auto-merge | 3-way merge | Seguridad sobre conveniencia |
| Secrets | Ignore + doctor | Vault integration | Simplicidad para MVP |
| Config format | YAML | TOML | Ecosistema Go, familiaridad |
| Cross-compile | goreleaser | Makefile manual | Automatización de 4+ targets |
