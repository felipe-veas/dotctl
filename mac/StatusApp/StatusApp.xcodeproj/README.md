# Xcode Project Scaffold

Este directorio existe para mantener la estructura esperada del milestone M3.

El build reproducible en este repo se hace con:

```bash
./scripts/build-app-macos.sh
```

Si quieres abrir/editar en Xcode, crea un proyecto `App` llamado `StatusApp` y agrega:

- `StatusApp/AppDelegate.swift`
- `StatusApp/StatusBarController.swift`
- `StatusApp/DotctlBridge.swift`
- `StatusApp/Info.plist`

y copia `mac/StatusApp/bin/dotctl` al bundle en `Contents/Resources/dotctl`.
