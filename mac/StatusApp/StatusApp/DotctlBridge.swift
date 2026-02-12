import Foundation

struct DotctlStatus: Decodable {
    let profile: String
    let os: String
    let arch: String
    let repo: RepoStatus
    let symlinks: SymlinkStatus
    let auth: AuthStatus
    let warnings: [String]?
    let errors: [String]
}

struct RepoStatus: Decodable {
    let url: String
    let status: String
    let branch: String?
    let lastCommit: String?
    let lastSync: String?

    enum CodingKeys: String, CodingKey {
        case url
        case status
        case branch
        case lastCommit = "last_commit"
        case lastSync = "last_sync"
    }
}

struct SymlinkStatus: Decodable {
    let total: Int
    let ok: Int
    let broken: Int
    let drift: Int
}

struct AuthStatus: Decodable {
    let method: String
    let user: String?
    let ok: Bool
}

enum DotctlBridgeError: LocalizedError {
    case binaryNotFound
    case commandFailed(String)
    case invalidOutput(String)

    var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "dotctl binary not found (set DOTCTL_BIN or place dotctl in PATH)."
        case .commandFailed(let message):
            return message
        case .invalidOutput(let message):
            return message
        }
    }
}

final class DotctlBridge {
    private let binaryPath: String
    private let decoder: JSONDecoder

    init() throws {
        self.binaryPath = try DotctlBridge.resolveBinaryPath()
        self.decoder = JSONDecoder()
    }

    func status() throws -> DotctlStatus {
        let output = try run(["status", "--json"])
        do {
            return try decoder.decode(DotctlStatus.self, from: output)
        } catch {
            let raw = String(data: output, encoding: .utf8) ?? "<non-utf8>"
            throw DotctlBridgeError.invalidOutput("Failed to parse status JSON: \(raw)")
        }
    }

    func sync() throws {
        _ = try run(["sync", "--json"])
    }

    func pull() throws {
        _ = try run(["pull", "--json"])
    }

    func push() throws {
        _ = try run(["push", "--json"])
    }

    func doctor() throws {
        _ = try run(["doctor", "--json"])
    }

    func openRepo() throws {
        _ = try run(["open"])
    }

    static func configDir() -> String {
        let env = ProcessInfo.processInfo.environment
        if let xdg = env["XDG_CONFIG_HOME"], !xdg.isEmpty {
            return URL(fileURLWithPath: xdg).appendingPathComponent("dotctl").path
        }

        if let home = env["HOME"], !home.isEmpty {
            return URL(fileURLWithPath: home).appendingPathComponent(".config/dotctl").path
        }

        return NSHomeDirectory().appending("/.config/dotctl")
    }

    private static func resolveBinaryPath() throws -> String {
        let fm = FileManager.default
        let env = ProcessInfo.processInfo.environment

        if let override = env["DOTCTL_BIN"], !override.isEmpty, fm.isExecutableFile(atPath: override) {
            return override
        }

        if let bundled = Bundle.main.path(forResource: "dotctl", ofType: nil), fm.isExecutableFile(atPath: bundled) {
            return bundled
        }

        let pathEntries = (env["PATH"] ?? "").split(separator: ":").map(String.init)
        for entry in pathEntries {
            let candidate = URL(fileURLWithPath: entry).appendingPathComponent("dotctl").path
            if fm.isExecutableFile(atPath: candidate) {
                return candidate
            }
        }

        let fallbacks = [
            "/usr/local/bin/dotctl",
            "/opt/homebrew/bin/dotctl",
            "\(NSHomeDirectory())/.local/bin/dotctl",
        ]
        for candidate in fallbacks where fm.isExecutableFile(atPath: candidate) {
            return candidate
        }

        throw DotctlBridgeError.binaryNotFound
    }

    @discardableResult
    private func run(_ arguments: [String]) throws -> Data {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        process.arguments = arguments

        let stdout = Pipe()
        let stderr = Pipe()
        process.standardOutput = stdout
        process.standardError = stderr

        try process.run()
        process.waitUntilExit()

        let out = stdout.fileHandleForReading.readDataToEndOfFile()
        let err = stderr.fileHandleForReading.readDataToEndOfFile()
        if process.terminationStatus != 0 {
            let stderrText = String(data: err, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let stdoutText = String(data: out, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let details = !stderrText.isEmpty ? stderrText : stdoutText
            throw DotctlBridgeError.commandFailed("dotctl \(arguments.joined(separator: " ")) failed: \(details)")
        }

        return out
    }
}
