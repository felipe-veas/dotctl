# StatusApp (macOS status bar)

`mac/StatusApp` contains the Swift status bar app scaffold.

## Build

```bash
./scripts/build-app-macos.sh
```

The script:

1. Builds a universal `dotctl` binary (`arm64` + `amd64`).
2. Compiles the Swift `StatusApp` binary.
3. Assembles `bin/StatusApp.app` with embedded `dotctl` at `Contents/Resources/dotctl`.

## Autostart (opt-in)

- LaunchAgent plist: `mac/StatusApp/LaunchAgents/com.felipeveas.dotctl.statusapp.plist`
- Install command:

```bash
./scripts/install-launchagent-macos.sh
```
