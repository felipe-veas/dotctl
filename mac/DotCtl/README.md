# DotCtl App (macOS status bar)

`mac/DotCtl` contains the Swift status bar app scaffold.

## Build

```bash
./scripts/build-app-macos.sh
```

The script:

1. Builds a universal `dotctl` binary (`arm64` + `amd64`).
2. Compiles the Swift `DotCtl` binary.
3. Assembles `bin/DotCtl.app` with embedded `dotctl` at `Contents/Resources/dotctl`.

## Autostart (opt-in)

- LaunchAgent plist: `mac/DotCtl/LaunchAgents/com.felipeveas.dotctl.app.plist`
- Install command:

```bash
./scripts/install-launchagent-macos.sh
```
