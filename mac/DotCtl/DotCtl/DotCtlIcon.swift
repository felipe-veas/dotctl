import AppKit

enum DotCtlIconState {
    case synced
    case warning
    case error
    case syncing
}

enum DotCtlIconFactory {
    static func appIcon(size: CGFloat = 128) -> NSImage {
        let canvas = NSSize(width: size, height: size)
        let image = NSImage(size: canvas)

        image.lockFocus()
        defer { image.unlockFocus() }

        let inset = size * 0.06
        let iconRect = NSRect(x: inset, y: inset, width: size - (inset * 2), height: size - (inset * 2))
        let iconPath = NSBezierPath(
            roundedRect: iconRect,
            xRadius: size * 0.22,
            yRadius: size * 0.22
        )

        let gradient = NSGradient(colors: [
            NSColor(calibratedRed: 0.16, green: 0.46, blue: 0.95, alpha: 1.0),
            NSColor(calibratedRed: 0.11, green: 0.29, blue: 0.74, alpha: 1.0),
        ])
        gradient?.draw(in: iconPath, angle: -90)

        NSColor(calibratedWhite: 1.0, alpha: 0.24).setStroke()
        iconPath.lineWidth = max(1.0, size * 0.02)
        iconPath.stroke()

        let glyphInset = size * 0.28
        let glyphRect = NSRect(
            x: glyphInset,
            y: glyphInset,
            width: size - (glyphInset * 2),
            height: size - (glyphInset * 2)
        )
        let glyph = NSBezierPath(
            roundedRect: glyphRect,
            xRadius: size * 0.09,
            yRadius: size * 0.09
        )
        NSColor.white.withAlphaComponent(0.95).setStroke()
        glyph.lineWidth = max(1.6, size * 0.055)
        glyph.stroke()

        let dotSize = size * 0.12
        let dotRect = NSRect(
            x: (size - dotSize) / 2,
            y: (size - dotSize) / 2,
            width: dotSize,
            height: dotSize
        )
        let dot = NSBezierPath(ovalIn: dotRect)
        NSColor.white.withAlphaComponent(0.95).setFill()
        dot.fill()

        return image
    }

    static func statusBarIcon(for state: DotCtlIconState) -> NSImage {
        let size = NSSize(width: 16, height: 16)
        let image = NSImage(size: size)

        image.lockFocus()
        defer { image.unlockFocus() }

        NSColor.black.setStroke()
        NSColor.black.setFill()

        let base = NSBezierPath(
            roundedRect: NSRect(x: 1.5, y: 1.5, width: 13, height: 13),
            xRadius: 3.4,
            yRadius: 3.4
        )
        base.lineWidth = 1.4
        base.stroke()

        let dot = NSBezierPath(ovalIn: NSRect(x: 6.0, y: 6.0, width: 4.0, height: 4.0))
        dot.fill()

        let badgeText: String
        switch state {
        case .synced:
            badgeText = "\u{2713}"
        case .warning:
            badgeText = "!"
        case .error:
            badgeText = "\u{00D7}"
        case .syncing:
            badgeText = "\u{21BB}"
        }

        let attrs: [NSAttributedString.Key: Any] = [
            .font: NSFont.systemFont(ofSize: 7.5, weight: .bold),
            .foregroundColor: NSColor.black,
        ]
        let textSize = (badgeText as NSString).size(withAttributes: attrs)
        let textRect = NSRect(
            x: size.width - textSize.width - 0.7,
            y: 0.2,
            width: textSize.width,
            height: textSize.height
        )
        (badgeText as NSString).draw(in: textRect, withAttributes: attrs)

        image.isTemplate = true
        return image
    }
}
