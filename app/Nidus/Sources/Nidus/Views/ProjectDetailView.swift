import SwiftUI

struct ProjectDetailView: View {
    let project: Project
    @EnvironmentObject var auth: AuthService
    @State private var deployments: [Deployment] = []
    @State private var metrics: Metrics?
    @State private var envVars = ""
    @State private var envDirty = false
    @State private var deploying = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                // Header
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(project.name).font(.title).bold()
                        HStack {
                            StatusBadge(status: project.status)
                            if let fw = project.framework { Text(fw).font(.caption).foregroundColor(.secondary) }
                        }
                    }
                    Spacer()
                    Button(action: deploy) {
                        if deploying { ProgressView().scaleEffect(0.8) }
                        else { Label("Deploy Now", systemImage: "play.fill") }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(deploying)
                }

                // Metrics
                if let m = metrics {
                    LazyVGrid(columns: [.init(.flexible()), .init(.flexible()), .init(.flexible()), .init(.flexible())], spacing: 12) {
                        MetricCard(label: "Status", value: m.status, icon: "circle.fill", color: m.running ? .green : .red)
                        MetricCard(label: "CPU", value: "\(String(format: "%.1f", m.cpu))%", icon: "cpu", color: .blue)
                        MetricCard(label: "RAM", value: "\(String(format: "%.1f", m.memory.percent))%", icon: "memorychip", color: .purple)
                        MetricCard(label: "Uptime", value: m.uptime > 3600 ? "\(String(format: "%.1f", Double(m.uptime)/3600))h" : "\(m.uptime/60)m", icon: "clock", color: .orange)
                    }
                }

                // Env Vars
                VStack(alignment: .leading, spacing: 8) {
                    Text("Environment Variables").font(.headline)
                    TextEditor(text: $envVars)
                        .font(.system(.caption, design: .monospaced))
                        .frame(height: 100)
                        .border(Color.gray.opacity(0.2))
                        .cornerRadius(6)
                    HStack {
                        Spacer()
                        if envDirty {
                            Button("Salvar") { saveEnv() }.buttonStyle(.bordered)
                        }
                    }
                }

                // Deployments
                Text("Deployments").font(.headline)
                if deployments.isEmpty {
                    Text("Nenhum deployment ainda").foregroundColor(.secondary).padding()
                }
                ForEach(deployments) { dep in
                    DeploymentRow(dep: dep)
                }
            }
            .padding()
        }
        .task { await load() }
    }

    func load() async {
        do {
            async let deps: [Deployment] = auth.get("/api/projects/\(project.id)/deployments")
            async let mets: Metrics? = try? await auth.get("/api/projects/\(project.id)/metrics")
            let (d, m) = await (try deps, try? mets)
            await MainActor.run {
                deployments = d; metrics = m
                envVars = project.envVars ?? ""
            }
        } catch {}
    }

    func deploy() {
        deploying = true
        Task {
            do {
                let _: Deployment = try await auth.post("/api/projects/\(project.id)/deploy", body: [:])
                await load()
            } catch {}
            await MainActor.run { deploying = false }
        }
    }

    func saveEnv() {
        Task {
            do {
                let _: Project = try await auth.patch("/api/projects/\(project.id)", body: ["envVars": envVars])
                await MainActor.run { envDirty = false }
            } catch {}
        }
    }
}

struct StatusBadge: View {
    let status: String
    var body: some View {
        Text(status.lowercased())
            .font(.caption).bold()
            .padding(.horizontal, 8).padding(.vertical, 2)
            .background(status == "ACTIVE" ? Color.green.opacity(0.2) : status == "FAILED" ? Color.red.opacity(0.2) : Color.orange.opacity(0.2))
            .foregroundColor(status == "ACTIVE" ? .green : status == "FAILED" ? .red : .orange)
            .cornerRadius(4)
    }
}

struct MetricCard: View {
    let label: String
    let value: String
    let icon: String
    let color: Color
    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon).foregroundColor(color)
            Text(value).font(.title3).bold()
            Text(label).font(.caption).foregroundColor(.secondary)
        }
        .padding()
        .frame(maxWidth: .infinity)
        .background(Color.gray.opacity(0.1))
        .cornerRadius(8)
    }
}

struct DeploymentRow: View {
    let dep: Deployment
    @State private var showLogs = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                StatusBadge(status: dep.status.uppercased())
                if let date = dep.createdAt { Text(date).font(.caption).foregroundColor(.secondary) }
                Spacer()
                if let url = dep.url {
                    Link(dep.url ?? "", destination: URL(string: url)!)
                        .font(.caption)
                }
                Button(showLogs ? "Ocultar" : "Logs") { showLogs.toggle() }
                    .buttonStyle(.borderless)
                    .font(.caption)
            }
            if showLogs, let logs = dep.logs {
                ScrollView(.horizontal) {
                    Text(logs).font(.system(.caption2, design: .monospaced))
                }
                .frame(maxHeight: 150)
                .padding(8)
                .background(Color.black.opacity(0.3))
                .cornerRadius(6)
            }
        }
        .padding(12)
        .background(Color.gray.opacity(0.05))
        .cornerRadius(8)
    }
}
