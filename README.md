# dotctl

CLI + tray app (macOS menubar / Linux system tray) para sincronizar dotfiles entre dispositivos usando un repo privado de GitHub como source of truth.

## Status

**En planificación** — ver [docs/plan/](./docs/plan/) para el plan completo de implementación.

## Concepto

```
dotctl init --repo github.com/user/dotfiles --profile macstudio
dotctl sync          # pull → apply symlinks → push
dotctl status        # ver estado actual
dotctl doctor        # diagnóstico completo
dotctl open          # abrir repo en browser
```

- **Declarativo**: `manifest.yaml` define qué se sincroniza y cómo
- **Idempotente**: ejecutar sync N veces produce el mismo resultado
- **Seguro**: backups automáticos, nunca expone tokens, file locking
- **Multi-perfil**: un repo sirve a N máquinas con reglas condicionales

## Plataformas

| OS | Arquitecturas | CLI | Tray App |
|---|---|---|---|
| macOS | arm64 (Apple Silicon) + amd64 (Intel) | Full | Swift NSStatusBar |
| Linux | amd64 + arm64 | Full | Go + libayatana AppIndicator |

## Stack

| Componente | Tecnología |
|---|---|
| CLI | Go 1.22+ (cobra, viper) |
| Tray macOS | Swift (NSStatusBar) |
| Tray Linux | Go (getlantern/systray) |
| Auth | gh CLI (Keychain / libsecret) |
| Config | YAML manifest |

## Documentación

- [Plan de implementación](./docs/plan/)
