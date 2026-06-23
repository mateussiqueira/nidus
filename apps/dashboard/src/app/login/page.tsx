"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { api, setToken, isAuthenticated } from "@/lib/api"

export default function LoginPage() {
  const router = useRouter()
  const [mode, setMode] = useState<"login" | "register">("login")
  const [email, setEmail] = useState("")
  const [name, setName] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isAuthenticated()) router.push("/dashboard")
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
