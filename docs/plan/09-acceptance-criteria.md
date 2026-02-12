# 9. Criterios de Aceptación — MVP Checklist

El MVP se considera completo cuando **todos** los items de esta checklist están verificados **en macOS y Linux**.

## Cross-Platform

- [ ] CLI compila y funciona en: macOS arm64, macOS amd64, Linux amd64, Linux arm64
- [ ] Tests pasan en CI matrix: `[macos-latest, ubuntu-latest]`
- [ ] `dotctl version` muestra OS/arch correctamente en cada plataforma
- [ ] `XDG_CONFIG_HOME` respetado en Linux
- [ ] Paths por defecto correctos en cada OS (ver [01-architecture.md](./01-architecture.md))

## Core CLI

- [ ] `dotctl init --repo <url> --profile <name>` clona el repo y crea config local
- [ ] `dotctl init` con repo SSH funciona sin `gh` (macOS y Linux)
- [ ] `dotctl status` muestra perfil, OS/arch, estado del repo, symlinks, auth
- [ ] `dotctl status --json` retorna JSON válido parseable por la tray app
- [ ] `dotctl sync` ejecuta: pull → apply manifest → push (idempotente)
- [ ] `dotctl sync --dry-run` muestra plan sin modificar filesystem
- [ ] `dotctl sync` ejecutado dos veces seguidas no produce cambios en la segunda
- [ ] `dotctl pull` ejecuta `git pull --rebase` y reporta resultado
- [ ] `dotctl push` hace stage + commit + push con mensaje auto-generado
- [ ] `dotctl push --message "..."` permite mensaje custom
- [ ] `dotctl doctor` verifica: git, gh, repo, manifest, symlinks, auth, OS/arch
- [ ] `dotctl open` abre URL del repo en browser:
  - [ ] macOS: usa `open`
  - [ ] Linux: usa `xdg-open`
- [ ] `dotctl bootstrap` ejecuta hooks de bootstrap del manifest
- [ ] `dotctl version` muestra versión + OS/arch del binario

## Flags Globales

- [ ] `--json` funciona en: status, sync, pull, push, doctor
- [ ] `--dry-run` funciona en: sync, push
- [ ] `--verbose` muestra comandos git ejecutados y paths completos
- [ ] `--profile` override funciona en todos los comandos
- [ ] `--force` omite confirmaciones en sync/push

## Manifest

- [ ] `manifest.yaml` se parsea correctamente con todos los campos documentados
- [ ] Condiciones `when.os: darwin` filtran correctamente en macOS
- [ ] Condiciones `when.os: linux` filtran correctamente en Linux
- [ ] Condiciones `when.profile` filtran por perfil activo
- [ ] Variables `{{ .var }}` se resuelven en targets
- [ ] `mode: symlink` crea symlinks (default) — funciona en macOS y Linux
- [ ] `mode: copy` copia archivos — funciona en macOS y Linux
- [ ] `ignore:` patterns previenen sincronización de archivos sensibles
- [ ] Hooks `pre_sync` y `post_sync` se ejecutan en orden
- [ ] Hooks `bootstrap` se ejecutan con `dotctl bootstrap`
- [ ] Hooks con `when.os` se filtran correctamente
- [ ] Manifest inválido produce error claro con línea/columna

## Symlinks & Backup

- [ ] Symlink a archivo funciona (macOS + Linux)
- [ ] Symlink a directorio funciona (macOS + Linux)
- [ ] Si target existe como archivo regular → backup + crear symlink
- [ ] Si target existe como directorio regular → backup + crear symlink
- [ ] Si target ya es symlink correcto → no-op
- [ ] Si target es symlink incorrecto (apunta a otro lugar) → backup + recrear
- [ ] Backups se guardan en `~/.config/dotctl/backups/<timestamp>/`
- [ ] `--dry-run` no crea backups ni symlinks

## Auth & Git

- [ ] `gh` CLI detectado y auth verificada (macOS + Linux)
- [ ] Si `gh` no está instalado → error claro con instrucción de instalación por OS:
  - [ ] macOS: sugiere `brew install gh`
  - [ ] Linux: sugiere link a docs de instalación
- [ ] Si `gh` no está autenticado → error claro con instrucción de login
- [ ] URL SSH funciona sin `gh` (macOS + Linux)
- [ ] `git pull --rebase` en repo limpio funciona
- [ ] `git pull --rebase` con cambios locales → error claro (no auto-stash)
- [ ] `git push` con conflicto → error claro con instrucciones
- [ ] Tokens nunca aparecen en stdout, stderr, ni logs

## Seguridad

- [ ] Archivos que matchean `ignore:` patterns no se sincronizan
- [ ] `doctor` advierte si archivos potencialmente sensibles están trackeados
- [ ] Log file no contiene tokens (redacción de patterns `gh[ps]_*`)
- [ ] No se ejecutan comandos arbitrarios excepto hooks definidos en manifest

## Hardening

- [ ] File lock previene ejecución concurrente de sync (macOS + Linux)
- [ ] Si sync falla a mitad → rollback de symlinks creados en esta sesión
- [ ] Logs se escriben a location correcta por OS:
  - [ ] macOS: `~/.config/dotctl/dotctl.log`
  - [ ] Linux: `$XDG_STATE_HOME/dotctl/dotctl.log`
- [ ] `--verbose` es útil para debugging
- [ ] Exit codes documentados y consistentes

## Tray App macOS

- [ ] App aparece como ícono en menubar (sin ícono en Dock)
- [ ] Menú muestra: estado actual, última sync, perfil
- [ ] Ícono refleja estado: verde (ok), amarillo (drift), rojo (error)
- [ ] "Sync Now" ejecuta sync y actualiza estado
- [ ] "Pull" ejecuta pull
- [ ] "Push" ejecuta push
- [ ] "Doctor" ejecuta doctor
- [ ] "Open Repo" abre el repo en browser (vía `open`)
- [ ] "Open Config" abre directorio de config en Finder
- [ ] "Quit" cierra la app
- [ ] Polling automático cada 60s actualiza estado
- [ ] Si el binario `dotctl` no se encuentra → error claro en menú
- [ ] Universal binary (arm64 + amd64) embebido en .app bundle

## Tray App Linux

- [ ] App aparece en system tray (GNOME con AppIndicator / KDE / XFCE)
- [ ] Menú muestra: estado actual, última sync, perfil
- [ ] Ícono refleja estado (PNG icons)
- [ ] "Sync Now" ejecuta sync y actualiza estado
- [ ] "Pull" ejecuta pull
- [ ] "Push" ejecuta push
- [ ] "Doctor" ejecuta doctor
- [ ] "Open Repo" abre el repo en browser (vía `xdg-open`)
- [ ] "Open Config" abre directorio de config en file manager
- [ ] "Quit" cierra la app
- [ ] Polling automático cada 60s actualiza estado
- [ ] `.desktop` file incluido para autostart

## Build & Distribution

- [ ] `goreleaser --snapshot` genera binarios para: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64
- [ ] macOS universal binary (lipo) funciona en Apple Silicon e Intel
- [ ] Linux tray app compila con CGO (libayatana-appindicator)
- [ ] CI matrix verde: macOS-latest + ubuntu-latest

## Calidad

- [ ] Tests unitarios para: manifest parser, condition resolver, linker, config, auth, platform
- [ ] Tests corren en CI matrix (macOS + Linux)
- [ ] `golangci-lint` pasa sin errores
- [ ] `go vet` pasa sin errores
- [ ] README con: instalación por OS, quickstart, comandos, configuración

## Definición de "Done"

El MVP está listo cuando:
1. Un usuario puede ejecutar `dotctl init` + `dotctl sync` en una máquina nueva (**macOS o Linux**) y tener sus dotfiles aplicados
2. Reglas `when.os` filtran correctamente — un mismo manifest sirve para ambos OS
3. La tray app muestra estado correcto y permite ejecutar acciones (**en ambos OS**)
4. Un segundo `dotctl sync` no produce cambios (idempotente)
5. `dotctl doctor` reporta sistema healthy
6. Ningún secreto se expone en ningún momento del flujo
7. Binarios disponibles para las 4 combinaciones OS/arch
