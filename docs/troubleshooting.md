# Troubleshooting

## `dotctl not initialized`

Run:

```bash
dotctl init --repo <url> --profile <name>
```

## `gh not authenticated`

Run:

```bash
gh auth login --web
```

## `repository has uncommitted changes`

Inside the active repo path, commit or stash local changes, then retry:

```bash
dotctl sync
```

## `decrypt entries require sops or age in PATH`

Install at least one decryption tool and retry.

## `doctor` reports drift

Inspect differences:

```bash
dotctl diff
dotctl diff --details
```

Then reconcile:

```bash
dotctl sync --dry-run
dotctl sync
```

## Manifest validation errors

Common causes:

- missing `target`
- unsupported `mode`
- duplicate `target` entries
- `decrypt: true` used without `mode: copy`
- encrypted source not containing `.enc.` in filename
