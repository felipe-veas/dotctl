import AppKit

@main
final class AppDelegate: NSObject, NSApplicationDelegate {
    private static var sharedStatusBarController: StatusBarController?

    static func main() {
        let app = NSApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        app.run()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.applicationIconImage = DotCtlIconFactory.appIcon(size: 512)
        NSApp.setActivationPolicy(.accessory)
        Self.sharedStatusBarController = StatusBarController()
    }

    func applicationWillTerminate(_ notification: Notification) {
        Self.sharedStatusBarController = nil
    }
}
