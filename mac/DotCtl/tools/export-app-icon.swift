import AppKit
import Foundation

@main
struct DotCtlIconExport {
    static func main() {
        guard CommandLine.arguments.count == 2 else {
            fputs("usage: dotctl-icon-export <output-png>\n", stderr)
            exit(2)
        }

        let outputPath = CommandLine.arguments[1]
        let image = DotCtlIconFactory.appIcon(size: 1024)

        guard
            let tiff = image.tiffRepresentation,
            let rep = NSBitmapImageRep(data: tiff),
            let png = rep.representation(using: .png, properties: [:])
        else {
            fputs("error: unable to encode app icon png\n", stderr)
            exit(1)
        }

        do {
            try png.write(to: URL(fileURLWithPath: outputPath))
        } catch {
            fputs("error: \(error.localizedDescription)\n", stderr)
            exit(1)
        }
    }
}
