import Foundation
import SwiftUI

@MainActor
class ChatService: ObservableObject {
    @Published var messages: [ChatMessage] = []
    @Published var isStreaming = false
    @Published var error: String?

    private var apiKey: String {
        UserDefaults.standard.string(forKey: "openrouter_key") ?? ""
    }

    var hasApiKey: Bool { !apiKey.isEmpty }

    private let baseURL = "https://openrouter.ai/api/v1"

    func send(text: String, images: [ChatAttachment] = [], model: String = "anthropic/claude-sonnet-4") {
        guard hasApiKey else { error = "Configure sua chave do OpenRouter em Settings"; return }

        let userMsg = ChatMessage(role: "user", content: buildContent(text: text, images: images))
        messages.append(userMsg)
        isStreaming = true
        error = nil

        Task {
            do {
                try await streamResponse(model: model)
            } catch {
                self.error = error.localizedDescription
                isStreaming = false
            }
        }
    }

    private func buildContent(text: String, images: [ChatAttachment]) -> [ContentPart] {
        var parts: [ContentPart] = [.text(text)]
        for img in images {
            if let data = img.data {
                let base64 = data.base64EncodedString()
                let mimeType = img.mimeType ?? "image/png"
                parts.append(.image("data:\(mimeType);base64,\(base64)"))
            }
        }
        return parts
    }

    private func streamResponse(model: String) async throws {
        let url = URL(string: "\(baseURL)/chat/completions")!
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
        req.setValue("nidus-macOS", forHTTPHeaderField: "HTTP-Referer")
        req.setValue("Nidus", forHTTPHeaderField: "X-Title")

        let assistantMsg = ChatMessage(role: "assistant", content: [.text("")])
        messages.append(assistantMsg)
        let idx = messages.count - 1

        let body: [String: Any] = [
            "model": model,
            "messages": messages.dropLast().map { $0.asDict },
            "stream": true,
        ]
        req.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (result, response) = try await URLSession.shared.bytes(for: req)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw NidusError.server("Erro na API (\((response as? HTTPURLResponse)?.statusCode ?? 0))")
        }

        for try await line in result.lines {
            if line.hasPrefix("data: ") {
                let data = String(line.dropFirst(6)).trimmingCharacters(in: .whitespacesAndNewlines)
                if data == "[DONE]" { break }
                if let jsonData = data.data(using: .utf8),
                   let chunk = try? JSONSerialization.jsonObject(with: jsonData) as? [String: Any],
                   let choices = chunk["choices"] as? [[String: Any]],
                   let delta = choices.first?["delta"] as? [String: Any],
                   let content = delta["content"] as? String {
                    var current = ""
                    if case .text(let t) = messages[idx].content.last {
                        current = t
                    }
                    messages[idx].content = [.text(current + content)]
                }
            }
        }
        isStreaming = false
    }

    static let models: [(name: String, id: String)] = [
        ("Claude Sonnet 4", "anthropic/claude-sonnet-4"),
        ("Claude Haiku 3.5", "anthropic/claude-3.5-haiku"),
        ("GPT-5", "openai/gpt-5"),
        ("GPT-4o", "openai/gpt-4o"),
        ("Gemini 2.5 Pro", "google/gemini-2.5-pro-exp-03-25"),
        ("DeepSeek V4", "deepseek/deepseek-chat"),
        ("Mistral Large", "mistral/mistral-large-2411"),
        ("Llama 4", "meta-llama/llama-4-maverick"),
    ]
}

struct ChatMessage: Identifiable, Sendable {
    let id = UUID()
    let role: String
    var content: [ContentPart]

    var asDict: [String: Any] {
        var parts: [[String: Any]] = []
        for part in content {
            switch part {
            case .text(let t):
                parts.append(["type": "text", "text": t])
            case .image(let url):
                parts.append(["type": "image_url", "image_url": ["url": url]])
            case .audio(let data, let mimeType):
                parts.append(["type": "input_audio", "data": data, "format": mimeType.contains("mp3") ? "mp3" : "wav"])
            }
        }
        if parts.isEmpty { return ["role": role, "content": ""] }
        return ["role": role, "content": parts]
    }
}

enum ContentPart: Sendable {
    case text(String)
    case image(String)
    case audio(Data, String)
}

struct ChatAttachment: Identifiable {
    let id = UUID()
    let data: Data?
    let mimeType: String?
    let thumbnail: NSImage?
}
