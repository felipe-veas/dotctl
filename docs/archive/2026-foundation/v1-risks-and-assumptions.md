# V1 Risks and Assumptions

## Core tradeoffs

- Use shell-out to `git` for operational parity and simplicity.
- Reuse `gh` auth flows instead of implementing custom OAuth.
- Default to symlinks; support `copy` for edge cases.
- Keep tray implementations minimal and platform-appropriate.

## Main risk clusters

- Git conflicts during multi-machine edits.
- Secret leakage through tracked files.
- Cross-platform packaging and desktop integration variance.
- Sync interruption and partial apply states.

## Mitigation direction

- Explicit error messaging.
- Locking + rollback.
- Doctor checks and sensitive-file warnings.
- Structured release and platform validation.
