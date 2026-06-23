import SwiftUI

@main
struct NidusApp: App {
    @StateObject private var auth = AuthService()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(auth)
                .frame(minWidth: 900, minHeight: 600)
        }
        .windowStyle(.titleBar)
        .windowResizability(.contentSize)
    }
}
