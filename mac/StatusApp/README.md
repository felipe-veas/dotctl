# StatusApp (macOS menubar)

`mac/StatusApp` contiene el scaffold Swift para la app de menubar.

## Build

```bash
./scripts/build-app-macos.sh
```

El script:

1. Construye `dotctl` universal (`arm64` + `amd64`)
2. Compila el binario Swift `StatusApp`
3. Ensambla `bin/StatusApp.app` con `dotctl` embebido en `Contents/Resources/dotctl`

## Autostart (opt-in)

- LaunchAgent plist: `mac/StatusApp/LaunchAgents/com.felipeveas.dotctl.statusapp.plist`
- Instalación:

```bash
./scripts/install-launchagent-macos.sh
```
