# Linux Tray App

`linux/tray` contiene la app de system tray para Linux.

## Build

La implementación real está detrás del build tag `tray` (usa `getlantern/systray`):

```bash
CGO_ENABLED=1 go build -tags tray -o bin/dotctl-tray ./linux/tray
```

O usando el script:

```bash
./scripts/build-tray-linux.sh
```

## Requisitos

- `pkg-config`
- `libayatana-appindicator3-dev` (o `libappindicator3-dev`)

## Autostart

- `.desktop`: `linux/tray/autostart/dotctl-tray.desktop`
- systemd user service: `linux/tray/systemd/dotctl-tray.service`

Instalación rápida:

```bash
./scripts/install-tray-autostart-linux.sh desktop
./scripts/install-tray-autostart-linux.sh systemd
```
