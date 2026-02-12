import AppKit
import Foundation
import UserNotifications

final class StatusBarController: NSObject, UNUserNotificationCenterDelegate {
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
    private let notificationCenter = UNUserNotificationCenter.current()
    private var notificationsAuthorized = false

    override init() {
        do {
            bridge = try DotctlBridge()
        } catch {
            fatalError("Unable to initialize DotctlBridge: \(error.localizedDescription)")
        }

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        super.init()

        configureNotifications()
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

    private func configureNotifications() {
        notificationCenter.delegate = self
        notificationCenter.requestAuthorization(options: [.alert, .sound]) { [weak self] granted, _ in
            self?.notificationsAuthorized = granted
        }
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

    private func runAction(actionName: String, statusText: String, refreshAfter: Bool, notify: Bool, _ action: @escaping () throws -> Void) {
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
                    if notify {
                        self.sendNotification(title: "dotctl \(actionName) failed", body: self.shortLine(actionError.localizedDescription, maxLength: 140))
                    }
                    return
                }
                if notify {
                    self.sendNotification(title: "dotctl \(actionName)", body: "completed successfully")
                }
                if refreshAfter {
                    self.refreshStatus()
                }
            }
        }
    }

    @objc private func syncAction() {
        runAction(actionName: "sync", statusText: "syncing", refreshAfter: true, notify: true) {
            try self.bridge.sync()
        }
    }

    @objc private func pullAction() {
        runAction(actionName: "pull", statusText: "pulling", refreshAfter: true, notify: true) {
            try self.bridge.pull()
        }
    }

    @objc private func pushAction() {
        runAction(actionName: "push", statusText: "pushing", refreshAfter: true, notify: true) {
            try self.bridge.push()
        }
    }

    @objc private func doctorAction() {
        runAction(actionName: "doctor", statusText: "running doctor", refreshAfter: true, notify: true) {
            try self.bridge.doctor()
        }
    }

    @objc private func openRepoAction() {
        runAction(actionName: "open repo", statusText: "opening repo", refreshAfter: false, notify: false) {
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

    private func sendNotification(title: String, body: String) {
        guard notificationsAuthorized else { return }

        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default

        let request = UNNotificationRequest(
            identifier: UUID().uuidString,
            content: content,
            trigger: nil
        )
        notificationCenter.add(request, withCompletionHandler: nil)
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) -> UNNotificationPresentationOptions {
        return [.banner, .list, .sound]
    }
}
