import Foundation

class AuthService: ObservableObject, @unchecked Sendable {
    @Published var isAuthenticated = false
    @Published var user: User?
    @Published var token: String?

    private let baseURL = "http://2.24.204.31:3001"

    var headers: [String: String] {
        var h = ["Content-Type": "application/json"]
        if let t = token { h["Authorization"] = "Bearer \(t)" }
        return h
    }

    func login(email: String, password: String) async throws {
        let body = ["email": email, "password": password]
        let data = try await request("/api/auth/login", method: "POST", body: body)
        let result = try JSONDecoder().decode(AuthResult.self, from: data)
        await MainActor.run {
            self.token = result.token
            self.user = result.user
            self.isAuthenticated = true
        }
    }

    func fetchMe() async throws {
        let data = try await request("/api/auth/me")
        let user = try JSONDecoder().decode(User.self, from: data)
        await MainActor.run { self.user = user; self.isAuthenticated = true }
    }

    func logout() {
        token = nil
        user = nil
        isAuthenticated = false
    }

    private func request(_ path: String, method: String = "GET", body: Any? = nil) async throws -> Data {
        guard let url = URL(string: "\(baseURL)\(path)") else { throw URLError(.badURL) }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let t = token { req.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization") }
        if let b = body { req.httpBody = try JSONSerialization.data(withJSONObject: b) }
        let (data, res) = try await URLSession.shared.data(for: req)
        guard let http = res as? HTTPURLResponse, (200...299).contains(http.statusCode) else {
            let msg = try? JSONDecoder().decode(ErrorResponse.self, from: data)
            throw NidusError.server(msg?.message ?? "Erro desconhecido")
        }
        return data
    }

    func get<T: Decodable>(_ path: String) async throws -> T {
        let data = try await request(path)
        return try JSONDecoder().decode(T.self, from: data)
    }

    func post<T: Decodable>(_ path: String, body: Any) async throws -> T {
        let data = try await request(path, method: "POST", body: body)
        return try JSONDecoder().decode(T.self, from: data)
    }

    func patch<T: Decodable>(_ path: String, body: Any) async throws -> T {
        let data = try await request(path, method: "PATCH", body: body)
        return try JSONDecoder().decode(T.self, from: data)
    }
}

struct User: Codable {
    let id: String
    let name: String
    let email: String
    let avatar: String?
}

struct AuthResult: Codable {
    let token: String
    let user: User
}

struct ErrorResponse: Codable {
    let message: String
}

struct Project: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let slug: String
    let framework: String?
    let status: String
    let domain: String?
    let repoUrl: String?
    let envVars: String?
    let createdAt: String?

    func hash(into hasher: inout Hasher) { hasher.combine(id) }
    static func == (lhs: Project, rhs: Project) -> Bool { lhs.id == rhs.id }

    enum CodingKeys: String, CodingKey {
        case id, name, slug, framework, status, domain
        case repoUrl = "repo_url"
        case envVars = "env_vars"
        case createdAt = "created_at"
    }
}

struct Deployment: Codable, Identifiable, Sendable {
    let id: String
    let status: String
    let url: String?
    let logs: String?
    let createdAt: String?
    let finishedAt: String?

    enum CodingKeys: String, CodingKey {
        case id, status, url, logs
        case createdAt = "created_at"
        case finishedAt = "finished_at"
    }
}

struct Metrics: Codable, Sendable {
    let status: String
    let running: Bool
    let uptime: Int
    let cpu: Double
    let memory: MemoryInfo
    let restartCount: Int
}

struct MemoryInfo: Codable, Sendable {
    let usage: String
    let limit: String
    let percent: Double
}

enum NidusError: LocalizedError {
    case server(String)
    var errorDescription: String? {
        switch self { case .server(let m): return m }
    }
}
