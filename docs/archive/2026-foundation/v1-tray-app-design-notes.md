# V1 Tray App Design Notes

## Platform choices

- macOS: native Swift status bar app.
- Linux: Go tray app via AppIndicator/systray.

## Shared behavior

- Poll `dotctl status --json` for state.
- Trigger sync/pull/push/doctor from tray actions.
- Keep tray code thin; CLI remains the source of truth.

## Autostart

- macOS LaunchAgent and Linux desktop/systemd options were planned as opt-in.
