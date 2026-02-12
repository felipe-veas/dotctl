import AppKit
import Foundation

final class StatusBarController: NSObject {
    private enum UIState {
        case synced
        case warning
        case error
        case syncing
    }

    private let bridge: DotctlBridge
    private let statusItem: NSStatusItem

    private let statusMenuItem = NSMenuItem(title: "Status: loading...", action: nil, keyEquivalent: "")
    private let lastSyncMenuItem = NSMenuItem(title: "Last sync: --", action: nil, keyEquivalent: "")
    private let profileMenuItem = NSMenuItem(title: "Profile: --", action: nil, keyEquivalent: "")

    private lazy var syncMenuItem = actionItem(title: "Sync Now", selector: #selector(syncAction))
    private lazy var pullMenuItem = actionItem(title: "Pull", selector: #selector(pullAction))
    private lazy var pushMenuItem = actionItem(title: "Push", selector: #selector(pushAction))
    private lazy var doctorMenuItem = actionItem(title: "Doctor", selector: #selector(doctorAction))
    private lazy var openRepoMenuItem = actionItem(title: "Open Repo", selector: #selector(openRepoAction))
    private lazy var openConfigMenuItem = actionItem(title: "Open Config", selector: #selector(openConfigAction))

    private var pollTimer: Timer?
    private var busy = false

    override init() {
        do {
            bridge = try DotctlBridge()
        } catch {
            fatalError("Unable to initialize DotctlBridge: \(error.localizedDescription)")
        }

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        super.init()

        configureMenu()
        setState(.syncing)
        startPolling()
    }

    deinit {
        pollTimer?.invalidate()
    }

    private func configureMenu() {
        let menu = NSMenu()

        statusMenuItem.isEnabled = false
        lastSyncMenuItem.isEnabled = false
        profileMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)
        menu.addItem(lastSyncMenuItem)
        menu.addItem(profileMenuItem)
        menu.addItem(.separator())

        menu.addItem(syncMenuItem)
        menu.addItem(pullMenuItem)
        menu.addItem(pushMenuItem)
        menu.addItem(doctorMenuItem)
        menu.addItem(.separator())
        menu.addItem(openRepoMenuItem)
        menu.addItem(openConfigMenuItem)
        menu.addItem(.separator())
        menu.addItem(actionItem(title: "Quit", selector: #selector(quitAction), key: "q"))

        statusItem.menu = menu
        setState(.syncing)
    }

    private func startPolling() {
        refreshStatus()
        pollTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            self?.refreshStatus()
        }
        if let pollTimer {
            RunLoop.main.add(pollTimer, forMode: .common)
        }
    }

    private func refreshStatus() {
        guard !busy else { return }
        DispatchQueue.global(qos: .utility).async { [weak self] in
            guard let self else { return }
            do {
                let status = try self.bridge.status()
                DispatchQueue.main.async {
                    self.updateUI(with: status)
                }
            } catch {
                DispatchQueue.main.async {
                    self.showError(error)
                }
            }
        }
    }

    private func updateUI(with status: DotctlStatus) {
        let state = resolveState(from: status)
        setState(state)

        let summary: String
        switch state {
        case .synced:
            summary = "synced (\(status.symlinks.ok)/\(status.symlinks.total) ok)"
        case .warning:
            summary = "drift (\(status.symlinks.ok)/\(status.symlinks.total) ok)"
        case .error:
            summary = "error"
        case .syncing:
            summary = "syncing"
        }

        statusMenuItem.title = "Status: \(summary)"
        lastSyncMenuItem.title = "Last sync: \(status.repo.lastSync ?? "never")"
        profileMenuItem.title = "Profile: \(status.profile.isEmpty ? "(unset)" : status.profile)"
    }

    private func resolveState(from status: DotctlStatus) -> UIState {
        if !status.errors.isEmpty || !status.auth.ok || status.symlinks.broken > 0 || status.repo.status == "error" || status.repo.status == "not_git_repo" {
            return .error
        }
        if status.symlinks.drift > 0 || status.repo.status == "dirty" {
            return .warning
        }
        if status.symlinks.total == 0 {
            return .warning
        }
        return .synced
    }

    private func showError(_ error: Error) {
        setState(.error)
        statusMenuItem.title = "Status: error"
        lastSyncMenuItem.title = "Error: \(shortLine(error.localizedDescription, maxLength: 90))"
    }

    private func setState(_ state: UIState) {
        let symbolName: String
        switch state {
        case .synced:
            symbolName = "checkmark.circle"
        case .warning:
            symbolName = "exclamationmark.triangle"
        case .error:
            symbolName = "xmark.circle"
        case .syncing:
            symbolName = "arrow.triangle.2.circlepath"
        }

        statusItem.button?.image = NSImage(systemSymbolName: symbolName, accessibilityDescription: "dotctl")
    }

    private func setBusy(_ value: Bool) {
        busy = value
        let items = [syncMenuItem, pullMenuItem, pushMenuItem, doctorMenuItem, openRepoMenuItem, openConfigMenuItem]
        for item in items {
            item.isEnabled = !value
        }
    }

    private func runAction(statusText: String, refreshAfter: Bool, _ action: @escaping () throws -> Void) {
        guard !busy else { return }
        setBusy(true)
        setState(.syncing)
        statusMenuItem.title = "Status: \(statusText)..."

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self else { return }
            var actionError: Error?
            do {
                try action()
            } catch {
                actionError = error
            }

            DispatchQueue.main.async {
                self.setBusy(false)
                if let actionError {
                    self.showError(actionError)
                    return
                }
                if refreshAfter {
                    self.refreshStatus()
                }
            }
        }
    }

    @objc private func syncAction() {
        runAction(statusText: "syncing", refreshAfter: true) {
            try self.bridge.sync()
        }
    }

    @objc private func pullAction() {
        runAction(statusText: "pulling", refreshAfter: true) {
            try self.bridge.pull()
        }
    }

    @objc private func pushAction() {
        runAction(statusText: "pushing", refreshAfter: true) {
            try self.bridge.push()
        }
    }

    @objc private func doctorAction() {
        runAction(statusText: "running doctor", refreshAfter: true) {
            try self.bridge.doctor()
        }
    }

    @objc private func openRepoAction() {
        runAction(statusText: "opening repo", refreshAfter: false) {
            try self.bridge.openRepo()
        }
    }

    @objc private func openConfigAction() {
        let dir = DotctlBridge.configDir()
        NSWorkspace.shared.open(URL(fileURLWithPath: dir, isDirectory: true))
    }

    @objc private func quitAction() {
        NSApplication.shared.terminate(nil)
    }

    private func actionItem(title: String, selector: Selector, key: String = "") -> NSMenuItem {
        let item = NSMenuItem(title: title, action: selector, keyEquivalent: key)
        item.target = self
        return item
    }

    private func shortLine(_ text: String, maxLength: Int) -> String {
        let normalized = text.replacingOccurrences(of: "\n", with: " ").trimmingCharacters(in: .whitespacesAndNewlines)
        if normalized.count <= maxLength {
            return normalized
        }
        let end = normalized.index(normalized.startIndex, offsetBy: maxLength - 3)
        return String(normalized[..<end]) + "..."
    }
}
