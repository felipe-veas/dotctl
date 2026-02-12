# Packaging

## Snap (Linux)

- Source config: `packaging/snap/snapcraft.yaml`
- Build command:

```bash
make build-snap
```

This requires `snapcraft` installed locally.

## AUR (Arch Linux)

- PKGBUILD: `packaging/aur/PKGBUILD`
- Regenerate `.SRCINFO` after PKGBUILD changes:

```bash
make build-aur-srcinfo
```

This requires `makepkg` (run on Arch/Manjaro or an Arch container).
