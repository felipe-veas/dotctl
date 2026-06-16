# Troubleshooting

## `dotctl not initialized`

Run:

```bash
dotctl init --repo <url>
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
- removed fields from older manifests such as `decrypt`

## Restore from backup

List available snapshots and preview the restore first:

```bash
dotctl backups list
dotctl backups restore <snapshot> --dry-run
dotctl backups restore <snapshot> --target ~/.zshrc --dry-run
```

If the plan is correct, restore with:

```bash
dotctl backups restore <snapshot> --target ~/.zshrc --force
dotctl backups restore <snapshot> --force
```

`--target` matches the backed-up target path exactly. Repeat it to restore multiple entries. Directory targets restore the directory entry when the snapshot has one. Restore only applies targets under your home directory.
