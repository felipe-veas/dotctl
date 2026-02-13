# Linux Tray App

`linux/tray` contains the Linux system tray application.

## Build

The real implementation is behind the `tray` build tag (`getlantern/systray`):

```bash
CGO_ENABLED=1 go build -tags tray -o bin/dotctl-tray ./linux/tray
```

Or with the helper script:

```bash
./scripts/build-tray-linux.sh
```

## Requirements

- `pkg-config`
- `libayatana-appindicator3-dev` (or `libappindicator3-dev`)

## Autostart

- Desktop entry: `linux/tray/autostart/dotctl-tray.desktop`
- user systemd service: `linux/tray/systemd/dotctl-tray.service`

Quick install:

```bash
./scripts/install-tray-autostart-linux.sh desktop
./scripts/install-tray-autostart-linux.sh systemd
```
