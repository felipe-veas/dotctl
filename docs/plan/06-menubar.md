# 6. Tray App (macOS Menubar + Linux System Tray)

## Estrategia por OS

| OS | Tecnología | Rationale |
|---|---|---|
| **macOS** | Swift nativo (NSStatusBar) | UX nativa, SF Symbols, dark mode, estabilidad |
| **Linux** | Go + `getlantern/systray` (libayatana-appindicator) | Un solo lenguaje, compatible con GNOME/KDE/XFCE |

Ambas apps son "remote controls" del binario `dotctl`. Ejecutan `dotctl <cmd> --json` y parsean stdout. Zero lógica de sync en la UI.

---

## Evaluación de Enfoques macOS

### Enfoque A: Swift nativo (NSStatusBar) + binario Go

| Aspecto | Evaluación |
|---|---|
| Estabilidad | Nativa. No CGO, no wrappers frágiles |
| Look & Feel | 100% nativo macOS. SF Symbols, dark mode gratis |
| Mantenimiento | Swift es el lenguaje oficial de macOS. Docs abundantes |
| Comunicación | Ejecuta binario Go y parsea JSON stdout |
| Empaquetado | `.app` bundle estándar. Se puede firmar y notarizar |
| Complejidad | Requiere Xcode. Dos lenguajes en el proyecto |

### Enfoque B: Go + systray (CGO)

| Aspecto | Evaluación |
|---|---|
| Estabilidad | `getlantern/systray` funciona pero depende de CGO en macOS |
| Look & Feel | Limitado. Menú básico, sin SF Symbols nativos |
| Mantenimiento | Librería con actividad moderada. Breaking changes posibles |
| Comunicación | Directa (todo en Go). Más simple en teoría |
| Empaquetado | Requiere `go build` con CGO_ENABLED=1. Cross-compile difícil |
| Complejidad | Un solo lenguaje pero más limitaciones de UI |

### Decisión macOS: Enfoque A (Swift nativo)

**Rationale**:
1. La menubar es intrínsecamente una feature macOS — usar el SDK nativo es lo correcto.
2. La app Swift es muy simple (~200 líneas). No justifica agregar CGO al proyecto Go.
3. Mejor UX: SF Symbols, animaciones nativas, accesibilidad gratis.
4. El binario Go es la fuente de verdad — la app Swift es solo un "remote control".

### Decisión Linux: Go + systray (libayatana)

**Rationale**:
1. `getlantern/systray` usa libayatana-appindicator que funciona en GNOME, KDE, XFCE, i3 (con trayer).
2. CGO requerido, pero solo para el binario tray — el CLI principal sigue siendo CGO_ENABLED=0.
3. Misma lógica de bridge que Swift (exec `dotctl --json`), mismo contrato JSON.
4. Alternativa era escribir un tray en Python/GTK pero agrega otra runtime dependency.

---

## Diseño Compartido (ambas plataformas)

### Estructura Visual del Menú

```
┌─────────────────────────┐
│  ⟳  dotctl              │  ← Ícono en tray
├─────────────────────────┤
│  ● Status: Synced       │  ← Verde/Amarillo/Rojo
│  Last sync: 2 min ago   │
│  Profile: macstudio     │
├─────────────────────────┤
│  ▶ Sync Now             │  ← Ejecuta: dotctl sync --json
│  ▶ Pull                 │  ← Ejecuta: dotctl pull --json
│  ▶ Push                 │  ← Ejecuta: dotctl push --json
│  ▶ Doctor               │  ← Ejecuta: dotctl doctor --json
├─────────────────────────┤
│  ◎ Open Repo            │  ← Ejecuta: dotctl open
│  ◎ Open Config          │  ← Abre ~/.config/dotctl/ en file manager
├─────────────────────────┤
│  Quit                   │
└─────────────────────────┘
```

### Estados del Ícono

| Estado | macOS (SF Symbol) | Linux (PNG) | Condición |
|---|---|---|---|
| Synced | `checkmark.circle` (verde) | `icon-ok.png` | Último status OK, sin drift |
| Drift | `exclamationmark.triangle` (amarillo) | `icon-warn.png` | Hay drift detectado |
| Error | `xmark.circle` (rojo) | `icon-error.png` | Último sync/pull falló |
| Syncing | `arrow.triangle.2.circlepath` (animado) | `icon-sync.png` | Operación en curso |

### Comunicación Tray ↔ Go

Idéntica en ambas plataformas:
```
Tray App                           dotctl binary
    │                                    │
    │── exec("dotctl status --json") ──→ │
    │                                    │── lee config
    │                                    │── verifica git
    │                                    │── verifica symlinks
    │← JSON en stdout ──────────────────│
    │
    │── parsea JSON
    │── actualiza UI
```

### Contrato JSON (consumido por ambas tray apps)

```json
{
  "profile": "macstudio",
  "os": "darwin",
  "arch": "arm64",
  "repo": {
    "url": "github.com/felipe-veas/dotfiles",
    "status": "clean",
    "last_sync": "2025-01-15T10:30:00Z"
  },
  "symlinks": { "total": 12, "ok": 10, "broken": 1, "drift": 1 },
  "auth": { "method": "gh", "user": "felipe-veas", "ok": true },
  "errors": []
}
```

---

## Implementación macOS (Swift)

### Pseudo-código Swift

```swift
// StatusApp/DotctlBridge.swift
import Foundation

struct DotctlStatus: Codable {
    let profile: String
    let os: String
    let arch: String
    let repo: RepoStatus
    let symlinks: SymlinkStatus
    let auth: AuthStatus
    let errors: [String]
}

struct RepoStatus: Codable {
    let url: String
    let status: String
    let lastSync: String?
}

struct SymlinkStatus: Codable {
    let total: Int
    let ok: Int
    let broken: Int
    let drift: Int
}

struct AuthStatus: Codable {
    let method: String
    let user: String
    let ok: Bool
}

class DotctlBridge {
    private let binaryPath: String

    init() {
        if let bundled = Bundle.main.path(forResource: "dotctl", ofType: nil) {
            self.binaryPath = bundled
        } else {
            self.binaryPath = "/usr/local/bin/dotctl"
        }
    }

    func status() async throws -> DotctlStatus {
        let output = try await run(["status", "--json"])
        return try JSONDecoder().decode(DotctlStatus.self, from: output)
    }

    func sync() async throws -> String {
        let output = try await run(["sync", "--json"])
        return String(data: output, encoding: .utf8) ?? ""
    }

    func openRepo() async throws {
        _ = try await run(["open"])
    }

    private func run(_ args: [String]) async throws -> Data {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        process.arguments = args

        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = FileHandle.nullDevice

        try process.run()
        process.waitUntilExit()

        return pipe.fileHandleForReading.readDataToEndOfFile()
    }
}
```

```swift
// StatusApp/StatusBarController.swift
import AppKit

class StatusBarController {
    private let statusItem: NSStatusItem
    private let bridge = DotctlBridge()

    init() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "checkmark.circle",
                                   accessibilityDescription: "dotctl")
        }

        setupMenu()
        startPolling()
    }

    private func setupMenu() {
        let menu = NSMenu()

        let statusItem = NSMenuItem(title: "Loading...", action: nil, keyEquivalent: "")
        statusItem.tag = 100
        menu.addItem(statusItem)
        menu.addItem(NSMenuItem.separator())

        menu.addItem(NSMenuItem(title: "Sync Now", action: #selector(syncAction), keyEquivalent: "s"))
        menu.addItem(NSMenuItem(title: "Pull", action: #selector(pullAction), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Push", action: #selector(pushAction), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Doctor", action: #selector(doctorAction), keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())

        menu.addItem(NSMenuItem(title: "Open Repo", action: #selector(openRepoAction), keyEquivalent: "o"))
        menu.addItem(NSMenuItem.separator())

        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))

        self.statusItem.menu = menu
    }

    private func startPolling() {
        Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            self?.refreshStatus()
        }
        refreshStatus()
    }

    private func refreshStatus() {
        Task {
            do {
                let status = try await bridge.status()
                await MainActor.run { updateUI(with: status) }
            } catch {
                await MainActor.run { updateUIError(error) }
            }
        }
    }

    private func updateUI(with status: DotctlStatus) {
        let icon: String
        if status.symlinks.broken > 0 || !status.errors.isEmpty {
            icon = "xmark.circle"
        } else if status.symlinks.drift > 0 {
            icon = "exclamationmark.triangle"
        } else {
            icon = "checkmark.circle"
        }

        statusItem.button?.image = NSImage(systemSymbolName: icon,
                                           accessibilityDescription: "dotctl")

        if let item = statusItem.menu?.item(withTag: 100) {
            item.title = "Profile: \(status.profile) — \(status.symlinks.ok)/\(status.symlinks.total) synced"
        }
    }

    @objc private func syncAction() {
        Task { try? await bridge.sync(); refreshStatus() }
    }

    @objc private func openRepoAction() {
        Task { try? await bridge.openRepo() }
    }
}
```

```swift
// StatusApp/AppDelegate.swift
import AppKit

@main
class AppDelegate: NSObject, NSApplicationDelegate {
    var statusBar: StatusBarController?

    func applicationDidFinishLaunching(_ notification: Notification) {
        statusBar = StatusBarController()
    }
}
```

### Empaquetado macOS

```
StatusApp.app/
└── Contents/
    ├── Info.plist
    ├── MacOS/
    │   └── StatusApp          # Binary Swift
    └── Resources/
        ├── dotctl             # Binary Go (universal: arm64+amd64)
        └── Assets.car
```

```bash
#!/bin/bash
# scripts/build-app-macos.sh
set -euo pipefail

# 1. Build universal Go binary
GOOS=darwin GOARCH=arm64 go build -o bin/dotctl-arm64 ./cmd/dotctl
GOOS=darwin GOARCH=amd64 go build -o bin/dotctl-amd64 ./cmd/dotctl
lipo -create bin/dotctl-arm64 bin/dotctl-amd64 -output bin/dotctl
rm bin/dotctl-arm64 bin/dotctl-amd64

# 2. Embed in app Resources
cp bin/dotctl mac/StatusApp/StatusApp/Resources/dotctl

# 3. Build Swift app (universal via Xcode)
cd mac/StatusApp
xcodebuild -scheme StatusApp -configuration Release \
    -archivePath build/StatusApp.xcarchive archive
```

---

## Implementación Linux (Go + systray)

### Código Go del Tray Linux

```go
// linux/tray/main.go
package main

import "github.com/getlantern/systray"

func main() {
    systray.Run(onReady, onExit)
}

func onReady() {
    systray.SetIcon(iconOK)
    systray.SetTitle("dotctl")
    systray.SetTooltip("dotctl - dotfiles sync")

    mStatus := systray.AddMenuItem("Loading...", "Current status")
    mStatus.Disable()

    systray.AddSeparator()

    mSync := systray.AddMenuItem("Sync Now", "Pull + apply + push")
    mPull := systray.AddMenuItem("Pull", "Git pull")
    mPush := systray.AddMenuItem("Push", "Git push")
    mDoctor := systray.AddMenuItem("Doctor", "Run diagnostics")

    systray.AddSeparator()

    mOpenRepo := systray.AddMenuItem("Open Repo", "Open in browser")
    mOpenConfig := systray.AddMenuItem("Open Config", "Open config dir")

    systray.AddSeparator()

    mQuit := systray.AddMenuItem("Quit", "Quit dotctl tray")

    bridge := NewBridge()

    // Poll status every 60s
    go bridge.PollStatus(mStatus, func(status string) {
        mStatus.SetTitle(status)
    }, func(state State) {
        switch state {
        case StateOK:
            systray.SetIcon(iconOK)
        case StateWarn:
            systray.SetIcon(iconWarn)
        case StateError:
            systray.SetIcon(iconError)
        }
    })

    go func() {
        for {
            select {
            case <-mSync.ClickedCh:
                bridge.RunSync()
            case <-mPull.ClickedCh:
                bridge.RunPull()
            case <-mPush.ClickedCh:
                bridge.RunPush()
            case <-mDoctor.ClickedCh:
                bridge.RunDoctor()
            case <-mOpenRepo.ClickedCh:
                bridge.OpenRepo()
            case <-mOpenConfig.ClickedCh:
                bridge.OpenConfig()
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}

func onExit() {}
```

```go
// linux/tray/bridge.go
package main

import (
    "encoding/json"
    "os/exec"
)

type Bridge struct {
    binaryPath string
}

func NewBridge() *Bridge {
    path, err := exec.LookPath("dotctl")
    if err != nil {
        path = "/usr/local/bin/dotctl"
    }
    return &Bridge{binaryPath: path}
}

func (b *Bridge) Status() (*DotctlStatus, error) {
    out, err := exec.Command(b.binaryPath, "status", "--json").Output()
    if err != nil {
        return nil, err
    }
    var status DotctlStatus
    if err := json.Unmarshal(out, &status); err != nil {
        return nil, err
    }
    return &status, nil
}

func (b *Bridge) RunSync()   { exec.Command(b.binaryPath, "sync").Run() }
func (b *Bridge) RunPull()   { exec.Command(b.binaryPath, "pull").Run() }
func (b *Bridge) RunPush()   { exec.Command(b.binaryPath, "push").Run() }
func (b *Bridge) RunDoctor() { exec.Command(b.binaryPath, "doctor").Run() }
func (b *Bridge) OpenRepo()  { exec.Command(b.binaryPath, "open").Run() }

func (b *Bridge) OpenConfig() {
    exec.Command("xdg-open", "$HOME/.config/dotctl").Run()
}
```

### Build Dependencies Linux

```bash
# Ubuntu/Debian
sudo apt install libayatana-appindicator3-dev gcc

# Fedora
sudo dnf install libayatana-appindicator-gtk3-devel gcc

# Arch
sudo pacman -S libayatana-appindicator
```

### .desktop file (Linux autostart)

```ini
# linux/tray/assets/dotctl.desktop
[Desktop Entry]
Type=Application
Name=dotctl Tray
Comment=Dotfiles sync status
Exec=/usr/bin/dotctl-tray
Icon=dotctl
Categories=Utility;
StartupNotify=false
X-GNOME-Autostart-enabled=true
```

Para autostart: copiar a `~/.config/autostart/dotctl.desktop`.

---

## Auto-start por OS

### macOS: LaunchAgent

```xml
<!-- ~/Library/LaunchAgents/com.dotctl.statusapp.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dotctl.statusapp</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/StatusApp.app/Contents/MacOS/StatusApp</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
```

### Linux: XDG Autostart

```bash
# Instalación
cp dotctl.desktop ~/.config/autostart/

# Desinstalación
rm ~/.config/autostart/dotctl.desktop
```

### Linux: systemd user service (alternativa)

```ini
# ~/.config/systemd/user/dotctl-tray.service
[Unit]
Description=dotctl Tray
After=graphical-session.target

[Service]
ExecStart=/usr/bin/dotctl-tray
Restart=on-failure
RestartSec=5

[Install]
WantedBy=graphical-session.target
```

```bash
systemctl --user enable dotctl-tray
systemctl --user start dotctl-tray
```

### Pros/Cons Auto-start

| Método | Pro | Con |
|---|---|---|
| macOS LaunchAgent | Estándar macOS, `launchctl` control | Otro plist que mantener |
| Linux .desktop | Simple, estándar XDG | Depende del DE soportar autostart |
| Linux systemd | Restart automático, logs con journalctl | Más complejo. No todos usan systemd |

**Recomendación**: ambos como opt-in via `dotctl bootstrap`. No activar por defecto.
