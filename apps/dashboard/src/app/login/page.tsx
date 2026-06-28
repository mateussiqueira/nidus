"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { api, setToken, isAuthenticated, clearToken } from "@/lib/api"

export default function LoginPage() {
  const router = useRouter()
  const [mode, setMode] = useState<"login" | "register">("login")
  const [email, setEmail] = useState("")
  const [name, setName] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isAuthenticated()) {
      router.push("/dashboard")
      return
    }
    // Check for OAuth token in URL
    const params = new URLSearchParams(window.location.search)
    const token = params.get("token")
    if (token) {
      setToken(token)
      window.history.replaceState({}, "", "/login")
      router.push("/dashboard")
    }
  }, [router])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const data = mode === "login"
        ? await api.auth.login(email, password)
        : await api.auth.register(email, name, password)
      setToken(data.token)
      router.push("/dashboard")
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001"

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-8">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold tracking-tight">Nidus</h1>
          <p className="mt-2 text-sm text-zinc-400">Sua PaaS pessoal</p>
        </div>

        <div className="card">
          <div className="mb-6 flex gap-1 rounded-lg bg-zinc-900 p-1">
            <button onClick={() => setMode("login")} className={`flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition ${mode === "login" ? "bg-zinc-700 text-white" : "text-zinc-400 hover:text-white"}`}>Entrar</button>
            <button onClick={() => setMode("register")} className={`flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition ${mode === "register" ? "bg-zinc-700 text-white" : "text-zinc-400 hover:text-white"}`}>Cadastrar</button>
          </div>

          <a
            href={`${API_URL}/api/auth/github/login`}
            className="btn flex w-full items-center justify-center gap-2 bg-zinc-800 text-white hover:bg-zinc-700 mb-4"
          >
            <svg viewBox="0 0 24 24" className="w-5 h-5 fill-current"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.385-1.335-1.755-1.335-1.755-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 21.795 24 17.295 24 12 24 5.37 18.63 0 12 0"/></svg>
            Entrar com GitHub
          </a>

          <div className="relative mb-4">
            <div className="absolute inset-0 flex items-center"><span className="w-full border-t border-zinc-800" /></div>
            <div className="relative flex justify-center text-xs"><span className="bg-card px-2 text-zinc-500">ou</span></div>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-zinc-400">Email</label>
              <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="seu@email.com" required />
            </div>

            {mode === "register" && (
              <div>
                <label className="mb-1 block text-sm font-medium text-zinc-400">Nome</label>
                <input className="input" type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="Seu nome" required />
              </div>
            )}

            <div>
              <label className="mb-1 block text-sm font-medium text-zinc-400">Senha</label>
              <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••" required />
            </div>

            {error && <p className="text-sm text-red-400">{error}</p>}

            <button type="submit" disabled={loading} className="btn btn-primary w-full">
              {loading ? "Aguarde..." : mode === "login" ? "Entrar" : "Criar conta"}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
