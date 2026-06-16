# Roadmap

This file tracks forward-looking product priorities. Historical implementation planning is archived under [docs/archive](./archive/README.md).

## Active direction

`dotctl` is being refocused as a small CLI-only dotfile manager for one private Git repository and one configuration set.

- Decision record: [ADR 0001: Refocus dotctl as a Simple Dotfile Manager](./adr/0001-refocus-dotctl-as-simple-dotfile-manager.md)
- Implementation plan: [Simplification Roadmap](./simplification-roadmap.md)

## Near-term

- Keep sync explicit, CLI-only, and easy to reason about.
- Stabilize the simplified command set and documentation.
- Backup snapshots now record restore metadata for exact logical entries.
- Backup restore now supports exact-target selection via repeatable `--target`.

## Medium-term

- Continue hardening path validation and sensitive-file guardrails.

## Long-term

- Keep the CLI small and focused.
- Consider removed capabilities only as separate projects or external integrations.
- Maintain clear documentation around what `dotctl` is and is not.
