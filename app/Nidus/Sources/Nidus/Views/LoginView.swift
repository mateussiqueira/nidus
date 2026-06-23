import SwiftUI

struct LoginView: View {
    @EnvironmentObject var auth: AuthService
    @State private var email = ""
    @State private var password = ""
    @State private var error = ""
    @State private var loading = false

    var body: some View {
        VStack(spacing: 20) {
            Spacer()
            Image(systemName: "leaf.fill").font(.system(size: 48)).foregroundColor(.green)
            Text("Nidus").font(.largeTitle).bold()

            VStack(spacing: 12) {
                TextField("Email", text: $email)
                    .textFieldStyle(.roundedBorder)
                    .frame(width: 280)
                SecureField("Senha", text: $password)
                    .textFieldStyle(.roundedBorder)
                    .frame(width: 280)

                if !error.isEmpty {
                    Text(error).foregroundColor(.red).font(.caption)
                }

                Button(action: login) {
                    if loading { ProgressView().scaleEffect(0.8) }
                    else { Text("Entrar").frame(width: 280) }
                }
                .buttonStyle(.borderedProminent)
                .disabled(loading)
            }
            Spacer()
        }
        .frame(width: 400, height: 400)
    }

    func login() {
        loading = true; error = ""
        Task {
            do {
                try await auth.login(email: email, password: password)
            } catch {
                await MainActor.run { self.error = error.localizedDescription }
            }
            await MainActor.run { loading = false }
        }
    }
}
