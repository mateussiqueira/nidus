import SwiftUI
import UniformTypeIdentifiers

struct ChatView: View {
    @StateObject private var chat = ChatService()
    @State private var input = ""
    @State private var selectedModel = ChatService.models[0].id
    @State private var attachments: [ChatAttachment] = []
    @State private var showingImagePicker = false
    @State private var showSettings = false
    @State private var apiKey = UserDefaults.standard.string(forKey: "openrouter_key") ?? ""

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Picker("Modelo", selection: $selectedModel) {
                    ForEach(ChatService.models, id: \.id) { model in
                        Text(model.name).tag(model.id)
                    }
                }
                .pickerStyle(.menu)
                .frame(width: 200)

                Spacer()

                if !chat.hasApiKey {
                    Button("🔑 Configurar API Key") { showSettings = true }
                        .buttonStyle(.borderedProminent)
                        .controlSize(.small)
                }

                Button(action: { chat.messages.removeAll() }) {
                    Image(systemName: "trash").foregroundColor(.red)
                }
                .buttonStyle(.borderless)
                .help("Limpar conversa")
            }
            .padding(12)
            .background(Color(NSColor.windowBackgroundColor))

            Divider()

            // Messages
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 12) {
                        ForEach(chat.messages) { msg in
                            MessageBubble(message: msg)
                                .id(msg.id)
                        }

                        if chat.isStreaming {
                            HStack {
                                DotLoader()
                                Spacer()
                            }
                            .padding(.horizontal)
                        }

                        if let error = chat.error {
                            HStack {
                                Text(error).foregroundColor(.red).font(.caption)
                                Spacer()
                            }
                            .padding(.horizontal)
                        }
                    }
                    .padding()
                }
                .onChange(of: chat.messages.count) {
                    if let last = chat.messages.last { proxy.scrollTo(last.id, anchor: .bottom) }
                }
            }

            Divider()

            // Attachments preview
            if !attachments.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(attachments) { att in
                            AttachmentPreview(attachment: att) {
                                attachments.removeAll { $0.id == att.id }
                            }
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                }
                .background(Color(NSColor.controlBackgroundColor))
            }

            // Input
            HStack(spacing: 8) {
                Button(action: { showingImagePicker = true }) {
                    Image(systemName: "photo").font(.system(size: 16))
                }
                .buttonStyle(.borderless)
                .help("Anexar imagem")

                Button(action: pickAudio) {
                    Image(systemName: "mic").font(.system(size: 16))
                }
                .buttonStyle(.borderless)
                .help("Anexar áudio")

                TextField("Pergunte ou peça um deploy...", text: $input)
                    .textFieldStyle(.plain)
                    .padding(8)
                    .background(Color(NSColor.controlBackgroundColor))
                    .cornerRadius(8)
                    .onSubmit(submit)

                Button(action: submit) {
                    if chat.isStreaming {
                        ProgressView().scaleEffect(0.8)
                    } else {
                        Image(systemName: "arrow.up.circle.fill")
                            .font(.system(size: 24))
                    }
                }
                .buttonStyle(.borderless)
                .disabled(input.trimmingCharacters(in: .whitespaces).isEmpty || chat.isStreaming)
            }
            .padding(12)
        }
        .fileImporter(isPresented: $showingImagePicker, allowedContentTypes: [.image, .png, .jpeg]) { result in
            if case .success(let url) = result {
                if let data = try? Data(contentsOf: url) {
                    attachments.append(ChatAttachment(data: data, mimeType: url.mimeType, thumbnail: NSImage(data: data)))
                }
            }
        }
        .sheet(isPresented: $showSettings) {
            SettingsView(apiKey: $apiKey)
        }
    }

    func submit() {
        let text = input.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }
        chat.send(text: text, images: attachments, model: selectedModel)
        input = ""
        attachments.removeAll()
    }

    func pickAudio() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.mp3, .wav, .aiff]
        panel.begin { result in
            if result == .OK, let url = panel.url, let data = try? Data(contentsOf: url) {
                attachments.append(ChatAttachment(data: data, mimeType: url.mimeType, thumbnail: nil))
            }
        }
    }
}

struct MessageBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack(alignment: .top, spacing: 8) {
            if message.role == "user" {
                Spacer(minLength: 60)
            } else {
                Image(systemName: "leaf.fill")
                    .foregroundColor(.green)
                    .font(.system(size: 16))
                    .padding(.top, 4)
            }

            VStack(alignment: message.role == "user" ? .trailing : .leading, spacing: 4) {
                ForEach(Array(message.content.enumerated()), id: \.offset) { _, part in
                    switch part {
                    case .text(let t):
                        Text(t)
                            .textSelection(.enabled)
                            .font(.system(size: 14))
                            .foregroundColor(message.role == "user" ? .white : .primary)
                    case .image(let url):
                        AsyncImage(url: URL(string: url)) { phase in
                            if let img = phase.image {
                                img.resizable().scaledToFit().frame(maxHeight: 200).cornerRadius(8)
                            } else {
                                ProgressView()
                            }
                        }
                    case .audio(let data, _):
                        HStack {
                            Image(systemName: "waveform").foregroundColor(.blue)
                            Text("Áudio (\(data.count / 1024) KB)").font(.caption)
                        }
                        .padding(8)
                        .background(Color.gray.opacity(0.1))
                        .cornerRadius(8)
                    }
                }
            }
            .padding(12)
            .background(message.role == "user" ? Color.blue : Color(NSColor.controlBackgroundColor))
            .foregroundColor(message.role == "user" ? .white : .primary)
            .cornerRadius(16)

            if message.role == "assistant" {
                Spacer(minLength: 60)
            }
        }
    }
}

struct AttachmentPreview: View {
    let attachment: ChatAttachment
    let onRemove: () -> Void

    var body: some View {
        ZStack(alignment: .topTrailing) {
            if let thumb = attachment.thumbnail {
                Image(nsImage: thumb)
                    .resizable()
                    .scaledToFill()
                    .frame(width: 60, height: 60)
                    .cornerRadius(8)
            } else {
                VStack {
                    Image(systemName: "waveform").foregroundColor(.blue)
                    Text("Áudio").font(.caption2)
                }
                .frame(width: 60, height: 60)
                .background(Color.gray.opacity(0.1))
                .cornerRadius(8)
            }

            Button(action: onRemove) {
                Image(systemName: "xmark.circle.fill")
                    .foregroundColor(.red)
                    .background(Circle().fill(.white))
            }
            .buttonStyle(.borderless)
            .offset(x: 6, y: -6)
        }
    }
}

struct DotLoader: View {
    @State private var animate = false
    var body: some View {
        HStack(spacing: 4) {
            ForEach(0..<3) { i in
                Circle()
                    .fill(Color.gray)
                    .frame(width: 6, height: 6)
                    .scaleEffect(animate ? 1.0 : 0.5)
                    .animation(.easeInOut(duration: 0.6).repeatForever().delay(Double(i) * 0.2), value: animate)
            }
        }
        .onAppear { animate = true }
    }
}

struct SettingsView: View {
    @Binding var apiKey: String
    @Environment(\.dismiss) var dismiss

    var body: some View {
        VStack(spacing: 16) {
            Text("Configuração do OpenRouter").font(.headline)
            Text("Insira sua chave de API do OpenRouter para usar o chat com IA.")
                .font(.caption).foregroundColor(.secondary)

            SecureField("sk-or-v1-...", text: $apiKey)
                .textFieldStyle(.roundedBorder)

            HStack {
                Button("Cancelar") { dismiss() }
                Button("Salvar") {
                    UserDefaults.standard.set(apiKey, forKey: "openrouter_key")
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding()
        .frame(width: 400)
    }
}

extension URL {
    var mimeType: String {
        if pathExtension == "png" { return "image/png" }
        if ["jpg", "jpeg"].contains(pathExtension) { return "image/jpeg" }
        if pathExtension == "mp3" { return "audio/mpeg" }
        if pathExtension == "wav" { return "audio/wav" }
        if pathExtension == "m4a" { return "audio/mp4" }
        return "application/octet-stream"
    }
}
