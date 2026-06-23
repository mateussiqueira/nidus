// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "Nidus",
    platforms: [.macOS(.v14)],
    targets: [
        .executableTarget(
            name: "Nidus",
            path: "Sources/Nidus"
        )
    ]
)
