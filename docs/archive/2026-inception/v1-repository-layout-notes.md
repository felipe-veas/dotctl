# V1 Repository Layout Notes

## Intended structure

- Clear separation between command layer and internal packages.
- Platform-specific tray implementations under `mac/` and `linux/`.
- Packaging and release support under `packaging/` and scripts.
- Test fixtures under `testdata/`.

## Documentation intent

- User-facing docs separate from implementation planning notes.
- Stable docs for install and usage.
- Historical notes preserved without driving current behavior.
