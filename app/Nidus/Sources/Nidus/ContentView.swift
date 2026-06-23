import SwiftUI

struct ContentView: View {
    @EnvironmentObject var auth: AuthService
    @State private var selectedProject: Project?
    @State private var tab = "chat"

    var body: some View {
        if auth.isAuthenticated {
            NavigationSplitView {
                List(selection: $tab) {
                    Label("Chat IA", systemImage: "message").tag("chat")
                    Label("Projetos", systemImage: "folder").tag("projects")
                }
                .listStyle(.sidebar)
                .frame(minWidth: 180)
            } detail: {
                switch tab {
                case "chat":
                    ChatView()
                case "projects":
                    if let project = selectedProject {
                        ProjectDetailView(project: project)
                    } else {
                        ProjectsListView(selectedProject: $selectedProject)
                    }
                default:
                    EmptyView()
                }
            }
            .toolbar {
                ToolbarItem {
                    Button(action: { tab = "chat" }) {
                        Image(systemName: "message")
                    }
                }
                ToolbarItem {
                    Button(action: { tab = "projects" }) {
                        Image(systemName: "folder")
                    }
                }
                ToolbarItem {
                    Button(action: auth.logout) {
                        Text("Sair").foregroundColor(.red)
                    }
                }
            }
        } else {
            LoginView()
        }
    }
}

struct ProjectsListView: View {
    @EnvironmentObject var auth: AuthService
    @Binding var selectedProject: Project?
    @State private var projects: [Project] = []

    var body: some View {
        List(selection: $selectedProject) {
            ForEach(projects) { project in
                HStack(spacing: 10) {
                    Circle()
                        .fill(project.status == "ACTIVE" ? Color.green : project.status == "FAILED" ? Color.red : Color.orange)
                        .frame(width: 8, height: 8)
                    VStack(alignment: .leading) {
                        Text(project.name).font(.system(size: 13, weight: .medium))
                        if let fw = project.framework {
                            Text(fw).font(.caption).foregroundColor(.secondary)
                        }
                    }
                }
                .tag(project)
            }
        }
        .task {
            do {
                let p: [Project] = try await auth.get("/api/projects")
                await MainActor.run { projects = p }
            } catch {}
        }
    }
}
