# V1 Scope and Goals

## Product summary

`dotctl` was defined as a CLI-first dotfiles synchronization tool with optional tray apps for macOS and Linux.

## In-scope goals

- Initialize a local machine with profile-aware repository configuration.
- Sync repository-managed files to local targets using declarative rules.
- Support idempotent `sync` behavior.
- Provide health checks (`doctor`) and status reporting (`status`).
- Support private GitHub repositories and secure auth flows.

## Out-of-scope goals (for initial version)

- Package manager orchestration.
- Real-time background daemon as default behavior.
- Automatic merge conflict resolution.
- Full secret vault functionality.
- Windows support.

## Initial platform target

- macOS: arm64 and amd64
- Linux: arm64 and amd64
