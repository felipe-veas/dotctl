# V1 Manifest Design Notes

## Design goals

- Declarative source-to-target mapping.
- Conditional inclusion by OS and profile.
- Safe defaults (`symlink`, backups enabled).

## Key model elements

- `version`
- `vars`
- `files[]`
- `ignore[]`
- `hooks` (`pre_sync`, `post_sync`, `bootstrap`)

## Validation constraints

- `source` and `target` are required.
- `mode` must be `symlink` or `copy`.
- `decrypt: true` requires `mode: copy` and encrypted filename convention.
- Duplicate targets are rejected.
