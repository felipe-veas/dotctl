# V1 Acceptance Criteria

## Functional baseline

- Initialization succeeds for profile + repository setup.
- Sync can apply manifest rules predictably.
- Status and doctor provide actionable diagnostics.
- Pull/push flows work with clear failure messages.

## Cross-platform baseline

- Command behavior is consistent on macOS and Linux.
- Platform-specific integrations (`open`, tray, paths) behave correctly.

## Safety baseline

- Backups created before destructive overwrite paths.
- Sensitive tracked file warnings available in diagnostics.
- No token leakage in logs/output.

## Operational baseline

- JSON output available for machine consumption.
- Verbose mode useful for troubleshooting and support.
